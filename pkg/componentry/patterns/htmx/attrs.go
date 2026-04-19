// Package htmx provides typed hx-* attribute builders for HTMX-enhanced elements.
package htmx

import (
	"encoding/json"
	"strconv"

	g "maragu.dev/gomponents"
)

// AttrsProps describes typed hx-* attributes for a single enhanced element.
type AttrsProps struct {
	Get         string
	Post        string
	Put         string
	Patch       string
	Delete      string
	Target      string
	Swap        string
	Select      string
	SelectOOB   string
	Trigger     string
	Include     string
	Indicator   string
	DisabledElt string
	Sync        string
	Confirm     string
	Encoding    string
	PushURL     string
	ReplaceURL  string
	Params      string
	Values      map[string]string
	Headers     map[string]string
	Boost       *bool
	Extra       []g.Node
}

// Attrs renders hx-* attributes for p.
func Attrs(p AttrsProps) []g.Node {
	nodes := make([]g.Node, 0, len(p.Extra)+20)
	appendAttr := func(name, value string) {
		if value != "" {
			nodes = append(nodes, g.Attr(name, value))
		}
	}

	appendAttr("hx-get", p.Get)
	appendAttr("hx-post", p.Post)
	appendAttr("hx-put", p.Put)
	appendAttr("hx-patch", p.Patch)
	appendAttr("hx-delete", p.Delete)
	appendAttr("hx-target", p.Target)
	appendAttr("hx-swap", p.Swap)
	appendAttr("hx-select", p.Select)
	appendAttr("hx-select-oob", p.SelectOOB)
	appendAttr("hx-trigger", p.Trigger)
	appendAttr("hx-include", p.Include)
	appendAttr("hx-indicator", p.Indicator)
	appendAttr("hx-disabled-elt", p.DisabledElt)
	appendAttr("hx-sync", p.Sync)
	appendAttr("hx-confirm", p.Confirm)
	appendAttr("hx-encoding", p.Encoding)
	appendAttr("hx-push-url", p.PushURL)
	appendAttr("hx-replace-url", p.ReplaceURL)
	appendAttr("hx-params", p.Params)

	if encoded := encodeMap(p.Values); encoded != "" {
		nodes = append(nodes, g.Attr("hx-vals", encoded))
	}
	if encoded := encodeMap(p.Headers); encoded != "" {
		nodes = append(nodes, g.Attr("hx-headers", encoded))
	}
	if p.Boost != nil {
		nodes = append(nodes, g.Attr("hx-boost", strconv.FormatBool(*p.Boost)))
	}

	nodes = append(nodes, g.Group(p.Extra))
	return nodes
}

func encodeMap(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return string(raw)
}
