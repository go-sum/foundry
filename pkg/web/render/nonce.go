package render

import (
	g "maragu.dev/gomponents"
)

// NonceAttr returns a nonce="<nonce>" attribute node for use in script and style elements.
// Pass secure.Nonce(c) as the nonce value.
//
//	<script nonce="<nonce>">...</script>
func NonceAttr(nonce string) g.Node {
	return g.Attr("nonce", nonce)
}
