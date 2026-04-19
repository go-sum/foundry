package feedback

import (
	"fmt"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ProgressVariant selects the colour of a progress bar.
type ProgressVariant string

const (
	ProgressDefault ProgressVariant = "default"
	ProgressSuccess ProgressVariant = "success"
	ProgressDanger  ProgressVariant = "danger"
	ProgressWarning ProgressVariant = "warning"
)

// ProgressSize selects the height of a progress bar.
type ProgressSize string

const (
	ProgressSm ProgressSize = "sm"
	ProgressLg ProgressSize = "lg"
)

// ProgressProps configures a progress bar.
type ProgressProps struct {
	ID        string
	Max       int
	Value     int
	Label     string
	ShowValue bool
	Size      ProgressSize
	Variant   ProgressVariant
	Extra     []g.Node
}

var _progressSizeClasses = map[ProgressSize]string{
	ProgressSm: "h-1",
	ProgressLg: "h-4",
}

func progressSizeClass(s ProgressSize) string {
	if c, ok := _progressSizeClasses[s]; ok {
		return c
	}
	return "h-2.5"
}

var _progressVariantClasses = map[ProgressVariant]string{
	ProgressSuccess: "progress-success",
	ProgressDanger:  "progress-danger",
	ProgressWarning: "progress-warning",
}

func progressVariantClass(v ProgressVariant) string {
	if c, ok := _progressVariantClasses[v]; ok {
		return c
	}
	return "progress-default"
}

func progressMax(max int) int {
	if max <= 0 {
		return 100
	}
	return max
}

func progressPercent(value, max int) int {
	m := progressMax(max)
	if value < 0 {
		value = 0
	}
	if value > m {
		value = m
	}
	return (value * 100) / m
}

// Progress renders a labelled progress bar.
func Progress(p ProgressProps) g.Node {
	pct := progressPercent(p.Value, p.Max)
	nodes := []g.Node{h.Class("w-full")}
	if p.ID != "" {
		nodes = append(nodes, h.ID(p.ID))
	}
	nodes = append(nodes, g.Group(p.Extra))
	if p.Label != "" || p.ShowValue {
		labelNodes := []g.Node{h.Class("flex justify-between items-center mb-1")}
		if p.Label != "" {
			labelNodes = append(labelNodes, h.Span(h.Class("text-sm font-medium"), g.Text(p.Label)))
		}
		if p.ShowValue {
			labelNodes = append(labelNodes, h.Span(h.Class("text-sm font-medium"), g.Textf("%d%%", pct)))
		}
		nodes = append(nodes, h.Div(labelNodes...))
	}
	nodes = append(nodes, h.Progress(
		h.Class("progress-bar "+progressSizeClass(p.Size)+" "+progressVariantClass(p.Variant)),
		g.Attr("max", fmt.Sprintf("%d", progressMax(p.Max))),
		g.Attr("value", fmt.Sprintf("%d", p.Value)),
	))
	return h.Div(nodes...)
}
