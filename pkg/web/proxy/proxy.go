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
)

// defaultClient is a production-safe HTTP client used when Options.Client is nil.
// It has bounded timeouts and connection pool settings. Callers may provide their
// own client via Options.Client to override.
var defaultClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConnsPerHost:   32,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          128,
	},
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
	// ForwardProto, when true, sets the X-Forwarded-Proto request header to
	// the scheme of the incoming request before forwarding. This allows
	// upstream applications to construct correct absolute URLs and redirects.
	// The header is only set when not already present.
	ForwardProto bool
}

// resolveClient returns the HTTP client to use for upstream requests.
// It returns opts.Client when provided. When MaxResponseHeaderBytes is set it
// builds a fresh client with the same transport settings as defaultClient but
// with the custom header size limit applied. Otherwise it returns defaultClient.
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
	return defaultClient
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

// Reverse returns a web.Handler that proxies every request to target.
//
// It:
//   - Copies the incoming method, URL (path + query), headers, and body to a new upstream request.
//   - Appends X-Forwarded-For and X-Forwarded-Host headers (does not overwrite if already set).
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

		// Copy incoming headers, skipping static and dynamic hop-by-hop headers.
		// RFC 7230 6.1: strip any header named in the Connection field.
		dynamicReq := dynamicHopHeaders(c.Request.Headers.Get("Connection"))
		c.Request.Headers.ForEach(func(name string, values []string) {
			lower := strings.ToLower(name)
			if hopByHopHeaders[lower] || dynamicReq[lower] {
				return
			}
			for _, v := range values {
				upstreamReq.Header.Add(name, v)
			}
		})

		// Set X-Forwarded-Host from the incoming request Host.
		if upstreamReq.Header.Get("X-Forwarded-Host") == "" {
			host := c.Request.Host()
			if host != "" {
				upstreamReq.Header.Set("X-Forwarded-Host", host)
			}
		}

		// Set X-Forwarded-Proto from the incoming request scheme when enabled.
		if opts.ForwardProto && upstreamReq.Header.Get("X-Forwarded-Proto") == "" {
			scheme := "http"
			if c.URL() != nil && c.URL().Scheme != "" {
				scheme = c.URL().Scheme
			} else if proto := c.Request.Headers.Get("X-Forwarded-Proto"); proto != "" {
				scheme = strings.ToLower(strings.TrimSpace(proto))
			}
			upstreamReq.Header.Set("X-Forwarded-Proto", scheme)
		}

		// Set or append X-Forwarded-For with the client IP from RemoteAddr.
		remoteAddr := c.Request.RemoteAddr()
		clientIP := ""
		if remoteAddr != "" {
			if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
				clientIP = host
			}
		}
		if existing := c.Request.Headers.Get("X-Forwarded-For"); existing != "" {
			if clientIP != "" {
				upstreamReq.Header.Set("X-Forwarded-For", existing+", "+clientIP)
			} else {
				upstreamReq.Header.Set("X-Forwarded-For", existing)
			}
		} else if clientIP != "" {
			upstreamReq.Header.Set("X-Forwarded-For", clientIP)
		}

		// Emit RFC 7239 Forwarded header alongside X-Forwarded-* for load
		// balancers and proxies that prefer the standardised format.
		// Append to any existing Forwarded value rather than replacing it.
		if clientIP != "" {
			forwardedProto := "http"
			if c.URL() != nil && c.URL().Scheme != "" {
				forwardedProto = c.URL().Scheme
			} else if proto := c.Request.Headers.Get("X-Forwarded-Proto"); proto != "" {
				forwardedProto = strings.ToLower(strings.TrimSpace(proto))
			}
			forwardedHost := c.Request.Host()
			forwardedValue := fmt.Sprintf("for=%s;host=%s;proto=%s",
				quoteForwardedNode(clientIP), forwardedHost, forwardedProto)
			if existing := c.Request.Headers.Get("Forwarded"); existing != "" {
				upstreamReq.Header.Set("Forwarded", existing+", "+forwardedValue)
			} else {
				upstreamReq.Header.Set("Forwarded", forwardedValue)
			}
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
					outHeaders.Append(name, rewriteSetCookie(v, c.Request.Host(), opts.PathPrefix))
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
	host := targetHost
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
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
