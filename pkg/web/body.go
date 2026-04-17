package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
)

// defaultMaxBodyBytes is the maximum number of body bytes read by Bytes().
// Reads exceeding this limit return ErrBodyTooLarge.
const defaultMaxBodyBytes = 4 * 1024 * 1024 // 4 MiB

// bodyState is shared across value-copies of Request via the state pointer.
// Modifications to bodyState are visible to all copies that share the pointer.
type bodyState struct {
	bodyUsed bool
}

// trackedBody is an io.ReadCloser that flips bodyState.bodyUsed on the first
// Read call. SetBody installs this wrapper so that direct reads of Request.Body
// are tracked the same way as the typed body methods (Bytes, Text, JSON, FormData).
type trackedBody struct {
	rc    io.ReadCloser
	state *bodyState
}

func (t *trackedBody) Read(p []byte) (int, error) {
	t.state.bodyUsed = true
	return t.rc.Read(p)
}

func (t *trackedBody) Close() error {
	return t.rc.Close()
}

// Bytes reads the full request body and returns it as a byte slice.
// The body is disturbed after this call — subsequent calls to Bytes, Text,
// JSON, or FormData return ErrBodyConsumed. Returns (nil, nil) if Body is nil.
// Returns ErrBodyTooLarge if the body exceeds defaultMaxBodyBytes (4 MiB).
func (r Request) Bytes() ([]byte, error) {
	if r.state == nil {
		return nil, nil
	}
	if r.state.bodyUsed {
		return nil, ErrBodyConsumed
	}
	r.state.bodyUsed = true
	if r.Body == nil {
		return nil, nil
	}
	defer func() {
		_ = r.Body.Close()
	}()

	// Warn on Content-Length vs Transfer-Encoding conflict.
	cl := r.Headers.Get("Content-Length")
	te := r.Headers.Get("Transfer-Encoding")
	if cl != "" && te != "" && !strings.EqualFold(te, "identity") {
		slog.Warn("web: request has both Content-Length and Transfer-Encoding; reading body with transfer encoding semantics")
	}

	// Bounded read: detect overflow via LimitedReader.
	lr := &io.LimitedReader{R: r.Body, N: defaultMaxBodyBytes + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, wrapBodyReadError("reading request body", err)
	}
	if lr.N == 0 {
		return nil, ErrBodyTooLarge
	}
	return data, nil
}

// Text reads the full request body as a UTF-8 string.
// The body is disturbed after this call. Returns ("", nil) if Body is nil.
func (r Request) Text() (string, error) {
	data, err := r.Bytes()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// JSON reads the full request body and decodes it as JSON into dest.
// Returns ErrEmptyBody if the body is nil or empty.
// Returns ErrBodyConsumed if the body has already been disturbed.
// Returns an error if the Content-Type specifies a non-UTF-8 charset.
func (r Request) JSON(dest any) error {
	// Validate charset before reading the body.
	ct := r.Headers.Get("Content-Type")
	if ct != "" {
		if err := checkJSONCharset(ct); err != nil {
			return err
		}
	}

	data, err := r.Bytes()
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return ErrEmptyBody
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("web: decoding JSON body: %w", err)
	}
	return nil
}

// JSONStrict decodes the body as JSON into dest, rejecting unknown fields.
// Returns ErrEmptyBody if the body is nil or empty.
// Returns ErrBodyConsumed if the body has already been disturbed.
func (r Request) JSONStrict(dest any) error {
	ct := r.Headers.Get("Content-Type")
	if ct != "" {
		if err := checkJSONCharset(ct); err != nil {
			return err
		}
	}

	data, err := r.Bytes()
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return ErrEmptyBody
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return fmt.Errorf("web: decoding JSON body (strict): %w", err)
	}
	// Verify no trailing content follows the first JSON value. json.Decoder
	// is designed for streams and intentionally does not reject trailing bytes
	// after a successful Decode; we must check manually to match the strictness
	// of json.Unmarshal.
	if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return fmt.Errorf("web: JSON body (strict): unexpected trailing content after JSON value")
	}
	return nil
}

// checkJSONCharset returns an error if the Content-Type header specifies a
// charset that is not UTF-8 for a JSON media type.
func checkJSONCharset(ct string) error {
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return nil // not our job to reject unparseable content types here
	}
	isJSON := strings.HasSuffix(mediaType, "json") || mediaType == "application/json"
	if !isJSON {
		return nil
	}
	charset, ok := params["charset"]
	if !ok {
		return nil // no charset specified — UTF-8 is the JSON default
	}
	if strings.EqualFold(charset, "utf-8") || strings.EqualFold(charset, "utf8") {
		return nil
	}
	return fmt.Errorf("web: JSON body charset %q is not UTF-8", charset)
}

func wrapBodyReadError(action string, err error) error {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return fmt.Errorf("web: %s: %w", action, ErrBodyTooLarge)
	}
	return fmt.Errorf("web: %s: %w", action, err)
}
