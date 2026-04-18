package web_test

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-sum/web"
	"github.com/go-sum/web/serve"
	"github.com/go-sum/web/htmx"
	"github.com/go-sum/web/router"
	"github.com/go-sum/web/secure"
	"github.com/go-sum/web/session"
	xhtml "golang.org/x/net/html"
)

type integrationApp struct {
	router *router.Router
	client *http.Client
	base   string
}

type formPage struct {
	status  int
	headers http.Header
	body    string
	token   string
}

type requestOption func(*http.Request)

func (a integrationApp) url(name string, params map[string]string) string {
	return a.base + a.router.MustReverse(name, params)
}

func newServerAndClient(t *testing.T, r *router.Router) integrationApp {
	t.Helper()

	srv := httptest.NewServer(serve.ToHTTPHandler(r.Serve))
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}

	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return integrationApp{
		router: r,
		client: client,
		base:   srv.URL,
	}
}

func newSessionRouter(t *testing.T, csrfCfg secure.CSRFConfig) *router.Router {
	t.Helper()

	store := session.NewMemoryStore()
	t.Cleanup(store.Stop)

	r := router.New()
	r.Use(session.Middleware(session.Config{
		Store: store,
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}))
	r.Use(secure.CSRF(csrfCfg))

	r.GET("/ping", "ping", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "pong"), nil
	})

	r.GET("/form", "form.show", func(c *web.Context) (web.Response, error) {
		body := fmt.Sprintf(
			`<form method="post" action="%s"><input type="hidden" name="_csrf" value="%s" /><input name="name" /></form>`,
			r.MustReverse("form.submit", nil),
			html.EscapeString(secure.CSRFToken(c)),
		)
		return web.HTML(http.StatusOK, body), nil
	})

	r.POST("/submit", "form.submit", func(c *web.Context) (web.Response, error) {
		fd, err := c.Request.FormData()
		if err != nil {
			return web.Text(http.StatusBadRequest, "bad form"), nil
		}
		defer fd.Close()

		name := fd.Values.Get("name")
		if name == "" {
			return web.Text(http.StatusUnprocessableEntity, "name required"), nil
		}

		sess, ok := session.FromContext(c)
		if !ok || sess == nil {
			return web.Text(http.StatusInternalServerError, "session missing"), nil
		}
		if err := sess.Set("user", name); err != nil {
			return web.Text(http.StatusInternalServerError, "session write failed"), nil
		}

		if htmx.IsHTMX(c) {
			return web.Text(http.StatusOK, "saved:"+name), nil
		}
		return web.SeeOther(r.MustReverse("me.show", nil)), nil
	})

	r.GET("/me", "me.show", func(c *web.Context) (web.Response, error) {
		sess, ok := session.FromContext(c)
		if !ok || sess == nil {
			return web.Text(http.StatusInternalServerError, "session missing"), nil
		}

		user, ok, err := session.Get[string](sess, "user")
		if err != nil {
			return web.Text(http.StatusInternalServerError, "session read failed"), nil
		}
		if !ok {
			return web.Text(http.StatusUnauthorized, "anonymous"), nil
		}
		return web.Text(http.StatusOK, user), nil
	})

	return r
}

func newStatelessRouter(t *testing.T, csrfCfg secure.CSRFConfig) *router.Router {
	t.Helper()

	r := router.New()
	r.Use(secure.CSRF(csrfCfg))

	r.GET("/form", "form.show", func(c *web.Context) (web.Response, error) {
		body := fmt.Sprintf(
			`<form method="post" action="%s"><input type="hidden" name="_csrf" value="%s" /><input name="name" /></form>`,
			r.MustReverse("form.submit", nil),
			html.EscapeString(secure.CSRFToken(c)),
		)
		return web.HTML(http.StatusOK, body), nil
	})

	r.POST("/submit", "form.submit", func(c *web.Context) (web.Response, error) {
		fd, err := c.Request.FormData()
		if err != nil {
			return web.Text(http.StatusBadRequest, "bad form"), nil
		}
		defer fd.Close()

		name := fd.Values.Get("name")
		if name == "" {
			return web.Text(http.StatusUnprocessableEntity, "name required"), nil
		}

		if htmx.IsHTMX(c) {
			return web.Text(http.StatusOK, "saved:"+name), nil
		}
		return web.Text(http.StatusOK, "accepted:"+name), nil
	})

	return r
}

func newSessionApp(t *testing.T) integrationApp {
	t.Helper()
	return newServerAndClient(t, newSessionRouter(t, mustCSRFConfig(t, mustGenerateKeyHex(t))))
}

func newPreparedSessionApp(t *testing.T) (integrationApp, formPage) {
	t.Helper()

	app := newSessionApp(t)
	page := mustFetchFormPage(t, app)
	return app, page
}

func newPreparedStatelessApp(t *testing.T, csrfCfg secure.CSRFConfig) (integrationApp, formPage) {
	t.Helper()

	app := newServerAndClient(t, newStatelessRouter(t, csrfCfg))
	page := mustFetchFormPage(t, app)
	return app, page
}

func mustGenerateKeyHex(t *testing.T) string {
	t.Helper()

	keyHex, err := secure.GenerateKeyHex()
	if err != nil {
		t.Fatalf("secure.GenerateKeyHex: %v", err)
	}
	return keyHex
}

func mustCSRFConfig(t *testing.T, keyHex string, previousKeysHex ...string) secure.CSRFConfig {
	t.Helper()

	cfg, err := secure.NewCSRFConfigFromHex(keyHex, strings.Join(previousKeysHex, ","))
	if err != nil {
		t.Fatalf("secure.NewCSRFConfigFromHex: %v", err)
	}
	return cfg
}

func mustFetchFormPage(t *testing.T, app integrationApp) formPage {
	t.Helper()

	resp, err := app.client.Get(app.url("form.show", nil))
	if err != nil {
		t.Fatalf("GET %s: %v", app.url("form.show", nil), err)
	}
	body := mustReadBody(t, resp)

	return formPage{
		status:  resp.StatusCode,
		headers: resp.Header.Clone(),
		body:    body,
		token:   mustCSRFToken(t, body),
	}
}

func mustCookie(t *testing.T, app integrationApp, name string) *http.Cookie {
	t.Helper()

	baseURL, err := url.Parse(app.base)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", app.base, err)
	}
	for _, cookie := range app.client.Jar.Cookies(baseURL) {
		if cookie.Name == name {
			return cookie
		}
	}

	t.Fatalf("cookie %q not found in jar", name)
	return nil
}

func mustDo(t *testing.T, client *http.Client, req *http.Request) *http.Response {
	t.Helper()

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL.String(), err)
	}
	return resp
}

func mustReadBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer closeBody(t, resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	return string(body)
}

func closeBody(t *testing.T, resp *http.Response) {
	t.Helper()

	if resp == nil || resp.Body == nil {
		return
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("resp.Body.Close: %v", err)
	}
}

func mustCSRFToken(t *testing.T, body string) string {
	t.Helper()

	doc, err := xhtml.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}

	var walk func(*xhtml.Node) string
	walk = func(n *xhtml.Node) string {
		if n.Type == xhtml.ElementNode && n.Data == "input" {
			var name string
			var value string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "name":
					name = attr.Val
				case "value":
					value = attr.Val
				}
			}
			if name == "_csrf" && value != "" {
				return value
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if token := walk(child); token != "" {
				return token
			}
		}
		return ""
	}

	token := walk(doc)
	if token == "" {
		t.Fatalf("CSRF token not found in body: %q", body)
	}
	return token
}

func newFormPost(t *testing.T, target string, values url.Values, opts ...requestOption) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, target, strings.NewReader(values.Encode()))
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, opt := range opts {
		opt(req)
	}
	return req
}

func withOrigin(origin string) requestOption {
	return func(req *http.Request) {
		req.Header.Set("Origin", origin)
	}
}

func withHeader(name, value string) requestOption {
	return func(req *http.Request) {
		req.Header.Set(name, value)
	}
}

func withCookie(cookie *http.Cookie) requestOption {
	return func(req *http.Request) {
		req.AddCookie(cookie)
	}
}

func TestIntegration_SessionAndCSRFFlow(t *testing.T) {
	t.Parallel()

	t.Run("form renders token and security headers", func(t *testing.T) {
		t.Parallel()

		_, page := newPreparedSessionApp(t)

		if page.status != http.StatusOK {
			t.Fatalf("GET /form status = %d, want %d", page.status, http.StatusOK)
		}
		if page.token == "" {
			t.Fatal("GET /form returned empty CSRF token")
		}
		if got := page.headers.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Fatalf("X-Content-Type-Options = %q, want %q", got, "nosniff")
		}
	})

	t.Run("missing form token returns 403", func(t *testing.T) {
		t.Parallel()

		app, _ := newPreparedSessionApp(t)
		resp := mustDo(t, app.client, newFormPost(
			t,
			app.url("form.submit", nil),
			url.Values{"name": {"alice"}},
			withOrigin(app.base),
		))
		body := mustReadBody(t, resp)

		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("POST /submit status = %d, want %d", resp.StatusCode, http.StatusForbidden)
		}
		if !strings.Contains(body, "CSRF token missing") {
			t.Fatalf("POST /submit missing-token response body = %q, want it to contain %q", body, "CSRF token missing")
		}
	})

	t.Run("cross origin request is rejected", func(t *testing.T) {
		t.Parallel()

		app, page := newPreparedSessionApp(t)
		resp := mustDo(t, app.client, newFormPost(
			t,
			app.url("form.submit", nil),
			url.Values{
				"_csrf": {page.token},
				"name":  {"alice"},
			},
			withOrigin("https://evil.example"),
		))
		body := mustReadBody(t, resp)

		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("cross-origin POST /submit status = %d, want %d", resp.StatusCode, http.StatusForbidden)
		}
		if !strings.Contains(body, "CSRF origin invalid") {
			t.Fatalf("cross-origin POST /submit response body = %q, want it to contain %q", body, "CSRF origin invalid")
		}
	})

	t.Run("htmx header token passes through", func(t *testing.T) {
		t.Parallel()

		app, page := newPreparedSessionApp(t)
		resp := mustDo(t, app.client, newFormPost(
			t,
			app.url("form.submit", nil),
			url.Values{"name": {"alice"}},
			withOrigin(app.base),
			withHeader("HX-Request", "true"),
			withHeader("X-CSRF-Token", page.token),
		))
		body := mustReadBody(t, resp)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("HTMX POST /submit status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if body != "saved:alice" {
			t.Fatalf("HTMX POST /submit body = %q, want %q", body, "saved:alice")
		}
	})

	t.Run("valid form token redirects", func(t *testing.T) {
		t.Parallel()

		app, page := newPreparedSessionApp(t)
		resp := mustDo(t, app.client, newFormPost(
			t,
			app.url("form.submit", nil),
			url.Values{
				"_csrf": {page.token},
				"name":  {"alice"},
			},
			withOrigin(app.base),
		))
		closeBody(t, resp)

		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("POST /submit status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
		}
		if got := resp.Header.Get("Location"); got != app.router.MustReverse("me.show", nil) {
			t.Fatalf("POST /submit Location = %q, want %q", got, app.router.MustReverse("me.show", nil))
		}
	})

	t.Run("follow-up request reads session state", func(t *testing.T) {
		t.Parallel()

		app, page := newPreparedSessionApp(t)
		resp := mustDo(t, app.client, newFormPost(
			t,
			app.url("form.submit", nil),
			url.Values{
				"_csrf": {page.token},
				"name":  {"alice"},
			},
			withOrigin(app.base),
		))
		closeBody(t, resp)

		meResp := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.url("me.show", nil)))
		body := mustReadBody(t, meResp)

		if meResp.StatusCode != http.StatusOK {
			t.Fatalf("GET /me status = %d, want %d", meResp.StatusCode, http.StatusOK)
		}
		if body != "alice" {
			t.Fatalf("GET /me body = %q, want %q", body, "alice")
		}
	})
}

func TestIntegration_StatelessCSRFFlow(t *testing.T) {
	t.Parallel()

	t.Run("issues csrf cookie and accepts matching token", func(t *testing.T) {
		t.Parallel()

		app, page := newPreparedStatelessApp(t, mustCSRFConfig(t, mustGenerateKeyHex(t)))
		cookie := mustCookie(t, app, "csrf")
		resp := mustDo(t, app.client, newFormPost(
			t,
			app.url("form.submit", nil),
			url.Values{
				"_csrf": {page.token},
				"name":  {"bob"},
			},
			withOrigin(app.base),
		))
		body := mustReadBody(t, resp)

		if page.status != http.StatusOK {
			t.Fatalf("GET /form status = %d, want %d", page.status, http.StatusOK)
		}
		if cookie.Value == "" {
			t.Fatal("csrf cookie value is empty")
		}
		if cookie.Value != page.token {
			t.Fatalf("csrf cookie value = %q, want token %q", cookie.Value, page.token)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("POST /submit status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if body != "accepted:bob" {
			t.Fatalf("POST /submit body = %q, want %q", body, "accepted:bob")
		}
	})
}

func TestIntegration_StatelessCSRFKeyRotation(t *testing.T) {
	t.Parallel()

	oldKey := mustGenerateKeyHex(t)
	oldApp, page := newPreparedStatelessApp(t, mustCSRFConfig(t, oldKey))
	oldCookie := mustCookie(t, oldApp, "csrf")

	newKey := mustGenerateKeyHex(t)
	rotatedApp := newServerAndClient(t, newStatelessRouter(t, mustCSRFConfig(t, newKey, oldKey)))
	resp := mustDo(t, rotatedApp.client, newFormPost(
		t,
		rotatedApp.url("form.submit", nil),
		url.Values{
			"_csrf": {page.token},
			"name":  {"carol"},
		},
		withOrigin(rotatedApp.base),
		withCookie(&http.Cookie{Name: oldCookie.Name, Value: oldCookie.Value}),
	))
	body := mustReadBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("rotated POST /submit status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if body != "accepted:carol" {
		t.Fatalf("rotated POST /submit body = %q, want %q", body, "accepted:carol")
	}
}

func TestIntegration_RouterHTTPContract(t *testing.T) {
	t.Parallel()

	t.Run("HEAD falls back to GET without body", func(t *testing.T) {
		t.Parallel()

		app := newSessionApp(t)
		resp := mustDo(t, app.client, mustRequest(t, http.MethodHead, app.url("ping", nil)))
		body := mustReadBody(t, resp)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("HEAD /ping status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if body != "" {
			t.Fatalf("HEAD /ping body = %q, want empty", body)
		}
	})

	t.Run("OPTIONS returns Allow header", func(t *testing.T) {
		t.Parallel()

		app := newSessionApp(t)
		resp := mustDo(t, app.client, mustRequest(t, http.MethodOptions, app.url("ping", nil)))
		closeBody(t, resp)

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("OPTIONS /ping status = %d, want %d", resp.StatusCode, http.StatusNoContent)
		}
		if got := resp.Header.Get("Allow"); got != "GET, HEAD, OPTIONS" {
			t.Fatalf("OPTIONS /ping Allow = %q, want %q", got, "GET, HEAD, OPTIONS")
		}
	})

	t.Run("405 returns non-empty body", func(t *testing.T) {
		t.Parallel()

		app := newSessionApp(t)
		resp := mustDo(t, app.client, mustRequest(t, http.MethodPost, app.url("ping", nil)))
		body := mustReadBody(t, resp)

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("POST /ping status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
		}
		if !strings.Contains(body, "Method Not Allowed") {
			t.Fatalf("POST /ping body = %q, want it to contain %q", body, "Method Not Allowed")
		}
	})

	t.Run("404 returns non-empty body", func(t *testing.T) {
		t.Parallel()

		app := newSessionApp(t)
		resp := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.base+"/missing"))
		body := mustReadBody(t, resp)

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("GET /missing status = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
		if !strings.Contains(body, "Not Found") {
			t.Fatalf("GET /missing body = %q, want it to contain %q", body, "Not Found")
		}
	})
}

func mustRequest(t *testing.T, method, target string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, target, nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	return req
}

// ---------------------------------------------------------------------------
// Security: Sec-Fetch-Site enforcement
// ---------------------------------------------------------------------------

func TestIntegration_SecFetchSiteBlocksCrossOrigin(t *testing.T) {
	t.Parallel()

	app, page := newPreparedSessionApp(t)
	resp := mustDo(t, app.client, newFormPost(
		t,
		app.url("form.submit", nil),
		url.Values{
			"_csrf": {page.token},
			"name":  {"alice"},
		},
		withHeader("Sec-Fetch-Site", "cross-site"),
	))
	body := mustReadBody(t, resp)

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Sec-Fetch-Site:cross-site POST status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
	if !strings.Contains(body, "CSRF origin invalid") {
		t.Fatalf("Sec-Fetch-Site:cross-site POST body = %q, want it to contain %q", body, "CSRF origin invalid")
	}
}

func TestIntegration_SecFetchSiteSameOriginStillRequiresToken(t *testing.T) {
	t.Parallel()

	// Sec-Fetch-Site: same-origin bypasses the origin check but the CSRF
	// middleware still requires a valid token in the submitted request.
	// A POST with no token at all must be rejected with 403 even though
	// the Sec-Fetch-Site header claims same-origin.
	app, _ := newPreparedSessionApp(t)
	resp := mustDo(t, app.client, newFormPost(
		t,
		app.url("form.submit", nil),
		url.Values{"name": {"alice"}},
		withHeader("Sec-Fetch-Site", "same-origin"),
		// No Origin header, no _csrf field.
	))
	body := mustReadBody(t, resp)

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("same-origin no-token POST status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
	if !strings.Contains(body, "CSRF token missing") {
		t.Fatalf("same-origin no-token POST body = %q, want it to contain %q", body, "CSRF token missing")
	}
}

// ---------------------------------------------------------------------------
// Session: Regenerate (session fixation defence)
// ---------------------------------------------------------------------------

func TestIntegration_SessionRegenerate(t *testing.T) {
	t.Parallel()

	store := session.NewMemoryStore()
	t.Cleanup(store.Stop)

	r := router.New()
	r.Use(session.Middleware(session.Config{
		Store: store,
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}))
	r.Use(secure.CSRF(mustCSRFConfig(t, mustGenerateKeyHex(t))))

	r.GET("/form", "form.show", func(c *web.Context) (web.Response, error) {
		body := fmt.Sprintf(
			`<form method="post" action="/login"><input type="hidden" name="_csrf" value="%s" /></form>`,
			html.EscapeString(secure.CSRFToken(c)),
		)
		return web.HTML(http.StatusOK, body), nil
	})

	r.POST("/login", "login.submit", func(c *web.Context) (web.Response, error) {
		sess, ok := session.FromContext(c)
		if !ok || sess == nil {
			return web.Text(http.StatusInternalServerError, "session missing"), nil
		}
		sess.Regenerate()
		if err := sess.Set("user", "alice"); err != nil {
			return web.Text(http.StatusInternalServerError, "set failed"), nil
		}
		return web.Text(http.StatusOK, "logged in"), nil
	})

	r.GET("/me", "me.show", func(c *web.Context) (web.Response, error) {
		sess, ok := session.FromContext(c)
		if !ok || sess == nil {
			return web.Text(http.StatusInternalServerError, "session missing"), nil
		}
		user, found, err := session.Get[string](sess, "user")
		if err != nil || !found {
			return web.Text(http.StatusUnauthorized, "anonymous"), nil
		}
		return web.Text(http.StatusOK, user), nil
	})

	app := newServerAndClient(t, r)

	// Step 1: GET /form — capture initial session cookie value.
	page := mustFetchFormPage(t, app)
	if page.status != http.StatusOK {
		t.Fatalf("GET /form status = %d, want %d", page.status, http.StatusOK)
	}
	oldCookieValue := mustCookie(t, app, "sess").Value

	// Step 2: POST /login — the handler calls Regenerate(); expect new cookie.
	resp := mustDo(t, app.client, newFormPost(
		t,
		app.base+"/login",
		url.Values{
			"_csrf": {page.token},
		},
		withOrigin(app.base),
	))
	body := mustReadBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /login status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if body != "logged in" {
		t.Fatalf("POST /login body = %q, want %q", body, "logged in")
	}

	newCookieValue := mustCookie(t, app, "sess").Value
	if newCookieValue == oldCookieValue {
		t.Fatal("session cookie value did not change after Regenerate — session fixation risk")
	}

	// Step 3: GET /me — the new session should carry the user value.
	meResp := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.url("me.show", nil)))
	meBody := mustReadBody(t, meResp)

	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /me status = %d, want %d", meResp.StatusCode, http.StatusOK)
	}
	if meBody != "alice" {
		t.Fatalf("GET /me body = %q, want %q", meBody, "alice")
	}
}

// ---------------------------------------------------------------------------
// Session: Destroy
// ---------------------------------------------------------------------------

func TestIntegration_SessionDestroy(t *testing.T) {
	t.Parallel()

	store := session.NewMemoryStore()
	t.Cleanup(store.Stop)

	r := router.NewWithoutSecureDefaults()
	r.Use(session.Middleware(session.Config{
		Store: store,
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}))

	// First set a value so the session is persisted (dirty).
	r.GET("/seed", "seed", func(c *web.Context) (web.Response, error) {
		sess, ok := session.FromContext(c)
		if !ok || sess == nil {
			return web.Text(http.StatusInternalServerError, "session missing"), nil
		}
		if err := sess.Set("user", "alice"); err != nil {
			return web.Text(http.StatusInternalServerError, "set failed"), nil
		}
		return web.Text(http.StatusOK, "seeded"), nil
	})

	r.POST("/logout", "logout", func(c *web.Context) (web.Response, error) {
		sess, ok := session.FromContext(c)
		if !ok || sess == nil {
			return web.Text(http.StatusInternalServerError, "session missing"), nil
		}
		sess.Destroy()
		return web.Text(http.StatusOK, "logged out"), nil
	})

	app := newServerAndClient(t, r)

	// Seed a session.
	seedResp := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.base+"/seed"))
	mustReadBody(t, seedResp)

	// POST /logout.
	logoutReq, err := http.NewRequest(http.MethodPost, app.base+"/logout", nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	logoutResp := mustDo(t, app.client, logoutReq)
	mustReadBody(t, logoutResp)

	if logoutResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /logout status = %d, want %d", logoutResp.StatusCode, http.StatusOK)
	}

	// Inspect Set-Cookie header from the logout response.
	// web.Cookie serializes MaxAge=-1 as Max-Age=0 (stdlib convention: negative → delete).
	setCookie := logoutResp.Header.Get("Set-Cookie")
	if !strings.Contains(setCookie, "Max-Age=0") {
		t.Fatalf("Set-Cookie on logout = %q — expected Max-Age=0 (deleted cookie)", setCookie)
	}

	// Inspect Clear-Site-Data header.
	clearSiteData := logoutResp.Header.Get("Clear-Site-Data")
	if !strings.Contains(clearSiteData, "cookies") {
		t.Fatalf("Clear-Site-Data = %q — expected to contain %q", clearSiteData, "cookies")
	}
	if !strings.Contains(clearSiteData, "storage") {
		t.Fatalf("Clear-Site-Data = %q — expected to contain %q", clearSiteData, "storage")
	}
}

// ---------------------------------------------------------------------------
// Rate limiting
// ---------------------------------------------------------------------------

func TestIntegration_RateLimit(t *testing.T) {
	t.Parallel()

	store := secure.NewMemoryStore(secure.MemoryStoreConfig{
		Rate:  1,
		Burst: 2,
	})

	r := router.NewWithoutSecureDefaults()
	r.Use(secure.RateLimit(secure.RateLimitConfig{
		Store:          store,
		IdentifierFunc: func(_ *web.Context) string { return "test-client" },
	}))
	r.GET("/ping", "ping", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "pong"), nil
	})

	app := newServerAndClient(t, r)

	// First two requests must succeed (burst=2).
	for i := range 2 {
		resp := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.base+"/ping"))
		body := mustReadBody(t, resp)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", i+1, resp.StatusCode, http.StatusOK)
		}
		if body != "pong" {
			t.Fatalf("request %d: body = %q, want %q", i+1, body, "pong")
		}
	}

	// Third request must be rate-limited.
	resp := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.base+"/ping"))
	body := mustReadBody(t, resp)

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("third request status = %d, want %d", resp.StatusCode, http.StatusTooManyRequests)
	}
	if !strings.Contains(body, "Too Many Requests") {
		t.Fatalf("third request body = %q, want it to contain %q", body, "Too Many Requests")
	}
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("Retry-After header missing on 429 response")
	}
	// Retry-After must be a non-negative integer.
	for _, ch := range retryAfter {
		if ch < '0' || ch > '9' {
			t.Fatalf("Retry-After = %q — expected numeric value, got non-digit character %q", retryAfter, string(ch))
		}
	}
}

// ---------------------------------------------------------------------------
// MaxBodyBytes → 413
// ---------------------------------------------------------------------------

func TestIntegration_MaxBodyBytes413(t *testing.T) {
	t.Parallel()

	handler := func(c *web.Context) (web.Response, error) {
		if _, err := c.Request.Bytes(); err != nil {
			return web.Text(http.StatusRequestEntityTooLarge, "too large"), nil
		}
		return web.Text(http.StatusOK, "ok"), nil
	}

	srv := httptest.NewServer(serve.ToHTTPHandlerWithConfig(handler, serve.Config{
		MaxRequestBodyBytes: 100,
	}))
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	client := &http.Client{Jar: jar}

	body := strings.Repeat("x", 200)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/upload", strings.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	respBody := mustReadBody(t, resp)

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusRequestEntityTooLarge)
	}
	if respBody != "too large" {
		t.Fatalf("body = %q, want %q", respBody, "too large")
	}
}

// ---------------------------------------------------------------------------
// CSP nonce uniqueness
// ---------------------------------------------------------------------------

func TestIntegration_CSPNonceUniqueness(t *testing.T) {
	t.Parallel()

	r := router.New() // router.New() installs SecureDefaults which includes CSPNonce
	r.GET("/page", "page", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "hello"), nil
	})

	app := newServerAndClient(t, r)

	extractNonce := func(csp string) string {
		const prefix = "nonce-"
		idx := strings.Index(csp, prefix)
		if idx == -1 {
			return ""
		}
		rest := csp[idx+len(prefix):]
		// Nonce ends at the first ' or space.
		end := strings.IndexAny(rest, "' ")
		if end == -1 {
			return rest
		}
		return rest[:end]
	}

	resp1 := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.base+"/page"))
	csp1 := resp1.Header.Get("Content-Security-Policy")
	mustReadBody(t, resp1)

	resp2 := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.base+"/page"))
	csp2 := resp2.Header.Get("Content-Security-Policy")
	mustReadBody(t, resp2)

	nonce1 := extractNonce(csp1)
	nonce2 := extractNonce(csp2)

	if nonce1 == "" {
		t.Fatalf("first response CSP has no nonce: %q", csp1)
	}
	if nonce2 == "" {
		t.Fatalf("second response CSP has no nonce: %q", csp2)
	}
	if nonce1 == nonce2 {
		t.Fatalf("CSP nonces are identical across requests: %q", nonce1)
	}
}

// ---------------------------------------------------------------------------
// Secure defaults headers
// ---------------------------------------------------------------------------

func TestIntegration_SecureDefaultsHeaders(t *testing.T) {
	t.Parallel()

	r := router.New() // SecureDefaults installed by New()
	r.GET("/check", "check", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	})

	app := newServerAndClient(t, r)
	resp := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.base+"/check"))
	mustReadBody(t, resp)

	checks := []struct {
		header string
		want   string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload"},
		{"Cross-Origin-Opener-Policy", "same-origin"},
		{"Cross-Origin-Embedder-Policy", "require-corp"},
		{"Cross-Origin-Resource-Policy", "same-origin"},
	}

	for _, tc := range checks {
		got := resp.Header.Get(tc.header)
		if got != tc.want {
			t.Fatalf("%s = %q, want %q", tc.header, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Panic recovery
// ---------------------------------------------------------------------------

func TestIntegration_PanicRecovery(t *testing.T) {
	t.Parallel()

	r := router.New()
	// ErrorBoundary must be installed explicitly to recover panics and render
	// a 500 response. Without it, the panic propagates to net/http which closes
	// the connection.
	r.Use(web.ErrorBoundary(web.BoundaryConfig{}))
	r.GET("/panic", "panic.show", func(_ *web.Context) (web.Response, error) {
		panic("intentional test panic")
	})

	app := newServerAndClient(t, r)
	resp := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.base+"/panic"))
	body := mustReadBody(t, resp)

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
	if !strings.Contains(body, "Internal Server Error") {
		t.Fatalf("body = %q, want it to contain %q", body, "Internal Server Error")
	}
}

// ---------------------------------------------------------------------------
// CSRF: X-XSRF-Token header accepted
// ---------------------------------------------------------------------------

func TestIntegration_XSRFTokenHeaderAccepted(t *testing.T) {
	t.Parallel()

	app, page := newPreparedSessionApp(t)
	resp := mustDo(t, app.client, newFormPost(
		t,
		app.url("form.submit", nil),
		url.Values{"name": {"dave"}},
		withOrigin(app.base),
		withHeader("X-XSRF-Token", page.token),
	))
	body := mustReadBody(t, resp)

	if resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusOK {
		t.Fatalf("X-XSRF-Token POST status = %d, want 200 or 303", resp.StatusCode)
	}
	// Ensure it was not rejected with 403.
	if resp.StatusCode == http.StatusForbidden {
		t.Fatalf("X-XSRF-Token POST was rejected with 403; body = %q", body)
	}
}

// ---------------------------------------------------------------------------
// CSRF: session mode issues no separate csrf cookie
// ---------------------------------------------------------------------------

func TestIntegration_SessionCSRFNoCsrfCookie(t *testing.T) {
	t.Parallel()

	app, _ := newPreparedSessionApp(t)

	baseURL, err := url.Parse(app.base)
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}

	for _, cookie := range app.client.Jar.Cookies(baseURL) {
		if cookie.Name == "csrf" {
			t.Fatalf("session mode should not issue a 'csrf' double-submit cookie, but found one: %v", cookie)
		}
	}
}

// ---------------------------------------------------------------------------
// CSRF: stateless mode cookie flags
// ---------------------------------------------------------------------------

func TestIntegration_StatelessCSRFCookieFlags(t *testing.T) {
	t.Parallel()

	app, _ := newPreparedStatelessApp(t, mustCSRFConfig(t, mustGenerateKeyHex(t)))

	// Use the raw Set-Cookie response header from the form GET to inspect attributes.
	// The cookie jar stores parsed cookies, but we need the raw header for flag inspection.
	// Re-issue a fresh GET to capture the raw response headers.
	resp, err := app.client.Get(app.url("form.show", nil))
	if err != nil {
		t.Fatalf("GET /form: %v", err)
	}
	mustReadBody(t, resp)

	rawSetCookie := resp.Header.Get("Set-Cookie")
	if rawSetCookie == "" {
		t.Fatal("Set-Cookie header not present on GET /form in stateless mode")
	}

	// HttpOnly must NOT be present (JS must read the cookie to send in XHR).
	upperRaw := strings.ToLower(rawSetCookie)
	if strings.Contains(upperRaw, "httponly") {
		t.Fatalf("CSRF cookie must not have HttpOnly flag; Set-Cookie = %q", rawSetCookie)
	}

	// Path=/ must be present.
	if !strings.Contains(rawSetCookie, "Path=/") {
		t.Fatalf("CSRF cookie must have Path=/; Set-Cookie = %q", rawSetCookie)
	}

	// SameSite attribute must be present.
	if !strings.Contains(rawSetCookie, "SameSite=") {
		t.Fatalf("CSRF cookie must have SameSite attribute; Set-Cookie = %q", rawSetCookie)
	}
}

// ---------------------------------------------------------------------------
// Session: lazy emission (no Set-Cookie when session is read-only)
// ---------------------------------------------------------------------------

func TestIntegration_SessionLazyEmission(t *testing.T) {
	t.Parallel()

	store := session.NewMemoryStore()
	t.Cleanup(store.Stop)

	r := router.NewWithoutSecureDefaults()
	r.Use(session.Middleware(session.Config{
		Store: store,
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}))

	r.GET("/readonly", "readonly", func(c *web.Context) (web.Response, error) {
		sess, ok := session.FromContext(c)
		if !ok || sess == nil {
			return web.Text(http.StatusInternalServerError, "session missing"), nil
		}
		// Read without writing — session must not be committed.
		_, _, _ = session.Get[string](sess, "anything")
		return web.Text(http.StatusOK, "read"), nil
	})

	app := newServerAndClient(t, r)
	resp := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.base+"/readonly"))
	mustReadBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /readonly status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Set-Cookie"); got != "" {
		t.Fatalf("GET /readonly Set-Cookie = %q, want empty (lazy emission)", got)
	}
}

// ---------------------------------------------------------------------------
// Session: flash (one-shot consumption)
// ---------------------------------------------------------------------------

func TestIntegration_SessionFlash(t *testing.T) {
	t.Parallel()

	store := session.NewMemoryStore()
	t.Cleanup(store.Stop)

	r := router.NewWithoutSecureDefaults()
	r.Use(session.Middleware(session.Config{
		Store: store,
		CookieTemplate: web.Cookie{
			Name:     "sess",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
		TTL: time.Hour,
	}))

	r.POST("/flash-set", "flash.set", func(c *web.Context) (web.Response, error) {
		sess, ok := session.FromContext(c)
		if !ok || sess == nil {
			return web.Text(http.StatusInternalServerError, "session missing"), nil
		}
		if err := sess.Flash("msg", "hello"); err != nil {
			return web.Text(http.StatusInternalServerError, "flash failed"), nil
		}
		return web.Text(http.StatusOK, "set"), nil
	})

	r.GET("/flash-get", "flash.get", func(c *web.Context) (web.Response, error) {
		sess, ok := session.FromContext(c)
		if !ok || sess == nil {
			return web.Text(http.StatusInternalServerError, "session missing"), nil
		}
		value, found, err := session.FlashPop[string](sess, "msg")
		if err != nil {
			return web.Text(http.StatusInternalServerError, "flash pop failed"), nil
		}
		if !found {
			return web.Text(http.StatusOK, "empty"), nil
		}
		return web.Text(http.StatusOK, value), nil
	})

	app := newServerAndClient(t, r)

	// POST /flash-set.
	setReq, err := http.NewRequest(http.MethodPost, app.base+"/flash-set", nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	setResp := mustDo(t, app.client, setReq)
	mustReadBody(t, setResp)
	if setResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /flash-set status = %d, want %d", setResp.StatusCode, http.StatusOK)
	}

	// First GET /flash-get — expect "hello".
	get1Resp := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.base+"/flash-get"))
	get1Body := mustReadBody(t, get1Resp)
	if get1Resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /flash-get (1) status = %d, want %d", get1Resp.StatusCode, http.StatusOK)
	}
	if get1Body != "hello" {
		t.Fatalf("GET /flash-get (1) body = %q, want %q", get1Body, "hello")
	}

	// Second GET /flash-get — flash was consumed; expect "empty".
	get2Resp := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.base+"/flash-get"))
	get2Body := mustReadBody(t, get2Resp)
	if get2Resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /flash-get (2) status = %d, want %d", get2Resp.StatusCode, http.StatusOK)
	}
	if get2Body != "empty" {
		t.Fatalf("GET /flash-get (2) body = %q, want %q (flash must be consumed exactly once)", get2Body, "empty")
	}
}

// ---------------------------------------------------------------------------
// Router: path parameter extraction
// ---------------------------------------------------------------------------

func TestIntegration_RouterPathParam(t *testing.T) {
	t.Parallel()

	r := router.NewWithoutSecureDefaults()
	r.GET("/users/{id}", "user.show", func(c *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, c.Param("id")), nil
	})

	app := newServerAndClient(t, r)
	resp := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.base+"/users/abc123"))
	body := mustReadBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /users/abc123 status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if body != "abc123" {
		t.Fatalf("GET /users/abc123 body = %q, want %q", body, "abc123")
	}
}

// ---------------------------------------------------------------------------
// Router: Allow header for GET+POST route
// ---------------------------------------------------------------------------

func TestIntegration_RouterAllowHeaderExact(t *testing.T) {
	t.Parallel()

	r := router.NewWithoutSecureDefaults()
	r.GET("/resource", "resource.get", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "get"), nil
	})
	r.POST("/resource", "resource.post", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "post"), nil
	})

	app := newServerAndClient(t, r)
	resp := mustDo(t, app.client, mustRequest(t, http.MethodOptions, app.base+"/resource"))
	closeBody(t, resp)

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("OPTIONS /resource status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	allowHeader := resp.Header.Get("Allow")
	allowMethods := strings.Split(allowHeader, ", ")
	allowSet := make(map[string]struct{}, len(allowMethods))
	for _, m := range allowMethods {
		allowSet[strings.TrimSpace(m)] = struct{}{}
	}

	for _, required := range []string{"GET", "HEAD", "POST", "OPTIONS"} {
		if _, ok := allowSet[required]; !ok {
			t.Fatalf("Allow header %q missing %q (full value: %q)", allowHeader, required, allowHeader)
		}
	}
}

// ---------------------------------------------------------------------------
// Router: PUT, PATCH, DELETE methods
// ---------------------------------------------------------------------------

func TestIntegration_RouterPUTPATCHDELETE(t *testing.T) {
	t.Parallel()

	r := router.NewWithoutSecureDefaults()
	r.PUT("/put", "res.put", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	})
	r.PATCH("/patch", "res.patch", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	})
	r.DELETE("/delete", "res.delete", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	})

	app := newServerAndClient(t, r)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPut, "/put"},
		{http.MethodPatch, "/patch"},
		{http.MethodDelete, "/delete"},
	}

	for _, tc := range cases {
		resp := mustDo(t, app.client, mustRequest(t, tc.method, app.base+tc.path))
		body := mustReadBody(t, resp)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s %s status = %d, want %d", tc.method, tc.path, resp.StatusCode, http.StatusOK)
		}
		if body != "ok" {
			t.Fatalf("%s %s body = %q, want %q", tc.method, tc.path, body, "ok")
		}
	}
}

// ---------------------------------------------------------------------------
// HTMX: response headers
// ---------------------------------------------------------------------------

func TestIntegration_HTMXResponseHeaders(t *testing.T) {
	t.Parallel()

	r := router.NewWithoutSecureDefaults()
	r.GET("/htmx-action", "htmx.action", func(c *web.Context) (web.Response, error) {
		resp := web.Text(http.StatusOK, "fragment")
		htmx.SetRedirect(&resp, "/new-url")
		htmx.SetRefresh(&resp)
		htmx.SetTrigger(&resp, "myEvent", nil)
		return resp, nil
	})

	app := newServerAndClient(t, r)
	req := mustRequest(t, http.MethodGet, app.base+"/htmx-action")
	req.Header.Set("HX-Request", "true")
	resp := mustDo(t, app.client, req)
	mustReadBody(t, resp)

	if got := resp.Header.Get("HX-Redirect"); got != "/new-url" {
		t.Fatalf("HX-Redirect = %q, want %q", got, "/new-url")
	}
	if got := resp.Header.Get("HX-Refresh"); got != "true" {
		t.Fatalf("HX-Refresh = %q, want %q", got, "true")
	}
	if got := resp.Header.Get("HX-Trigger"); got != "myEvent" {
		t.Fatalf("HX-Trigger = %q, want %q", got, "myEvent")
	}
}

// ---------------------------------------------------------------------------
// HTMX: VaryMiddleware adds HX-Request to Vary
// ---------------------------------------------------------------------------

func TestIntegration_HTMXVaryMiddleware(t *testing.T) {
	t.Parallel()

	r := router.NewWithoutSecureDefaults()
	r.Use(htmx.VaryMiddleware())
	r.GET("/page", "vary.page", func(_ *web.Context) (web.Response, error) {
		resp := web.Text(http.StatusOK, "content")
		resp.Headers.Set("Vary", "Accept")
		return resp, nil
	})

	app := newServerAndClient(t, r)
	resp := mustDo(t, app.client, mustRequest(t, http.MethodGet, app.base+"/page"))
	mustReadBody(t, resp)

	// The adapter writes each header value with w.Header().Add — multiple Vary
	// values may arrive as separate header lines. Collect all of them.
	varySet := make(map[string]struct{})
	for _, headerLine := range resp.Header.Values("Vary") {
		for _, part := range strings.Split(headerLine, ",") {
			varySet[strings.TrimSpace(part)] = struct{}{}
		}
	}

	if _, ok := varySet["Accept"]; !ok {
		t.Fatalf("Vary values %v missing %q", resp.Header.Values("Vary"), "Accept")
	}
	if _, ok := varySet["HX-Request"]; !ok {
		t.Fatalf("Vary values %v missing %q", resp.Header.Values("Vary"), "HX-Request")
	}
}
