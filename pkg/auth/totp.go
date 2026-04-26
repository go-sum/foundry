package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // RFC 6238 TOTP uses HMAC-SHA1.
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"time"
)

func generateTOTPCode(secret string, issuedAt time.Time, periodSeconds int) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("decode verification secret: %w", err)
	}

	period := int64(periodSeconds)
	if period <= 0 {
		period = 300
	}

	counter := uint64(issuedAt.UTC().Unix() / period)
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)

	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(msg[:])
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	code := truncated % 1_000_000
	return fmt.Sprintf("%06d", code), nil
}

func validateTOTPCode(secret string, issuedAt, expiresAt time.Time, code string, periodSeconds int, now time.Time) error {
	if now.After(expiresAt) {
		return ErrVerificationExpired
	}

	expected, err := generateTOTPCode(secret, issuedAt, periodSeconds)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
		return nil
	}
	return ErrInvalidVerificationCode
}

func randomSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}
