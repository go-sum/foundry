package auth

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-sum/foundry/pkg/web/cookiecodec"
)

// ParseTokenKeys decodes hex-encoded AEAD key material for use as auth token
// secrets. keyHex must be non-empty and decode to at least 32 bytes.
// Returns ErrTokenKeyMissing when keyHex is empty and ErrTokenKeyInvalid
// when the value is malformed or too short.
func ParseTokenKeys(keyHex string) ([][]byte, error) {
	if keyHex == "" {
		return nil, ErrTokenKeyMissing
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenKeyInvalid, err)
	}
	if len(key) < 32 {
		return nil, fmt.Errorf("%w: got %d bytes, need at least 32", ErrTokenKeyInvalid, len(key))
	}
	return [][]byte{key}, nil
}

// TokenCodec encodes and decodes verification tokens.
type TokenCodec interface {
	Encode(VerificationToken) (string, error)
	Decode(string) (VerificationToken, error)
}

type cookieCodecTokenCodec struct {
	codec *cookiecodec.Codec
}

// NewTokenCodec returns a TokenCodec backed by XChaCha20-Poly1305 AEAD encryption.
func NewTokenCodec(secrets [][]byte) (*cookieCodecTokenCodec, error) {
	codec, err := cookiecodec.New(cookiecodec.Config{
		Name:    "auth.verify",
		Secrets: secrets,
		Mode:    cookiecodec.AEAD,
	})
	if err != nil {
		return nil, err
	}
	return &cookieCodecTokenCodec{codec: codec}, nil
}

func (c *cookieCodecTokenCodec) Encode(token VerificationToken) (string, error) {
	payload, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return c.codec.Serialize(string(payload), token.ExpiresAt)
}

func (c *cookieCodecTokenCodec) Decode(raw string) (VerificationToken, error) {
	plaintext, err := c.codec.Parse(raw)
	if err != nil {
		if errors.Is(err, cookiecodec.ErrExpired) {
			return VerificationToken{}, ErrVerificationExpired
		}
		return VerificationToken{}, ErrVerificationMissing
	}
	var token VerificationToken
	if err := json.Unmarshal([]byte(plaintext), &token); err != nil {
		return VerificationToken{}, ErrVerificationMissing
	}
	return token, nil
}
