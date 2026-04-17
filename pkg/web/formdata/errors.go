package formdata

import "fmt"

// MaxFileSizeExceededError is returned when a single file part exceeds MaxFileSize.
type MaxFileSizeExceededError struct {
	Field string
	Limit int64
}

func (e *MaxFileSizeExceededError) Error() string {
	return fmt.Sprintf("formdata: file %q exceeds %d byte limit", e.Field, e.Limit)
}

// MaxPartsExceededError is returned when the number of parts exceeds MaxParts.
type MaxPartsExceededError struct{ Limit int }

func (e *MaxPartsExceededError) Error() string {
	return fmt.Sprintf("formdata: multipart parts exceed %d limit", e.Limit)
}

// MaxTotalSizeExceededError is returned when the aggregate body exceeds MaxTotalSize.
type MaxTotalSizeExceededError struct{ Limit int64 }

func (e *MaxTotalSizeExceededError) Error() string {
	return fmt.Sprintf("formdata: multipart total size exceeds %d byte limit", e.Limit)
}

// MaxHeaderSizeExceededError is returned when a part's MIME headers exceed MaxHeaderSize.
type MaxHeaderSizeExceededError struct{}

func (e *MaxHeaderSizeExceededError) Error() string {
	return "formdata: part header size limit exceeded"
}

// MalformedMultipartError is returned when the multipart body is structurally invalid.
type MalformedMultipartError struct{ Reason string }

func (e *MalformedMultipartError) Error() string {
	return "formdata: malformed multipart: " + e.Reason
}
