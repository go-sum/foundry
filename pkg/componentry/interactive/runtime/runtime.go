// Package runtime provides the componentry micro-runtime script for embedding
// in Go HTML templates. The runtime implements a lightweight Stimulus-style
// controller system with HTMX lifecycle integration.
//
// Include Script() at the end of <body> in your page layout. The bundled file
// is produced by the asset pipeline (assets build js) and must be present
// before this package can be compiled.
package runtime

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"strings"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

//go:embed componentry.min.js
var scriptContent string

// ScriptCSPHash is the ready-to-embed CSP token for the runtime inline script.
// Add it to script-src in your Content-Security-Policy header:
//
//	script-src 'self' <ScriptCSPHash>
var ScriptCSPHash string

func init() {
	scriptContent = strings.TrimRight(scriptContent, "\r\n")
	var b strings.Builder
	if err := Script().Render(&b); err != nil {
		panic("runtime.Script render: " + err.Error())
	}
	rendered := b.String()
	inner := strings.TrimPrefix(rendered, "<script>")
	inner = strings.TrimSuffix(inner, "</script>")
	ScriptCSPHash = cspHash(inner)
}

func cspHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

// Script returns an inline <script> containing the componentry micro-runtime.
// Suitable for zero-dependency deployments or when a CSP hash is preferred over
// a file URL. Use ScriptSrc for cacheable file-based delivery in production.
func Script() g.Node {
	return h.Script(g.Raw(scriptContent))
}

// ScriptSrc returns a deferred <script src="url"> for file-based delivery.
// The browser can cache this across page loads; no CSP hash is needed as long
// as script-src includes 'self'. Pair with the js.bundles entry in .assets.yaml
// so the file is present in public/static/js/.
func ScriptSrc(url string) g.Node {
	return h.Script(h.Src(url), h.Defer())
}
