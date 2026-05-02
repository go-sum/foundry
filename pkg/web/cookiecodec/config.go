package cookiecodec

// Mode selects the security mode of the codec.
type Mode int

const (
	// Signed uses HMAC-SHA256. The cookie value is visible to the client.
	Signed Mode = iota
	// AEAD uses XChaCha20-Poly1305. The cookie value is encrypted.
	AEAD
)

// Config configures a Codec.
type Config struct {
	// Name is the cookie name. It is bound into every MAC/AAD so a cookie
	// value signed for one name cannot be reused for another.
	Name string
	// Secrets is the list of signing/encryption keys. At least one is required.
	// Secrets[0] is used for new values; all are tried during verification.
	// Rotate by prepending a new secret.
	Secrets [][]byte `validate:"required,min=1"`
	// Mode selects Signed (HMAC-only) or AEAD (encrypted) mode.
	Mode Mode
}
