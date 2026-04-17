// Package formdata provides streaming multipart and URL-encoded form parsers.
// The multipart parser never buffers a full file part — each part body is
// exposed as an io.Reader. By default, file parts are spilled to disk above
// 1 MiB. Text fields are always read into memory.
package formdata
