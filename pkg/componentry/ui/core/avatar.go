package core

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

const avatarRootClass = "relative flex h-10 w-10 shrink-0 overflow-hidden rounded-full"
const avatarImageClass = "aspect-square h-full w-full object-cover"

type avatarNS struct{}

// Avatar groups explicit avatar constructors under a namespace.
var Avatar avatarNS

// Image renders a complete avatar for a known-good image source.
func (avatarNS) Image(src, alt string, extra ...g.Node) g.Node {
	return h.Span(
		h.Class(avatarRootClass),
		g.Group(extra),
		h.Img(
			h.Class(avatarImageClass),
			h.Src(src),
			h.Alt(alt),
		),
	)
}

// Fallback renders a complete avatar placeholder.
func (avatarNS) Fallback(children ...g.Node) g.Node {
	return h.Span(
		h.Class(avatarRootClass),
		h.Span(
			h.Class("flex h-full w-full items-center justify-center rounded-full bg-muted"),
			g.Group(children),
		),
	)
}
