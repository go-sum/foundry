package feedback

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ToastVariant selects the colour of a toast.
type ToastVariant string

const (
	ToastDefault ToastVariant = "default"
	ToastSuccess ToastVariant = "success"
	ToastError   ToastVariant = "error"
	ToastWarning ToastVariant = "warning"
	ToastInfo    ToastVariant = "info"
)

// ToastPosition selects where the toast appears on screen.
type ToastPosition string

const (
	PositionTopRight     ToastPosition = "top-right"
	PositionTopLeft      ToastPosition = "top-left"
	PositionTopCenter    ToastPosition = "top-center"
	PositionBottomRight  ToastPosition = "bottom-right"
	PositionBottomLeft   ToastPosition = "bottom-left"
	PositionBottomCenter ToastPosition = "bottom-center"
)

// ToastProps configures a static server-rendered toast notification.
type ToastProps struct {
	ID          string
	Title       string
	Description string
	Variant     ToastVariant
	Position    ToastPosition
	Dismissible bool
	Extra       []g.Node
}

var _toastVariantClasses = map[ToastVariant]string{
	ToastSuccess: "backdrop-blur-sm border-success/30 bg-success/20 text-success min-w-sm",
	ToastError:   "backdrop-blur-sm border-destructive/30 bg-destructive/20 text-destructive min-w-sm",
	ToastWarning: "backdrop-blur-sm border-warning/30 bg-warning/20 text-warning min-w-sm",
	ToastInfo:    "backdrop-blur-sm border-primary/30 bg-primary/20 text-primary min-w-sm",
}

func toastVariantClasses(v ToastVariant) string {
	if c, ok := _toastVariantClasses[v]; ok {
		return c
	}
	return "backdrop-blur-sm border-primary/30 bg-background text-muted-foreground min-w-sm"
}

func toastAnnouncementAttrs(v ToastVariant) []g.Node {
	if v == ToastError || v == ToastWarning {
		return []g.Node{
			h.Role("alert"),
			g.Attr("aria-live", "assertive"),
			g.Attr("aria-atomic", "true"),
		}
	}
	return []g.Node{
		h.Role("status"),
		g.Attr("aria-live", "polite"),
		g.Attr("aria-atomic", "true"),
	}
}

var _toastPositionClasses = map[ToastPosition]string{
	PositionTopRight:     "top-4 right-4",
	PositionTopLeft:      "top-4 left-4",
	PositionTopCenter:    "top-4 left-1/2 -translate-x-1/2",
	PositionBottomLeft:   "bottom-4 left-4",
	PositionBottomCenter: "bottom-4 left-1/2 -translate-x-1/2",
	PositionBottomRight:  "bottom-4 right-4",
}

func toastPositionClasses(p ToastPosition) string {
	if c, ok := _toastPositionClasses[p]; ok {
		return c
	}
	return ""
}

func toastDismissButton() g.Node {
	return h.Button(
		g.Attr("data-dismiss", "alert"),
		h.Class("absolute top-2 right-2 opacity-50 hover:opacity-100 transition-opacity outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50"),
		h.Type("button"),
		g.Attr("aria-label", "Dismiss"),
		g.Text("×"),
	)
}

// Toast renders a toast notification. When Position is set, the toast is
// fixed-positioned and self-contained. When Position is "" (zero value), the
// toast renders as a plain card suitable for injection into a container div.
func Toast(p ToastProps) g.Node {
	cardCls := "relative rounded-lg border p-4 shadow-md " + toastVariantClasses(p.Variant)
	var fixedCls string
	if p.Position != "" {
		fixedCls = "fixed z-50 max-w-sm " + toastPositionClasses(p.Position) + " "
	}
	cls := fixedCls + cardCls
	nodes := []g.Node{h.Class(cls)}
	nodes = append(nodes, toastAnnouncementAttrs(p.Variant)...)
	if p.ID != "" {
		nodes = append(nodes, h.ID(p.ID))
	}
	if p.Dismissible {
		nodes = append(nodes, g.Attr("data-dismissible", ""))
	}
	nodes = append(nodes, g.Group(p.Extra))
	if p.Title != "" {
		nodes = append(nodes, h.P(h.Class("font-medium text-sm"), g.Text(p.Title)))
	}
	if p.Description != "" {
		nodes = append(nodes, h.P(h.Class("text-sm mt-1 opacity-80"), g.Text(p.Description)))
	}
	if p.Dismissible {
		nodes = append(nodes, toastDismissButton())
	}
	return h.Div(nodes...)
}
