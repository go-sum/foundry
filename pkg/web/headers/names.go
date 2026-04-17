package headers

import "strings"

// specialCases maps lowercase segment tokens to their canonical form.
var specialCases = map[string]string{
	"etag": "ETag",
	"www":  "WWW",
	"te":   "TE",
	"dnt":  "DNT",
	"csp":  "CSP",
	"atb":  "ATB",
}

// CanonicalName returns the canonical Title-Case HTTP header name.
// Examples: "content-type" → "Content-Type", "x-request-id" → "X-Request-Id".
// Special cases (case-preserved): "ETag", "WWW-Authenticate", "TE", "DNT".
func CanonicalName(name string) string {
	segments := strings.Split(name, "-")
	for i, seg := range segments {
		lower := strings.ToLower(seg)
		if special, ok := specialCases[lower]; ok {
			segments[i] = special
		} else if len(seg) > 0 {
			segments[i] = strings.ToUpper(seg[:1]) + strings.ToLower(seg[1:])
		}
	}
	return strings.Join(segments, "-")
}

// forbiddenRequestHeaders is the WHATWG fetch forbidden request header list.
// https://fetch.spec.whatwg.org/#forbidden-request-header
var forbiddenRequestHeaders = map[string]bool{
	"accept-charset":                true,
	"accept-encoding":               true,
	"access-control-request-headers": true,
	"access-control-request-method": true,
	"connection":                    true,
	"content-length":                true,
	"cookie":                        true,
	"cookie2":                       true,
	"date":                          true,
	"dnt":                           true,
	"expect":                        true,
	"host":                          true,
	"keep-alive":                    true,
	"origin":                        true,
	"referer":                       true,
	"set-cookie":                    true,
	"te":                            true,
	"trailer":                       true,
	"transfer-encoding":             true,
	"upgrade":                       true,
	"via":                           true,
}

// IsForbiddenRequestHeader reports whether a header name is one that
// fetch callers must not set (controlled by the browser/transport).
// See: https://fetch.spec.whatwg.org/#forbidden-request-header
func IsForbiddenRequestHeader(name string) bool {
	return forbiddenRequestHeaders[strings.ToLower(name)]
}

// forbiddenResponseHeaders is the set of response headers forbidden from
// being exposed in fetch responses.
var forbiddenResponseHeaders = map[string]bool{
	"set-cookie":  true,
	"set-cookie2": true,
}

// IsForbiddenResponseHeader reports whether a header is forbidden
// to expose in fetch responses.
func IsForbiddenResponseHeader(name string) bool {
	return forbiddenResponseHeaders[strings.ToLower(name)]
}
