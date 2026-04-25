package auth

import (
	"encoding/json"
	"errors"

	"github.com/go-sum/web/cookiecodec"
)

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
