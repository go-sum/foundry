package webtest

import (
	"net/http"
	"testing"
)

func TestBuildRequest_WithHeaderAndBody(t *testing.T) {
	req := BuildRequest(
		http.MethodPost,
		"https://example.com/form?step=1",
		WithHeader("X-Test", "yes"),
		WithBody("text/plain", "hello"),
	)

	if got, want := req.Method, http.MethodPost; got != want {
		t.Fatalf("Method = %q, want %q", got, want)
	}
	if got, want := req.URL.String(), "https://example.com/form?step=1"; got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
	if got, want := req.Headers.Get("X-Test"), "yes"; got != want {
		t.Fatalf("X-Test = %q, want %q", got, want)
	}
	if got, want := req.Headers.Get("Content-Type"), "text/plain"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
	body, err := req.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}
	if got, want := body, "hello"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestAssertHelpers_SuccessPaths(t *testing.T) {
	AssertNoCRLF(t, "safe", "plain text")
	AssertExactHTML(t, "<div>ok</div>", "<div>ok</div>")
}

func TestCRLFCorpus(t *testing.T) {
	got := CRLFCorpus()
	if len(got) == 0 {
		t.Fatal("CRLFCorpus() = empty, want payloads")
	}
	if got[0] != "innocent\r\nSet-Cookie: evil=1" {
		t.Fatalf("CRLFCorpus()[0] = %q, want first attack sample", got[0])
	}
}
