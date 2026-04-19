package email

import (
	"strings"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// H1 renders an email-safe heading at the largest size with inline styles.
func H1(text string) g.Node {
	return h.H1(
		g.Attr("style", "font-family:Arial,Helvetica,sans-serif;font-size:28px;font-weight:bold;color:#111827;margin:0 0 16px 0;"),
		g.Text(text),
	)
}

// H2 renders an email-safe heading at the medium size with inline styles.
func H2(text string) g.Node {
	return h.H2(
		g.Attr("style", "font-family:Arial,Helvetica,sans-serif;font-size:22px;font-weight:bold;color:#111827;margin:0 0 12px 0;"),
		g.Text(text),
	)
}

// P renders an email-safe paragraph with inline styles.
func P(text string) g.Node {
	return h.P(
		g.Attr("style", "font-family:Arial,Helvetica,sans-serif;font-size:16px;line-height:1.6;color:#374151;margin:0 0 16px 0;"),
		g.Text(text),
	)
}

// Button renders a CTA button using a table-cell approach for Outlook
// compatibility. text is the button label; href is the link destination.
func Button(text, href string) g.Node {
	return h.Table(
		g.Attr("cellpadding", "0"),
		g.Attr("cellspacing", "0"),
		g.Attr("role", "presentation"),
		h.TBody(
			h.Tr(
				h.Td(
					g.Attr("style", "background-color:#111827;border-radius:6px;text-align:center;"),
					h.A(
						h.Href(href),
						g.Attr("style", "color:#ffffff;text-decoration:none;font-weight:600;padding:12px 24px;display:inline-block;font-family:Arial,Helvetica,sans-serif;font-size:16px;"),
						g.Text(text),
					),
				),
			),
		),
	)
}

// A renders a styled inline hyperlink.
func A(text, href string) g.Node {
	return h.A(
		h.Href(href),
		g.Attr("style", "color:#2563EB;text-decoration:underline;"),
		g.Text(text),
	)
}

// HR renders an email-safe horizontal rule using a table border.
func HR() g.Node {
	return h.Table(
		g.Attr("width", "100%"),
		g.Attr("cellpadding", "0"),
		g.Attr("cellspacing", "0"),
		g.Attr("role", "presentation"),
		h.TBody(
			h.Tr(
				h.Td(
					g.Attr("style", "border-top:1px solid #E5E7EB;padding:16px 0;"),
				),
			),
		),
	)
}

// previewPadding is appended after the preview text to prevent email clients
// from pulling body content into the inbox snippet.
const previewPadding = "&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;"

// PreviewText renders a hidden inbox-preview snippet. Email clients display up
// to ~200 characters of body text in the inbox list; placing this element at
// the top of the body controls what snippet is shown. text should be 100
// characters or fewer — the remainder is padded automatically.
func PreviewText(text string) g.Node {
	var sb strings.Builder
	sb.WriteString(text)
	sb.WriteString("&zwnj;&nbsp;")
	sb.WriteString(previewPadding)
	return g.El("div",
		g.Attr("style", "display:none;max-height:0;overflow:hidden;mso-hide:all;"),
		g.Raw(sb.String()),
	)
}
