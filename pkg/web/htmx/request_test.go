package htmx

import (
	"context"
	"net/url"
	"testing"

	"github.com/go-sum/web"
)

func newContext(headers map[string]string) *web.Context {
	req := web.NewRequest("GET", &url.URL{Path: "/"})
	for name, value := range headers {
		req.Headers.Set(name, value)
	}
	return web.NewContext(context.Background(), req)
}

func TestIsHTMX(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   bool
	}{
		{name: "no header", header: "", want: false},
		{name: "true", header: "true", want: true},
		{name: "True mixed case", header: "True", want: true},
		{name: "false", header: "false", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.header != "" {
				headers["HX-Request"] = tt.header
			}
			c := newContext(headers)
			if got := IsHTMX(c); got != tt.want {
				t.Errorf("IsHTMX() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsBoosted(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   bool
	}{
		{name: "no header", header: "", want: false},
		{name: "true", header: "true", want: true},
		{name: "TRUE uppercase", header: "TRUE", want: true},
		{name: "false", header: "false", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.header != "" {
				headers["HX-Boosted"] = tt.header
			}
			c := newContext(headers)
			if got := IsBoosted(c); got != tt.want {
				t.Errorf("IsBoosted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHistoryRestore(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   bool
	}{
		{name: "no header", header: "", want: false},
		{name: "true", header: "true", want: true},
		{name: "false", header: "false", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.header != "" {
				headers["HX-History-Restore-Request"] = tt.header
			}
			c := newContext(headers)
			if got := HistoryRestore(c); got != tt.want {
				t.Errorf("HistoryRestore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCurrentURL(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{name: "no header", header: "", want: ""},
		{name: "with URL", header: "https://example.com/page", want: "https://example.com/page"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.header != "" {
				headers["HX-Current-URL"] = tt.header
			}
			c := newContext(headers)
			if got := CurrentURL(c); got != tt.want {
				t.Errorf("CurrentURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrompt(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{name: "no header", header: "", want: ""},
		{name: "with value", header: "user input here", want: "user input here"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.header != "" {
				headers["HX-Prompt"] = tt.header
			}
			c := newContext(headers)
			if got := Prompt(c); got != tt.want {
				t.Errorf("Prompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTarget(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{name: "no header", header: "", want: ""},
		{name: "with id", header: "main-content", want: "main-content"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.header != "" {
				headers["HX-Target"] = tt.header
			}
			c := newContext(headers)
			if got := Target(c); got != tt.want {
				t.Errorf("Target() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTrigger(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{name: "no header", header: "", want: ""},
		{name: "with id", header: "submit-btn", want: "submit-btn"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.header != "" {
				headers["HX-Trigger"] = tt.header
			}
			c := newContext(headers)
			if got := Trigger(c); got != tt.want {
				t.Errorf("Trigger() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTriggerName(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{name: "no header", header: "", want: ""},
		{name: "with name", header: "search", want: "search"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.header != "" {
				headers["HX-Trigger-Name"] = tt.header
			}
			c := newContext(headers)
			if got := TriggerName(c); got != tt.want {
				t.Errorf("TriggerName() = %q, want %q", got, tt.want)
			}
		})
	}
}
