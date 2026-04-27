package kv

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"unicode/utf8"

	kvstore "github.com/go-sum/foundry/pkg/kv"
)

const maxDisplayLen = 2000

// KeyEntry represents a single key in the listing.
type KeyEntry struct {
	Name string
}

// KeyListResult holds a paginated list of keys.
type KeyListResult struct {
	Keys  []KeyEntry
	Total int
}

// KeyDetail holds the full detail for a single key's value.
type KeyDetail struct {
	Key       string
	Value     string
	RawBytes  []byte
	Size      int
	ValueType string // "json", "text", or "binary"
	Exists    bool
}

func listKeys(ctx context.Context, store kvstore.Store, pattern string, page, perPage int) (KeyListResult, error) {
	scanner, ok := store.(kvstore.Scanner)
	if !ok {
		return KeyListResult{}, fmt.Errorf("store does not support key scanning")
	}

	var allKeys []string
	if err := scanner.Scan(ctx, pattern, func(key string) error {
		allKeys = append(allKeys, key)
		return nil
	}); err != nil {
		return KeyListResult{}, fmt.Errorf("scan keys: %w", err)
	}
	sort.Strings(allKeys)

	total := len(allKeys)
	offset := (page - 1) * perPage
	if offset >= total {
		offset = 0
	}
	end := offset + perPage
	if end > total {
		end = total
	}

	keys := make([]KeyEntry, end-offset)
	for i, k := range allKeys[offset:end] {
		keys[i] = KeyEntry{Name: k}
	}
	return KeyListResult{Keys: keys, Total: total}, nil
}

func getKeyDetail(ctx context.Context, store kvstore.Store, key string) (KeyDetail, error) {
	raw, err := store.Get(ctx, key)
	if errors.Is(err, kvstore.ErrNotFound) {
		return KeyDetail{Key: key, Exists: false}, nil
	}
	if err != nil {
		return KeyDetail{}, fmt.Errorf("get key: %w", err)
	}

	vt := detectValueType(raw)
	return KeyDetail{
		Key:       key,
		Value:     formatValue(raw, vt),
		RawBytes:  raw,
		Size:      len(raw),
		ValueType: vt,
		Exists:    true,
	}, nil
}

func detectValueType(raw []byte) string {
	if len(raw) == 0 {
		return "text"
	}
	if json.Valid(raw) && (raw[0] == '{' || raw[0] == '[') {
		return "json"
	}
	if utf8.Valid(raw) {
		return "text"
	}
	return "binary"
}

func formatValue(raw []byte, valueType string) string {
	switch valueType {
	case "json":
		var buf []byte
		var v any
		if err := json.Unmarshal(raw, &v); err == nil {
			if pretty, err := json.MarshalIndent(v, "", "  "); err == nil {
				buf = pretty
			}
		}
		if buf == nil {
			buf = raw
		}
		s := string(buf)
		if len(s) > maxDisplayLen {
			return s[:maxDisplayLen] + "…"
		}
		return s
	case "binary":
		s := hex.EncodeToString(raw)
		if len(s) > maxDisplayLen {
			return s[:maxDisplayLen] + "…"
		}
		return s
	default:
		s := string(raw)
		if len(s) > maxDisplayLen {
			return s[:maxDisplayLen] + "…"
		}
		return s
	}
}
