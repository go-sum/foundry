package hello

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/page"
	"github.com/go-sum/web"
	"github.com/go-sum/web/render"
	"github.com/go-sum/web/router"
)

func testRouter(t *testing.T) *router.Router {
	t.Helper()
	rt := router.New()
	router.Register(rt,
		router.GET("/hello/greeting", "hello.greeting", nil),
		router.GET("/", "home.show", nil),
	)
	return rt
}

func TestHandlerGreeting(t *testing.T) {
	h := NewHandler(nil)

	u, _ := url.Parse("/hello/greeting?name=Alice")
	req := web.NewRequest(http.MethodGet, u)
	resp, err := h.Greeting(web.NewContext(context.Background(), req))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	want := render.RenderNode(t, page.HelloPartial("Alice"))
	if string(body) != want {
		t.Fatalf("body mismatch")
	}
}

func TestHandlerGreeting_EmptyName_DefaultsToWorld(t *testing.T) {
	h := NewHandler(nil)

	u, _ := url.Parse("/hello/greeting")
	req := web.NewRequest(http.MethodGet, u)
	resp, err := h.Greeting(web.NewContext(context.Background(), req))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	want := render.RenderNode(t, page.HelloPartial("World"))
	if string(body) != want {
		t.Fatalf("body mismatch\ngot:  %s\nwant: %s", string(body), want)
	}
}

func TestHandlerShow(t *testing.T) {
	h := NewHandler(testRouter(t))

	u, _ := url.Parse("/hello/Alice")
	req := web.NewRequest(http.MethodGet, u)
	c := web.NewContext(context.Background(), req)
	c.SetParam("name", "Alice")
	resp, err := h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	vr := view.NewRequest(c)
	want := render.RenderNode(t, page.HelloPage(vr, "Alice", "/hello/greeting", "/"))
	if string(body) != want {
		t.Fatalf("body mismatch")
	}

	req.Headers.Set("HX-Request", "true")
	c = web.NewContext(context.Background(), req)
	c.SetParam("name", "Alice")
	resp, err = h.Show(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	want = render.RenderNode(t, page.HelloPartial("Alice"))
	if string(body) != want {
		t.Fatalf("partial body mismatch")
	}
}

func TestHandlerShow_InvalidName_ReturnsBadRequest(t *testing.T) {
	h := NewHandler(nil)

	u, _ := url.Parse("/hello/bad!!!")
	req := web.NewRequest(http.MethodGet, u)
	c := web.NewContext(context.Background(), req)
	c.SetParam("name", "bad!!!")

	resp, err := h.Show(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Response should be zero value when an error is returned.
	if resp.Status != 0 {
		t.Errorf("expected zero response, got status %d", resp.Status)
	}
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", webErr.Status, http.StatusBadRequest)
	}
	if webErr.Code != web.CodeBadRequest {
		t.Errorf("Code = %q, want %q", webErr.Code, web.CodeBadRequest)
	}
}

func TestHandlerShow_LongName_ReturnsBadRequest(t *testing.T) {
	h := NewHandler(nil)

	longName := strings.Repeat("a", 65) // 65 chars — exceeds 64-char limit
	u, _ := url.Parse("/hello/" + longName)
	req := web.NewRequest(http.MethodGet, u)
	c := web.NewContext(context.Background(), req)
	c.SetParam("name", longName)

	resp, err := h.Show(c)
	if err == nil {
		t.Fatal("expected error for 65-char name, got nil")
	}
	if resp.Status != 0 {
		t.Errorf("expected zero response for error case, got status %d", resp.Status)
	}
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", webErr.Status, http.StatusBadRequest)
	}
	if webErr.Code != web.CodeBadRequest {
		t.Errorf("Code = %q, want %q", webErr.Code, web.CodeBadRequest)
	}
}

func TestIsValidName(t *testing.T) {
	cases := []struct {
		name  string
		valid bool
	}{
		{"Alice", true},
		{"World", true},
		{"", false},
		{"bad!!!", false},
		{"123", false},
		{"Alice123", false},
		{"Ünïcödé", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isValidName(tc.name)
			if got != tc.valid {
				t.Errorf("isValidName(%q) = %v, want %v", tc.name, got, tc.valid)
			}
		})
	}
}
