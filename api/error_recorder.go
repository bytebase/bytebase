package api

import (
	"container/ring"
	"sync"
)

// errorRecordCount is the count limit for error records.
const errorRecordCount = 100

// ErrorRecordRing is the struct to store error records in memory.
type ErrorRecordRing struct {
	Ring  *ring.Ring
	Mutex sync.RWMutex
}

// NewErrorRecordRing creates an error record ring.
func NewErrorRecordRing() ErrorRecordRing {
	return ErrorRecordRing{
		Ring:  ring.New(errorRecordCount),
		Mutex: sync.RWMutex{},
	}
}

// ErrorRecord is the struct to record an error's useful details.
type ErrorRecord struct {
	RecordTs    int64  `jsonapi:"attr,recordTs"`
	Method      string `jsonapi:"attr,method"`
	RequestPath string `jsonapi:"attr,requestPath"`
	Role        Role   `jsonapi:"attr,role"`
	Error       string `jsonapi:"attr,error"`
	StackTrace  string `jsonapi:"attr,stackTrace"`
}
