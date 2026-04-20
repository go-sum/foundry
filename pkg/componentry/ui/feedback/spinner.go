package feedback

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// SpinnerSize selects the dimensions of a spinner.
type SpinnerSize string

const (
	SpinnerSm SpinnerSize = "sm"
	SpinnerMd SpinnerSize = "md"
	SpinnerLg SpinnerSize = "lg"
)

// SpinnerProps configures a Spinner.
type SpinnerProps struct {
	ID    string
	Size  SpinnerSize
	Label string
	Extra []g.Node
}

var _spinnerSizeClasses = map[SpinnerSize]string{
	SpinnerSm: "size-4",
	SpinnerLg: "size-8",
}

func spinnerSizeClass(s SpinnerSize) string {
	if c, ok := _spinnerSizeClasses[s]; ok {
		return c
	}
	return "size-5"
}

// Spinner renders an animated loading indicator using a spinning SVG circle.
// Add class "htmx-indicator" via Extra to auto-show during HTMX requests.
func Spinner(p SpinnerProps) g.Node {
	size := spinnerSizeClass(p.Size)
	svgNodes := []g.Node{
		h.Class("animate-spin " + size),
		g.Attr("viewBox", "0 0 24 24"),
		g.Attr("fill", "none"),
		g.Attr("xmlns", "http://www.w3.org/2000/svg"),
	}
	if p.Label != "" {
		svgNodes = append(svgNodes, g.Attr("role", "img"), g.Attr("aria-label", p.Label))
	} else {
		svgNodes = append(svgNodes, g.Attr("aria-hidden", "true"))
	}

	svgNodes = append(svgNodes,
		g.El("circle",
			g.Attr("cx", "12"), g.Attr("cy", "12"), g.Attr("r", "10"),
			g.Attr("stroke", "currentColor"), g.Attr("stroke-width", "4"),
			h.Class("opacity-25"),
		),
		g.El("path",
			g.Attr("fill", "currentColor"),
			h.Class("opacity-75"),
			g.Attr("d", "M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"),
		),
	)

	nodes := []g.Node{}
	if p.ID != "" {
		nodes = append(nodes, h.ID(p.ID))
	}
	nodes = append(nodes, g.Group(p.Extra))
	nodes = append(nodes, h.SVG(svgNodes...))

	return h.Span(nodes...)
}
