package web

import (
	"context"
	"io"
	"net/url"
	"strings"
	"testing"
)

func TestMethodOverride(t *testing.T) {
	tests := []struct {
		name       string
		cfg        MethodOverrideConfig
		method     string
		header     string
		headerVal  string
		body       string
		wantMethod string
		// wantBody is what the next handler should be able to read from the body.
		wantBody string
	}{
		{
			name:       "POST with X-HTTP-Method-Override DELETE header becomes DELETE",
			cfg:        MethodOverrideConfig{},
			method:     "POST",
			header:     "X-HTTP-Method-Override",
			headerVal:  "DELETE",
			body:       "",
			wantMethod: "DELETE",
			wantBody:   "",
		},
		{
			name:       "POST with form body _method=PATCH becomes PATCH; original body still readable",
			cfg:        MethodOverrideConfig{},
			method:     "POST",
			body:       "_method=PATCH&name=alice",
			wantMethod: "PATCH",
			wantBody:   "_method=PATCH&name=alice",
		},
		{
			name:       "POST with form body _method=PUT becomes PUT",
			cfg:        MethodOverrideConfig{},
			method:     "POST",
			body:       "_method=PUT",
			wantMethod: "PUT",
			wantBody:   "_method=PUT",
		},
		{
			name:       "POST with form body _method=put lowercase becomes PUT",
			cfg:        MethodOverrideConfig{},
			method:     "POST",
			body:       "_method=put",
			wantMethod: "PUT",
			wantBody:   "_method=put",
		},
		{
			name:       "POST with header GET not in AllowedMethods stays POST",
			cfg:        MethodOverrideConfig{},
			method:     "POST",
			header:     "X-HTTP-Method-Override",
			headerVal:  "GET",
			body:       "",
			wantMethod: "POST",
			wantBody:   "",
		},
		{
			name:       "GET with _method=DELETE stays GET",
			cfg:        MethodOverrideConfig{},
			method:     "GET",
			body:       "_method=DELETE",
			wantMethod: "GET",
			wantBody:   "_method=DELETE",
		},
		{
			name:       "POST with no override field or header stays POST",
			cfg:        MethodOverrideConfig{},
			method:     "POST",
			body:       "name=alice",
			wantMethod: "POST",
			wantBody:   "name=alice",
		},
		{
			name: "custom FormField and Header config works",
			cfg: MethodOverrideConfig{
				FormField:      "_m",
				Header:         "X-Override",
				AllowedMethods: []string{"PUT", "PATCH", "DELETE"},
			},
			method:     "POST",
			header:     "X-Override",
			headerVal:  "PATCH",
			body:       "",
			wantMethod: "PATCH",
			wantBody:   "",
		},
		{
			name: "custom FormField resolved from body",
			cfg: MethodOverrideConfig{
				FormField:      "_m",
				Header:         "X-Override",
				AllowedMethods: []string{"PUT", "PATCH", "DELETE"},
			},
			method:     "POST",
			body:       "_m=DELETE&id=5",
			wantMethod: "DELETE",
			wantBody:   "_m=DELETE&id=5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := MethodOverride(tt.cfg)

			u := &url.URL{Path: "/"}
			req := NewRequest(tt.method, u)

			if tt.header != "" {
				req.Headers.Set(tt.header, tt.headerVal)
			}

			if tt.body != "" {
				req.SetBody(io.NopCloser(strings.NewReader(tt.body)))
			}

			c := NewContext(context.Background(), req)

			var gotMethod string
			var gotBody string

			handler := func(c *Context) (Response, error) {
				gotMethod = c.Method()
				if c.Request.Body != nil {
					data, _ := c.Request.Text()
					gotBody = data
				}
				return Respond(200), nil
			}

			mw(handler)(c) //nolint:errcheck

			if gotMethod != tt.wantMethod {
				t.Errorf("Method = %q, want %q", gotMethod, tt.wantMethod)
			}

			if tt.body != "" && gotBody != tt.wantBody {
				t.Errorf("body seen by next handler = %q, want %q", gotBody, tt.wantBody)
			}

			// Verify c.Request.Method matches c.Method.
			if c.Request.Method != gotMethod {
				t.Errorf("c.Request.Method = %q, want %q", c.Request.Method, gotMethod)
			}
		})
	}
}

func TestMethodOverride_DefaultConfig(t *testing.T) {
	mw := MethodOverride(MethodOverrideConfig{})

	u := &url.URL{Path: "/"}
	req := NewRequest("POST", u)
	req.SetBody(io.NopCloser(strings.NewReader("_method=DELETE&x=1")))

	c := NewContext(context.Background(), req)

	var gotMethod string
	handler := func(c *Context) (Response, error) {
		gotMethod = c.Method()
		return Respond(200), nil
	}

	mw(handler)(c) //nolint:errcheck

	if gotMethod != "DELETE" {
		t.Errorf("Method = %q, want DELETE", gotMethod)
	}
}

func TestMethodOverride_NextHandlerReceivesFullBody(t *testing.T) {
	mw := MethodOverride(MethodOverrideConfig{})

	body := "_method=PATCH&field1=value1&field2=value2"
	u := &url.URL{Path: "/"}
	req := NewRequest("POST", u)
	req.SetBody(io.NopCloser(strings.NewReader(body)))

	c := NewContext(context.Background(), req)

	var gotMethod string
	var gotBody string
	handler := func(c *Context) (Response, error) {
		gotMethod = c.Method()
		data, err := c.Request.Text()
		if err != nil {
			t.Errorf("next handler Text() error = %v", err)
		}
		gotBody = data
		return Respond(200), nil
	}

	mw(handler)(c) //nolint:errcheck

	if gotMethod != "PATCH" {
		t.Errorf("Method = %q, want PATCH", gotMethod)
	}
	if gotBody != body {
		t.Errorf("body = %q, want %q", gotBody, body)
	}
}
