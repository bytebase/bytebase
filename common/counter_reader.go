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
	count uint64
}

// NewCountingReader creates a new CountingReader from an existing io.Reader.
func NewCountingReader(r io.Reader) *CountingReader {
	return &CountingReader{r: r}
}

// Read implements the io.Reader interface.
func (r *CountingReader) Read(buf []byte) (int, error) {
	n, err := r.r.Read(buf)
	if n > 0 {
		atomic.AddUint64(&r.count, uint64(n))
	}
	return n, err
}

// Count returns the number of read bytes.
func (r *CountingReader) Count() uint64 {
	return atomic.LoadUint64(&r.count)
}
