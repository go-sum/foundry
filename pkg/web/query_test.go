package web

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

// mustURL parses a URL string and panics on error. For use in tests only.
func mustURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic("mustURL: " + err.Error())
	}
	return u
}

// newTestContext builds a *Context with the given URL and optionally sets route params.
func newTestContext(rawURL string) *Context {
	req := NewRequest(http.MethodGet, mustURL(rawURL))
	return NewContext(context.Background(), req)
}

// ---- PathParam tests --------------------------------------------------------

func TestPathParam_Int(t *testing.T) {
	tests := []struct {
		name    string
		param   string
		wantVal int
		wantErr bool
	}{
		{"valid positive", "42", 42, false},
		{"valid negative", "-7", -7, false},
		{"valid zero", "0", 0, false},
		{"missing param", "", 0, true},
		{"not an integer", "abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContext("http://example.com/items/42")
			if tt.param != "" {
				c.SetParam("id", tt.param)
			}
			got, err := PathParam[int](c, "id")
			if tt.wantErr {
				if err == nil {
					t.Fatal("PathParam[int]() error = nil, want non-nil")
				}
				e, ok := err.(*Error)
				if !ok {
					t.Fatalf("err type = %T, want *Error", err)
				}
				if e.Status != http.StatusBadRequest {
					t.Errorf("Status = %d, want 400", e.Status)
				}
				if e.Code != CodeBadRequest {
					t.Errorf("Code = %q, want %q", e.Code, CodeBadRequest)
				}
			} else {
				if err != nil {
					t.Fatalf("PathParam[int]() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("PathParam[int]() = %d, want %d", got, tt.wantVal)
				}
			}
		})
	}
}

func TestPathParam_String(t *testing.T) {
	tests := []struct {
		name    string
		param   string
		wantVal string
		wantErr bool
	}{
		{"valid slug", "hello-world", "hello-world", false},
		{"missing param", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContext("http://example.com/posts/hello-world")
			if tt.param != "" {
				c.SetParam("slug", tt.param)
			}
			got, err := PathParam[string](c, "slug")
			if tt.wantErr {
				if err == nil {
					t.Fatal("PathParam[string]() error = nil, want non-nil")
				}
				e, ok := err.(*Error)
				if !ok {
					t.Fatalf("err type = %T, want *Error", err)
				}
				if e.Status != http.StatusBadRequest {
					t.Errorf("Status = %d, want 400", e.Status)
				}
			} else {
				if err != nil {
					t.Fatalf("PathParam[string]() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("PathParam[string]() = %q, want %q", got, tt.wantVal)
				}
			}
		})
	}
}

func TestPathParam_Bool(t *testing.T) {
	tests := []struct {
		name    string
		param   string
		wantVal bool
		wantErr bool
	}{
		{"true", "true", true, false},
		{"1", "1", true, false},
		{"false", "false", false, false},
		{"0", "0", false, false},
		{"missing", "", false, true},
		{"invalid", "maybe", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContext("http://example.com/toggle/true")
			if tt.param != "" {
				c.SetParam("flag", tt.param)
			}
			got, err := PathParam[bool](c, "flag")
			if tt.wantErr {
				if err == nil {
					t.Fatal("PathParam[bool]() error = nil, want non-nil")
				}
				e, ok := err.(*Error)
				if !ok {
					t.Fatalf("err type = %T, want *Error", err)
				}
				if e.Status != http.StatusBadRequest {
					t.Errorf("Status = %d, want 400", e.Status)
				}
			} else {
				if err != nil {
					t.Fatalf("PathParam[bool]() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("PathParam[bool]() = %v, want %v", got, tt.wantVal)
				}
			}
		})
	}
}

// ---- QueryParam tests -------------------------------------------------------

func TestQueryParam_Int(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantVal int
		wantErr bool
	}{
		{"valid page", "http://example.com/items?page=3", 3, false},
		{"missing param", "http://example.com/items", 0, true},
		{"invalid value", "http://example.com/items?page=abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContext(tt.rawURL)
			got, err := QueryParam[int](c, "page")
			if tt.wantErr {
				if err == nil {
					t.Fatal("QueryParam[int]() error = nil, want non-nil")
				}
				e, ok := err.(*Error)
				if !ok {
					t.Fatalf("err type = %T, want *Error", err)
				}
				if e.Status != http.StatusBadRequest {
					t.Errorf("Status = %d, want 400", e.Status)
				}
				if e.Code != CodeBadRequest {
					t.Errorf("Code = %q, want %q", e.Code, CodeBadRequest)
				}
			} else {
				if err != nil {
					t.Fatalf("QueryParam[int]() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("QueryParam[int]() = %d, want %d", got, tt.wantVal)
				}
			}
		})
	}
}

// ---- QueryParamOr tests -----------------------------------------------------

func TestQueryParamOr_Int(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		fallback int
		wantVal  int
		wantErr  bool
	}{
		{"present and valid", "http://example.com/items?page=5", 1, 5, false},
		{"missing uses fallback", "http://example.com/items", 1, 1, false},
		{"present but invalid", "http://example.com/items?page=bad", 1, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContext(tt.rawURL)
			got, err := QueryParamOr[int](c, "page", tt.fallback)
			if tt.wantErr {
				if err == nil {
					t.Fatal("QueryParamOr[int]() error = nil, want non-nil")
				}
				e, ok := err.(*Error)
				if !ok {
					t.Fatalf("err type = %T, want *Error", err)
				}
				if e.Status != http.StatusBadRequest {
					t.Errorf("Status = %d, want 400", e.Status)
				}
			} else {
				if err != nil {
					t.Fatalf("QueryParamOr[int]() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("QueryParamOr[int]() = %d, want %d", got, tt.wantVal)
				}
			}
		})
	}
}

// ---- QueryParams tests ------------------------------------------------------

func TestQueryParams_Int(t *testing.T) {
	t.Run("multi-value returns slice", func(t *testing.T) {
		c := newTestContext("http://example.com/items?id=1&id=2&id=3")
		got, err := QueryParams[int](c, "id")
		if err != nil {
			t.Fatalf("QueryParams[int]() error = %v, want nil", err)
		}
		want := []int{1, 2, 3}
		if len(got) != len(want) {
			t.Fatalf("QueryParams[int]() len = %d, want %d", len(got), len(want))
		}
		for i, v := range want {
			if got[i] != v {
				t.Errorf("QueryParams[int]()[%d] = %d, want %d", i, got[i], v)
			}
		}
	})

	t.Run("absent key returns nil", func(t *testing.T) {
		c := newTestContext("http://example.com/items")
		got, err := QueryParams[int](c, "id")
		if err != nil {
			t.Fatalf("QueryParams[int]() error = %v, want nil", err)
		}
		if got != nil {
			t.Errorf("QueryParams[int]() = %v, want nil", got)
		}
	})

	t.Run("invalid value returns error", func(t *testing.T) {
		c := newTestContext("http://example.com/items?id=1&id=bad")
		_, err := QueryParams[int](c, "id")
		if err == nil {
			t.Fatal("QueryParams[int]() error = nil, want non-nil")
		}
		e, ok := err.(*Error)
		if !ok {
			t.Fatalf("err type = %T, want *Error", err)
		}
		if e.Status != http.StatusBadRequest {
			t.Errorf("Status = %d, want 400", e.Status)
		}
	})
}

// ---- HeaderParam tests ------------------------------------------------------

func TestHeaderParam_String(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		value   string
		wantVal string
		wantErr bool
	}{
		{"present header", "X-Tenant-ID", "acme", "acme", false},
		{"missing header", "X-Tenant-ID", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContext("http://example.com/")
			if tt.value != "" {
				c.Request.Headers.Set(tt.header, tt.value)
			}
			got, err := HeaderParam[string](c, tt.header)
			if tt.wantErr {
				if err == nil {
					t.Fatal("HeaderParam[string]() error = nil, want non-nil")
				}
				e, ok := err.(*Error)
				if !ok {
					t.Fatalf("err type = %T, want *Error", err)
				}
				if e.Status != http.StatusBadRequest {
					t.Errorf("Status = %d, want 400", e.Status)
				}
				if e.Code != CodeBadRequest {
					t.Errorf("Code = %q, want %q", e.Code, CodeBadRequest)
				}
			} else {
				if err != nil {
					t.Fatalf("HeaderParam[string]() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("HeaderParam[string]() = %q, want %q", got, tt.wantVal)
				}
			}
		})
	}
}

// ---- HeaderParamOr tests ----------------------------------------------------

func TestHeaderParamOr_String(t *testing.T) {
	tests := []struct {
		name     string
		setValue string
		fallback string
		wantVal  string
		wantErr  bool
	}{
		{"present returns value", "tenant-a", "default", "tenant-a", false},
		{"missing returns fallback", "", "default", "default", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContext("http://example.com/")
			if tt.setValue != "" {
				c.Request.Headers.Set("X-Tenant-ID", tt.setValue)
			}
			got, err := HeaderParamOr[string](c, "X-Tenant-ID", tt.fallback)
			if tt.wantErr {
				if err == nil {
					t.Fatal("HeaderParamOr[string]() error = nil, want non-nil")
				}
			} else {
				if err != nil {
					t.Fatalf("HeaderParamOr[string]() error = %v, want nil", err)
				}
				if got != tt.wantVal {
					t.Errorf("HeaderParamOr[string]() = %q, want %q", got, tt.wantVal)
				}
			}
		})
	}
}
