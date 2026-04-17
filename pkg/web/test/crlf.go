package webtest

// CRLFCorpus returns a set of payloads that attempt header/cookie injection
// via CR, LF, and CRLF sequences. Use this corpus in security tests.
func CRLFCorpus() []string {
	return []string{
		"innocent\r\nSet-Cookie: evil=1",
		"innocent\rSet-Cookie: evil=1",
		"innocent\nSet-Cookie: evil=1",
		"innocent\r\n\r\n<script>alert(1)</script>",
		"\r\n",
		"\r",
		"\n",
		"a\x00b",
		"a\x0db",
		"a\x0ab",
	}
}
