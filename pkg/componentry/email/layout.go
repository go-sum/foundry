// Package email provides Gomponents-based building blocks for composing
// HTML transactional emails. All components use table-based layout and inline
// styles for maximum email client compatibility.
package email

import (
	"fmt"
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// LayoutProps configures a [Layout] email document.
type LayoutProps struct {
	// Title appears in the document <title> element and as a preview fallback.
	Title string
	// BgColor is the outer background colour. Defaults to "#f4f4f4".
	BgColor string
	// ContentWidth is the inner content table width in px. Defaults to 600.
	ContentWidth int
	// FontFamily is the CSS font-family stack. Defaults to "Arial,Helvetica,sans-serif".
	FontFamily string
	// Footer is optional footer content rendered below the main body.
	Footer g.Node
}

func (p *LayoutProps) applyDefaults() {
	if p.BgColor == "" {
		p.BgColor = "#f4f4f4"
	}
	if p.ContentWidth == 0 {
		p.ContentWidth = 600
	}
	if p.FontFamily == "" {
		p.FontFamily = "Arial,Helvetica,sans-serif"
	}
}

// Layout renders a complete email-safe HTML document using table-based layout
// and inline styles. body nodes are placed inside the centred content column.
func Layout(p LayoutProps, body ...g.Node) g.Node {
	p.applyDefaults()

	widthStr := strconv.Itoa(p.ContentWidth)
	innerStyle := fmt.Sprintf(
		"background-color:#ffffff;border-radius:4px;max-width:%dpx;width:100%%;",
		p.ContentWidth,
	)
	outerBodyStyle := fmt.Sprintf(
		"background-color:%s;margin:0;padding:0;",
		p.BgColor,
	)
	outerTableStyle := fmt.Sprintf("background-color:%s;", p.BgColor)

	var footerRow g.Node
	if p.Footer != nil {
		footerRow = h.Tr(
			h.Td(
				g.Attr("align", "center"),
				g.Attr("style", "padding:16px;"),
				p.Footer,
			),
		)
	}

	return g.Group([]g.Node{
		g.Raw(`<!DOCTYPE html>`),
		h.HTML(
			h.Lang("en"),
			h.Head(
				h.Meta(g.Attr("charset", "utf-8")),
				h.Meta(
					g.Attr("name", "viewport"),
					g.Attr("content", "width=device-width,initial-scale=1"),
				),
				h.TitleEl(g.Text(p.Title)),
			),
			h.Body(
				g.Attr("bgcolor", p.BgColor),
				g.Attr("style", outerBodyStyle),
				h.Table(
					g.Attr("width", "100%"),
					g.Attr("cellpadding", "0"),
					g.Attr("cellspacing", "0"),
					g.Attr("role", "presentation"),
					g.Attr("style", outerTableStyle),
					h.TBody(
						h.Tr(
							h.Td(
								g.Attr("align", "center"),
								g.Attr("style", "padding:40px 16px;"),
								h.Table(
									g.Attr("width", widthStr),
									g.Attr("cellpadding", "0"),
									g.Attr("cellspacing", "0"),
									g.Attr("role", "presentation"),
									g.Attr("style", innerStyle),
									h.TBody(
										h.Tr(
											h.Td(
												g.Attr("style", fmt.Sprintf(
													"padding:40px;font-family:%s;font-size:16px;line-height:1.6;color:#374151;",
													p.FontFamily,
												)),
												g.Group(body),
											),
										),
										footerRow,
									),
								),
							),
						),
					),
				),
			),
		),
	})
}
