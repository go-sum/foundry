package email

import "strings"

// PlainText joins lines with CRLF line endings as required by RFC 5322.
// Use it to compose the plain-text fallback body of an email message.
//
//	body := email.PlainText(
//	    "Hello, "+name+"!",
//	    "",
//	    "Thanks for reaching out.",
//	)
func PlainText(lines ...string) string {
	return strings.Join(lines, "\r\n")
}
