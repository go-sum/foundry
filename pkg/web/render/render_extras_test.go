package render

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// ---- CSRFField ----

func TestCSRFField_RendersHiddenInput(t *testing.T) {
	node := CSRFField("tok123")
	got := RenderNode(t, node)
	want := `<input type="hidden" name="_csrf" value="tok123">`
	if got != want {
		t.Fatalf("CSRFField = %q, want %q", got, want)
	}
}

// ---- NonceAttr ----

func TestNonceAttr_RenderedInsideScript(t *testing.T) {
	got := renderAttrInTag(t, "script", NonceAttr("abc123"))
	want := `<script nonce="abc123"></script>`
	if got != want {
		t.Fatalf("NonceAttr() = %q, want %q", got, want)
	}
}

// ---- HXForm ----

func TestHXForm_ContainsHXPost(t *testing.T) {
	node := HXForm("/submit")
	got := RenderNode(t, node)
	want := `<form hx-post="/submit" hx-swap="outerHTML"></form>`
	if got != want {
		t.Fatalf("HXForm() = %q, want %q", got, want)
	}
}

func TestHXAttributes(t *testing.T) {
	cases := []struct {
		name string
		node func() string
		want string
	}{
		{"HXGet", func() string { return renderAttrInDiv(t, HXGet("/path")) }, `hx-get="/path"`},
		{"HXTarget", func() string { return renderAttrInDiv(t, HXTarget("#container")) }, `hx-target="#container"`},
		{"HXSwap", func() string { return renderAttrInDiv(t, HXSwap("innerHTML")) }, `hx-swap="innerHTML"`},
		{"HXTrigger", func() string { return renderAttrInDiv(t, HXTrigger("click")) }, `hx-trigger="click"`},
		{"HXBoost", func() string { return renderAttrInDiv(t, HXBoost()) }, `hx-boost="true"`},
		{"HXPushURL", func() string { return renderAttrInDiv(t, HXPushURL("/page")) }, `hx-push-url="/page"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.node()
			want := `<div ` + tc.want + `></div>`
			if got != want {
				t.Errorf("%s: rendered %q, want %q", tc.name, got, want)
			}
		})
	}
}

// ---- SSEEvent ----

func TestSSEEvent_Encode_BasicEvent(t *testing.T) {
	ev := SSEEvent{Event: "update", Data: "hello"}
	var sb strings.Builder
	if err := ev.Encode(&sb); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if got, want := sb.String(), "event: update\ndata: hello\n\n"; got != want {
		t.Fatalf("Encode() = %q, want %q", got, want)
	}
}

func TestSSEEvent_Encode_AllFields(t *testing.T) {
	ev := SSEEvent{ID: "42", Event: "msg", Data: "line1\nline2", Retry: 3000}
	var sb strings.Builder
	if err := ev.Encode(&sb); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	want := "id: 42\nevent: msg\nretry: 3000\ndata: line1\ndata: line2\n\n"
	if got := sb.String(); got != want {
		t.Fatalf("Encode() = %q, want %q", got, want)
	}
}

func TestSSEEvent_Encode_NoOptionalFields(t *testing.T) {
	ev := SSEEvent{Data: "payload"}
	var sb strings.Builder
	if err := ev.Encode(&sb); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if got, want := sb.String(), "data: payload\n\n"; got != want {
		t.Fatalf("Encode() = %q, want %q", got, want)
	}
}

type errWriter struct {
	err error
}

func (w errWriter) Write(_ []byte) (int, error) {
	return 0, w.err
}

func TestSSEEvent_Encode_PropagatesWriterError(t *testing.T) {
	wantErr := errors.New("write failed")
	err := SSEEvent{Data: "payload"}.Encode(errWriter{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Encode() error = %v, want %v", err, wantErr)
	}
}

func TestNewSSEResponse_Headers(t *testing.T) {
	resp, sse := NewSSEResponse()
	defer sse.Close()

	if ct := resp.Headers.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	if cc := resp.Headers.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}
	if resp.Status != 200 {
		t.Errorf("status = %d, want 200", resp.Status)
	}
}

func TestNewSSEResponse_RoundTrip(t *testing.T) {
	resp, sse := NewSSEResponse()

	go func() {
		_ = sse.Send(SSEEvent{Event: "hello", Data: "world"})
		sse.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if got, want := string(data), "event: hello\ndata: world\n\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

// ---- HXCSRFHeaders ----

func TestHXCSRFHeaders_BasicToken(t *testing.T) {
	got := renderAttrInDiv(t, HXCSRFHeaders("abc123"))
	want := `<div hx-headers="{&#34;X-CSRF-Token&#34;:&#34;abc123&#34;}"></div>`
	if got != want {
		t.Errorf("HXCSRFHeaders = %q, want %q", got, want)
	}
}

func TestHXCSRFHeaders_TokenWithQuote(t *testing.T) {
	got := renderAttrInDiv(t, HXCSRFHeaders(`tok"en`))
	want := `<div hx-headers="{&#34;X-CSRF-Token&#34;:&#34;tok\&#34;en&#34;}"></div>`
	if got != want {
		t.Errorf("HXCSRFHeaders = %q, want %q", got, want)
	}
}

// ---- HXCSRFMeta ----

func TestHXCSRFMeta_BasicToken(t *testing.T) {
	node := HXCSRFMeta("tok")
	got := RenderNode(t, node)
	want := `<meta name="htmx-config" content="{&#34;antiForgery&#34;:{&#34;headerName&#34;:&#34;X-CSRF-Token&#34;,&#34;parameterName&#34;:&#34;_csrf&#34;,&#34;token&#34;:&#34;tok&#34;}}">`
	if got != want {
		t.Errorf("HXCSRFMeta = %q, want %q", got, want)
	}
}

func TestHXCSRFMeta_TokenAppearsInContent(t *testing.T) {
	node := HXCSRFMeta("mytoken42")
	got := RenderNode(t, node)
	want := `<meta name="htmx-config" content="{&#34;antiForgery&#34;:{&#34;headerName&#34;:&#34;X-CSRF-Token&#34;,&#34;parameterName&#34;:&#34;_csrf&#34;,&#34;token&#34;:&#34;mytoken42&#34;}}">`
	if got != want {
		t.Errorf("HXCSRFMeta = %q, want %q", got, want)
	}
}

// renderAttrInDiv renders an attribute node inside a div to get proper output.
func renderAttrInDiv(t *testing.T, attr interface{ Render(io.Writer) error }) string {
	return renderAttrInTag(t, "div", attr)
}

func renderAttrInTag(t *testing.T, tag string, attr interface{ Render(io.Writer) error }) string {
	t.Helper()
	import_ := `<` + tag
	var sb strings.Builder
	sb.WriteString(import_)
	_ = attr.Render(&sb)
	sb.WriteString(`></` + tag + `>`)
	return sb.String()
}
