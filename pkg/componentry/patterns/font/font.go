// Package font provides Gomponents helpers for loading web fonts in <head>.
//
// It supports three remote providers and one self-hosted strategy:
//
//   - [Google]: Google Fonts via preconnect + stylesheet link
//   - [Bunny]: Bunny Fonts (privacy-friendly Google Fonts alternative)
//   - [Adobe]: Adobe Fonts (Typekit) via stylesheet link
//   - [Self]: Self-hosted fonts via preload hints + @font-face declarations
//
// All provider types implement [Provider] so callers can collect them with [Nodes]:
//
//	nodes := font.Nodes(
//	    font.Google("Inter", "Roboto Mono"),
//	    font.Self(font.Face{
//	        Family: "MyFont",
//	        Src:    "/public/fonts/myfont-regular.woff2",
//	    }),
//	)
//
// The returned []g.Node slice is suitable for inclusion in a <head> element.
package font

import (
	"fmt"
	"strings"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Provider is implemented by any font loading strategy.
type Provider interface {
	// Nodes returns the HTML nodes to render (link/style tags).
	Nodes() []g.Node
	// CSPSources returns the Content-Security-Policy additions this provider requires.
	CSPSources() CSPSources
}

// CSPSources holds the CSP directive values a provider requires.
type CSPSources struct {
	StyleSrc []string
	FontSrc  []string
}

// Face describes a single font face for self-hosted fonts.
type Face struct {
	// Family is the CSS font-family name, e.g. "Inter".
	Family string
	// Src is the URL to the font file.
	Src string
	// Style is the CSS font-style value. Defaults to "normal" when empty.
	Style string
	// Weight is the CSS font-weight value. Defaults to "400" when empty.
	Weight string
	// Display is the CSS font-display value. Defaults to "swap" when empty.
	Display string
}

// Nodes collects and returns all <head> nodes from the given providers.
// The returned slice is nil when no providers are supplied.
func Nodes(providers ...Provider) []g.Node {
	var all []g.Node
	for _, p := range providers {
		all = append(all, p.Nodes()...)
	}
	return all
}

// CollectCSPSources merges the CSPSources from all providers into a single
// result, deduplicating within each directive.
func CollectCSPSources(providers ...Provider) CSPSources {
	var result CSPSources
	for _, p := range providers {
		srcs := p.CSPSources()
		result.StyleSrc = appendUnique(result.StyleSrc, srcs.StyleSrc...)
		result.FontSrc = appendUnique(result.FontSrc, srcs.FontSrc...)
	}
	return result
}

// appendUnique appends values to dst, skipping any already present.
func appendUnique(dst []string, values ...string) []string {
	for _, v := range values {
		if !contains(dst, v) {
			dst = append(dst, v)
		}
	}
	return dst
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// --- Google Fonts ---

// googleProvider loads fonts from Google Fonts.
type googleProvider struct {
	families []string
}

// Google returns a Provider that loads the given font families from Google Fonts.
// Family names are URL-encoded (spaces replaced with +).
func Google(families ...string) Provider {
	return &googleProvider{families: families}
}

func (p *googleProvider) Nodes() []g.Node {
	if len(p.families) == 0 {
		return nil
	}
	encoded := make([]string, len(p.families))
	for i, f := range p.families {
		encoded[i] = strings.ReplaceAll(f, " ", "+")
	}
	href := "https://fonts.googleapis.com/css2?family=" +
		strings.Join(encoded, "&family=") +
		"&display=swap"
	return []g.Node{
		h.Link(h.Rel("preconnect"), h.Href("https://fonts.googleapis.com")),
		h.Link(h.Rel("preconnect"), h.Href("https://fonts.gstatic.com"), g.Attr("crossorigin", "")),
		h.Link(h.Rel("stylesheet"), h.Href(href)),
	}
}

func (p *googleProvider) CSPSources() CSPSources {
	if len(p.families) == 0 {
		return CSPSources{}
	}
	return CSPSources{
		StyleSrc: []string{"https://fonts.googleapis.com"},
		FontSrc:  []string{"https://fonts.gstatic.com"},
	}
}

// --- Bunny Fonts ---

// bunnyProvider loads fonts from Bunny Fonts (https://fonts.bunny.net),
// a GDPR-friendly alternative to Google Fonts.
type bunnyProvider struct {
	families []string
}

// Bunny returns a Provider that loads the given font families from Bunny Fonts.
func Bunny(families ...string) Provider {
	return &bunnyProvider{families: families}
}

func (p *bunnyProvider) Nodes() []g.Node {
	if len(p.families) == 0 {
		return nil
	}
	href := "https://fonts.bunny.net/css?family=" +
		strings.Join(p.families, "&family=") +
		"&display=swap"
	return []g.Node{
		h.Link(h.Rel("preconnect"), h.Href("https://fonts.bunny.net")),
		h.Link(h.Rel("stylesheet"), h.Href(href)),
	}
}

func (p *bunnyProvider) CSPSources() CSPSources {
	if len(p.families) == 0 {
		return CSPSources{}
	}
	return CSPSources{
		StyleSrc: []string{"https://fonts.bunny.net"},
		FontSrc:  []string{"https://fonts.bunny.net"},
	}
}

// --- Adobe Fonts ---

// adobeProvider loads fonts from Adobe Fonts (Typekit).
type adobeProvider struct {
	projectID string
}

// Adobe returns a Provider that loads fonts from the given Adobe Fonts project.
func Adobe(projectID string) Provider {
	return &adobeProvider{projectID: projectID}
}

func (p *adobeProvider) Nodes() []g.Node {
	if p.projectID == "" {
		return nil
	}
	return []g.Node{
		h.Link(h.Rel("stylesheet"), h.Href("https://use.typekit.net/"+p.projectID+".css")),
	}
}

func (p *adobeProvider) CSPSources() CSPSources {
	if p.projectID == "" {
		return CSPSources{}
	}
	return CSPSources{
		StyleSrc: []string{"https://use.typekit.net"},
		FontSrc:  []string{"https://use.typekit.net", "https://p.typekit.net"},
	}
}

// --- Self-hosted Fonts ---

// selfProvider renders preload hints and @font-face declarations for
// self-hosted font files.
type selfProvider struct {
	faces []Face
}

// Self returns a Provider that renders preload hints and @font-face declarations
// for the given font faces. Style defaults to "normal", Weight to "400", and
// Display to "swap" when empty.
func Self(faces ...Face) Provider {
	return &selfProvider{faces: faces}
}

func (p *selfProvider) Nodes() []g.Node {
	if len(p.faces) == 0 {
		return nil
	}

	nodes := make([]g.Node, 0, len(p.faces)+1)

	// Emit one <link rel="preload"> per face.
	for _, f := range p.faces {
		if f.Src == "" {
			continue
		}
		nodes = append(nodes,
			h.Link(
				h.Rel("preload"),
				g.Attr("as", "font"),
				h.Href(f.Src),
				g.Attr("crossorigin", ""),
			),
		)
	}

	// Emit a single <style> block with all @font-face rules.
	css := buildFontFaceCSS(p.faces)
	if css != "" {
		nodes = append(nodes, h.StyleEl(g.Raw(css)))
	}
	return nodes
}

func (p *selfProvider) CSPSources() CSPSources {
	return CSPSources{}
}

// buildFontFaceCSS returns the raw CSS string for all @font-face rules.
func buildFontFaceCSS(faces []Face) string {
	var sb strings.Builder
	for _, f := range faces {
		if f.Src == "" || f.Family == "" {
			continue
		}
		style := orDefault(f.Style, "normal")
		weight := orDefault(f.Weight, "400")
		display := orDefault(f.Display, "swap")

		fmt.Fprintf(&sb, "@font-face { font-family: '%s'; src: url('%s'); font-style: %s; font-weight: %s; font-display: %s; }",
			f.Family, f.Src, style, weight, display)
	}
	return sb.String()
}

func orDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
