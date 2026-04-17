package headers

import (
	"testing"
)

func intPtr(n int) *int { return &n }

func TestParseCacheControl(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  CacheControl
	}{
		{
			name:  "empty input",
			input: "",
			want:  CacheControl{},
		},
		{
			name:  "no-cache no-store",
			input: "no-cache, no-store",
			want:  CacheControl{NoCache: true, NoStore: true},
		},
		{
			name:  "max-age",
			input: "max-age=3600",
			want:  CacheControl{MaxAge: intPtr(3600)},
		},
		{
			name:  "s-maxage",
			input: "s-maxage=86400",
			want:  CacheControl{SMaxAge: intPtr(86400)},
		},
		{
			name:  "public max-age",
			input: "public, max-age=600",
			want:  CacheControl{Public: true, MaxAge: intPtr(600)},
		},
		{
			name:  "private no-transform",
			input: "private, no-transform",
			want:  CacheControl{Private: true, NoTransform: true},
		},
		{
			name:  "must-revalidate",
			input: "must-revalidate",
			want:  CacheControl{MustRevalidate: true},
		},
		{
			name:  "proxy-revalidate",
			input: "proxy-revalidate",
			want:  CacheControl{ProxyRevalidate: true},
		},
		{
			name:  "must-understand",
			input: "must-understand",
			want:  CacheControl{MustUnderstand: true},
		},
		{
			name:  "immutable",
			input: "immutable",
			want:  CacheControl{Immutable: true},
		},
		{
			name:  "stale-while-revalidate",
			input: "stale-while-revalidate=60",
			want:  CacheControl{StaleWhileRevalidate: intPtr(60)},
		},
		{
			name:  "stale-if-error",
			input: "stale-if-error=300",
			want:  CacheControl{StaleIfError: intPtr(300)},
		},
		{
			name:  "only-if-cached",
			input: "only-if-cached",
			want:  CacheControl{OnlyIfCached: true},
		},
		{
			name:  "max-stale with value",
			input: "max-stale=100",
			want:  CacheControl{MaxStale: intPtr(100)},
		},
		{
			name:  "max-stale without value",
			input: "max-stale",
			want:  CacheControl{MaxStale: intPtr(0)},
		},
		{
			name:  "min-fresh",
			input: "min-fresh=60",
			want:  CacheControl{MinFresh: intPtr(60)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCacheControl(tt.input)
			if err != nil {
				t.Fatalf("ParseCacheControl(%q) error: %v", tt.input, err)
			}
			// Compare bool fields
			if got.NoCache != tt.want.NoCache {
				t.Errorf("NoCache: got %v, want %v", got.NoCache, tt.want.NoCache)
			}
			if got.NoStore != tt.want.NoStore {
				t.Errorf("NoStore: got %v, want %v", got.NoStore, tt.want.NoStore)
			}
			if got.Public != tt.want.Public {
				t.Errorf("Public: got %v, want %v", got.Public, tt.want.Public)
			}
			if got.Private != tt.want.Private {
				t.Errorf("Private: got %v, want %v", got.Private, tt.want.Private)
			}
			if got.Immutable != tt.want.Immutable {
				t.Errorf("Immutable: got %v, want %v", got.Immutable, tt.want.Immutable)
			}
			if got.MustRevalidate != tt.want.MustRevalidate {
				t.Errorf("MustRevalidate: got %v, want %v", got.MustRevalidate, tt.want.MustRevalidate)
			}
			if got.ProxyRevalidate != tt.want.ProxyRevalidate {
				t.Errorf("ProxyRevalidate: got %v, want %v", got.ProxyRevalidate, tt.want.ProxyRevalidate)
			}
			if got.MustUnderstand != tt.want.MustUnderstand {
				t.Errorf("MustUnderstand: got %v, want %v", got.MustUnderstand, tt.want.MustUnderstand)
			}
			if got.NoTransform != tt.want.NoTransform {
				t.Errorf("NoTransform: got %v, want %v", got.NoTransform, tt.want.NoTransform)
			}
			if got.OnlyIfCached != tt.want.OnlyIfCached {
				t.Errorf("OnlyIfCached: got %v, want %v", got.OnlyIfCached, tt.want.OnlyIfCached)
			}
			// Compare pointer fields
			compareIntPtr(t, "MaxAge", got.MaxAge, tt.want.MaxAge)
			compareIntPtr(t, "SMaxAge", got.SMaxAge, tt.want.SMaxAge)
			compareIntPtr(t, "StaleWhileRevalidate", got.StaleWhileRevalidate, tt.want.StaleWhileRevalidate)
			compareIntPtr(t, "StaleIfError", got.StaleIfError, tt.want.StaleIfError)
			compareIntPtr(t, "MaxStale", got.MaxStale, tt.want.MaxStale)
			compareIntPtr(t, "MinFresh", got.MinFresh, tt.want.MinFresh)
		})
	}
}

func compareIntPtr(t *testing.T, name string, got, want *int) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Errorf("%s: got %v, want %v", name, got, want)
		return
	}
	if got != nil && *got != *want {
		t.Errorf("%s: got %d, want %d", name, *got, *want)
	}
}

func TestCacheControlRoundTrip(t *testing.T) {
	inputs := []string{
		"public, max-age=3600",
		"no-cache, no-store",
		"private, max-age=0, no-transform",
		"s-maxage=86400, stale-while-revalidate=60",
		"max-stale=100, min-fresh=60",
		"immutable",
		"only-if-cached",
		"must-revalidate, proxy-revalidate, must-understand",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			cc, err := ParseCacheControl(input)
			if err != nil {
				t.Fatalf("ParseCacheControl error: %v", err)
			}
			serialized := cc.String()
			cc2, err := ParseCacheControl(serialized)
			if err != nil {
				t.Fatalf("round-trip ParseCacheControl error: %v", err)
			}
			// Verify round trip produces same struct
			if cc2.String() != serialized {
				t.Errorf("round-trip mismatch:\n  original serialized: %q\n  re-serialized:       %q", serialized, cc2.String())
			}
		})
	}
}
