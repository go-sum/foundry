package web

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"os"
)

const (
	defaultMaxFormFields   = 1000
	defaultMaxFormFiles    = 20
	defaultMaxFormValueB   = 1 << 20
	defaultMaxFormFileB    = 8 << 20
	defaultMaxFormTotalB   = 32 << 20
	defaultTempFilePattern = "web-upload-*"
)

// FormDataOptions configures Request.FormDataWithOptions.
type FormDataOptions struct {
	// MaxFields is the maximum number of non-file form values accepted.
	// Defaults to 1000.
	MaxFields int

	// MaxFiles is the maximum number of uploaded files accepted.
	// Defaults to 20.
	MaxFiles int

	// MaxValueBytes is the maximum size of a single non-file form value.
	// Defaults to 1 MiB.
	MaxValueBytes int64

	// MaxFileBytes is the maximum size of a single uploaded file.
	// Defaults to 8 MiB.
	MaxFileBytes int64

	// MaxTotalBytes is the maximum total size read from the request body.
	// Defaults to 32 MiB.
	MaxTotalBytes int64

	// InMemoryFileBytes is the per-file threshold below which uploaded files are
	// kept in memory. Files exceeding this size are spilled to a temporary file
	// on disk. Defaults to 1 MiB (1 << 20). Set to 1 to always spill to disk.
	// Ignored when UploadHandler is set.
	InMemoryFileBytes int64

	// TempDir is the directory used for temporary upload files when a file
	// exceeds InMemoryFileBytes. Defaults to os.TempDir() when empty. Useful
	// for read-only container environments that mount a writable volume
	// elsewhere. Ignored when UploadHandler is set.
	TempDir string

	// UploadHandler streams uploaded file content to caller-defined storage.
	// If nil, uploaded files are stored in memory up to InMemoryFileBytes,
	// then spilled to disk beyond that threshold.
	UploadHandler UploadHandler
}

// UploadHandler consumes an uploaded file stream and returns the FormFile
// representation that should be stored in the parsed FormData. Returning nil
// skips the file.
type UploadHandler func(part UploadPart) (*FormFile, error)

// UploadPart describes a multipart file part while it is being parsed.
// The Reader is one-shot and must not be used after the handler returns.
type UploadPart struct {
	FieldName string
	Filename  string
	Header    Headers
	Reader    io.Reader
}

// FormData holds parsed form values and uploaded files.
type FormData struct {
	Values url.Values
	Files  map[string][]*FormFile
}

// Close removes any temporary resources held by uploaded files.
func (fd FormData) Close() error {
	var errs []error
	for _, files := range fd.Files {
		for _, file := range files {
			if err := file.Remove(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// FormFile represents an uploaded file from a multipart form.
type FormFile struct {
	Filename string
	Header   Headers
	Size     int64

	content []byte
	open    func() (io.ReadCloser, error)
	remove  func() error
}

// NewMemoryFormFile creates a FormFile backed by in-memory bytes.
func NewMemoryFormFile(filename string, header Headers, data []byte) *FormFile {
	cp := make([]byte, len(data))
	copy(cp, data)
	return &FormFile{
		Filename: filename,
		Header:   header.Clone(),
		Size:     int64(len(cp)),
		content:  cp,
	}
}

// NewTempFormFile creates a FormFile backed by a file on disk. Call Remove or
// FormData.Close when the file is no longer needed.
func NewTempFormFile(filename string, header Headers, path string, size int64) *FormFile {
	return &FormFile{
		Filename: filename,
		Header:   header.Clone(),
		Size:     size,
		open: func() (io.ReadCloser, error) {
			return os.Open(path)
		},
		remove: func() error {
			err := os.Remove(path)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return nil
		},
	}
}

// TempFileUploadHandler returns an UploadHandler that streams each uploaded
// file to a temporary file on disk.
func TempFileUploadHandler(dir string) UploadHandler {
	return func(part UploadPart) (*FormFile, error) {
		file, err := os.CreateTemp(dir, defaultTempFilePattern)
		if err != nil {
			return nil, fmt.Errorf("web: creating temp upload file: %w", err)
		}

		var size int64
		copyErr := func() error {
			n, err := io.Copy(file, part.Reader)
			size = n
			if err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
			return nil
		}()
		if copyErr != nil {
			_ = file.Close()
			_ = os.Remove(file.Name())
			if errors.Is(copyErr, ErrFormFileTooLarge) || errors.Is(copyErr, ErrBodyTooLarge) {
				return nil, copyErr
			}
			return nil, fmt.Errorf("web: storing uploaded file: %w", copyErr)
		}

		return NewTempFormFile(part.Filename, part.Header, file.Name(), size), nil
	}
}

// Open returns a reader for the file content.
func (f *FormFile) Open() (io.ReadCloser, error) {
	switch {
	case f == nil:
		return nil, fmt.Errorf("web: nil form file")
	case f.open != nil:
		return f.open()
	default:
		return io.NopCloser(bytes.NewReader(f.content)), nil
	}
}

// Remove deletes any backing resources held by the file.
func (f *FormFile) Remove() error {
	if f == nil || f.remove == nil {
		return nil
	}
	return f.remove()
}

// FormData parses the request body according to Content-Type using safe
// production defaults.
func (r Request) FormData() (FormData, error) {
	return r.FormDataWithOptions(FormDataOptions{})
}

// FormDataWithOptions parses the request body according to Content-Type.
// Supports application/x-www-form-urlencoded and multipart/form-data.
//
// The body is disturbed after this call — subsequent calls to FormData, Bytes,
// Text, or JSON return ErrBodyConsumed.
func (r Request) FormDataWithOptions(opts FormDataOptions) (FormData, error) {
	if r.state == nil {
		return FormData{}, fmt.Errorf("web: missing Content-Type header")
	}
	if r.state.bodyUsed {
		return FormData{}, ErrBodyConsumed
	}

	ct := r.Headers.Get("Content-Type")
	if ct == "" {
		return FormData{}, fmt.Errorf("web: missing Content-Type header")
	}
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return FormData{}, fmt.Errorf("web: invalid Content-Type: %w", err)
	}

	opts = normalizeFormDataOptions(opts)
	r.state.bodyUsed = true

	switch mediaType {
	case "application/x-www-form-urlencoded":
		return r.parseURLEncodedForm(opts)
	case "multipart/form-data":
		boundary := params["boundary"]
		if boundary == "" {
			return FormData{}, fmt.Errorf("web: missing multipart boundary")
		}
		return r.parseMultipartForm(boundary, opts)
	default:
		r.state.bodyUsed = false // unsupported content-type is not a disturbance
		return FormData{}, fmt.Errorf("web: unsupported Content-Type: %s", mediaType)
	}
}

func normalizeFormDataOptions(opts FormDataOptions) FormDataOptions {
	if opts.MaxFields <= 0 {
		opts.MaxFields = defaultMaxFormFields
	}
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = defaultMaxFormFiles
	}
	if opts.MaxValueBytes <= 0 {
		opts.MaxValueBytes = defaultMaxFormValueB
	}
	if opts.MaxFileBytes <= 0 {
		opts.MaxFileBytes = defaultMaxFormFileB
	}
	if opts.MaxTotalBytes <= 0 {
		opts.MaxTotalBytes = defaultMaxFormTotalB
	}
	if opts.InMemoryFileBytes <= 0 {
		opts.InMemoryFileBytes = 1 << 20
	}
	return opts
}

func (r Request) parseURLEncodedForm(opts FormDataOptions) (FormData, error) {
	if r.Body == nil {
		return FormData{Values: url.Values{}}, nil
	}
	defer func() {
		_ = r.Body.Close()
	}()

	bodyReader := newLimitedReader(r.Body, opts.MaxTotalBytes, ErrBodyTooLarge)
	data, err := io.ReadAll(bodyReader)
	if err != nil {
		return FormData{}, wrapBodyReadError("reading form body", err)
	}

	values, err := url.ParseQuery(string(data))
	if err != nil {
		return FormData{}, fmt.Errorf("web: parsing urlencoded body: %w", err)
	}
	if err := validateFormValues(values, opts); err != nil {
		return FormData{}, err
	}
	return FormData{Values: values}, nil
}

func (r Request) parseMultipartForm(boundary string, opts FormDataOptions) (FormData, error) {
	if r.Body == nil {
		return FormData{Values: url.Values{}}, nil
	}
	defer func() {
		_ = r.Body.Close()
	}()

	reader := multipart.NewReader(
		newLimitedReader(r.Body, opts.MaxTotalBytes, ErrBodyTooLarge),
		boundary,
	)

	values := url.Values{}
	files := make(map[string][]*FormFile)
	fieldCount := 0
	fileCount := 0

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return FormData{}, wrapBodyReadError("reading multipart form", err)
		}

		name := part.FormName()
		if name == "" {
			_ = part.Close()
			continue
		}

		partHeaders := headersFromTextproto(part.Header)
		filename := part.FileName()
		if filename == "" {
			fieldCount++
			if fieldCount > opts.MaxFields {
				_ = part.Close()
				return FormData{}, ErrFormFieldsExceeded
			}
			data, err := readAllWithLimit(part, opts.MaxValueBytes, ErrFormValueTooLarge)
			_ = part.Close()
			if err != nil {
				return FormData{}, wrapMultipartReadError(name, err)
			}
			values.Add(name, string(data))
			continue
		}

		fileCount++
		if fileCount > opts.MaxFiles {
			part.Close() //nolint:errcheck
			return FormData{}, ErrFormFilesExceeded
		}

		file, err := parseMultipartFilePart(part, name, filename, partHeaders, opts)
		part.Close() //nolint:errcheck
		if err != nil {
			return FormData{}, wrapMultipartReadError(name, err)
		}
		if file != nil {
			files[name] = append(files[name], file)
		}
	}

	if err := validateFormValues(values, opts); err != nil {
		return FormData{}, err
	}
	if len(files) == 0 {
		files = nil
	}
	return FormData{Values: values, Files: files}, nil
}

func validateFormValues(values url.Values, opts FormDataOptions) error {
	fieldCount := 0
	for _, allValues := range values {
		for _, value := range allValues {
			fieldCount++
			if fieldCount > opts.MaxFields {
				return ErrFormFieldsExceeded
			}
			if int64(len(value)) > opts.MaxValueBytes {
				return ErrFormValueTooLarge
			}
		}
	}
	return nil
}

func parseMultipartFilePart(
	part *multipart.Part,
	fieldName string,
	filename string,
	headers Headers,
	opts FormDataOptions,
) (*FormFile, error) {
	limited := newLimitedReader(part, opts.MaxFileBytes, ErrFormFileTooLarge)
	if opts.UploadHandler != nil {
		file, err := opts.UploadHandler(UploadPart{
			FieldName: fieldName,
			Filename:  filename,
			Header:    headers.Clone(),
			Reader:    limited,
		})
		if drainErr := drainReader(limited); err == nil && drainErr != nil {
			err = drainErr
		}
		return file, err
	}

	// Read up to the in-memory threshold into a buffer.
	memLimit := opts.InMemoryFileBytes
	if opts.MaxFileBytes < memLimit {
		memLimit = opts.MaxFileBytes
	}
	var memBuf bytes.Buffer
	n, err := io.CopyN(&memBuf, limited, memLimit)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	// Check if there are more bytes beyond the in-memory threshold.
	var probe [1]byte
	probeN, probeErr := limited.Read(probe[:])
	exhausted := probeN == 0 && errors.Is(probeErr, io.EOF)

	if exhausted {
		// The entire file fit within the in-memory threshold.
		return NewMemoryFormFile(filename, headers, memBuf.Bytes()), nil
	}
	if probeErr != nil && !errors.Is(probeErr, io.EOF) {
		return nil, probeErr
	}

	// File exceeds in-memory threshold — spill to disk.
	tmpFile, err := os.CreateTemp(opts.TempDir, defaultTempFilePattern)
	if err != nil {
		return nil, fmt.Errorf("web: creating temp upload file: %w", err)
	}

	spillErr := func() error {
		// Write the already-buffered bytes first.
		if _, err := tmpFile.Write(memBuf.Bytes()); err != nil {
			return err
		}
		// Write the probe byte that confirmed more data exists.
		if _, err := tmpFile.Write(probe[:probeN]); err != nil {
			return err
		}
		// Copy remaining bytes from the limited reader.
		if _, err := io.Copy(tmpFile, limited); err != nil {
			return err
		}
		return tmpFile.Close()
	}()
	if spillErr != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return nil, spillErr
	}

	totalSize := n + int64(probeN)
	// Re-open the file to determine its actual size after writing.
	fi, err := os.Stat(tmpFile.Name())
	if err == nil {
		totalSize = fi.Size()
	}

	return NewTempFormFile(filename, headers, tmpFile.Name(), totalSize), nil
}

func headersFromTextproto(header textproto.MIMEHeader) Headers {
	out := NewHeaders()
	for key, values := range header {
		for _, value := range values {
			out.Append(key, value)
		}
	}
	return out
}

func readAllWithLimit(r io.Reader, limit int64, limitErr error) ([]byte, error) {
	limited := newLimitedReader(r, limit, limitErr)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func drainReader(r io.Reader) error {
	_, err := io.Copy(io.Discard, r)
	return err
}

func wrapMultipartReadError(name string, err error) error {
	switch {
	case errors.Is(err, ErrBodyTooLarge),
		errors.Is(err, ErrFormFileTooLarge),
		errors.Is(err, ErrFormValueTooLarge),
		errors.Is(err, ErrFormFieldsExceeded),
		errors.Is(err, ErrFormFilesExceeded):
		return err
	default:
		return fmt.Errorf("web: reading multipart field %s: %w", name, err)
	}
}

type limitedReader struct {
	reader    io.Reader
	remaining int64
	limitErr  error
}

func newLimitedReader(reader io.Reader, limit int64, limitErr error) *limitedReader {
	return &limitedReader{
		reader:    reader,
		remaining: limit,
		limitErr:  limitErr,
	}
}

func (r *limitedReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		var probe [1]byte
		n, err := r.reader.Read(probe[:])
		switch {
		case n == 0 && errors.Is(err, io.EOF):
			return 0, io.EOF
		case err != nil && !errors.Is(err, io.EOF):
			return 0, err
		default:
			return 0, r.limitErr
		}
	}
	if int64(len(p)) > r.remaining {
		p = p[:int(r.remaining)]
	}
	n, err := r.reader.Read(p)
	r.remaining -= int64(n)
	if err != nil {
		return n, err
	}
	return n, nil
}
