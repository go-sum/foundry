package authn

import (
	"encoding/json"
	"errors"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/web/cookiecodec"
)

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

func (c *cookieCodecTokenCodec) Encode(token auth.VerificationToken) (string, error) {
	payload, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return c.codec.Serialize(string(payload), token.ExpiresAt)
}

func (c *cookieCodecTokenCodec) Decode(raw string) (auth.VerificationToken, error) {
	plaintext, err := c.codec.Parse(raw)
	if err != nil {
		if errors.Is(err, cookiecodec.ErrExpired) {
			return auth.VerificationToken{}, auth.ErrVerificationExpired
		}
		return auth.VerificationToken{}, auth.ErrVerificationMissing
	}
	var token auth.VerificationToken
	if err := json.Unmarshal([]byte(plaintext), &token); err != nil {
		return auth.VerificationToken{}, auth.ErrVerificationMissing
	}
	return token, nil
}
