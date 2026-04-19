package core

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// IconProps configures an Icon rendered as an SVG <use> reference into a sprite file.
type IconProps struct {
	Src   string
	ID    string
	Size  string
	Label string
	Extra []g.Node
}

// Icon renders an accessible <svg><use href="sprite#id"/></svg> element.
// Decorative icons (no Label) get aria-hidden="true"; labelled icons get role="img".
func Icon(p IconProps) g.Node {
	size := iconSizeClass(p.Size)
	nodes := []g.Node{h.Class(size)}
	if p.Label != "" {
		nodes = append(nodes, g.Attr("role", "img"), g.Attr("aria-label", p.Label))
	} else {
		nodes = append(nodes, g.Attr("aria-hidden", "true"))
	}
	nodes = append(nodes, g.Group(p.Extra))

	href := p.Src + "#" + p.ID
	if p.Src == "" {
		href = "#" + p.ID
	}
	nodes = append(nodes, g.El("use", h.Href(href)))

	return h.SVG(nodes...)
}

func iconSizeClass(size string) string {
	if size != "" {
		return size
	}
	return "size-4"
}
