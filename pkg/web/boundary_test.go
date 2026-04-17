package web_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-sum/web"
	"github.com/go-sum/web/adapt"
	"github.com/go-sum/web/router"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newBoundaryServer builds an httptest.Server with ErrorBoundary installed,
// a single GET /test route that returns the handler's result, and an HTTP
// client that never follows redirects.
func newBoundaryServer(t *testing.T, cfg web.BoundaryConfig, handler web.Handler) (srv *httptest.Server, client *http.Client) {
	t.Helper()

	r := router.NewWithoutSecureDefaults()
	r.Use(web.WithRequestID())
	r.Use(web.ErrorBoundary(cfg))
	r.GET("/test", "test", handler)

	srv = httptest.NewServer(adapt.ToHTTPHandler(r.Serve))
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	client = &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return srv, client
}

func doGet(t *testing.T, client *http.Client, url string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do GET %s: %v", url, err)
	}
	return resp
}

func readAllBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	return string(b)
}

func decodeJSON(t *testing.T, body string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	return m
}

// stubRenderer is an ErrorRenderer that returns a fixed HTML fragment.
type stubRenderer struct {
	body   string
	called bool
}

func (s *stubRenderer) RenderError(_ *web.Context, _ *web.Error) web.Response {
	s.called = true
	return web.HTML(http.StatusForbidden, s.body)
}

// statusCodeRenderer returns an HTML body encoding the status code.
// Used to verify Accept negotiation drives renderer selection.
type statusCodeRenderer struct{}

func (r *statusCodeRenderer) RenderError(_ *web.Context, e *web.Error) web.Response {
	return web.HTML(e.Status, fmt.Sprintf("<p>%d</p>", e.Status))
}

// ---------------------------------------------------------------------------
// a. No 5xx cause leak
// ---------------------------------------------------------------------------

func TestBoundary_No5xxCauseLeak(t *testing.T) {
	t.Parallel()

	handler := func(_ *web.Context) (web.Response, error) {
		return web.Response{}, web.ErrInternal(errors.New("db connection failed"))
	}

	srv, client := newBoundaryServer(t, web.BoundaryConfig{}, handler)
	resp := doGet(t, client, srv.URL+"/test", map[string]string{
		"Accept": "application/json",
	})
	body := readAllBody(t, resp)

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}

	if strings.Contains(body, "db connection failed") {
		t.Fatalf("body leaks cause text: %q", body)
	}

	doc := decodeJSON(t, body)
	if detail, ok := doc["detail"]; ok && detail != "" {
		t.Fatalf("body contains non-empty 'detail' field for 5xx: %q", detail)
	}
}

// ---------------------------------------------------------------------------
// b. Panic uses same problem+json envelope
// ---------------------------------------------------------------------------

func TestBoundary_PanicUsesProblemJSON(t *testing.T) {
	t.Parallel()

	panicInvoked := false
	var onPanic func(any, []byte) = func(val any, stack []byte) {
		panicInvoked = true
	}

	handler := func(_ *web.Context) (web.Response, error) {
		panic("intentional boom")
	}

	srv, client := newBoundaryServer(t, web.BoundaryConfig{OnPanic: onPanic}, handler)
	resp := doGet(t, client, srv.URL+"/test", map[string]string{
		"Accept": "application/json",
	})
	body := readAllBody(t, resp)

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}

	if !panicInvoked {
		t.Fatal("OnPanic callback was not invoked")
	}

	if rid := resp.Header.Get("X-Request-Id"); rid == "" {
		t.Fatal("X-Request-Id header is absent on panic response")
	}

	if strings.Contains(body, "intentional boom") {
		t.Fatalf("body leaks panic value: %q", body)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/problem+json") {
		t.Fatalf("Content-Type = %q, want application/problem+json", ct)
	}
}

// ---------------------------------------------------------------------------
// c. Sentinel classification (table-driven)
// ---------------------------------------------------------------------------

func TestBoundary_SentinelClassification(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "ErrBodyTooLarge wrapped",
			err:        fmt.Errorf("wrapped: %w", web.ErrBodyTooLarge),
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "ErrContentTypeMismatch",
			err:        web.ErrContentTypeMismatch,
			wantStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:       "context.DeadlineExceeded wrapped",
			err:        fmt.Errorf("timed out: %w", context.DeadlineExceeded),
			wantStatus: 499,
		},
		{
			name:       "context.Canceled wrapped",
			err:        fmt.Errorf("canceled: %w", context.Canceled),
			wantStatus: 499,
		},
		{
			name:       "unknown error",
			err:        errors.New("unknown"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "ErrBadRequest",
			err:        web.ErrBadRequest("bad input"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "ErrNotFound",
			err:        web.ErrNotFound(""),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "ErrForbidden",
			err:        web.ErrForbidden("no access"),
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			capturedErr := tc.err
			handler := func(_ *web.Context) (web.Response, error) {
				return web.Response{}, capturedErr
			}

			srv, client := newBoundaryServer(t, web.BoundaryConfig{}, handler)
			resp := doGet(t, client, srv.URL+"/test", map[string]string{
				"Accept": "application/json",
			})
			_ = readAllBody(t, resp)

			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// d. Request-ID correlation
// ---------------------------------------------------------------------------

func TestBoundary_RequestIDCorrelation(t *testing.T) {
	t.Parallel()

	handler := func(_ *web.Context) (web.Response, error) {
		return web.Response{}, web.ErrBadRequest("bad")
	}

	srv, client := newBoundaryServer(t, web.BoundaryConfig{}, handler)
	resp := doGet(t, client, srv.URL+"/test", map[string]string{
		"Accept": "application/json",
	})
	body := readAllBody(t, resp)

	rid := resp.Header.Get("X-Request-Id")
	if rid == "" {
		t.Fatal("X-Request-Id response header is absent")
	}

	doc := decodeJSON(t, body)
	docRID, _ := doc["request_id"].(string)
	if docRID == "" {
		t.Fatalf("JSON body missing 'request_id' field; body = %q", body)
	}
	if docRID != rid {
		t.Fatalf("request_id in body = %q, want header value %q", docRID, rid)
	}
}

// ---------------------------------------------------------------------------
// e. Problem+JSON is fully buffered (Content-Length matches body)
// ---------------------------------------------------------------------------

func TestBoundary_ProblemJSONIsBuffered(t *testing.T) {
	t.Parallel()

	handler := func(_ *web.Context) (web.Response, error) {
		return web.Response{}, web.ErrBadRequest("some error")
	}

	srv, client := newBoundaryServer(t, web.BoundaryConfig{}, handler)
	resp := doGet(t, client, srv.URL+"/test", map[string]string{
		"Accept": "application/json",
	})
	body := readAllBody(t, resp)

	cl := resp.ContentLength
	if cl < 0 {
		t.Fatalf("Content-Length not set (got %d) — response may be using io.Pipe", cl)
	}
	if int(cl) != len(body) {
		t.Fatalf("Content-Length = %d, body len = %d — mismatch indicates streaming", cl, len(body))
	}
}

// ---------------------------------------------------------------------------
// f. HEAD method — body must be empty
// ---------------------------------------------------------------------------

func TestBoundary_HEADBodyEmpty(t *testing.T) {
	t.Parallel()

	r := router.NewWithoutSecureDefaults()
	r.Use(web.ErrorBoundary(web.BoundaryConfig{}))
	r.GET("/test", "test", func(_ *web.Context) (web.Response, error) {
		return web.Response{}, web.ErrNotFound("")
	})

	srv := httptest.NewServer(adapt.ToHTTPHandler(r.Serve))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodHead, srv.URL+"/test", nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HEAD /test: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
	if len(body) != 0 {
		t.Fatalf("HEAD body = %q, want empty", body)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/problem+json") {
		t.Fatalf("Content-Type = %q, want application/problem+json", ct)
	}
}

// ---------------------------------------------------------------------------
// g. HTMX fragment mode
// ---------------------------------------------------------------------------

func TestBoundary_HTMXFragmentMode(t *testing.T) {
	t.Parallel()

	const fragmentBody = "<div>error fragment</div>"
	renderer := &stubRenderer{body: fragmentBody}

	// Override the renderer so it always returns 403 for the fixed body.
	// We need a renderer that preserves the status from the error.
	fixedRenderer := &fixedStatusRenderer{body: fragmentBody}

	handler := func(_ *web.Context) (web.Response, error) {
		return web.Response{}, web.ErrForbidden("nope")
	}

	srv, client := newBoundaryServer(t, web.BoundaryConfig{Renderer: fixedRenderer}, handler)
	resp := doGet(t, client, srv.URL+"/test", map[string]string{
		"HX-Request": "true",
	})
	body := readAllBody(t, resp)

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
	if body != fragmentBody {
		t.Fatalf("body = %q, want %q", body, fragmentBody)
	}
	_ = renderer

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
}

// fixedStatusRenderer returns the given HTML body with the status from the error.
type fixedStatusRenderer struct {
	body string
}

func (r *fixedStatusRenderer) RenderError(_ *web.Context, e *web.Error) web.Response {
	return web.HTML(e.Status, r.body)
}

// ---------------------------------------------------------------------------
// h. Accept negotiation
// ---------------------------------------------------------------------------

func TestBoundary_AcceptNegotiation(t *testing.T) {
	t.Parallel()

	handler := func(_ *web.Context) (web.Response, error) {
		return web.Response{}, web.ErrNotFound("")
	}

	t.Run("Accept application/json uses problem+json", func(t *testing.T) {
		t.Parallel()
		// No renderer needed — JSON path does not call renderer.
		srv, client := newBoundaryServer(t, web.BoundaryConfig{}, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "application/json",
		})
		_ = readAllBody(t, resp)

		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "application/problem+json") {
			t.Fatalf("Content-Type = %q, want application/problem+json", ct)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("Accept text/html uses renderer", func(t *testing.T) {
		t.Parallel()
		srv, client := newBoundaryServer(t, web.BoundaryConfig{Renderer: &statusCodeRenderer{}}, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "text/html",
		})
		body := readAllBody(t, resp)

		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			t.Fatalf("Content-Type = %q, want text/html", ct)
		}
		// Renderer returns "<p>404</p>" for status 404.
		if body != "<p>404</p>" {
			t.Fatalf("body = %q, want %q — renderer was not called", body, "<p>404</p>")
		}
	})

	t.Run("Accept wildcard uses renderer", func(t *testing.T) {
		t.Parallel()
		// Empty Accept header (browser-friendly default) should use renderer.
		srv, client := newBoundaryServer(t, web.BoundaryConfig{Renderer: &statusCodeRenderer{}}, handler)
		resp := doGet(t, client, srv.URL+"/test", nil)
		body := readAllBody(t, resp)

		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			t.Fatalf("Content-Type = %q, want text/html (renderer used for empty Accept)", ct)
		}
		if body != "<p>404</p>" {
			t.Fatalf("body = %q, want %q — renderer was not called", body, "<p>404</p>")
		}
	})
}

// ---------------------------------------------------------------------------
// i. Retry-After header
// ---------------------------------------------------------------------------

func TestBoundary_RetryAfterHeader(t *testing.T) {
	t.Parallel()

	handler := func(_ *web.Context) (web.Response, error) {
		return web.Response{}, web.ErrTooManyRequests(5 * time.Second)
	}

	srv, client := newBoundaryServer(t, web.BoundaryConfig{}, handler)
	resp := doGet(t, client, srv.URL+"/test", map[string]string{
		"Accept": "application/json",
	})
	body := readAllBody(t, resp)

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusTooManyRequests)
	}

	ra := resp.Header.Get("Retry-After")
	if ra != "5" {
		t.Fatalf("Retry-After = %q, want %q", ra, "5")
	}

	doc := decodeJSON(t, body)
	retryAfterVal, ok := doc["retry_after"]
	if !ok {
		t.Fatalf("JSON body missing 'retry_after' field; body = %q", body)
	}
	// JSON numbers unmarshal as float64.
	if retryAfterVal.(float64) != 5 {
		t.Fatalf("retry_after = %v, want 5", retryAfterVal)
	}
}

// ---------------------------------------------------------------------------
// j. 4xx logged at Warn, 5xx logged at Error
// ---------------------------------------------------------------------------

func TestBoundary_LogLevels(t *testing.T) {
	t.Parallel()

	t.Run("4xx logged at WARN", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		handler := func(_ *web.Context) (web.Response, error) {
			return web.Response{}, web.ErrBadRequest("bad")
		}

		srv, client := newBoundaryServer(t, web.BoundaryConfig{Logger: logger}, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "application/json",
		})
		_ = readAllBody(t, resp)

		logged := buf.String()
		if !strings.Contains(logged, "WARN") {
			t.Fatalf("log output does not contain WARN; got:\n%s", logged)
		}
		if strings.Contains(logged, "ERROR") {
			t.Fatalf("log output must not contain ERROR for 4xx; got:\n%s", logged)
		}
	})

	t.Run("5xx logged at ERROR", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		handler := func(_ *web.Context) (web.Response, error) {
			return web.Response{}, web.ErrInternal(errors.New("x"))
		}

		srv, client := newBoundaryServer(t, web.BoundaryConfig{Logger: logger}, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "application/json",
		})
		_ = readAllBody(t, resp)

		logged := buf.String()
		if !strings.Contains(logged, "ERROR") {
			t.Fatalf("log output does not contain ERROR; got:\n%s", logged)
		}
	})
}

// ---------------------------------------------------------------------------
// G2 — Client cancellation is non-fault (DEBUG level)
// ---------------------------------------------------------------------------

func TestBoundary_ClientCancellation_LogsDebug(t *testing.T) {
	t.Parallel()

	t.Run("context.Canceled logs at DEBUG not WARN", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		handler := func(_ *web.Context) (web.Response, error) {
			return web.Response{}, fmt.Errorf("request aborted: %w", context.Canceled)
		}

		srv, client := newBoundaryServer(t, web.BoundaryConfig{Logger: logger}, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "application/json",
		})
		_ = readAllBody(t, resp)

		logged := buf.String()
		if !strings.Contains(logged, "DEBUG") {
			t.Fatalf("expected DEBUG log for context.Canceled; got:\n%s", logged)
		}
		if strings.Contains(logged, "WARN") {
			t.Fatalf("must not log at WARN for context.Canceled; got:\n%s", logged)
		}
	})

	t.Run("status 499 logs at DEBUG", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		// Classify(context.Canceled) maps to 499; confirm no WARN-level entry.
		handler := func(_ *web.Context) (web.Response, error) {
			return web.Response{}, context.Canceled
		}

		srv, client := newBoundaryServer(t, web.BoundaryConfig{Logger: logger}, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "application/json",
		})
		_ = readAllBody(t, resp)

		if resp.StatusCode != 499 {
			t.Fatalf("status = %d, want 499", resp.StatusCode)
		}

		logged := buf.String()
		if strings.Contains(logged, "WARN") {
			t.Fatalf("must not log at WARN for 499 response; got:\n%s", logged)
		}
		if !strings.Contains(logged, "DEBUG") {
			t.Fatalf("expected DEBUG log for 499 response; got:\n%s", logged)
		}
	})

	t.Run("5xx still logs at ERROR", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		handler := func(_ *web.Context) (web.Response, error) {
			return web.Response{}, errors.New("unexpected failure")
		}

		srv, client := newBoundaryServer(t, web.BoundaryConfig{Logger: logger}, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "application/json",
		})
		_ = readAllBody(t, resp)

		if resp.StatusCode != 500 {
			t.Fatalf("status = %d, want 500", resp.StatusCode)
		}

		logged := buf.String()
		if !strings.Contains(logged, "ERROR") {
			t.Fatalf("expected ERROR log for 5xx; got:\n%s", logged)
		}
	})
}

// ---------------------------------------------------------------------------
// G5 — CaptureStack flag
// ---------------------------------------------------------------------------

func TestBoundary_CaptureStack(t *testing.T) {
	t.Parallel()

	t.Run("CaptureStack true: stack attr present in http.error on 5xx", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		handler := func(_ *web.Context) (web.Response, error) {
			return web.Response{}, errors.New("internal failure")
		}

		srv, client := newBoundaryServer(t, web.BoundaryConfig{
			CaptureStack: true,
			Logger:       logger,
		}, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "application/json",
		})
		_ = readAllBody(t, resp)

		if resp.StatusCode != 500 {
			t.Fatalf("status = %d, want 500", resp.StatusCode)
		}

		logged := buf.String()
		if !strings.Contains(logged, "stack=") {
			t.Fatalf("expected 'stack=' attr in log for 5xx with CaptureStack=true; got:\n%s", logged)
		}
	})

	t.Run("CaptureStack true: no stack attr on 4xx", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		handler := func(_ *web.Context) (web.Response, error) {
			return web.Response{}, web.ErrBadRequest("bad input")
		}

		srv, client := newBoundaryServer(t, web.BoundaryConfig{
			CaptureStack: true,
			Logger:       logger,
		}, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "application/json",
		})
		_ = readAllBody(t, resp)

		if resp.StatusCode != 400 {
			t.Fatalf("status = %d, want 400", resp.StatusCode)
		}

		logged := buf.String()
		if strings.Contains(logged, "stack=") {
			t.Fatalf("must not emit 'stack=' attr for 4xx; got:\n%s", logged)
		}
	})

	t.Run("CaptureStack false: no stack attr on 5xx", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		handler := func(_ *web.Context) (web.Response, error) {
			return web.Response{}, errors.New("internal failure")
		}

		// CaptureStack defaults to false.
		srv, client := newBoundaryServer(t, web.BoundaryConfig{Logger: logger}, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "application/json",
		})
		_ = readAllBody(t, resp)

		if resp.StatusCode != 500 {
			t.Fatalf("status = %d, want 500", resp.StatusCode)
		}

		logged := buf.String()
		if strings.Contains(logged, "stack=") {
			t.Fatalf("must not emit 'stack=' attr when CaptureStack=false; got:\n%s", logged)
		}
	})
}

// ---------------------------------------------------------------------------
// G12 — Per-field extractor funcs
// ---------------------------------------------------------------------------

func TestBoundary_ExtractorFuncs(t *testing.T) {
	t.Parallel()

	t.Run("nil extractors: op/subsystem/trace_id/span_id/dedupe_key absent from log", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		handler := func(_ *web.Context) (web.Response, error) {
			return web.Response{}, errors.New("failure")
		}

		// All extractors are nil by default.
		srv, client := newBoundaryServer(t, web.BoundaryConfig{Logger: logger}, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "application/json",
		})
		_ = readAllBody(t, resp)

		logged := buf.String()
		for _, field := range []string{"op=", "subsystem=", "trace_id=", "span_id=", "dedupe_key="} {
			if strings.Contains(logged, field) {
				t.Errorf("log must not contain %q when extractor is nil; got:\n%s", field, logged)
			}
		}
	})

	t.Run("non-nil extractors: all five fields emitted when non-empty", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		handler := func(_ *web.Context) (web.Response, error) {
			return web.Response{}, errors.New("failure")
		}

		cfg := web.BoundaryConfig{
			Logger:    logger,
			Op:        func(_ *web.Context) string { return "user.create" },
			Subsystem: func(_ *web.Context) string { return "auth" },
			TraceID:   func(_ *web.Context) string { return "trace-abc" },
			SpanID:    func(_ *web.Context) string { return "span-xyz" },
			DedupeKey: func(_ *web.Context) string { return "dedup-1" },
		}

		srv, client := newBoundaryServer(t, cfg, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "application/json",
		})
		_ = readAllBody(t, resp)

		logged := buf.String()
		for field, want := range map[string]string{
			"op":         "user.create",
			"subsystem":  "auth",
			"trace_id":   "trace-abc",
			"span_id":    "span-xyz",
			"dedupe_key": "dedup-1",
		} {
			if !strings.Contains(logged, field+"="+want) {
				t.Errorf("expected log to contain %s=%s; got:\n%s", field, want, logged)
			}
		}
	})

	t.Run("extractor returning empty string omits the field", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		handler := func(_ *web.Context) (web.Response, error) {
			return web.Response{}, errors.New("failure")
		}

		cfg := web.BoundaryConfig{
			Logger: logger,
			Op:     func(_ *web.Context) string { return "" }, // returns empty — must be omitted
		}

		srv, client := newBoundaryServer(t, cfg, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "application/json",
		})
		_ = readAllBody(t, resp)

		logged := buf.String()
		if strings.Contains(logged, "op=") {
			t.Fatalf("log must not contain 'op=' when extractor returns empty string; got:\n%s", logged)
		}
	})

	t.Run("extractors work in panic path too", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		handler := func(_ *web.Context) (web.Response, error) {
			panic("deliberate panic for extractor test")
		}

		cfg := web.BoundaryConfig{
			Logger: logger,
			Op:     func(_ *web.Context) string { return "panic.op" },
		}

		srv, client := newBoundaryServer(t, cfg, handler)
		resp := doGet(t, client, srv.URL+"/test", map[string]string{
			"Accept": "application/json",
		})
		_ = readAllBody(t, resp)

		if resp.StatusCode != 500 {
			t.Fatalf("status = %d, want 500 on panic", resp.StatusCode)
		}

		logged := buf.String()
		if !strings.Contains(logged, "op=panic.op") {
			t.Fatalf("expected 'op=panic.op' in panic log; got:\n%s", logged)
		}
	})
}
