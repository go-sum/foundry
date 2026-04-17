package headers

import "testing"

func TestCanonicalName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"content-type", "Content-Type"},
		{"x-request-id", "X-Request-Id"},
		{"accept-encoding", "Accept-Encoding"},
		{"etag", "ETag"},
		{"www-authenticate", "WWW-Authenticate"},
		{"te", "TE"},
		{"dnt", "DNT"},
		{"content-length", "Content-Length"},
		{"authorization", "Authorization"},
		{"set-cookie", "Set-Cookie"},
		{"cache-control", "Cache-Control"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := CanonicalName(tt.input)
			if got != tt.want {
				t.Errorf("CanonicalName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsForbiddenRequestHeader(t *testing.T) {
	forbidden := []string{
		"Accept-Charset",
		"Accept-Encoding",
		"Access-Control-Request-Headers",
		"Access-Control-Request-Method",
		"Connection",
		"Content-Length",
		"Cookie",
		"Cookie2",
		"Date",
		"DNT",
		"Expect",
		"Host",
		"Keep-Alive",
		"Origin",
		"Referer",
		"Set-Cookie",
		"TE",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
		"Via",
	}
	for _, h := range forbidden {
		if !IsForbiddenRequestHeader(h) {
			t.Errorf("IsForbiddenRequestHeader(%q) = false, want true", h)
		}
	}

	allowed := []string{"Authorization", "Content-Type", "X-Custom-Header", "Accept"}
	for _, h := range allowed {
		if IsForbiddenRequestHeader(h) {
			t.Errorf("IsForbiddenRequestHeader(%q) = true, want false", h)
		}
	}
}

func TestIsForbiddenResponseHeader(t *testing.T) {
	if !IsForbiddenResponseHeader("Set-Cookie") {
		t.Error("expected Set-Cookie to be forbidden response header")
	}
	if !IsForbiddenResponseHeader("set-cookie2") {
		t.Error("expected set-cookie2 to be forbidden response header")
	}
	if IsForbiddenResponseHeader("Content-Type") {
		t.Error("expected Content-Type to not be a forbidden response header")
	}
}
