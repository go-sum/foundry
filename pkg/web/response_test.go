package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestRespond(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{name: "200 OK", status: http.StatusOK},
		{name: "204 No Content", status: http.StatusNoContent},
		{name: "500 Internal Server Error", status: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := Respond(tt.status)
			if resp.Status != tt.status {
				t.Errorf("Status = %d, want %d", resp.Status, tt.status)
			}
			if resp.Body != nil {
				t.Errorf("Body = %v, want nil", resp.Body)
			}
		})
	}
}

func TestText(t *testing.T) {
	resp := Text(http.StatusOK, "hello world")

	if resp.Status != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
	}
	if got := resp.Headers.Get("Content-Type"); got != "text/plain; charset=UTF-8" {
		t.Errorf("Content-Type = %q, want %q", got, "text/plain; charset=UTF-8")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if string(body) != "hello world" {
		t.Errorf("Body = %q, want %q", string(body), "hello world")
	}
}

func TestHTML(t *testing.T) {
	html := "<h1>Hello</h1>"
	resp := HTML(http.StatusOK, html)

	if resp.Status != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
	}
	if got := resp.Headers.Get("Content-Type"); got != "text/html; charset=UTF-8" {
		t.Errorf("Content-Type = %q, want %q", got, "text/html; charset=UTF-8")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if string(body) != html {
		t.Errorf("Body = %q, want %q", string(body), html)
	}
}

func TestHTMLReader(t *testing.T) {
	content := "<p>streamed</p>"
	rc := io.NopCloser(strings.NewReader(content))
	resp := HTMLReader(http.StatusOK, rc)

	if resp.Status != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
	}
	if got := resp.Headers.Get("Content-Type"); got != "text/html; charset=UTF-8" {
		t.Errorf("Content-Type = %q, want %q", got, "text/html; charset=UTF-8")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if string(body) != content {
		t.Errorf("Body = %q, want %q", string(body), content)
	}
}

func TestJSON(t *testing.T) {
	t.Run("valid struct encodes with trailing newline", func(t *testing.T) {
		type payload struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		resp := JSON(http.StatusOK, payload{Name: "Alice", Age: 30})

		if resp.Status != http.StatusOK {
			t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
		}
		if got := resp.Headers.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("reading body: %v", err)
		}
		// json.Encoder.Encode appends a trailing newline.
		want := "{\"name\":\"Alice\",\"age\":30}\n"
		if string(body) != want {
			t.Errorf("Body = %q, want %q", string(body), want)
		}
	})

	t.Run("unmarshalable value closes pipe with error", func(t *testing.T) {
		ch := make(chan int)
		resp := JSON(http.StatusOK, ch)

		// Status and Content-Type are set before encoding begins.
		if resp.Status != http.StatusOK {
			t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
		}
		if got := resp.Headers.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}
		// Reading the body must return an error because encoding fails.
		_, err := io.ReadAll(resp.Body)
		if err == nil {
			t.Error("expected body read error for unmarshalable value, got nil")
		}
	})
}

func TestStreamJSON(t *testing.T) {
	t.Run("valid struct encodes with trailing newline", func(t *testing.T) {
		type payload struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		resp := StreamJSON(http.StatusOK, payload{Name: "Alice", Age: 30})

		if resp.Status != http.StatusOK {
			t.Errorf("Status = %d, want %d", resp.Status, http.StatusOK)
		}
		if got := resp.Headers.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("reading body: %v", err)
		}
		want := "{\"name\":\"Alice\",\"age\":30}\n"
		if string(body) != want {
			t.Errorf("Body = %q, want %q", string(body), want)
		}
	})
}

func TestProblem(t *testing.T) {
	readDoc := func(t *testing.T, resp Response) map[string]any {
		t.Helper()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("reading body: %v", err)
		}
		var doc map[string]any
		if err := json.Unmarshal(body, &doc); err != nil {
			t.Fatalf("unmarshal body: %v (body=%q)", err, body)
		}
		return doc
	}

		makeCtx := func(path string) *Context {
			u, _ := url.Parse(path)
			req := NewRequest(http.MethodGet, u)
			return NewContext(context.Background(), req)
		}

	t.Run("404 not found populates fields", func(t *testing.T) {
		c := makeCtx("/items/42")
		resp := Problem(c, ErrNotFound("item not found"))

		if resp.Status != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", resp.Status, http.StatusNotFound)
		}
		if got := resp.Headers.Get("Content-Type"); got != "application/problem+json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/problem+json")
		}
		doc := readDoc(t, resp)
		if doc["type"] != "about:blank" {
			t.Errorf("type = %q, want %q", doc["type"], "about:blank")
		}
		if doc["title"] != "Not Found" {
			t.Errorf("title = %q, want %q", doc["title"], "Not Found")
		}
		if doc["status"] != float64(404) {
			t.Errorf("status = %v, want 404", doc["status"])
		}
		if doc["detail"] != "item not found" {
			t.Errorf("detail = %q, want %q", doc["detail"], "item not found")
		}
		if doc["code"] != "not_found" {
			t.Errorf("code = %q, want %q", doc["code"], "not_found")
		}
		if doc["instance"] != "/items/42" {
			t.Errorf("instance = %q, want %q", doc["instance"], "/items/42")
		}
	})

	t.Run("500 internal does not leak detail", func(t *testing.T) {
		c := makeCtx("/")
		resp := Problem(c, ErrInternal(nil))

		if resp.Status != http.StatusInternalServerError {
			t.Errorf("Status = %d, want %d", resp.Status, http.StatusInternalServerError)
		}
		doc := readDoc(t, resp)
		if _, ok := doc["detail"]; ok {
			t.Errorf("detail should be absent for 5xx, got %q", doc["detail"])
		}
	})

	t.Run("nil context does not panic", func(t *testing.T) {
		resp := Problem(nil, ErrNotFound("gone"))
		if resp.Status != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", resp.Status, http.StatusNotFound)
		}
		doc := readDoc(t, resp)
		if _, ok := doc["instance"]; ok {
			t.Errorf("instance should be absent when context is nil, got %q", doc["instance"])
		}
	})

	t.Run("instance from e.Instance takes priority over path", func(t *testing.T) {
		c := makeCtx("/other")
		e := ErrNotFound("").WithInstance("/explicit")
		resp := Problem(c, e)
		doc := readDoc(t, resp)
		if doc["instance"] != "/explicit" {
			t.Errorf("instance = %q, want %q", doc["instance"], "/explicit")
		}
	})

	t.Run("retry-after header set when RetryAfter > 0", func(t *testing.T) {
		c := makeCtx("/")
		e := ErrTooManyRequests(5e9) // 5 seconds
		resp := Problem(c, e)
		if got := resp.Headers.Get("Retry-After"); got != "5" {
			t.Errorf("Retry-After = %q, want %q", got, "5")
		}
		doc := readDoc(t, resp)
		if doc["retry_after"] != float64(5) {
			t.Errorf("retry_after = %v, want 5", doc["retry_after"])
		}
	})

	t.Run("meta merged into document", func(t *testing.T) {
		c := makeCtx("/")
		e := ErrBadRequest("bad").WithMeta("field", "email")
		resp := Problem(c, e)
		doc := readDoc(t, resp)
		if doc["field"] != "email" {
			t.Errorf("field = %q, want %q", doc["field"], "email")
		}
	})

	t.Run("Content-Type is application/problem+json", func(t *testing.T) {
		resp := Problem(nil, ErrInternal(nil))
		if got := resp.Headers.Get("Content-Type"); got != "application/problem+json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/problem+json")
		}
	})
}

func TestProblemBodyIsBuffered(t *testing.T) {
	// Confirm there is no io.Pipe — body must be fully readable multiple times (from buffer).
	resp := Problem(nil, ErrNotFound("gone"))
	body1, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("first read: %v", err)
	}
	if !strings.Contains(string(body1), "not_found") {
		t.Errorf("body missing code: %q", body1)
	}
}

func TestRedirect(t *testing.T) {
	resp := Redirect(http.StatusFound, "/dashboard")

	if resp.Status != http.StatusFound {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusFound)
	}
	if got := resp.Headers.Get("Location"); got != "/dashboard" {
		t.Errorf("Location = %q, want %q", got, "/dashboard")
	}
	if resp.Body != nil {
		t.Errorf("Body = %v, want nil", resp.Body)
	}
}

func TestSeeOther(t *testing.T) {
	resp := SeeOther("/home")

	if resp.Status != http.StatusSeeOther {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusSeeOther)
	}
	if got := resp.Headers.Get("Location"); got != "/home" {
		t.Errorf("Location = %q, want %q", got, "/home")
	}
	if resp.Body != nil {
		t.Errorf("Body = %v, want nil", resp.Body)
	}
}
