package serve_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-sum/web/serve"
)

// fakeInfoResponder is a mock http.ResponseWriter that also implements the
// optional informationalResponder interface used by WriteEarlyHints.
type fakeInfoResponder struct {
	*httptest.ResponseRecorder
	capturedStatus int
	capturedHeader http.Header
	callCount      int
}

func (f *fakeInfoResponder) WriteInfoHeader(statusCode int, header http.Header) error {
	f.callCount++
	f.capturedStatus = statusCode
	f.capturedHeader = header
	return nil
}

func TestWriteEarlyHints_NilLinks(t *testing.T) {
	rec := httptest.NewRecorder()
	// Must not panic and must be a no-op.
	serve.WriteEarlyHints(rec, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (recorder default)", rec.Code, http.StatusOK)
	}
}

func TestWriteEarlyHints_EmptyLinks(t *testing.T) {
	rec := httptest.NewRecorder()
	serve.WriteEarlyHints(rec, []string{})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (recorder default)", rec.Code, http.StatusOK)
	}
}

func TestWriteEarlyHints_StandardRecorder_NoOp(t *testing.T) {
	// httptest.ResponseRecorder does not implement informationalResponder,
	// so WriteEarlyHints must be a no-op — no status change, no body write.
	rec := httptest.NewRecorder()
	serve.WriteEarlyHints(rec, []string{`</static/app.css>; rel=preload; as=style`})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (recorder default)", rec.Code, http.StatusOK)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
}

func TestWriteEarlyHints_InformationalResponder_SingleLink(t *testing.T) {
	fake := &fakeInfoResponder{ResponseRecorder: httptest.NewRecorder()}
	link := `</static/app.css>; rel=preload; as=style`

	serve.WriteEarlyHints(fake, []string{link})

	if fake.callCount != 1 {
		t.Fatalf("WriteInfoHeader called %d times, want 1", fake.callCount)
	}
	if fake.capturedStatus != http.StatusEarlyHints {
		t.Fatalf("status = %d, want %d", fake.capturedStatus, http.StatusEarlyHints)
	}
	links := fake.capturedHeader.Values("Link")
	if len(links) != 1 {
		t.Fatalf("Link header count = %d, want 1", len(links))
	}
	if links[0] != link {
		t.Fatalf("Link[0] = %q, want %q", links[0], link)
	}
}

func TestWriteEarlyHints_InformationalResponder_MultipleLinks(t *testing.T) {
	fake := &fakeInfoResponder{ResponseRecorder: httptest.NewRecorder()}
	linkCSS := `</static/app.css>; rel=preload; as=style`
	linkJS := `</static/app.js>; rel=preload; as=script`

	serve.WriteEarlyHints(fake, []string{linkCSS, linkJS})

	if fake.callCount != 1 {
		t.Fatalf("WriteInfoHeader called %d times, want 1", fake.callCount)
	}
	if fake.capturedStatus != http.StatusEarlyHints {
		t.Fatalf("status = %d, want %d", fake.capturedStatus, http.StatusEarlyHints)
	}
	links := fake.capturedHeader.Values("Link")
	if len(links) != 2 {
		t.Fatalf("Link header count = %d, want 2", len(links))
	}
	if links[0] != linkCSS {
		t.Fatalf("Link[0] = %q, want %q", links[0], linkCSS)
	}
	if links[1] != linkJS {
		t.Fatalf("Link[1] = %q, want %q", links[1], linkJS)
	}
}
