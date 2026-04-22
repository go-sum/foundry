package formdata

import (
	"cmp"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"strings"
)

// ParseOptions controls limits for multipart parsing.
type ParseOptions struct {
	MaxMemory    int64         // default 1 MiB — spill threshold per file
	MaxFiles     int           // default 20
	MaxFileSize  int64         // default 8 MiB per file
	MaxParts     int           // default 1000 (fields + files)
	MaxTotalSize int64         // default 0 = MaxFiles*MaxFileSize + 1 MiB
	Upload       UploadHandler // nil = DefaultUploadHandler(MaxMemory)
}

// DefaultOptions are sensible production defaults.
var DefaultOptions = ParseOptions{
	MaxMemory:   1 << 20, // 1 MiB
	MaxFiles:    20,
	MaxFileSize: 8 << 20, // 8 MiB
	MaxParts:    1000,
}

// FormData holds the parsed result of a form submission.
type FormData struct {
	Values map[string][]string // URL-encoded or text multipart fields
	Files  map[string][]*LazyFile
}

// Close releases all disk-backed temp files.
func (fd *FormData) Close() error {
	var last error
	for _, files := range fd.Files {
		for _, f := range files {
			if err := f.Close(); err != nil {
				last = err
			}
		}
	}
	return last
}

// Parse parses an HTTP request body as either application/x-www-form-urlencoded
// or multipart/form-data. Returns a *FormData that the caller must Close.
// body is the raw request body; contentType is the Content-Type header value.
func Parse(body io.Reader, contentType string, opts ParseOptions) (*FormData, error) {
	if opts.MaxMemory == 0 {
		opts.MaxMemory = DefaultOptions.MaxMemory
	}
	if opts.MaxFiles == 0 {
		opts.MaxFiles = DefaultOptions.MaxFiles
	}
	if opts.MaxFileSize == 0 {
		opts.MaxFileSize = DefaultOptions.MaxFileSize
	}
	if opts.MaxParts == 0 {
		opts.MaxParts = DefaultOptions.MaxParts
	}
	if opts.MaxTotalSize == 0 {
		opts.MaxTotalSize = int64(opts.MaxFiles)*opts.MaxFileSize + (1 << 20)
	}
	if opts.Upload == nil {
		opts.Upload = DefaultUploadHandler(opts.MaxMemory)
	}

	if contentType == "" {
		return &FormData{Values: make(map[string][]string), Files: make(map[string][]*LazyFile)}, nil
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("formdata: parse Content-Type: %w", err)
	}

	switch {
	case mediaType == "application/x-www-form-urlencoded":
		return parseURLEncoded(body, opts)
	case strings.HasPrefix(mediaType, "multipart/"):
		boundary := params["boundary"]
		if boundary == "" {
			return nil, &MalformedMultipartError{Reason: "missing boundary parameter"}
		}
		return parseMultipart(body, boundary, opts)
	default:
		return &FormData{Values: make(map[string][]string), Files: make(map[string][]*LazyFile)}, nil
	}
}

func parseURLEncoded(body io.Reader, opts ParseOptions) (*FormData, error) {
	limited := io.LimitReader(body, opts.MaxTotalSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > opts.MaxTotalSize {
		return nil, &MaxTotalSizeExceededError{Limit: opts.MaxTotalSize}
	}
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string, len(values))
	for k, v := range values {
		result[k] = v
	}
	return &FormData{Values: result, Files: make(map[string][]*LazyFile)}, nil
}

func parseMultipart(body io.Reader, boundary string, opts ParseOptions) (*FormData, error) {
	result := &FormData{
		Values: make(map[string][]string),
		Files:  make(map[string][]*LazyFile),
	}

	mr := multipart.NewReader(body, boundary)
	var totalBytes int64
	var fileCount int
	var partCount int

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Clean up on error
			result.Close() //nolint:errcheck
			return nil, &MalformedMultipartError{Reason: err.Error()}
		}

		partCount++
		if partCount > opts.MaxParts {
			part.Close()   //nolint:errcheck
			result.Close() //nolint:errcheck
			return nil, &MaxPartsExceededError{Limit: opts.MaxParts}
		}

		fieldName := part.FormName()
		filename := part.FileName()

		if filename == "" {
			// Text field
			limited := io.LimitReader(part, opts.MaxFileSize+1)
			data, err := io.ReadAll(limited)
			if err != nil {
				part.Close()   //nolint:errcheck
				result.Close() //nolint:errcheck
				return nil, err
			}
			if int64(len(data)) > opts.MaxFileSize {
				part.Close()   //nolint:errcheck
				result.Close() //nolint:errcheck
				return nil, &MaxFileSizeExceededError{Field: fieldName, Limit: opts.MaxFileSize}
			}
			totalBytes += int64(len(data))
			if totalBytes > opts.MaxTotalSize {
				part.Close()   //nolint:errcheck
				result.Close() //nolint:errcheck
				return nil, &MaxTotalSizeExceededError{Limit: opts.MaxTotalSize}
			}
			result.Values[fieldName] = append(result.Values[fieldName], string(data))
		} else {
			// File field
			fileCount++
			if fileCount > opts.MaxFiles {
				part.Close()   //nolint:errcheck
				result.Close() //nolint:errcheck
				return nil, &MaxPartsExceededError{Limit: opts.MaxFiles}
			}

			ct := cmp.Or(part.Header.Get("Content-Type"), "application/octet-stream")

			// Wrap with file-size limit
			limited := &limitedReader{R: io.LimitReader(part, opts.MaxFileSize+1), limit: opts.MaxFileSize}
			lf, err := opts.Upload(fieldName, filename, ct, limited)
			if limited.exceeded {
				part.Close() //nolint:errcheck
				if lf != nil {
					lf.Close() //nolint:errcheck
				}
				result.Close() //nolint:errcheck
				return nil, &MaxFileSizeExceededError{Field: fieldName, Limit: opts.MaxFileSize}
			}
			if err != nil {
				part.Close()   //nolint:errcheck
				result.Close() //nolint:errcheck
				return nil, err
			}
			if lf != nil {
				totalBytes += lf.Size
				if totalBytes > opts.MaxTotalSize {
					lf.Close()     //nolint:errcheck
					part.Close()   //nolint:errcheck
					result.Close() //nolint:errcheck
					return nil, &MaxTotalSizeExceededError{Limit: opts.MaxTotalSize}
				}
				result.Files[fieldName] = append(result.Files[fieldName], lf)
			}
		}
		part.Close() //nolint:errcheck
	}
	return result, nil
}

// limitedReader wraps an io.Reader and sets exceeded=true if limit is reached.
type limitedReader struct {
	R        io.Reader
	limit    int64
	read     int64
	exceeded bool
}

func (l *limitedReader) Read(p []byte) (int, error) {
	n, err := l.R.Read(p)
	l.read += int64(n)
	if l.read > l.limit {
		l.exceeded = true
	}
	return n, err
}
