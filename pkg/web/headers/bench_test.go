package headers

import (
	"testing"
)

var sinkContentType ContentType
var sinkAccept Accept
var sinkCacheControl CacheControl

func BenchmarkParseContentType(b *testing.B) {
	input := "application/json; charset=utf-8"
	b.ResetTimer()
	for range b.N {
		ct, _ := ParseContentType(input)
		sinkContentType = ct
	}
}

func BenchmarkContentTypeString(b *testing.B) {
	ct := ContentType{MediaType: "application/json", Params: map[string]string{"charset": "utf-8"}}
	b.ResetTimer()
	for range b.N {
		_ = ct.String()
	}
}

func BenchmarkParseAccept(b *testing.B) {
	input := "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"
	b.ResetTimer()
	for range b.N {
		a, _ := ParseAccept(input)
		sinkAccept = a
	}
}

func BenchmarkParseAccept_Negotiate(b *testing.B) {
	a, _ := ParseAccept("text/html,application/json;q=0.9,*/*;q=0.8")
	offered := []string{"application/json", "text/html", "text/plain"}
	b.ResetTimer()
	for range b.N {
		_ = a.Negotiate(offered...)
	}
}

func BenchmarkParseCacheControl(b *testing.B) {
	input := "max-age=3600, must-revalidate, no-transform"
	b.ResetTimer()
	for range b.N {
		cc, _ := ParseCacheControl(input)
		sinkCacheControl = cc
	}
}
