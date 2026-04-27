package flash_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/componentry/patterns/flash"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestRender(t *testing.T) {
	t.Run("empty slice renders empty text", func(t *testing.T) {
		got := testutil.RenderNode(t, flash.Render(nil))
		if got != "" {
			t.Errorf("expected empty string, got: %q", got)
		}
	})

	t.Run("empty slice does not panic", func(t *testing.T) {
		got := testutil.RenderNode(t, flash.Render([]flash.Message{}))
		if got != "" {
			t.Errorf("expected empty string, got: %q", got)
		}
	})

	t.Run("success type uses default alert variant", func(t *testing.T) {
		got := testutil.RenderNode(t, flash.Render([]flash.Message{
			{Type: flash.TypeSuccess, Text: "Saved successfully"},
		}))
		want := testutil.LoadGolden(t)
		testutil.AssertEqualHTML(t, want, got)
	})

	t.Run("error type uses destructive alert variant", func(t *testing.T) {
		got := testutil.RenderNode(t, flash.Render([]flash.Message{
			{Type: flash.TypeError, Text: "Something went wrong"},
		}))
		want := testutil.LoadGolden(t)
		testutil.AssertEqualHTML(t, want, got)
	})

	t.Run("info type uses default alert variant", func(t *testing.T) {
		got := testutil.RenderNode(t, flash.Render([]flash.Message{
			{Type: flash.TypeInfo, Text: "Please review your settings"},
		}))
		want := testutil.LoadGolden(t)
		testutil.AssertEqualHTML(t, want, got)
	})

	t.Run("warning type uses default alert variant", func(t *testing.T) {
		got := testutil.RenderNode(t, flash.Render([]flash.Message{
			{Type: flash.TypeWarning, Text: "Low disk space"},
		}))
		want := testutil.LoadGolden(t)
		testutil.AssertEqualHTML(t, want, got)
	})

	t.Run("multiple messages", func(t *testing.T) {
		got := testutil.RenderNode(t, flash.Render([]flash.Message{
			{Type: flash.TypeSuccess, Text: "Item created"},
			{Type: flash.TypeError, Text: "Could not send email"},
		}))
		want := testutil.LoadGolden(t)
		testutil.AssertEqualHTML(t, want, got)
	})
}

func TestRenderOOB(t *testing.T) {
	t.Run("empty renders empty text", func(t *testing.T) {
		got := testutil.RenderNode(t, flash.RenderOOB(nil))
		if got != "" {
			t.Errorf("expected empty string, got: %q", got)
		}
	})

	t.Run("adds hx-swap-oob attribute", func(t *testing.T) {
		got := testutil.RenderNode(t, flash.RenderOOB([]flash.Message{
			{Type: flash.TypeSuccess, Text: "Saved"},
		}))
		want := testutil.LoadGolden(t)
		testutil.AssertEqualHTML(t, want, got)
	})
}

func TestRenderContainer(t *testing.T) {
	got := testutil.RenderNode(t, flash.RenderContainer(nil))
	want := `<div id="flash" class="grid gap-2"></div>`
	if got != want {
		t.Fatalf("RenderContainer(nil) = %q, want %q", got, want)
	}
}

func TestSet(t *testing.T) {
	w := httptest.NewRecorder()
	msgs := []flash.Message{
		{Type: flash.TypeSuccess, Text: "Saved"},
		{Type: flash.TypeError, Text: "Failed"},
	}
	err := flash.Set(w, msgs)
	if err != nil {
		t.Fatalf("Set: unexpected error: %v", err)
	}

	resp := w.Result()
	var cookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "flash" {
			cookie = c
			break
		}
	}
	if cookie == nil {
		t.Fatal("Set: expected flash cookie to be set")
	}

	// Decode and verify round-trip
	data, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		t.Fatalf("Set: cookie value decode error: %v", err)
	}
	var got []flash.Message
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Set: unmarshal error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Set: expected 2 messages, got %d", len(got))
	}
	if got[0].Text != "Saved" || got[0].Type != flash.TypeSuccess {
		t.Errorf("Set: first message = %+v, want {TypeSuccess, Saved}", got[0])
	}
	if got[1].Text != "Failed" || got[1].Type != flash.TypeError {
		t.Errorf("Set: second message = %+v, want {TypeError, Failed}", got[1])
	}
}

func TestGetAll_noCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	msgs, err := flash.GetAll(r, w)
	if err != nil {
		t.Fatalf("GetAll no cookie: unexpected error: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("GetAll no cookie: expected empty slice, got %v", msgs)
	}
}

func TestGetAll_withCookie(t *testing.T) {
	// First set the cookie
	setW := httptest.NewRecorder()
	msgs := []flash.Message{{Type: flash.TypeInfo, Text: "Hello"}}
	if err := flash.Set(setW, msgs); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Build a request with the cookie
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range setW.Result().Cookies() {
		r.AddCookie(c)
	}

	w := httptest.NewRecorder()
	got, err := flash.GetAll(r, w)
	if err != nil {
		t.Fatalf("GetAll with cookie: unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("GetAll with cookie: expected 1 message, got %d", len(got))
	}
	if got[0].Text != "Hello" || got[0].Type != flash.TypeInfo {
		t.Errorf("GetAll with cookie: got %+v, want {TypeInfo, Hello}", got[0])
	}

	// The cookie should be cleared (Max-Age=0 or deleted)
	resp := w.Result()
	setCookieHdr := resp.Header.Get("Set-Cookie")
	if setCookieHdr == "" {
		t.Error("GetAll: expected Set-Cookie header to clear cookie")
	}
	if !strings.Contains(setCookieHdr, "Max-Age=0") {
		t.Errorf("GetAll: expected Max-Age=0 to delete cookie, got: %s", setCookieHdr)
	}
}

func TestGetAll_invalidCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "flash", Value: "!!!not-base64!!!"})
	w := httptest.NewRecorder()

	_, err := flash.GetAll(r, w)
	if err == nil {
		t.Error("GetAll invalid cookie: expected error, got nil")
	}
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	if err := flash.Success(w, "Done!"); err != nil {
		t.Fatalf("Success: %v", err)
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range w.Result().Cookies() {
		r.AddCookie(c)
	}
	msgs, err := flash.GetAll(r, httptest.NewRecorder())
	if err != nil {
		t.Fatalf("Success GetAll: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Type != flash.TypeSuccess || msgs[0].Text != "Done!" {
		t.Errorf("Success: got %+v, want [{TypeSuccess Done!}]", msgs)
	}
}

func TestInfo(t *testing.T) {
	w := httptest.NewRecorder()
	if err := flash.Info(w, "FYI"); err != nil {
		t.Fatalf("Info: %v", err)
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range w.Result().Cookies() {
		r.AddCookie(c)
	}
	msgs, err := flash.GetAll(r, httptest.NewRecorder())
	if err != nil {
		t.Fatalf("Info GetAll: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Type != flash.TypeInfo || msgs[0].Text != "FYI" {
		t.Errorf("Info: got %+v, want [{TypeInfo FYI}]", msgs)
	}
}

func TestWarning(t *testing.T) {
	w := httptest.NewRecorder()
	if err := flash.Warning(w, "Careful"); err != nil {
		t.Fatalf("Warning: %v", err)
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range w.Result().Cookies() {
		r.AddCookie(c)
	}
	msgs, err := flash.GetAll(r, httptest.NewRecorder())
	if err != nil {
		t.Fatalf("Warning GetAll: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Type != flash.TypeWarning || msgs[0].Text != "Careful" {
		t.Errorf("Warning: got %+v, want [{TypeWarning Careful}]", msgs)
	}
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	if err := flash.Error(w, "Oops"); err != nil {
		t.Fatalf("Error: %v", err)
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range w.Result().Cookies() {
		r.AddCookie(c)
	}
	msgs, err := flash.GetAll(r, httptest.NewRecorder())
	if err != nil {
		t.Fatalf("Error GetAll: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Type != flash.TypeError || msgs[0].Text != "Oops" {
		t.Errorf("Error: got %+v, want [{TypeError Oops}]", msgs)
	}
}
