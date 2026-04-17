package cookiecodec

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const versionAEAD byte = 0x82
const versionAEAD2 byte = 0x83

func serializeAEAD(name string, secret []byte, value string, exp time.Time) (string, error) {
	key, err := deriveAEADKey(secret)
	if err != nil {
		return "", fmt.Errorf("cookiecodec: derive key: %w", err)
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", fmt.Errorf("cookiecodec: create AEAD: %w", err)
	}

	var nonce [chacha20poly1305.NonceSizeX]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return "", fmt.Errorf("cookiecodec: generate nonce: %w", err)
	}

	iat := time.Now().Unix()
	var expUnix int64
	if !exp.IsZero() {
		expUnix = exp.Unix()
	}

	var iatExp [16]byte
	binary.BigEndian.PutUint64(iatExp[0:8], uint64(iat))
	binary.BigEndian.PutUint64(iatExp[8:16], uint64(expUnix))

	// AAD binds name + version + iat + exp
	aad := buildAEADAAD(name, versionAEAD2, iatExp[:])

	plaintext := []byte(value)
	ciphertext := aead.Seal(nil, nonce[:], plaintext, aad)

	// blob = version(1) || nonce(24) || iat_int64(8) || exp_int64(8) || ciphertext+tag
	blob := make([]byte, 1+len(nonce)+16+len(ciphertext))
	blob[0] = versionAEAD2
	copy(blob[1:], nonce[:])
	copy(blob[1+len(nonce):], iatExp[:])
	copy(blob[1+len(nonce)+16:], ciphertext)

	return base64.RawURLEncoding.EncodeToString(blob), nil
}

func parseAEAD(name string, secrets [][]byte, encoded string) (string, error) {
	if len(encoded) > maxEncodedSize {
		return "", ErrInvalid
	}
	blob, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrInvalid
	}
	nonceSize := chacha20poly1305.NonceSizeX

	if len(blob) < 1 {
		return "", ErrInvalid
	}

	version := blob[0]
	if version != versionAEAD && version != versionAEAD2 {
		return "", ErrInvalid
	}

	var iatExpSize int
	if version == versionAEAD {
		iatExpSize = 8 // two uint32 fields
	} else {
		iatExpSize = 16 // two int64 fields
	}

	minLen := 1 + nonceSize + iatExpSize + 16 // version + nonce + iat+exp + poly1305 tag
	if len(blob) < minLen {
		return "", ErrInvalid
	}

	nonce := blob[1 : 1+nonceSize]
	iatExp := blob[1+nonceSize : 1+nonceSize+iatExpSize]
	ciphertext := blob[1+nonceSize+iatExpSize:]

	var expUnix int64
	if version == versionAEAD {
		expUnix = int64(binary.BigEndian.Uint32(iatExp[4:8]))
	} else {
		expUnix = int64(binary.BigEndian.Uint64(iatExp[8:16]))
	}

	aad := buildAEADAAD(name, version, iatExp)

	for _, secret := range secrets {
		key, err := deriveAEADKey(secret)
		if err != nil {
			continue
		}
		aead, err := chacha20poly1305.NewX(key)
		if err != nil {
			continue
		}
		plaintext, err := aead.Open(nil, nonce, ciphertext, aad)
		if err != nil {
			continue
		}
		// Auth succeeded
		if expUnix != 0 && time.Now().Unix() > expUnix {
			return "", ErrExpired
		}
		return string(plaintext), nil
	}
	return "", ErrInvalid
}

func deriveAEADKey(secret []byte) ([]byte, error) {
	r := hkdf.New(sha256.New, secret, nil, []byte("web/cookiecodec/v2/aead"))
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

func buildAEADAAD(name string, version byte, iatExp []byte) []byte {
	aad := make([]byte, len(name)+2+len(iatExp))
	copy(aad, []byte(name))
	aad[len(name)] = 0x00
	aad[len(name)+1] = version
	copy(aad[len(name)+2:], iatExp)
	return aad
}
