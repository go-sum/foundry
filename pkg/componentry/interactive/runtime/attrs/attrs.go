// Package attrs provides typed helpers for wiring data-controller, data-action,
// and data-target attributes used by the componentry micro-runtime.
package attrs

import (
	"fmt"
	"strings"

	g "maragu.dev/gomponents"
)

// Attrs is a map of HTML attribute name to value for controller wiring.
// Values for data-controller and data-action are space-separated lists
// and are concatenated correctly by Compose.
type Attrs map[string]string

// Nodes converts the Attrs into a slice of gomponents attribute nodes.
func (a Attrs) Nodes() []g.Node {
	nodes := make([]g.Node, 0, len(a))
	for k, v := range a {
		nodes = append(nodes, g.Attr(k, v))
	}
	return nodes
}

// Controller returns Attrs with data-controller set to the space-joined names.
//
//	attrs.Controller("tabs")
//	attrs.Controller("tabs", "hotkeys")
func Controller(names ...string) Attrs {
	return Attrs{"data-controller": strings.Join(names, " ")}
}

// Action returns Attrs with data-action set to "event->controller#method".
//
//	attrs.Action("click", "dialog", "open")  // data-action="click->dialog#open"
func Action(event, controller, method string) Attrs {
	return Attrs{"data-action": fmt.Sprintf("%s->%s#%s", event, controller, method)}
}

// Target returns Attrs with data-{controller}-target="{name}".
//
//	attrs.Target("tabs", "panel")  // data-tabs-target="panel"
func Target(controller, name string) Attrs {
	return Attrs{fmt.Sprintf("data-%s-target", controller): name}
}

// Compose merges multiple Attrs into one. Values for data-controller and
// data-action are space-concatenated; all other keys are last-write-wins.
func Compose(sets ...Attrs) Attrs {
	out := Attrs{}
	for _, a := range sets {
		for k, v := range a {
			switch k {
			case "data-controller", "data-action":
				if existing, ok := out[k]; ok {
					out[k] = existing + " " + v
				} else {
					out[k] = v
				}
			default:
				out[k] = v
			}
		}
	}
	return out
}
