// Package proxy provides a reverse proxy helper that implements web.Handler.
package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/headers"
	"github.com/go-sum/foundry/pkg/web/serve"
)

// defaultTransport is the shared, concurrent-safe transport for upstream requests.
// It is intentionally package-level so all Reverse instances without a custom
// client share one connection pool.
var defaultTransport = &http.Transport{
	ResponseHeaderTimeout: 10 * time.Second,
	IdleConnTimeout:       90 * time.Second,
	MaxIdleConnsPerHost:   32,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	MaxIdleConns:          128,
}

// hopByHopHeaders are the static hop-by-hop headers defined by RFC 7230 6.1
// that must never be forwarded between proxy hops.
var hopByHopHeaders = map[string]bool{
	"connection":          true,
	"keep-alive":          true,
	"proxy-authenticate":  true,
	"proxy-authorization": true,
	"te":                  true,
	"trailers":            true,
	"transfer-encoding":   true,
	"upgrade":             true,
}

// strippedClientHeaders lists forwarding headers that are always stripped from
// inbound requests and rebuilt from server-known connection metadata. Clients
// must never be allowed to supply these values because upstream services use
// them for IP-based rate limiting, ACLs, and audit trails.
var strippedClientHeaders = map[string]bool{
	"forwarded":         true,
	"x-forwarded-for":   true,
	"x-forwarded-host":  true,
	"x-forwarded-proto": true,
}

// dynamicHopHeaders returns the set of header names listed in a Connection
// header value. Per RFC 7230 6.1 a proxy must strip not only the static
// hop-by-hop set above but also any header named by the Connection field.
func dynamicHopHeaders(connectionHeader string) map[string]bool {
	if connectionHeader == "" {
		return nil
	}
	out := make(map[string]bool)
	for _, token := range strings.Split(connectionHeader, ",") {
		if name := strings.ToLower(strings.TrimSpace(token)); name != "" {
			out[name] = true
		}
	}
	return out
}

// Options configures Reverse.
type Options struct {
	// Client is the HTTP client used to send upstream requests.
	// If nil, a hardened package-default client with bounded timeouts is used.
	Client *http.Client
	// MaxResponseHeaderBytes limits the size of response headers from the
	// upstream. Defaults to 1 MiB. Only applies when Client is nil (the
	// package default transport is used).
	MaxResponseHeaderBytes int64
	// ModifyRequest is called before the upstream request is sent.
	// Use it to add auth headers, rewrite paths, etc.
	ModifyRequest func(r *http.Request)
	// ModifyResponse is called after the upstream response is received.
	ModifyResponse func(r *http.Response)
	// PathPrefix is the public URL path prefix under which this proxy is
	// mounted (e.g. "/app"). When non-empty, Set-Cookie Path attributes from
	// the upstream are rewritten so that cookies set with absolute paths
	// (e.g. Path=/) remain valid under the prefix (e.g. Path=/app).
	PathPrefix string
	// TrustedProxies lists parsed CIDR ranges of trusted upstream proxies. When
	// the incoming connection's RemoteAddr falls within one of these ranges, the
	// proxy extends the existing X-Forwarded-For and Forwarded chains instead of
	// replacing them. This allows end-user IPs to propagate correctly when this
	// proxy sits behind a trusted load balancer or CDN. Build this slice via
	// serve.ParseTrustedProxies or net.ParseCIDR directly.
	TrustedProxies []*net.IPNet
	// Authority is the canonical public hostname used for outbound forwarding
	// headers and cookie domain rewriting. When set, it replaces c.Request.Host()
	// in X-Forwarded-Host, Forwarded host=, and Set-Cookie Domain rewrites.
	// Callers that validate the inbound Host header via AllowedHosts middleware
	// may leave this empty — the validated request host is used instead.
	Authority string
}

// resolveClient returns the HTTP client to use for upstream requests.
// A new *http.Client value is always returned; when MaxResponseHeaderBytes is
// not set the shared defaultTransport is used so all instances pool connections.
// When MaxResponseHeaderBytes is set a dedicated transport is created so the
// header limit does not affect other proxy instances.
func resolveClient(opts Options) *http.Client {
	if opts.Client != nil {
		return opts.Client
	}
	if opts.MaxResponseHeaderBytes > 0 {
		return &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				ResponseHeaderTimeout:  10 * time.Second,
				IdleConnTimeout:        90 * time.Second,
				MaxIdleConnsPerHost:    32,
				TLSHandshakeTimeout:    10 * time.Second,
				ExpectContinueTimeout:  1 * time.Second,
				MaxIdleConns:           128,
				MaxResponseHeaderBytes: opts.MaxResponseHeaderBytes,
			},
		}
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: defaultTransport,
	}
}

// quoteForwardedNode formats an IP address for use as a Forwarded header
// node identifier per RFC 7239 6. IPv6 addresses are enclosed in square
// brackets and quoted; IPv4 addresses are returned bare.
func quoteForwardedNode(ip string) string {
	if strings.Contains(ip, ":") {
		// IPv6 — must be enclosed in brackets and the whole token quoted.
		return `"[` + ip + `]"`
	}
	return ip
}


func sanitizedXForwardedForChain(raw string) ([]string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}

	parts := strings.Split(raw, ",")
	chain := make([]string, 0, len(parts))
	for _, part := range parts {
		ip, ok := serve.NormalizeProxyIP(part)
		if !ok {
			return nil, false
		}
		chain = append(chain, ip)
	}
	return chain, true
}

func sanitizedForwardedChain(raw string) ([]string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}

	elements, err := headers.ParseForwarded(raw)
	if err != nil || len(elements) == 0 {
		return nil, false
	}

	out := make([]string, 0, len(elements))
	for _, element := range elements {
		ip, ok := serve.NormalizeProxyIP(element.For)
		if !ok {
			return nil, false
		}

		parts := []string{"for=" + quoteForwardedNode(ip)}
		if element.By != "" {
			parts = append(parts, "by="+element.By)
		}
		if element.Host != "" {
			parts = append(parts, "host="+element.Host)
		}
		if element.Proto != "" {
			parts = append(parts, "proto="+element.Proto)
		}
		out = append(out, strings.Join(parts, ";"))
	}
	return out, true
}

// Reverse returns a web.Handler that proxies every request to target.
//
// It:
//   - Copies the incoming method, URL (path + query), headers, and body to a new upstream request.
//   - Strips client-supplied X-Forwarded-For, Forwarded, X-Forwarded-Host, and X-Forwarded-Proto headers.
//   - Rebuilds X-Forwarded-Host and X-Forwarded-Proto from server-resolved state (inbound values are never passed through).
//   - Preserves and extends inbound X-Forwarded-For / Forwarded chains only when the immediate peer is trusted and the inbound chain validates.
//   - Rewrites Set-Cookie Domain to match the incoming request host (public-facing origin).
//   - Returns the upstream status, headers, and body verbatim.
//   - Removes hop-by-hop headers (Connection, Upgrade, Transfer-Encoding, etc.) from both directions.
func Reverse(target *url.URL, opts Options) web.Handler {
	client := resolveClient(opts)
	return func(c *web.Context) (web.Response, error) {
		upstreamURL := buildUpstreamURL(target, c.Request)

		upstreamReq, err := http.NewRequestWithContext(
			c.Context(),
			c.Request.Method,
			upstreamURL.String(),
			c.Request.Body,
		)
		if err != nil {
			return web.Response{}, web.ErrBadGateway(err)
		}

		// Capture forwarding chain from inbound headers before the copy loop strips
		// them. Used below to extend or preserve the chain when the peer is trusted.
		inboundXFF := c.Request.Headers.Get("X-Forwarded-For")
		inboundForwarded := c.Request.Headers.Get("Forwarded")

		// Copy incoming headers, skipping static and dynamic hop-by-hop headers.
		// RFC 7230 6.1: strip any header named in the Connection field.
		dynamicReq := dynamicHopHeaders(c.Request.Headers.Get("Connection"))
		c.Request.Headers.ForEach(func(name string, values []string) {
			lower := strings.ToLower(name)
			if hopByHopHeaders[lower] || dynamicReq[lower] || strippedClientHeaders[lower] {
				return
			}
			for _, v := range values {
				upstreamReq.Header.Add(name, v)
			}
		})

		// Resolve the immediate peer IP from RemoteAddr.
		clientIP, _ := serve.NormalizeProxyIP(c.Request.RemoteAddr())
		trusted := serve.IsTrustedProxy(clientIP, opts.TrustedProxies)

		// resolvedHost is the canonical public hostname for outbound headers.
		// Authority takes precedence when set; otherwise the validated request host.
		resolvedHost := c.Request.Host()
		if opts.Authority != "" {
			resolvedHost = opts.Authority
		}

		// Set X-Forwarded-Host from the resolved public host.
		// Inbound X-Forwarded-Host is never preserved — always rebuild to prevent host-header injection.
		if resolvedHost != "" {
			upstreamReq.Header.Set("X-Forwarded-Host", resolvedHost)
		}

		// Set X-Forwarded-Proto from the server-resolved scheme.
		// Always rebuilt from server-known state — inbound values are never passed through.
		scheme := "http"
		if c.URL() != nil && c.URL().Scheme != "" {
			scheme = c.URL().Scheme
		}
		upstreamReq.Header.Set("X-Forwarded-Proto", scheme)

		// Set X-Forwarded-For: extend the inbound chain when the peer is a trusted
		// proxy (so end-user IPs survive a CDN/LB hop); start fresh otherwise.
		// Client-supplied X-Forwarded-For is stripped in the copy loop above.
		if clientIP != "" {
			chain := []string{clientIP}
			if trusted {
				if forwardedChain, ok := sanitizedXForwardedForChain(inboundXFF); ok {
					chain = append(forwardedChain, clientIP)
				}
			}
			upstreamReq.Header.Set("X-Forwarded-For", strings.Join(chain, ", "))
		}

		// Emit RFC 7239 Forwarded header. Extend the inbound chain when trusted.
		// Client-supplied Forwarded is stripped in the copy loop above.
		if clientIP != "" {
			forwardedProto := "http"
			if c.URL() != nil && c.URL().Scheme != "" {
				forwardedProto = c.URL().Scheme
			}
			forwardedHost := resolvedHost
			var thisHop string
			if forwardedHost != "" {
				thisHop = fmt.Sprintf("for=%s;host=%s;proto=%s",
					quoteForwardedNode(clientIP), forwardedHost, forwardedProto)
			} else {
				thisHop = fmt.Sprintf("for=%s;proto=%s",
					quoteForwardedNode(clientIP), forwardedProto)
			}
			chain := []string{thisHop}
			if trusted {
				if forwardedChain, ok := sanitizedForwardedChain(inboundForwarded); ok {
					chain = append(forwardedChain, thisHop)
				}
			}
			upstreamReq.Header.Set("Forwarded", strings.Join(chain, ", "))
		}

		if opts.ModifyRequest != nil {
			opts.ModifyRequest(upstreamReq)
		}

		resp, err := client.Do(upstreamReq)
		if err != nil {
			return web.Response{}, web.ErrBadGateway(err)
		}

		if opts.ModifyResponse != nil {
			opts.ModifyResponse(resp)
		}

		// Build the web.Response, copying upstream headers minus static and
		// dynamic hop-by-hop headers. RFC 7230 6.1: strip headers named in
		// the upstream response's Connection field.
		dynamicResp := dynamicHopHeaders(resp.Header.Get("Connection"))
		outHeaders := web.NewHeaders()
		for name, values := range resp.Header {
			lower := strings.ToLower(name)
			if hopByHopHeaders[lower] || dynamicResp[lower] {
				continue
			}
			if lower == "set-cookie" {
				for _, v := range values {
					outHeaders.Append(name, rewriteSetCookie(v, resolvedHost, opts.PathPrefix))
				}
				continue
			}
			for _, v := range values {
				outHeaders.Append(name, v)
			}
		}

		return web.Response{
			Status:  resp.StatusCode,
			Headers: outHeaders,
			Body:    resp.Body,
		}, nil
	}
}

// buildUpstreamURL combines the target base URL with the incoming request path and query.
func buildUpstreamURL(target *url.URL, req web.Request) *url.URL {
	u := *target // shallow copy

	incomingPath := ""
	if req.URL != nil {
		incomingPath = req.URL.Path
	}

	if target.Path != "" {
		u.Path = strings.TrimRight(target.Path, "/") + "/" + strings.TrimLeft(incomingPath, "/")
	} else {
		u.Path = incomingPath
	}

	if req.URL != nil {
		u.RawQuery = req.URL.RawQuery
	}

	return &u
}

// rewriteSetCookie rewrites Domain and (optionally) Path attributes in a
// Set-Cookie header value.
//
//   - Domain is replaced with targetHost (port stripped) when a Domain
//     attribute is present. This prevents upstream-internal host names from
//     leaking to clients.
//   - Path is rewritten relative to pathPrefix when pathPrefix is non-empty,
//     so cookies set by an app at its own root remain valid when the app is
//     mounted under a URL prefix. For example: upstream Path=/ with
//     pathPrefix="/app" becomes Path=/app.
func rewriteSetCookie(setCookie, targetHost, pathPrefix string) string {
	// Strip port from targetHost for the Domain value.
	// Use SplitHostPort to correctly handle IPv6 addresses like [::1]:8080.
	host := targetHost
	if h, _, err := net.SplitHostPort(targetHost); err == nil {
		host = h
	}

	parts := strings.Split(setCookie, ";")
	modified := false
	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		lower := strings.ToLower(trimmed)
		if host != "" && strings.HasPrefix(lower, "domain=") {
			parts[i] = " Domain=" + host
			modified = true
		} else if pathPrefix != "" && strings.HasPrefix(lower, "path=") {
			upstreamPath := trimmed[5:] // value after the 5-char "path=" prefix
			var newPath string
			if upstreamPath == "/" {
				newPath = pathPrefix
			} else {
				newPath = strings.TrimRight(pathPrefix, "/") + "/" + strings.TrimLeft(upstreamPath, "/")
			}
			parts[i] = " Path=" + newPath
			modified = true
		}
	}
	if !modified {
		return setCookie
	}
	return strings.Join(parts, ";")
}
