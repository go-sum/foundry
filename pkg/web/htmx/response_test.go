package htmx

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-sum/foundry/pkg/web"
)

func newResponse() *web.Response {
	resp := web.Respond(http.StatusOK)
	return &resp
}

func TestSetLocation(t *testing.T) {
	resp := newResponse()
	SetLocation(resp, "/dashboard")
	if got := resp.Headers.Get("HX-Location"); got != "/dashboard" {
		t.Errorf("HX-Location = %q, want %q", got, "/dashboard")
	}
}

func TestSetPushURL(t *testing.T) {
	resp := newResponse()
	SetPushURL(resp, "/new-path")
	if got := resp.Headers.Get("HX-Push-Url"); got != "/new-path" {
		t.Errorf("HX-Push-Url = %q, want %q", got, "/new-path")
	}
}

func TestSetPushURL_False(t *testing.T) {
	resp := newResponse()
	SetPushURL(resp, "false")
	if got := resp.Headers.Get("HX-Push-Url"); got != "false" {
		t.Errorf("HX-Push-Url = %q, want %q", got, "false")
	}
}

func TestSetReplaceURL(t *testing.T) {
	resp := newResponse()
	SetReplaceURL(resp, "/replaced")
	if got := resp.Headers.Get("HX-Replace-Url"); got != "/replaced" {
		t.Errorf("HX-Replace-Url = %q, want %q", got, "/replaced")
	}
}

func TestSetReplaceURL_False(t *testing.T) {
	resp := newResponse()
	SetReplaceURL(resp, "false")
	if got := resp.Headers.Get("HX-Replace-Url"); got != "false" {
		t.Errorf("HX-Replace-Url = %q, want %q", got, "false")
	}
}

func TestSetRedirect(t *testing.T) {
	resp := newResponse()
	SetRedirect(resp, "/login")
	if got := resp.Headers.Get("HX-Redirect"); got != "/login" {
		t.Errorf("HX-Redirect = %q, want %q", got, "/login")
	}
}

func TestSetRefresh(t *testing.T) {
	resp := newResponse()
	SetRefresh(resp)
	if got := resp.Headers.Get("HX-Refresh"); got != "true" {
		t.Errorf("HX-Refresh = %q, want %q", got, "true")
	}
}

func TestSetReswap(t *testing.T) {
	tests := []struct {
		strategy string
	}{
		{"innerHTML"},
		{"outerHTML"},
		{"beforebegin"},
		{"afterbegin"},
		{"beforeend"},
		{"afterend"},
		{"delete"},
		{"none"},
	}
	for _, tt := range tests {
		t.Run(tt.strategy, func(t *testing.T) {
			resp := newResponse()
			SetReswap(resp, tt.strategy)
			if got := resp.Headers.Get("HX-Reswap"); got != tt.strategy {
				t.Errorf("HX-Reswap = %q, want %q", got, tt.strategy)
			}
		})
	}
}

func TestSetRetarget(t *testing.T) {
	resp := newResponse()
	SetRetarget(resp, "#content")
	if got := resp.Headers.Get("HX-Retarget"); got != "#content" {
		t.Errorf("HX-Retarget = %q, want %q", got, "#content")
	}
}

func TestSetReselect(t *testing.T) {
	resp := newResponse()
	SetReselect(resp, ".result")
	if got := resp.Headers.Get("HX-Reselect"); got != ".result" {
		t.Errorf("HX-Reselect = %q, want %q", got, ".result")
	}
}

func TestSetTrigger_NilPayload(t *testing.T) {
	resp := newResponse()
	SetTrigger(resp, "myEvent", nil)
	if got := resp.Headers.Get("HX-Trigger"); got != "myEvent" {
		t.Errorf("HX-Trigger = %q, want %q", got, "myEvent")
	}
}

func TestSetTrigger_WithPayload(t *testing.T) {
	resp := newResponse()
	SetTrigger(resp, "myEvent", map[string]string{"key": "value"})
	got := resp.Headers.Get("HX-Trigger")

	var parsed map[string]any
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("HX-Trigger is not valid JSON: %v (got %q)", err, got)
	}
	if _, ok := parsed["myEvent"]; !ok {
		t.Errorf("HX-Trigger JSON does not contain key %q: %s", "myEvent", got)
	}
}

func TestSetTrigger_WithScalarPayload(t *testing.T) {
	resp := newResponse()
	SetTrigger(resp, "count", 42)
	got := resp.Headers.Get("HX-Trigger")

	var parsed map[string]any
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("HX-Trigger is not valid JSON: %v (got %q)", err, got)
	}
	val, ok := parsed["count"]
	if !ok {
		t.Fatalf("HX-Trigger JSON missing key %q: %s", "count", got)
	}
	// JSON numbers unmarshal to float64.
	if val != float64(42) {
		t.Errorf("HX-Trigger JSON[count] = %v, want 42", val)
	}
}

func TestSetTriggerAfterSettle_NilPayload(t *testing.T) {
	resp := newResponse()
	SetTriggerAfterSettle(resp, "settleEvent", nil)
	if got := resp.Headers.Get("HX-Trigger-After-Settle"); got != "settleEvent" {
		t.Errorf("HX-Trigger-After-Settle = %q, want %q", got, "settleEvent")
	}
}

func TestSetTriggerAfterSettle_WithPayload(t *testing.T) {
	resp := newResponse()
	SetTriggerAfterSettle(resp, "settleEvent", "data")
	got := resp.Headers.Get("HX-Trigger-After-Settle")

	var parsed map[string]any
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("HX-Trigger-After-Settle is not valid JSON: %v (got %q)", err, got)
	}
	if _, ok := parsed["settleEvent"]; !ok {
		t.Errorf("HX-Trigger-After-Settle JSON missing key %q: %s", "settleEvent", got)
	}
}

func TestSetTriggerAfterSwap_NilPayload(t *testing.T) {
	resp := newResponse()
	SetTriggerAfterSwap(resp, "swapEvent", nil)
	if got := resp.Headers.Get("HX-Trigger-After-Swap"); got != "swapEvent" {
		t.Errorf("HX-Trigger-After-Swap = %q, want %q", got, "swapEvent")
	}
}

func TestSetTriggerAfterSwap_WithPayload(t *testing.T) {
	resp := newResponse()
	SetTriggerAfterSwap(resp, "swapEvent", []string{"a", "b"})
	got := resp.Headers.Get("HX-Trigger-After-Swap")

	var parsed map[string]any
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("HX-Trigger-After-Swap is not valid JSON: %v (got %q)", err, got)
	}
	if _, ok := parsed["swapEvent"]; !ok {
		t.Errorf("HX-Trigger-After-Swap JSON missing key %q: %s", "swapEvent", got)
	}
}
