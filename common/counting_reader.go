package common

import (
	"io"
	"sync/atomic"
)

var (
	_ io.Reader = (*CountingReader)(nil)
)

// CountingReader is a reader that counts the read bytes in a thread safe way.
type CountingReader struct {
	r     io.Reader
	count int64
}

// NewCountingReader creates a new CountingReader from an existing io.Reader.
func NewCountingReader(r io.Reader) *CountingReader {
	return &CountingReader{r: r}
}

// Read implements the io.Reader interface.
func (r *CountingReader) Read(buf []byte) (int, error) {
	n, err := r.r.Read(buf)
	// Read() should always return a non-negative `n`.
	// But since `n` is a signed integer, some custom
	// implementation of an io.Reader may return negative
	// values.
	//
	// Excluding such invalid values from counting,
	// thus `if n >= 0`:
	if n >= 0 {
		atomic.AddInt64(&r.count, int64(n))
	}
	return n, err
}

// Count returns the number of read bytes.
func (r *CountingReader) Count() int64 {
	return atomic.LoadInt64(&r.count)
}
