package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"
)

// errReader is a fake io.ReadCloser that returns an error on the first Read.
type errReader struct {
	err error
}

func (e *errReader) Read(p []byte) (int, error) {
	return 0, e.err
}

func (e *errReader) Close() error {
	return nil
}

func newBodyRequest(body string) Request {
	req := NewRequest("POST", &url.URL{Path: "/"})
	req.SetBody(io.NopCloser(strings.NewReader(body)))
	return req
}

func TestRequest_Bytes_ReadsBodyOnce(t *testing.T) {
	req := newBodyRequest("hello")

	got, err := req.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v, want nil", err)
	}
	if string(got) != "hello" {
		t.Errorf("Bytes() = %q, want %q", string(got), "hello")
	}
	if !req.BodyUsed() {
		t.Error("BodyUsed() = false after Bytes(), want true")
	}
}

func TestRequest_Bytes_SecondCall_Fails(t *testing.T) {
	req := newBodyRequest("hello")

	_, _ = req.Bytes()
	got, err := req.Bytes()
	if got != nil {
		t.Errorf("Bytes() data = %v, want nil", got)
	}
	if !errors.Is(err, ErrBodyConsumed) {
		t.Errorf("Bytes() error = %v, want ErrBodyConsumed", err)
	}
}

func TestRequest_Bytes_NilBody(t *testing.T) {
	req := NewRequest("GET", &url.URL{Path: "/"})
	// No SetBody call — Body remains nil.

	got, err := req.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("Bytes() = %v, want nil", got)
	}
	if !req.BodyUsed() {
		t.Error("BodyUsed() = false after Bytes() on nil body, want true")
	}
}

func TestRequest_Bytes_ReadError(t *testing.T) {
	req := NewRequest("POST", &url.URL{Path: "/"})
	req.SetBody(&errReader{err: io.ErrUnexpectedEOF})

	_, err := req.Bytes()
	if err == nil {
		t.Fatal("Bytes() error = nil, want wrapped io.ErrUnexpectedEOF")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("Bytes() error = %v, want errors.Is(err, io.ErrUnexpectedEOF)", err)
	}
}

func TestRequest_Text_HappyPath(t *testing.T) {
	req := newBodyRequest("héllo")

	got, err := req.Text()
	if err != nil {
		t.Fatalf("Text() error = %v, want nil", err)
	}
	if got != "héllo" {
		t.Errorf("Text() = %q, want %q", got, "héllo")
	}
}

func TestRequest_Text_SecondCall_Fails(t *testing.T) {
	req := newBodyRequest("héllo")

	_, _ = req.Text()
	_, err := req.Text()
	if !errors.Is(err, ErrBodyConsumed) {
		t.Errorf("Text() error = %v, want ErrBodyConsumed", err)
	}
}

func TestRequest_JSON_HappyPath(t *testing.T) {
	req := newBodyRequest(`{"N":42}`)

	var dest struct{ N int }
	if err := req.JSON(&dest); err != nil {
		t.Fatalf("JSON() error = %v, want nil", err)
	}
	if dest.N != 42 {
		t.Errorf("dest.N = %d, want 42", dest.N)
	}
}

func TestRequest_JSON_EmptyBody_NilReader(t *testing.T) {
	req := NewRequest("POST", &url.URL{Path: "/"})
	// No SetBody — Body is nil.

	var dest struct{ N int }
	err := req.JSON(&dest)
	if !errors.Is(err, ErrEmptyBody) {
		t.Errorf("JSON() error = %v, want ErrEmptyBody", err)
	}
}

func TestRequest_JSON_EmptyBody_EmptyString(t *testing.T) {
	req := newBodyRequest("")

	var dest struct{ N int }
	err := req.JSON(&dest)
	if !errors.Is(err, ErrEmptyBody) {
		t.Errorf("JSON() error = %v, want ErrEmptyBody", err)
	}
}

func TestRequest_JSON_Malformed(t *testing.T) {
	req := newBodyRequest("notjson")

	var dest struct{ N int }
	err := req.JSON(&dest)
	if err == nil {
		t.Fatal("JSON() error = nil, want a *json.SyntaxError")
	}
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Errorf("JSON() error = %v, want errors.As(*json.SyntaxError) to match", err)
	}
}

func TestRequest_JSON_SecondCall_Fails(t *testing.T) {
	req := newBodyRequest(`{"N":1}`)

	var dest struct{ N int }
	_ = req.JSON(&dest)
	err := req.JSON(&dest)
	if !errors.Is(err, ErrBodyConsumed) {
		t.Errorf("JSON() error = %v, want ErrBodyConsumed", err)
	}
}

func TestRequest_BodyUsed_RawRead(t *testing.T) {
	req := newBodyRequest("raw")

	_, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("io.ReadAll error = %v", err)
	}
	if !req.BodyUsed() {
		t.Error("BodyUsed() = false after raw io.ReadAll, want true")
	}

	_, err = req.Bytes()
	if !errors.Is(err, ErrBodyConsumed) {
		t.Errorf("Bytes() after raw read error = %v, want ErrBodyConsumed", err)
	}
}

// TestP0_06_Body_BoundedRead verifies that Bytes() enforces defaultMaxBodyBytes.
func TestP0_06_Body_BoundedRead(t *testing.T) {
	t.Run("body exactly at limit passes", func(t *testing.T) {
		body := bytes.Repeat([]byte("x"), defaultMaxBodyBytes)
		req := NewRequest("POST", &url.URL{Path: "/"})
		req.SetBody(io.NopCloser(bytes.NewReader(body)))

		got, err := req.Bytes()
		if err != nil {
			t.Fatalf("Bytes() error = %v, want nil", err)
		}
		if len(got) != defaultMaxBodyBytes {
			t.Errorf("Bytes() len = %d, want %d", len(got), defaultMaxBodyBytes)
		}
	})

	t.Run("body at limit+1 returns ErrBodyTooLarge", func(t *testing.T) {
		body := bytes.Repeat([]byte("x"), defaultMaxBodyBytes+1)
		req := NewRequest("POST", &url.URL{Path: "/"})
		req.SetBody(io.NopCloser(bytes.NewReader(body)))

		_, err := req.Bytes()
		if !errors.Is(err, ErrBodyTooLarge) {
			t.Errorf("Bytes() error = %v, want ErrBodyTooLarge", err)
		}
	})
}

func TestBody_JSONStrict(t *testing.T) {
	t.Run("known fields accepted", func(t *testing.T) {
		req := newBodyRequest(`{"N":42}`)

		var dest struct{ N int }
		if err := req.JSONStrict(&dest); err != nil {
			t.Fatalf("JSONStrict() error = %v, want nil", err)
		}
		if dest.N != 42 {
			t.Errorf("dest.N = %d, want 42", dest.N)
		}
	})

	t.Run("unknown fields rejected", func(t *testing.T) {
		req := newBodyRequest(`{"N":42,"Unknown":"field"}`)

		var dest struct{ N int }
		err := req.JSONStrict(&dest)
		if err == nil {
			t.Fatal("JSONStrict() error = nil, want error for unknown fields")
		}
	})

	t.Run("empty body returns ErrEmptyBody", func(t *testing.T) {
		req := newBodyRequest("")
		var dest struct{ N int }
		err := req.JSONStrict(&dest)
		if !errors.Is(err, ErrEmptyBody) {
			t.Errorf("JSONStrict() error = %v, want ErrEmptyBody", err)
		}
	})

	t.Run("concatenated JSON values rejected", func(t *testing.T) {
		req := newBodyRequest(`{"N":1}{"N":2}`)
		var dest struct{ N int }
		err := req.JSONStrict(&dest)
		if err == nil {
			t.Fatal("JSONStrict() error = nil, want error for concatenated JSON")
		}
	})

	t.Run("trailing garbage rejected", func(t *testing.T) {
		req := newBodyRequest(`{"N":1}GARBAGE`)
		var dest struct{ N int }
		err := req.JSONStrict(&dest)
		if err == nil {
			t.Fatal("JSONStrict() error = nil, want error for trailing garbage")
		}
	})

	t.Run("trailing whitespace accepted", func(t *testing.T) {
		// json.Decoder treats trailing whitespace as EOF — this is consistent
		// with json.Unmarshal behavior and is not considered trailing content.
		req := newBodyRequest(`{"N":42}   `)
		var dest struct{ N int }
		if err := req.JSONStrict(&dest); err != nil {
			t.Fatalf("JSONStrict() error = %v, want nil for trailing whitespace", err)
		}
		if dest.N != 42 {
			t.Errorf("dest.N = %d, want 42", dest.N)
		}
	})
}

func TestBody_JSONCharset(t *testing.T) {
	t.Run("no charset accepted", func(t *testing.T) {
		req := newBodyRequest(`{"N":1}`)
		req.Headers.Set("Content-Type", "application/json")
		var dest struct{ N int }
		if err := req.JSON(&dest); err != nil {
			t.Fatalf("JSON() error = %v, want nil", err)
		}
	})

	t.Run("charset=utf-8 accepted", func(t *testing.T) {
		req := newBodyRequest(`{"N":1}`)
		req.Headers.Set("Content-Type", "application/json; charset=utf-8")
		var dest struct{ N int }
		if err := req.JSON(&dest); err != nil {
			t.Fatalf("JSON() error = %v, want nil", err)
		}
	})

	t.Run("charset=UTF-8 accepted (case-insensitive)", func(t *testing.T) {
		req := newBodyRequest(`{"N":1}`)
		req.Headers.Set("Content-Type", "application/json; charset=UTF-8")
		var dest struct{ N int }
		if err := req.JSON(&dest); err != nil {
			t.Fatalf("JSON() error = %v, want nil", err)
		}
	})

	t.Run("charset=iso-8859-1 rejected", func(t *testing.T) {
		req := newBodyRequest(`{"N":1}`)
		req.Headers.Set("Content-Type", "application/json; charset=iso-8859-1")
		var dest struct{ N int }
		err := req.JSON(&dest)
		if err == nil {
			t.Fatal("JSON() error = nil, want error for non-UTF-8 charset")
		}
		if errors.Is(err, ErrBodyConsumed) {
			t.Error("JSON() error is ErrBodyConsumed, but body should not have been read")
		}
	})
}
