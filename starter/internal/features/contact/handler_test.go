package contact

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"
	"github.com/go-sum/foundry/internal/view/page"
	"github.com/go-sum/foundry/internal/view/partial/contactpartial"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/ratelimit"
	"github.com/go-sum/foundry/pkg/web/render"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"
)

// fakeService is a manual implementation of Service for handler tests.
type fakeService struct {
	err error
}

func (f *fakeService) Submit(_ context.Context, _ ContactInput, _ string) error {
	return f.err
}

func testContactRouter(t *testing.T) *router.Router {
	t.Helper()
	rt := router.New()
	router.Register(rt,
		router.GET("/contact", "contact.form", nil),
		router.POST("/contact", "contact.submit", nil),
	)
	return rt
}

func newContactHandler(t *testing.T, svc Service) *Handler {
	t.Helper()
	rt := testContactRouter(t)
	val := validate.New()
	return NewHandler(rt, svc, val, ratelimit.RealIP)
}

func TestHandler_Form_RendersPage(t *testing.T) {
	h := newContactHandler(t, &fakeService{})

	u, _ := url.Parse("/contact")
	req := web.NewRequest(http.MethodGet, u)
	c := web.NewContext(context.Background(), req)

	resp, err := h.Form(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	got := string(body)

	vr := viewstate.NewRequest(c)
	submitURL := testContactRouter(t).MustReverse("contact.submit", nil)
	want := render.RenderNode(t, page.ContactPage(vr, submitURL, contactpartial.FormData{}))
	if got != want {
		t.Fatalf("body mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestHandler_Form_HTMXPartial(t *testing.T) {
	h := newContactHandler(t, &fakeService{})

	u, _ := url.Parse("/contact")
	req := web.NewRequest(http.MethodGet, u)
	req.Headers.Set("HX-Request", "true")
	c := web.NewContext(context.Background(), req)

	resp, err := h.Form(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	got := string(body)

	vr := viewstate.NewRequest(c)
	submitURL := testContactRouter(t).MustReverse("contact.submit", nil)
	want := render.RenderNode(t, contactpartial.ContactForm(vr, submitURL, contactpartial.FormData{}))
	if got != want {
		t.Fatalf("body mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

// TestHandler_Submit_ValidationError verifies that a POST with missing fields
// returns a 422 fragment containing field error markup.
func TestHandler_Submit_ValidationError(t *testing.T) {
	h := newContactHandler(t, &fakeService{})

	formBody := url.Values{
		"name":    {""},
		"email":   {""},
		"message": {""},
	}.Encode()

	u, _ := url.Parse("/contact")
	req := web.NewRequest(http.MethodPost, u)
	req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBody(io.NopCloser(strings.NewReader(formBody)))
	c := web.NewContext(context.Background(), req)

	resp, err := h.Submit(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != http.StatusUnprocessableEntity {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusUnprocessableEntity)
	}

	body, _ := io.ReadAll(resp.Body)
	got := string(body)

	vr := viewstate.NewRequest(c)
	submitURL := testContactRouter(t).MustReverse("contact.submit", nil)
	fieldErrors := map[string][]string{
		"name":    {"is required"},
		"email":   {"is required"},
		"message": {"is required"},
	}
	want := render.RenderNode(t, contactpartial.ContactForm(vr, submitURL, contactpartial.FormData{
		Errors: fieldErrors,
	}))
	if got != want {
		t.Fatalf("body mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

// TestHandler_Submit_Success verifies that a valid POST returns a 200 fragment
// showing the success state.
func TestHandler_Submit_Success(t *testing.T) {
	h := newContactHandler(t, &fakeService{err: nil})

	formBody := url.Values{
		"name":    {"Alice"},
		"email":   {"alice@example.com"},
		"message": {"Hello there"},
	}.Encode()

	u, _ := url.Parse("/contact")
	req := web.NewRequest(http.MethodPost, u)
	req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetRemoteAddr("127.0.0.1:1234")
	req.SetBody(io.NopCloser(strings.NewReader(formBody)))
	c := web.NewContext(context.Background(), req)

	resp, err := h.Submit(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	got := string(body)

	vr := viewstate.NewRequest(c)
	submitURL := testContactRouter(t).MustReverse("contact.submit", nil)
	want := render.RenderNode(t, contactpartial.ContactForm(vr, submitURL, contactpartial.FormData{Sent: true}))
	if got != want {
		t.Fatalf("body mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

// TestHandler_Submit_RateLimited verifies that ErrRateLimited produces a 429 *web.Error.
func TestHandler_Submit_RateLimited(t *testing.T) {
	h := newContactHandler(t, &fakeService{err: ErrRateLimited})

	formBody := url.Values{
		"name":    {"Alice"},
		"email":   {"alice@example.com"},
		"message": {"Hello there"},
	}.Encode()

	u, _ := url.Parse("/contact")
	req := web.NewRequest(http.MethodPost, u)
	req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetRemoteAddr("127.0.0.1:1234")
	req.SetBody(io.NopCloser(strings.NewReader(formBody)))
	c := web.NewContext(context.Background(), req)

	resp, err := h.Submit(c)
	if err == nil {
		t.Fatal("expected error for rate limited submission, got nil")
	}
	if resp.Status != 0 {
		t.Errorf("expected zero response on error, got status %d", resp.Status)
	}

	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusTooManyRequests {
		t.Errorf("Status = %d, want %d", webErr.Status, http.StatusTooManyRequests)
	}
	if webErr.Code != web.CodeTooManyRequests {
		t.Errorf("Code = %q, want %q", webErr.Code, web.CodeTooManyRequests)
	}
}

// TestHandler_Submit_ServiceError verifies that an unexpected service error
// produces a 503 *web.Error.
func TestHandler_Submit_ServiceError(t *testing.T) {
	unexpected := errors.New("database exploded")
	h := newContactHandler(t, &fakeService{err: unexpected})

	formBody := url.Values{
		"name":    {"Alice"},
		"email":   {"alice@example.com"},
		"message": {"Hello there"},
	}.Encode()

	u, _ := url.Parse("/contact")
	req := web.NewRequest(http.MethodPost, u)
	req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetRemoteAddr("127.0.0.1:1234")
	req.SetBody(io.NopCloser(strings.NewReader(formBody)))
	c := web.NewContext(context.Background(), req)

	resp, err := h.Submit(c)
	if err == nil {
		t.Fatal("expected error for service failure, got nil")
	}
	if resp.Status != 0 {
		t.Errorf("expected zero response on error, got status %d", resp.Status)
	}

	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want %d", webErr.Status, http.StatusServiceUnavailable)
	}
	if webErr.Code != web.CodeUnavailable {
		t.Errorf("Code = %q, want %q", webErr.Code, web.CodeUnavailable)
	}
	// Cause chain must reach the original error.
	if !errors.Is(webErr, unexpected) {
		t.Errorf("expected cause chain to contain original error")
	}
}

// TestHandler_Submit_ValidationError_PartialFields tests a partial submission
// (only some fields missing) still returns 422 with field errors.
func TestHandler_Submit_ValidationError_PartialFields(t *testing.T) {
	cases := []struct {
		name   string
		values url.Values
	}{
		{
			name:   "missing email and message",
			values: url.Values{"name": {"Alice"}, "email": {""}, "message": {""}},
		},
		{
			name:   "missing name only",
			values: url.Values{"name": {""}, "email": {"alice@example.com"}, "message": {"Hello"}},
		},
		{
			name:   "invalid email format",
			values: url.Values{"name": {"Alice"}, "email": {"not-an-email"}, "message": {"Hello"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newContactHandler(t, &fakeService{})

			u, _ := url.Parse("/contact")
			req := web.NewRequest(http.MethodPost, u)
			req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
			req.SetBody(io.NopCloser(strings.NewReader(tc.values.Encode())))
			c := web.NewContext(context.Background(), req)

			resp, err := h.Submit(c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Status != http.StatusUnprocessableEntity {
				t.Errorf("Status = %d, want %d", resp.Status, http.StatusUnprocessableEntity)
			}
		})
	}
}
