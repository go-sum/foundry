// Package cookiecodec provides tamper-evident signed and AEAD-encrypted
// HTTP cookie codecs. All values use a versioned wire format that binds
// the cookie name, issued-at, and expiry into the authentication tag,
// preventing cross-cookie replay attacks.
package cookiecodec
