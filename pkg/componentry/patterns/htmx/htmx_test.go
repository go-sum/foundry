package htmx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-sum/componentry/patterns/htmx"
)

func TestNewRequest_fullHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Request", "true")
	r.Header.Set("HX-Boosted", "true")
	r.Header.Set("HX-Trigger", "my-btn")
	r.Header.Set("HX-Target", "#result")
	r.Header.Set("HX-Trigger-Name", "submit-btn")
	r.Header.Set("HX-Current-URL", "https://example.com/page")

	req := htmx.NewRequest(r)

	if !req.Enabled {
		t.Errorf("NewRequest: Enabled = false, want true")
	}
	if !req.Boosted {
		t.Errorf("NewRequest: Boosted = false, want true")
	}
	if req.Trigger != "my-btn" {
		t.Errorf("NewRequest: Trigger = %q, want %q", req.Trigger, "my-btn")
	}
	if req.Target != "#result" {
		t.Errorf("NewRequest: Target = %q, want %q", req.Target, "#result")
	}
	if req.TriggerName != "submit-btn" {
		t.Errorf("NewRequest: TriggerName = %q, want %q", req.TriggerName, "submit-btn")
	}
	if req.CurrentURL != "https://example.com/page" {
		t.Errorf("NewRequest: CurrentURL = %q, want %q", req.CurrentURL, "https://example.com/page")
	}
}

func TestNewRequest_noHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	req := htmx.NewRequest(r)

	if req.Enabled {
		t.Errorf("NewRequest no headers: Enabled = true, want false")
	}
	if req.Boosted {
		t.Errorf("NewRequest no headers: Boosted = true, want false")
	}
	if req.Trigger != "" {
		t.Errorf("NewRequest no headers: Trigger = %q, want empty", req.Trigger)
	}
	if req.Target != "" {
		t.Errorf("NewRequest no headers: Target = %q, want empty", req.Target)
	}
	if req.TriggerName != "" {
		t.Errorf("NewRequest no headers: TriggerName = %q, want empty", req.TriggerName)
	}
	if req.CurrentURL != "" {
		t.Errorf("NewRequest no headers: CurrentURL = %q, want empty", req.CurrentURL)
	}
}

func TestRequest_IsPartial(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		boosted bool
		want    bool
	}{
		{name: "enabled not boosted", enabled: true, boosted: false, want: true},
		{name: "enabled and boosted", enabled: true, boosted: true, want: false},
		{name: "not enabled", enabled: false, boosted: false, want: false},
		{name: "not enabled boosted", enabled: false, boosted: true, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := htmx.Request{Enabled: tc.enabled, Boosted: tc.boosted}
			if got := req.IsPartial(); got != tc.want {
				t.Errorf("IsPartial = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResponse_Apply(t *testing.T) {
	tests := []struct {
		name       string
		resp       htmx.Response
		wantHeader string
		wantValue  string
		absentHdr  string
	}{
		{
			name:       "Redirect",
			resp:       htmx.Response{Redirect: "/new"},
			wantHeader: "HX-Redirect",
			wantValue:  "/new",
		},
		{
			name:       "Refresh",
			resp:       htmx.Response{Refresh: true},
			wantHeader: "HX-Refresh",
			wantValue:  "true",
		},
		{
			name:       "PushURL",
			resp:       htmx.Response{PushURL: "/pushed"},
			wantHeader: "HX-Push-Url",
			wantValue:  "/pushed",
		},
		{
			name:       "ReplaceURL",
			resp:       htmx.Response{ReplaceURL: "/replaced"},
			wantHeader: "HX-Replace-Url",
			wantValue:  "/replaced",
		},
		{
			name:       "Trigger",
			resp:       htmx.Response{Trigger: "reload"},
			wantHeader: "HX-Trigger",
			wantValue:  "reload",
		},
		{
			name:       "TriggerAfterSettle",
			resp:       htmx.Response{TriggerAfterSettle: "settled"},
			wantHeader: "HX-Trigger-After-Settle",
			wantValue:  "settled",
		},
		{
			name:       "TriggerAfterSwap",
			resp:       htmx.Response{TriggerAfterSwap: "swapped"},
			wantHeader: "HX-Trigger-After-Swap",
			wantValue:  "swapped",
		},
		{
			name:       "Retarget",
			resp:       htmx.Response{Retarget: "#new-target"},
			wantHeader: "HX-Retarget",
			wantValue:  "#new-target",
		},
		{
			name:       "Reswap",
			resp:       htmx.Response{Reswap: "outerHTML"},
			wantHeader: "HX-Reswap",
			wantValue:  "outerHTML",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.resp.Apply(w)
			got := w.Header().Get(tc.wantHeader)
			if got != tc.wantValue {
				t.Errorf("Apply %s: header %q = %q, want %q", tc.name, tc.wantHeader, got, tc.wantValue)
			}
		})
	}
}

func TestResponse_Apply_zeroFields(t *testing.T) {
	w := httptest.NewRecorder()
	htmx.Response{}.Apply(w)
	for _, hdr := range []string{"HX-Redirect", "HX-Refresh", "HX-Push-Url", "HX-Replace-Url",
		"HX-Trigger", "HX-Trigger-After-Settle", "HX-Trigger-After-Swap",
		"HX-Retarget", "HX-Reswap"} {
		if v := w.Header().Get(hdr); v != "" {
			t.Errorf("Apply zero: header %q = %q, want empty", hdr, v)
		}
	}
}

func TestSetRedirect(t *testing.T) {
	w := httptest.NewRecorder()
	htmx.SetRedirect(w, "/target")
	if got := w.Header().Get("HX-Redirect"); got != "/target" {
		t.Errorf("SetRedirect: HX-Redirect = %q, want /target", got)
	}
}

func TestSetRefresh(t *testing.T) {
	w := httptest.NewRecorder()
	htmx.SetRefresh(w)
	if got := w.Header().Get("HX-Refresh"); got != "true" {
		t.Errorf("SetRefresh: HX-Refresh = %q, want true", got)
	}
}

func TestSetTrigger(t *testing.T) {
	w := httptest.NewRecorder()
	htmx.SetTrigger(w, "my-event")
	if got := w.Header().Get("HX-Trigger"); got != "my-event" {
		t.Errorf("SetTrigger: HX-Trigger = %q, want my-event", got)
	}
}

func TestSetPushURL(t *testing.T) {
	w := httptest.NewRecorder()
	htmx.SetPushURL(w, "/new-url")
	if got := w.Header().Get("HX-Push-Url"); got != "/new-url" {
		t.Errorf("SetPushURL: HX-Push-Url = %q, want /new-url", got)
	}
}

func TestSetReplaceURL(t *testing.T) {
	w := httptest.NewRecorder()
	htmx.SetReplaceURL(w, "/replaced")
	if got := w.Header().Get("HX-Replace-Url"); got != "/replaced" {
		t.Errorf("SetReplaceURL: HX-Replace-Url = %q, want /replaced", got)
	}
}

func TestSetTriggerAfterSettle(t *testing.T) {
	w := httptest.NewRecorder()
	htmx.SetTriggerAfterSettle(w, "settled-event")
	if got := w.Header().Get("HX-Trigger-After-Settle"); got != "settled-event" {
		t.Errorf("SetTriggerAfterSettle: %q, want settled-event", got)
	}
}

func TestSetTriggerAfterSwap(t *testing.T) {
	w := httptest.NewRecorder()
	htmx.SetTriggerAfterSwap(w, "swapped-event")
	if got := w.Header().Get("HX-Trigger-After-Swap"); got != "swapped-event" {
		t.Errorf("SetTriggerAfterSwap: %q, want swapped-event", got)
	}
}

func TestSetRetarget(t *testing.T) {
	w := httptest.NewRecorder()
	htmx.SetRetarget(w, "#new-container")
	if got := w.Header().Get("HX-Retarget"); got != "#new-container" {
		t.Errorf("SetRetarget: %q, want #new-container", got)
	}
}

func TestSetReswap(t *testing.T) {
	w := httptest.NewRecorder()
	htmx.SetReswap(w, "innerHTML")
	if got := w.Header().Get("HX-Reswap"); got != "innerHTML" {
		t.Errorf("SetReswap: %q, want innerHTML", got)
	}
}

func TestIsRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if htmx.IsRequest(r) {
		t.Error("IsRequest without header: want false")
	}
	r.Header.Set("HX-Request", "true")
	if !htmx.IsRequest(r) {
		t.Error("IsRequest with header: want true")
	}
}

func TestIsBoosted(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if htmx.IsBoosted(r) {
		t.Error("IsBoosted without header: want false")
	}
	r.Header.Set("HX-Boosted", "true")
	if !htmx.IsBoosted(r) {
		t.Error("IsBoosted with header: want true")
	}
}

func TestGetTrigger(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Trigger", "the-trigger")
	if got := htmx.GetTrigger(r); got != "the-trigger" {
		t.Errorf("GetTrigger = %q, want the-trigger", got)
	}
}

func TestGetTarget(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Target", "#the-target")
	if got := htmx.GetTarget(r); got != "#the-target" {
		t.Errorf("GetTarget = %q, want #the-target", got)
	}
}

func TestGetTriggerName(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Trigger-Name", "submit-btn")
	if got := htmx.GetTriggerName(r); got != "submit-btn" {
		t.Errorf("GetTriggerName = %q, want submit-btn", got)
	}
}

func TestGetCurrentURL(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Current-URL", "https://example.com/page")
	if got := htmx.GetCurrentURL(r); got != "https://example.com/page" {
		t.Errorf("GetCurrentURL = %q, want https://example.com/page", got)
	}
}

func TestAttrs_Params(t *testing.T) {
	got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{Params: "*"}))
	if !containsStr(got, `hx-params="*"`) {
		t.Errorf("Attrs Params: expected hx-params=*, got:\n%s", got)
	}
}

func TestAttrs_Params_empty(t *testing.T) {
	got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{}))
	if containsStr(got, "hx-params") {
		t.Errorf("Attrs Params empty: expected no hx-params, got:\n%s", got)
	}
}

func TestAttrs_Select(t *testing.T) {
	got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{Select: "#fragment"}))
	if !containsStr(got, `hx-select="#fragment"`) {
		t.Errorf("Attrs Select: expected hx-select=#fragment, got:\n%s", got)
	}
}

func TestAttrs_SelectOOB(t *testing.T) {
	got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{SelectOOB: "#oob"}))
	if !containsStr(got, `hx-select-oob="#oob"`) {
		t.Errorf("Attrs SelectOOB: expected hx-select-oob=#oob, got:\n%s", got)
	}
}

func TestAttrs_Indicator(t *testing.T) {
	got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{Indicator: ".spinner"}))
	if !containsStr(got, `hx-indicator=".spinner"`) {
		t.Errorf("Attrs Indicator: expected hx-indicator=.spinner, got:\n%s", got)
	}
}

func TestAttrs_DisabledElt(t *testing.T) {
	got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{DisabledElt: "this"}))
	if !containsStr(got, `hx-disabled-elt="this"`) {
		t.Errorf("Attrs DisabledElt: expected hx-disabled-elt=this, got:\n%s", got)
	}
}

func TestAttrs_Sync(t *testing.T) {
	got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{Sync: "closest form"}))
	if !containsStr(got, `hx-sync`) {
		t.Errorf("Attrs Sync: expected hx-sync, got:\n%s", got)
	}
}

func TestAttrs_Encoding(t *testing.T) {
	got := renderAttrs(t, htmx.Attrs(htmx.AttrsProps{Encoding: "multipart/form-data"}))
	if !containsStr(got, `hx-encoding="multipart/form-data"`) {
		t.Errorf("Attrs Encoding: expected hx-encoding=multipart/form-data, got:\n%s", got)
	}
}
