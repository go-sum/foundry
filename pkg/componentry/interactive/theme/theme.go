// Package theme provides an inline FOUC-prevention script and a cycling
// theme-selector button (light → dark → system) for use in Go HTML templates.
package theme

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"fmt"
	"strings"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

//go:embed init.min.js
var initScriptContent string

//go:embed theme.min.js
var themeScriptContent string

// InitScriptCSPHash is the ready-to-embed CSP token for the InitScript inline
// script. Add it to script-src in your Content-Security-Policy header:
//
//	script-src 'self' <InitScriptCSPHash>
var InitScriptCSPHash string

// ThemeScriptCSPHash is the ready-to-embed CSP token for the ThemeScript inline
// script (the click handler).
var ThemeScriptCSPHash string

func init() {
	initScriptContent = strings.TrimRight(initScriptContent, "\r\n")
	themeScriptContent = strings.TrimRight(themeScriptContent, "\r\n")

	var buf strings.Builder
	if err := InitScript().Render(&buf); err != nil {
		panic(fmt.Sprintf("theme.InitScript render: %v", err))
	}
	rendered := buf.String()
	inner := strings.TrimPrefix(rendered, "<script>")
	inner = strings.TrimSuffix(inner, "</script>")
	InitScriptCSPHash = cspHash(inner)

	buf.Reset()
	if err := ThemeScript().Render(&buf); err != nil {
		panic(fmt.Sprintf("theme.ThemeScript render: %v", err))
	}
	rendered = buf.String()
	inner = strings.TrimPrefix(rendered, "<script>")
	inner = strings.TrimSuffix(inner, "</script>")
	ThemeScriptCSPHash = cspHash(inner)
}

// cspHash returns the 'sha256-...' token for an inline script value.
func cspHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

// InitScript returns a synchronous inline <script> that must be placed
// inside <head> before any body content renders. It reads the stored
// 'themePreference' key from localStorage ('light', 'dark', or 'system')
// and immediately adds the "dark" class to <html> when needed, preventing
// a flash of unstyled light-mode content on dark-preference page loads.
//
// The script is intentionally minified — it must not be deferred, and
// keeping it small reduces the blocking time to near zero.
func InitScript() g.Node {
	return h.Script(g.Raw(initScriptContent))
}

// ThemeSelectorProps configures the theme cycling button.
type ThemeSelectorProps struct {
	// LightIcon is the icon shown when data-theme-preference="light" on <html>.
	// Falls back to a text label when nil.
	LightIcon g.Node
	// DarkIcon is the icon shown when data-theme-preference="dark" on <html>.
	DarkIcon g.Node
	// SystemIcon is the icon shown when data-theme-preference="system" on <html>.
	SystemIcon g.Node
	// Extra nodes are appended inside the button element.
	Extra []g.Node
}

// ThemeSelector returns a button that cycles through light/dark/system themes
// on each click. The active state is persisted to localStorage and the .dark
// class on <html> is updated in place so the change is instant without a page
// reload.
//
// Icon visibility is controlled by CSS rules keyed on data-theme-preference on
// <html>. The click handler is provided by ThemeScript().
func ThemeSelector(p ThemeSelectorProps) g.Node {
	lightIcon := p.LightIcon
	if lightIcon == nil {
		lightIcon = g.Text("Light")
	}
	darkIcon := p.DarkIcon
	if darkIcon == nil {
		darkIcon = g.Text("Dark")
	}
	systemIcon := p.SystemIcon
	if systemIcon == nil {
		systemIcon = g.Text("System")
	}

	nodes := []g.Node{
		g.Attr("data-theme-selector", ""),
		h.Type("button"),
		g.Attr("aria-label", "Toggle theme"),
		// Light icon — visible when data-theme-preference="light" on <html>.
		h.Span(h.Class("theme-light-icon"), lightIcon),
		// Dark icon — visible when data-theme-preference="dark" on <html>.
		h.Span(h.Class("theme-dark-icon"), darkIcon),
		// System icon — visible when data-theme-preference="system" on <html>.
		h.Span(h.Class("theme-system-icon"), systemIcon),
	}
	for _, extra := range p.Extra {
		nodes = append(nodes, extra)
	}
	return h.Button(nodes...)
}

// ThemeScript returns the inline <script> for the ThemeSelector click handler.
// Place it after the ThemeSelector element (or at end of <body>).
// Separate from InitScript so apps can choose how to load each.
func ThemeScript() g.Node {
	return h.Script(g.Raw(themeScriptContent))
}
