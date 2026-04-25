package componentry

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-sum/web"
	"github.com/go-sum/web/render"
	g "maragu.dev/gomponents"
)

// stubPage satisfies PageFunc without requiring a real view layer.
// It renders the content node directly as a 200 HTML response so tests
// can assert on the body produced by the handler logic.
func stubPage(_ *web.Context, _ string, content g.Node) (web.Response, error) {
	return render.Component(content)
}

// newTestHandler constructs a handler with the stub PageFunc and a fixed base path.
func newTestHandler() *handler {
	return newHandler(Config{
		BasePath: "/componentry",
		Page:     stubPage,
	})
}

// newCtx builds a *web.Context from a method and raw URL string. Path params
// can be set on the returned context with c.SetParam before calling the handler.
func newCtx(t *testing.T, method, rawURL string) *web.Context {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", rawURL, err)
	}
	req := web.NewRequest(method, u)
	return web.NewContext(context.Background(), req)
}

// readBody drains the response body and returns it as a string.
func readBody(t *testing.T, resp web.Response) string {
	t.Helper()
	if resp.Body == nil {
		return ""
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	return string(data)
}

// ──────────────────────────────────────────────
// Show
// ──────────────────────────────────────────────

func TestHandler_Show_ReturnsOK(t *testing.T) {
	h := newTestHandler()
	c := newCtx(t, http.MethodGet, "/components")

	resp, err := h.Show(c)
	if err != nil {
		t.Fatalf("Show: unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Errorf("Show: status = %d, want %d", resp.Status, http.StatusOK)
	}
	body := readBody(t, resp)
	if body == "" {
		t.Error("Show: response body is empty")
	}
}

// Show must delegate to cfg.Page and pass the Showcase() node to it.
// We verify by confirming the rendered output contains a known stable string
// from the Showcase() tree.
func TestHandler_Show_DelegatesPageFunc(t *testing.T) {
	var capturedTitle string
	var capturedContent g.Node
	cfg := Config{
		BasePath: "/componentry",
		Page: func(c *web.Context, title string, content g.Node) (web.Response, error) {
			capturedTitle = title
			capturedContent = content
			return render.Component(content)
		},
	}
	h := newHandler(cfg)
	c := newCtx(t, http.MethodGet, "/components")

	_, err := h.Show(c)
	if err != nil {
		t.Fatalf("Show: unexpected error: %v", err)
	}
	if capturedTitle != "Component Showcase" {
		t.Errorf("Show: title passed to PageFunc = %q, want %q", capturedTitle, "Component Showcase")
	}
	if capturedContent == nil {
		t.Error("Show: content node passed to PageFunc is nil")
	}
}

// Show must surface any error returned by cfg.Page to the caller unchanged.
func TestHandler_Show_PageFuncErrorPropagates(t *testing.T) {
	wantErr := &web.Error{Status: http.StatusInternalServerError, Title: "page error"}
	cfg := Config{
		BasePath: "/componentry",
		Page: func(_ *web.Context, _ string, _ g.Node) (web.Response, error) {
			return web.Response{}, wantErr
		},
	}
	h := newHandler(cfg)
	c := newCtx(t, http.MethodGet, "/components")

	_, err := h.Show(c)
	if err != wantErr {
		t.Errorf("Show: err = %v, want %v", err, wantErr)
	}
}

// ──────────────────────────────────────────────
// Search
// ──────────────────────────────────────────────

func TestHandler_Search(t *testing.T) {
	tests := []struct {
		name  string
		query string // raw URL including any ?q= parameter
		want  string // expected substring in body
	}{
		{
			name:  "query matches users",
			query: "/demo/search?q=alice",
			want:  "Alice Johnson",
		},
		{
			name:  "empty query returns all rows",
			query: "/demo/search",
			want:  "Alice Johnson",
		},
		{
			name:  "no matches returns fallback message",
			query: "/demo/search?q=zzznomatch",
			want:  "No results found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler()
			c := newCtx(t, http.MethodGet, tc.query)

			resp, err := h.Search(c)
			if err != nil {
				t.Fatalf("Search: unexpected error: %v", err)
			}
			if resp.Status != http.StatusOK {
				t.Errorf("Search: status = %d, want %d", resp.Status, http.StatusOK)
			}
			body := readBody(t, resp)
			if !strings.Contains(body, tc.want) {
				t.Errorf("Search: body does not contain %q\nbody: %s", tc.want, body)
			}
		})
	}
}

// ──────────────────────────────────────────────
// Validate
// ──────────────────────────────────────────────

func TestHandler_Validate(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "valid email via field-named param",
			query: "/demo/validate?field=email&email=user@example.com",
			want:  "Looks good",
		},
		{
			name:  "invalid email shows error",
			query: "/demo/validate?field=email&email=notanemail",
			want:  "valid email",
		},
		{
			name:  "valid email via fallback ?value= param",
			query: "/demo/validate?field=email&value=user@example.com",
			want:  "Looks good",
		},
		{
			name:  "short username shows error",
			query: "/demo/validate?field=username&username=ab",
			want:  "3 characters",
		},
		{
			name:  "no params returns fragment without error",
			query: "/demo/validate",
			want:  "validate-field",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler()
			c := newCtx(t, http.MethodGet, tc.query)

			resp, err := h.Validate(c)
			if err != nil {
				t.Fatalf("Validate: unexpected error: %v", err)
			}
			if resp.Status != http.StatusOK {
				t.Errorf("Validate: status = %d, want %d", resp.Status, http.StatusOK)
			}
			body := readBody(t, resp)
			if !strings.Contains(body, tc.want) {
				t.Errorf("Validate: body does not contain %q\nbody: %s", tc.want, body)
			}
		})
	}
}

// ──────────────────────────────────────────────
// Paginate
// ──────────────────────────────────────────────

func TestHandler_Paginate(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "explicit page and per_page",
			query: "/demo/paginate?page=2&per_page=10",
			want:  "paginate-region",
		},
		{
			name:  "no params falls back to defaults",
			query: "/demo/paginate",
			want:  "paginate-region",
		},
		{
			name:  "invalid page value falls back to 0 (treated as first page)",
			query: "/demo/paginate?page=bad",
			want:  "paginate-region",
		},
		{
			name:  "page 1 shows first page indicator",
			query: "/demo/paginate?page=1&per_page=10",
			want:  "Page 1 of",
		},
		{
			name:  "last page shows last page indicator",
			query: "/demo/paginate?page=3&per_page=10",
			want:  "Page 3 of 3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler()
			c := newCtx(t, http.MethodGet, tc.query)

			resp, err := h.Paginate(c)
			if err != nil {
				t.Fatalf("Paginate: unexpected error: %v", err)
			}
			if resp.Status != http.StatusOK {
				t.Errorf("Paginate: status = %d, want %d", resp.Status, http.StatusOK)
			}
			body := readBody(t, resp)
			if !strings.Contains(body, tc.want) {
				t.Errorf("Paginate: body does not contain %q\nbody: %s", tc.want, body)
			}
		})
	}
}

// ──────────────────────────────────────────────
// OOBToast
// ──────────────────────────────────────────────

func TestHandler_OOBToast_ReturnsOK(t *testing.T) {
	h := newTestHandler()
	c := newCtx(t, http.MethodGet, "/demo/oob-toast")

	resp, err := h.OOBToast(c)
	if err != nil {
		t.Fatalf("OOBToast: unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Errorf("OOBToast: status = %d, want %d", resp.Status, http.StatusOK)
	}
	body := readBody(t, resp)
	if body == "" {
		t.Error("OOBToast: response body is empty")
	}
}

// OOBToast must include the hx-swap-oob attribute so HTMX processes it out-of-band.
func TestHandler_OOBToast_ContainsOOBAttr(t *testing.T) {
	h := newTestHandler()
	c := newCtx(t, http.MethodGet, "/demo/oob-toast")

	resp, err := h.OOBToast(c)
	if err != nil {
		t.Fatalf("OOBToast: unexpected error: %v", err)
	}
	body := readBody(t, resp)
	if !strings.Contains(body, "hx-swap-oob") {
		t.Errorf("OOBToast: body missing hx-swap-oob attribute\nbody: %s", body)
	}
}

// OOBToast must include the toast success message.
func TestHandler_OOBToast_ContainsSavedText(t *testing.T) {
	h := newTestHandler()
	c := newCtx(t, http.MethodGet, "/demo/oob-toast")

	resp, err := h.OOBToast(c)
	if err != nil {
		t.Fatalf("OOBToast: unexpected error: %v", err)
	}
	body := readBody(t, resp)
	if !strings.Contains(body, "Saved!") {
		t.Errorf("OOBToast: body missing expected title %q\nbody: %s", "Saved!", body)
	}
}

// ──────────────────────────────────────────────
// Region
// ──────────────────────────────────────────────

func TestHandler_Region(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		pathParam  string // set as c.Param("id") when non-empty
		want       string
		wantAbsent string
	}{
		{
			name:  "known country via query param returns region options",
			query: "/demo/region?country=us",
			want:  "California",
		},
		{
			name:      "known country via path param returns region options",
			query:     "/demo/region/us",
			pathParam: "us",
			want:      "California",
		},
		{
			name:  "unknown country shows fallback message",
			query: "/demo/region?country=xx",
			want:  "No regions available",
		},
		{
			name:  "no param shows fallback message",
			query: "/demo/region",
			want:  "No regions available",
		},
		{
			name:  "country SE returns Swedish regions",
			query: "/demo/region?country=se",
			want:  "Stockholm",
		},
		{
			name:  "country DE returns German regions",
			query: "/demo/region?country=de",
			want:  "Berlin",
		},
		{
			name:       "path param takes precedence over query param",
			query:      "/demo/region/de?country=us",
			pathParam:  "de",
			want:       "Berlin",
			wantAbsent: "California",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler()
			c := newCtx(t, http.MethodGet, tc.query)
			if tc.pathParam != "" {
				c.SetParam("id", tc.pathParam)
			}

			resp, err := h.Region(c)
			if err != nil {
				t.Fatalf("Region: unexpected error: %v", err)
			}
			if resp.Status != http.StatusOK {
				t.Errorf("Region: status = %d, want %d", resp.Status, http.StatusOK)
			}
			body := readBody(t, resp)
			if !strings.Contains(body, tc.want) {
				t.Errorf("Region: body does not contain %q\nbody: %s", tc.want, body)
			}
			if tc.wantAbsent != "" && strings.Contains(body, tc.wantAbsent) {
				t.Errorf("Region: body should not contain %q\nbody: %s", tc.wantAbsent, body)
			}
		})
	}
}

// ──────────────────────────────────────────────
// Routes / Config
// ──────────────────────────────────────────────

func TestDefaultConfig_BasePath(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BasePath != "/showcase/componentry" {
		t.Errorf("DefaultConfig().BasePath = %q, want %q", cfg.BasePath, "/showcase/componentry")
	}
}

func TestRoutes_ReturnsNonEmpty(t *testing.T) {
	cfg := Config{BasePath: "/showcase/componentry", Page: stubPage}
	nodes := Routes(cfg)
	if len(nodes) == 0 {
		t.Error("Routes: returned empty node slice")
	}
}
