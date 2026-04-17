package web

import (
	"bytes"
	"sync"
)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

var contextPool = sync.Pool{New: func() any { return &Context{} }}

// GetBuffer returns a *bytes.Buffer from the shared pool.
// The caller must call PutBuffer when done.
func GetBuffer() *bytes.Buffer {
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()
	return b
}

// PutBuffer returns b to the shared pool.
func PutBuffer(b *bytes.Buffer) {
	if b.Cap() > 64*1024 { // don't pool giant buffers
		return
	}
	bufPool.Put(b)
}
