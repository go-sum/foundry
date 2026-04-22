package formdata

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"testing"
)

// buildMultipart creates a multipart body with the given fields and returns
// the body bytes and content-type header value.
func buildMultipart(t *testing.T, fn func(w *multipart.Writer)) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fn(w)
	if err := w.Close(); err != nil {
		t.Fatalf("multipart.Writer.Close: %v", err)
	}
	return buf.Bytes(), "multipart/form-data; boundary=" + w.Boundary()
}

func TestFormData_URLEncoded(t *testing.T) {
	body := strings.NewReader("a=1&b=2")
	fd, err := Parse(body, "application/x-www-form-urlencoded", DefaultOptions)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := fd.Values["a"]; len(got) != 1 || got[0] != "1" {
		t.Errorf("a = %v, want [1]", got)
	}
	if got := fd.Values["b"]; len(got) != 1 || got[0] != "2" {
		t.Errorf("b = %v, want [2]", got)
	}
}

func TestFormData_Multipart_TextField(t *testing.T) {
	body, ct := buildMultipart(t, func(w *multipart.Writer) {
		if err := w.WriteField("username", "alice"); err != nil {
			t.Fatalf("WriteField: %v", err)
		}
	})

	fd, err := Parse(bytes.NewReader(body), ct, DefaultOptions)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := fd.Values["username"]; len(got) != 1 || got[0] != "alice" {
		t.Errorf("username = %v, want [alice]", got)
	}
	if err := fd.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestFormData_Multipart_FileField(t *testing.T) {
	fileContent := "hello file content"
	body, ct := buildMultipart(t, func(w *multipart.Writer) {
		part, err := w.CreateFormFile("document", "test.txt")
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		fmt.Fprint(part, fileContent)
	})

	fd, err := Parse(bytes.NewReader(body), ct, DefaultOptions)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	defer fd.Close() //nolint:errcheck

	files, ok := fd.Files["document"]
	if !ok || len(files) != 1 {
		t.Fatalf("expected 1 file for 'document', got %v", fd.Files)
	}

	lf := files[0]
	if lf.Filename != "test.txt" {
		t.Errorf("Filename = %q, want %q", lf.Filename, "test.txt")
	}
	if lf.Size != int64(len(fileContent)) {
		t.Errorf("Size = %d, want %d", lf.Size, len(fileContent))
	}

	rc, err := lf.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close() //nolint:errcheck
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != fileContent {
		t.Errorf("file content = %q, want %q", string(data), fileContent)
	}
}

func TestP0_07_Multipart_StreamingDefault(t *testing.T) {
	// 2 MiB file — should spill to disk (default threshold is 1 MiB)
	fileSize := 2 * 1024 * 1024
	fileData := bytes.Repeat([]byte("x"), fileSize)

	body, ct := buildMultipart(t, func(w *multipart.Writer) {
		part, err := w.CreateFormFile("bigfile", "big.bin")
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		if _, err := part.Write(fileData); err != nil {
			t.Fatalf("Write: %v", err)
		}
	})

	fd, err := Parse(bytes.NewReader(body), ct, DefaultOptions)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	defer fd.Close() //nolint:errcheck

	files, ok := fd.Files["bigfile"]
	if !ok || len(files) != 1 {
		t.Fatalf("expected 1 file for 'bigfile'")
	}

	lf := files[0]
	// Verify it is disk-backed (tmpf != nil, data == nil)
	if lf.data != nil {
		t.Error("expected disk-backed LazyFile, but data is in memory")
	}
	if lf.tmpf == nil {
		t.Error("expected disk-backed LazyFile, but tmpf is nil")
	}
	if lf.Size != int64(fileSize) {
		t.Errorf("Size = %d, want %d", lf.Size, fileSize)
	}
}

func TestFormData_Multipart_MaxFiles(t *testing.T) {
	body, ct := buildMultipart(t, func(w *multipart.Writer) {
		for i := 0; i < 3; i++ {
			part, err := w.CreateFormFile(fmt.Sprintf("file%d", i), fmt.Sprintf("f%d.txt", i))
			if err != nil {
				t.Fatalf("CreateFormFile: %v", err)
			}
			fmt.Fprint(part, "data")
		}
	})

	opts := DefaultOptions
	opts.MaxFiles = 2

	_, err := Parse(bytes.NewReader(body), ct, opts)
	if err == nil {
		t.Fatal("expected MaxPartsExceededError, got nil")
	}
	e, ok := err.(*MaxPartsExceededError)
	if !ok {
		t.Fatalf("error type = %T, want *MaxPartsExceededError", err)
	}
	if e.Limit != 2 {
		t.Errorf("Limit = %d, want 2", e.Limit)
	}
}

func TestFormData_Multipart_MaxFileSize(t *testing.T) {
	body, ct := buildMultipart(t, func(w *multipart.Writer) {
		part, err := w.CreateFormFile("doc", "big.txt")
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		fmt.Fprint(part, "hello world") // 11 bytes
	})

	opts := DefaultOptions
	opts.MaxFileSize = 5 // only 5 bytes allowed
	opts.MaxMemory = 3   // below file size, so spills to disk first

	_, err := Parse(bytes.NewReader(body), ct, opts)
	if err == nil {
		t.Fatal("expected MaxFileSizeExceededError, got nil")
	}
	e, ok := err.(*MaxFileSizeExceededError)
	if !ok {
		t.Fatalf("error type = %T, want *MaxFileSizeExceededError", err)
	}
	if e.Field != "doc" {
		t.Errorf("Field = %q, want %q", e.Field, "doc")
	}
	if e.Limit != 5 {
		t.Errorf("Limit = %d, want 5", e.Limit)
	}
}

func TestFormData_Multipart_MaxTotalSize(t *testing.T) {
	// Two files, each 100 bytes, total size limit 150 bytes
	body, ct := buildMultipart(t, func(w *multipart.Writer) {
		for i := 0; i < 2; i++ {
			part, err := w.CreateFormFile(fmt.Sprintf("f%d", i), fmt.Sprintf("file%d.txt", i))
			if err != nil {
				t.Fatalf("CreateFormFile: %v", err)
			}
			part.Write(bytes.Repeat([]byte("x"), 100)) //nolint:errcheck
		}
	})

	opts := DefaultOptions
	opts.MaxFileSize = 200
	opts.MaxFiles = 10
	opts.MaxTotalSize = 150

	_, err := Parse(bytes.NewReader(body), ct, opts)
	if err == nil {
		t.Fatal("expected MaxTotalSizeExceededError, got nil")
	}
	_, ok := err.(*MaxTotalSizeExceededError)
	if !ok {
		t.Fatalf("error type = %T, want *MaxTotalSizeExceededError", err)
	}
}

func TestFormData_Multipart_Close_CleansTempFiles(t *testing.T) {
	// File slightly above 1-byte threshold to force disk spill
	fileData := bytes.Repeat([]byte("z"), 10)

	body, ct := buildMultipart(t, func(w *multipart.Writer) {
		part, err := w.CreateFormFile("f", "data.bin")
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		part.Write(fileData) //nolint:errcheck
	})

	opts := DefaultOptions
	opts.MaxMemory = 5 // threshold of 5 bytes — 10-byte file spills to disk

	fd, err := Parse(bytes.NewReader(body), ct, opts)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	files := fd.Files["f"]
	if len(files) == 0 {
		t.Fatal("expected file 'f'")
	}
	lf := files[0]

	// Must be disk-backed
	if lf.tmpf == nil {
		t.Fatal("expected disk-backed file")
	}
	tmpName := lf.tmpf.Name()

	// Verify temp file exists before Close
	if _, err := os.Stat(tmpName); err != nil {
		t.Fatalf("temp file missing before Close: %v", err)
	}

	if err := fd.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}

	// Verify temp file is removed after Close
	if _, err := os.Stat(tmpName); !os.IsNotExist(err) {
		t.Errorf("temp file still exists after Close: %v", err)
	}
}
