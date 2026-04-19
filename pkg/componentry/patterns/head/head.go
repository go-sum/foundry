// Package head provides reusable HTML <head> element builders for page layouts.
// It stays generic by accepting concrete asset/meta values from callers rather
// than importing application-specific config or asset packages.
package head

import (
	"cmp"
	"strings"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Props configures a complete <head> element.
type Props struct {
	Meta        MetaProps
	Stylesheets []Stylesheet
	Scripts     []Script
	Extra       []g.Node // font nodes, theme script, CSP meta, etc.
}

// MetaProps configures the standard metadata emitted in <head>.
type MetaProps struct {
	Title       string
	Description string
	Favicon     string
	Keywords    []string
	Canonical   string     // <link rel="canonical">
	Robots      string     // noindex, nofollow, etc.
	OG          *OpenGraph
}

// OpenGraph configures Open Graph meta tags.
type OpenGraph struct {
	Title       string
	Type        string // default "website" when Title is set
	Description string
	Image       string
	URL         string
}

// Stylesheet configures a single <link rel="stylesheet"> tag.
type Stylesheet struct {
	Href      string
	Integrity string // SRI hash for CDN stylesheets
	Extra     []g.Node
}

// Script configures a single external <script> tag.
type Script struct {
	Src   string
	Defer bool
	Async bool
	Extra []g.Node
}

// Head renders a complete <head> element from typed metadata, assets, and caller extras.
// Always emits charset and viewport meta tags.
func Head(p Props) g.Node {
	nodes := []g.Node{
		h.Meta(h.Charset("UTF-8")),
		h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1.0")),
		Metatags(p.Meta),
		CSS(p.Stylesheets...),
		JS(p.Scripts...),
	}
	if len(p.Extra) > 0 {
		nodes = append(nodes, g.Group(p.Extra))
	}
	return h.Head(g.Group(nodes))
}

// Metatags renders the standard metadata block for a document head.
// Only emits tags for non-empty fields.
func Metatags(p MetaProps) g.Node {
	var nodes []g.Node
	if p.Title != "" {
		nodes = append(nodes, h.TitleEl(g.Text(p.Title)))
	}
	if p.Description != "" {
		nodes = append(nodes, h.Meta(h.Name("description"), h.Content(p.Description)))
	}
	if p.Favicon != "" {
		nodes = append(nodes, h.Link(h.Rel("icon"), h.Href(p.Favicon)))
	}
	if p.Canonical != "" {
		nodes = append(nodes, h.Link(h.Rel("canonical"), h.Href(p.Canonical)))
	}
	if p.Robots != "" {
		nodes = append(nodes, h.Meta(h.Name("robots"), h.Content(p.Robots)))
	}
	if len(p.Keywords) > 0 {
		nodes = append(nodes, h.Meta(h.Name("keywords"), h.Content(strings.Join(p.Keywords, ","))))
	}
	if p.OG != nil && p.OG.Title != "" {
		nodes = append(nodes, h.Meta(g.Attr("property", "og:title"), h.Content(p.OG.Title)))
		ogType := cmp.Or(p.OG.Type, "website")
		nodes = append(nodes, h.Meta(g.Attr("property", "og:type"), h.Content(ogType)))
		if p.OG.Description != "" {
			nodes = append(nodes, h.Meta(g.Attr("property", "og:description"), h.Content(p.OG.Description)))
		}
		if p.OG.Image != "" {
			nodes = append(nodes, h.Meta(g.Attr("property", "og:image"), h.Content(p.OG.Image)))
		}
		if p.OG.URL != "" {
			nodes = append(nodes, h.Meta(g.Attr("property", "og:url"), h.Content(p.OG.URL)))
		}
	}
	return g.Group(nodes)
}

// CSS renders stylesheet link tags for each provided Stylesheet.
// Skips entries where Href is empty.
// Adds integrity and crossorigin="anonymous" when Integrity is non-empty.
func CSS(stylesheets ...Stylesheet) g.Node {
	var nodes []g.Node
	for _, ss := range stylesheets {
		if ss.Href == "" {
			continue
		}
		attrs := []g.Node{h.Rel("stylesheet"), h.Href(ss.Href)}
		if ss.Integrity != "" {
			attrs = append(attrs, h.Integrity(ss.Integrity), h.CrossOrigin("anonymous"))
		}
		if len(ss.Extra) > 0 {
			attrs = append(attrs, ss.Extra...)
		}
		nodes = append(nodes, h.Link(attrs...))
	}
	return g.Group(nodes)
}

// JS renders external script tags for each provided Script.
// Skips entries where Src is empty.
func JS(scripts ...Script) g.Node {
	var nodes []g.Node
	for _, script := range scripts {
		if script.Src == "" {
			continue
		}
		attrs := []g.Node{h.Src(script.Src)}
		if script.Defer {
			attrs = append(attrs, h.Defer())
		}
		if script.Async {
			attrs = append(attrs, h.Async())
		}
		if len(script.Extra) > 0 {
			attrs = append(attrs, script.Extra...)
		}
		nodes = append(nodes, h.Script(attrs...))
	}
	return g.Group(nodes)
}
