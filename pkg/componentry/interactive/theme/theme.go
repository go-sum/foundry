// Package theme provides an inline FOUC-prevention script and a cycling
// theme-selector button (light → dark → system) for use in Go HTML templates.
package theme

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// themeScriptContent is the exact JavaScript emitted by ThemeScript().
// Defined as a constant so its SHA-256 hash is computed once (ScriptCSPHash)
// and the rendered <script> always matches what the CSP authorises.
const themeScriptContent = `(function(){var p=localStorage.getItem('themePreference');if(p==='dark'||(p==='system'||!p)&&window.matchMedia('(prefers-color-scheme: dark)').matches){document.documentElement.classList.add('dark')}})();`

// selectorScriptContent is the exact JavaScript emitted by SelectorScript().
const selectorScriptContent = `(function(){var btn=document.querySelector('[data-theme-selector]');if(!btn)return;btn.addEventListener('click',function(){var themes=['light','dark','system'];var cur=localStorage.getItem('themePreference')||'system';var next=themes[(themes.indexOf(cur)+1)%themes.length];localStorage.setItem('themePreference',next);document.documentElement.setAttribute('data-theme-preference',next);if(next==='dark'||(next==='system'&&window.matchMedia('(prefers-color-scheme: dark)').matches)){document.documentElement.classList.add('dark')}else{document.documentElement.classList.remove('dark')}})})();`

// ScriptCSPHash is the ready-to-embed CSP token for the ThemeScript inline
// script. Add it to script-src in your Content-Security-Policy header:
//
//	script-src 'self' <ScriptCSPHash>
var ScriptCSPHash string

// SelectorScriptCSPHash is the ready-to-embed CSP token for the SelectorScript
// inline script.
var SelectorScriptCSPHash string

func init() {
	// Hash the bytes the browser actually receives — the inner text of the
	// rendered <script> element — so any future change to ThemeScript() or its
	// gomponents wrapper automatically keeps the hash in sync.
	var buf strings.Builder
	if err := ThemeScript().Render(&buf); err != nil {
		panic(fmt.Sprintf("theme.ThemeScript render: %v", err))
	}
	rendered := buf.String()
	inner := strings.TrimPrefix(rendered, "<script>")
	inner = strings.TrimSuffix(inner, "</script>")
	ScriptCSPHash = cspHash(inner)

	buf.Reset()
	if err := SelectorScript().Render(&buf); err != nil {
		panic(fmt.Sprintf("theme.SelectorScript render: %v", err))
	}
	rendered = buf.String()
	inner = strings.TrimPrefix(rendered, "<script>")
	inner = strings.TrimSuffix(inner, "</script>")
	SelectorScriptCSPHash = cspHash(inner)
}

// cspHash returns the 'sha256-...' token for an inline script value.
func cspHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

// ThemeScript returns a synchronous inline <script> that must be placed
// inside <head> before any body content renders. It reads the stored
// 'themePreference' key from localStorage ('light', 'dark', or 'system')
// and immediately adds the "dark" class to <html> when needed, preventing
// a flash of unstyled light-mode content on dark-preference page loads.
//
// The script is intentionally minified — it must not be deferred, and
// keeping it small reduces the blocking time to near zero.
func ThemeScript() g.Node {
	return h.Script(g.Raw(themeScriptContent))
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
// <html>. The click handler is provided by SelectorScript().
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

// SelectorScript returns the inline <script> for the ThemeSelector click
// handler. Place it after the ThemeSelector element (or at end of <body>).
// Separate from ThemeScript so apps can choose how to load each.
func SelectorScript() g.Node {
	return h.Script(g.Raw(selectorScriptContent))
}
