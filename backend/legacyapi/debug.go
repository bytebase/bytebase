package api

import (
	"container/ring"
	"sync"
)

// errorRecordMaximum is the count limit for error records.
const errorRecordMaximum = 100

// ErrorRecordRing is the struct to store error records in memory.
type ErrorRecordRing struct {
	Ring *ring.Ring
	sync.RWMutex
}

// NewErrorRecordRing creates an error record ring.
func NewErrorRecordRing() ErrorRecordRing {
	return ErrorRecordRing{
		Ring:    ring.New(errorRecordMaximum),
		RWMutex: sync.RWMutex{},
	}
}
