package cookiecodec

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"time"
)

const versionSigned byte = 0x03

func serializeSigned(name string, secret []byte, value string, exp time.Time) (string, error) {
	payload := []byte(value)
	iat := time.Now().Unix()
	var expUnix int64
	if !exp.IsZero() {
		expUnix = exp.Unix()
	}

	// Build blob (without MAC): version(1) | iat_int64(8) | exp_int64(8) | payload
	blob := make([]byte, 17+len(payload))
	blob[0] = versionSigned
	binary.BigEndian.PutUint64(blob[1:9], uint64(iat))
	binary.BigEndian.PutUint64(blob[9:17], uint64(expUnix))
	copy(blob[17:], payload)

	// Compute MAC over blob + separator + name + separator
	mac := computeSignedMAC(secret, blob, name)
	blob = append(blob, mac...)

	return base64.RawURLEncoding.EncodeToString(blob), nil
}

// maxEncodedSize caps inbound cookie values to avoid allocations from forged headers.
const maxEncodedSize = 4096

func parseSigned(name string, secrets [][]byte, encoded string) (string, error) {
	if len(encoded) > maxEncodedSize {
		return "", ErrInvalid
	}
	blob, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(blob) < 1 {
		return "", ErrInvalid
	}

	version := blob[0]
	if version != versionSigned {
		return "", ErrInvalid
	}

	const headerSize = 17 // version(1) + iat_int64(8) + exp_int64(8)

	if len(blob) < headerSize+sha256.Size {
		return "", ErrInvalid
	}

	msgLen := len(blob) - sha256.Size
	msg := blob[:msgLen]
	mac := blob[msgLen:]

	// Try each secret
	verified := false
	for _, secret := range secrets {
		expected := computeSignedMAC(secret, msg, name)
		if hmac.Equal(mac, expected) {
			verified = true
			break
		}
	}
	if !verified {
		return "", ErrInvalid
	}

	expUnix := int64(binary.BigEndian.Uint64(msg[9:17]))

	if expUnix != 0 && time.Now().Unix() > expUnix {
		return "", ErrExpired
	}

	return string(msg[17:]), nil
}

func computeSignedMAC(secret, msg []byte, name string) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(msg)
	h.Write([]byte{0x1E})
	h.Write([]byte(name))
	h.Write([]byte{0x1E})
	return h.Sum(nil)
}
