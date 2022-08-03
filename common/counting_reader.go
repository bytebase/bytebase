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

	atomic.AddInt64(&r.count, int64(n))

	return n, err
}

// Count returns the number of read bytes.
func (r *CountingReader) Count() int64 {
	return atomic.LoadInt64(&r.count)
}
