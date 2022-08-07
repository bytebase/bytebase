package common

import (
	"io"
	"sync"
)

var (
	_ io.Writer = (*SignalWriter)(nil)
)

// SignalWriter sends a signal when write is called and the number of bytes written is greater than 0 for the first time,
// and the signal can be obtained from C.
type SignalWriter struct {
	w    io.Writer
	once sync.Once
	C    chan struct{}
}

// NewSignalWriter returns a new instance of SignalWriter.
func NewSignalWriter(w io.Writer) *SignalWriter {
	ch := make(chan struct{}, 1)
	return &SignalWriter{
		w:    w,
		once: sync.Once{},
		C:    ch,
	}
}

// Write implements the io.Write interface.
func (s *SignalWriter) Write(p []byte) (int, error) {
	b, err := s.w.Write(p)
	if b > 0 {
		s.once.Do(func() {
			s.C <- struct{}{}
		})
	}
	return b, err
}

// Close will close the chan in SignalWriter.
func (s *SignalWriter) Close() {
	close(s.C)
}
