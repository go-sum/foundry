package web

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequest_FormData_Urlencoded(t *testing.T) {
	t.Run("parses urlencoded body", func(t *testing.T) {
		req := NewRequest("POST", &url.URL{Path: "/submit"})
		req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBody(io.NopCloser(strings.NewReader("a=1&b=2")))

		fd, err := req.FormData()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := fd.Values.Get("a"); got != "1" {
			t.Errorf("a = %q, want %q", got, "1")
		}
		if got := fd.Values.Get("b"); got != "2" {
			t.Errorf("b = %q, want %q", got, "2")
		}
	})

	t.Run("nil body returns empty values", func(t *testing.T) {
		req := NewRequest("POST", &url.URL{Path: "/submit"})
		req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Body = nil

		fd, err := req.FormData()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(fd.Values) != 0 {
			t.Errorf("expected empty values, got %v", fd.Values)
		}
	})
}

func TestRequest_FormData_Multipart(t *testing.T) {
	t.Run("parses multipart body with text field", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		if err := writer.WriteField("username", "alice"); err != nil {
			t.Fatalf("writing field: %v", err)
		}
		writer.Close()

		req := NewRequest("POST", &url.URL{Path: "/upload"})
		req.Headers.Set("Content-Type", writer.FormDataContentType())
		req.SetBody(io.NopCloser(&buf))

		fd, err := req.FormData()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := fd.Values.Get("username"); got != "alice" {
			t.Errorf("username = %q, want %q", got, "alice")
		}
	})

	t.Run("parses multipart body with file upload", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		fileContent := "hello file content"
		part, err := writer.CreateFormFile("document", "test.txt")
		if err != nil {
			t.Fatalf("creating form file: %v", err)
		}
		fmt.Fprint(part, fileContent)
		writer.Close()

		req := NewRequest("POST", &url.URL{Path: "/upload"})
		req.Headers.Set("Content-Type", writer.FormDataContentType())
		req.SetBody(io.NopCloser(&buf))

		fd, err := req.FormData()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		files, ok := fd.Files["document"]
		if !ok {
			t.Fatal("expected files for key 'document'")
		}
		if len(files) != 1 {
			t.Fatalf("file count = %d, want 1", len(files))
		}

		f := files[0]
		if f.Filename != "test.txt" {
			t.Errorf("Filename = %q, want %q", f.Filename, "test.txt")
		}
		if f.Size != int64(len(fileContent)) {
			t.Errorf("Size = %d, want %d", f.Size, len(fileContent))
		}

		rc, err := f.Open()
		if err != nil {
			t.Fatalf("Open() error: %v", err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("reading file: %v", err)
		}
		if string(data) != fileContent {
			t.Errorf("file content = %q, want %q", string(data), fileContent)
		}
	})

	t.Run("multipart with nil body returns empty values", func(t *testing.T) {
		req := NewRequest("POST", &url.URL{Path: "/upload"})
		req.Headers.Set("Content-Type", "multipart/form-data; boundary=abc123")
		req.Body = nil

		fd, err := req.FormData()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(fd.Values) != 0 {
			t.Errorf("expected empty values, got %v", fd.Values)
		}
	})
}

func TestRequest_FormData_Errors(t *testing.T) {
	t.Run("returns error for missing Content-Type", func(t *testing.T) {
		req := NewRequest("POST", &url.URL{Path: "/upload"})
		req.SetBody(io.NopCloser(strings.NewReader("data")))

		_, err := req.FormData()
		if err == nil {
			t.Fatal("expected error for missing Content-Type")
		}
		want := "web: missing Content-Type header"
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("returns error for missing boundary", func(t *testing.T) {
		req := NewRequest("POST", &url.URL{Path: "/upload"})
		req.Headers.Set("Content-Type", "multipart/form-data")
		req.SetBody(io.NopCloser(strings.NewReader("data")))

		_, err := req.FormData()
		if err == nil {
			t.Fatal("expected error for missing boundary")
		}
		want := "web: missing multipart boundary"
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("returns error for unsupported Content-Type", func(t *testing.T) {
		req := NewRequest("POST", &url.URL{Path: "/upload"})
		req.Headers.Set("Content-Type", "application/json")
		req.SetBody(io.NopCloser(strings.NewReader(`{}`)))

		_, err := req.FormData()
		if err == nil {
			t.Fatal("expected error for unsupported Content-Type")
		}
		want := "web: unsupported Content-Type: application/json"
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("returns ErrBodyConsumed on second call", func(t *testing.T) {
		req := NewRequest("POST", &url.URL{Path: "/submit"})
		req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBody(io.NopCloser(strings.NewReader("key=value")))

		_, err := req.FormData()
		if err != nil {
			t.Fatalf("first call error: %v", err)
		}
		_, err = req.FormData()
		if err != ErrBodyConsumed {
			t.Errorf("second call error = %v, want ErrBodyConsumed", err)
		}
	})

	t.Run("returns ErrBodyConsumed after Bytes() was called first", func(t *testing.T) {
		req := NewRequest("POST", &url.URL{Path: "/submit"})
		req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBody(io.NopCloser(strings.NewReader("key=value")))

		_, err := req.Bytes()
		if err != nil {
			t.Fatalf("Bytes() error: %v", err)
		}
		_, err = req.FormData()
		if !errors.Is(err, ErrBodyConsumed) {
			t.Errorf("FormData() after Bytes() error = %v, want ErrBodyConsumed", err)
		}
	})

	t.Run("enforces per-value limits", func(t *testing.T) {
		req := NewRequest("POST", &url.URL{Path: "/submit"})
		req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBody(io.NopCloser(strings.NewReader("key=too-large")))

		_, err := req.FormDataWithOptions(FormDataOptions{MaxValueBytes: 3})
		if !errors.Is(err, ErrFormValueTooLarge) {
			t.Fatalf("FormDataWithOptions() error = %v, want ErrFormValueTooLarge", err)
		}
	})

	t.Run("enforces per-file limits", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		part, err := writer.CreateFormFile("document", "test.txt")
		if err != nil {
			t.Fatalf("CreateFormFile() error = %v", err)
		}
		fmt.Fprint(part, "hello")
		if err := writer.Close(); err != nil {
			t.Fatalf("writer.Close() error = %v", err)
		}

		req := NewRequest("POST", &url.URL{Path: "/upload"})
		req.Headers.Set("Content-Type", writer.FormDataContentType())
		req.SetBody(io.NopCloser(&buf))

		_, err = req.FormDataWithOptions(FormDataOptions{MaxFileBytes: 4})
		if !errors.Is(err, ErrFormFileTooLarge) {
			t.Fatalf("FormDataWithOptions() error = %v, want ErrFormFileTooLarge", err)
		}
	})

	t.Run("streams uploads via handler", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		part, err := writer.CreateFormFile("document", "test.txt")
		if err != nil {
			t.Fatalf("CreateFormFile() error = %v", err)
		}
		fmt.Fprint(part, "hello stream")
		if err := writer.Close(); err != nil {
			t.Fatalf("writer.Close() error = %v", err)
		}

		req := NewRequest("POST", &url.URL{Path: "/upload"})
		req.Headers.Set("Content-Type", writer.FormDataContentType())
		req.SetBody(io.NopCloser(&buf))

		fd, err := req.FormDataWithOptions(FormDataOptions{
			UploadHandler: TempFileUploadHandler(t.TempDir()),
		})
		if err != nil {
			t.Fatalf("FormDataWithOptions() error = %v", err)
		}
		defer fd.Close()

		files := fd.Files["document"]
		if len(files) != 1 {
			t.Fatalf("len(files) = %d, want 1", len(files))
		}
		rc, err := files[0].Open()
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("io.ReadAll() error = %v", err)
		}
		if string(data) != "hello stream" {
			t.Fatalf("file content = %q, want %q", string(data), "hello stream")
		}
	})
}

// makeMultipartRequest builds a multipart/form-data Request with a single file
// part whose content is fileData under field fieldName and filename filename.
func makeMultipartRequest(t *testing.T, fieldName, filename string, fileData []byte) Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(fileData); err != nil {
		t.Fatalf("writing file data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	req := NewRequest("POST", &url.URL{Path: "/upload"})
	req.Headers.Set("Content-Type", writer.FormDataContentType())
	req.SetBody(io.NopCloser(&buf))
	return req
}

func TestFormDataOptions_InMemorySpill(t *testing.T) {
	t.Run("file smaller than threshold stays in memory", func(t *testing.T) {
		content := []byte("small")
		req := makeMultipartRequest(t, "f", "small.txt", content)

		fd, err := req.FormDataWithOptions(FormDataOptions{
			InMemoryFileBytes: 1024,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer fd.Close()

		files := fd.Files["f"]
		if len(files) != 1 {
			t.Fatalf("file count = %d, want 1", len(files))
		}
		f := files[0]
		if f.content == nil {
			t.Error("expected in-memory content, got nil")
		}
		if f.open != nil {
			t.Error("expected nil open func for in-memory file")
		}
		if f.Size != int64(len(content)) {
			t.Errorf("Size = %d, want %d", f.Size, len(content))
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("Open() error: %v", err)
		}
		defer rc.Close()
		got, _ := io.ReadAll(rc)
		if string(got) != string(content) {
			t.Errorf("content = %q, want %q", got, content)
		}
	})

	t.Run("file exactly at threshold stays in memory", func(t *testing.T) {
		content := []byte("exactly")
		req := makeMultipartRequest(t, "f", "exact.txt", content)

		fd, err := req.FormDataWithOptions(FormDataOptions{
			InMemoryFileBytes: int64(len(content)),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer fd.Close()

		files := fd.Files["f"]
		if len(files) != 1 {
			t.Fatalf("file count = %d, want 1", len(files))
		}
		f := files[0]
		if f.content == nil {
			t.Error("expected in-memory content for file at threshold")
		}
		if f.open != nil {
			t.Error("expected nil open func for in-memory file")
		}
	})

	t.Run("file larger than threshold spills to disk", func(t *testing.T) {
		content := bytes.Repeat([]byte("x"), 100)
		req := makeMultipartRequest(t, "f", "large.txt", content)

		tmpDir := t.TempDir()
		fd, err := req.FormDataWithOptions(FormDataOptions{
			InMemoryFileBytes: 10,
			TempDir:           tmpDir,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		files := fd.Files["f"]
		if len(files) != 1 {
			t.Fatalf("file count = %d, want 1", len(files))
		}
		f := files[0]
		if f.open == nil {
			t.Error("expected open func for disk-backed file")
		}
		if f.content != nil {
			t.Error("expected nil content field for disk-backed file")
		}
		if f.Size != int64(len(content)) {
			t.Errorf("Size = %d, want %d", f.Size, len(content))
		}

		rc, err := f.Open()
		if err != nil {
			t.Fatalf("Open() error: %v", err)
		}
		got, _ := io.ReadAll(rc)
		rc.Close()
		if string(got) != string(content) {
			t.Errorf("content = %q, want %q", got, content)
		}

		// Close removes the temp file.
		if err := fd.Close(); err != nil {
			t.Fatalf("fd.Close() error: %v", err)
		}
		// After Close, Open() should fail (file removed).
		rc2, err := f.Open()
		if err == nil {
			rc2.Close()
			t.Error("expected error opening removed temp file, got nil")
		}
	})

	t.Run("file larger than MaxFileBytes returns ErrFormFileTooLarge", func(t *testing.T) {
		content := bytes.Repeat([]byte("x"), 20)
		req := makeMultipartRequest(t, "f", "toobig.txt", content)

		_, err := req.FormDataWithOptions(FormDataOptions{
			MaxFileBytes:      10,
			InMemoryFileBytes: 5,
		})
		if !errors.Is(err, ErrFormFileTooLarge) {
			t.Errorf("error = %v, want ErrFormFileTooLarge", err)
		}
	})

	t.Run("custom TempDir is used for spilled files", func(t *testing.T) {
		content := bytes.Repeat([]byte("y"), 50)
		req := makeMultipartRequest(t, "f", "spill.txt", content)

		tmpDir := t.TempDir()
		fd, err := req.FormDataWithOptions(FormDataOptions{
			InMemoryFileBytes: 10,
			TempDir:           tmpDir,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer fd.Close()

		files := fd.Files["f"]
		if len(files) != 1 {
			t.Fatalf("file count = %d, want 1", len(files))
		}

		entries, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Fatalf("ReadDir: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("expected 1 temp file in TempDir, got %d", len(entries))
		}
		if matched, _ := filepath.Match("web-upload-*", entries[0].Name()); !matched {
			t.Errorf("temp file name %q does not match pattern web-upload-*", entries[0].Name())
		}
	})

	t.Run("UploadHandler respected when provided", func(t *testing.T) {
		content := bytes.Repeat([]byte("z"), 200)
		req := makeMultipartRequest(t, "f", "handler.txt", content)

		handlerCalled := false
		fd, err := req.FormDataWithOptions(FormDataOptions{
			InMemoryFileBytes: 10, // ignored when UploadHandler is set
			UploadHandler: func(part UploadPart) (*FormFile, error) {
				handlerCalled = true
				data, err := io.ReadAll(part.Reader)
				if err != nil {
					return nil, err
				}
				return NewMemoryFormFile(part.Filename, part.Header, data), nil
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer fd.Close()

		if !handlerCalled {
			t.Error("expected UploadHandler to be called")
		}
		files := fd.Files["f"]
		if len(files) != 1 {
			t.Fatalf("file count = %d, want 1", len(files))
		}
		if files[0].Size != int64(len(content)) {
			t.Errorf("Size = %d, want %d", files[0].Size, len(content))
		}
	})
}
