package api

import (
	"container/ring"
	"sync"
)

// ErrorRecordRing is the struct to store error records in memory.
type ErrorRecordRing struct {
	Ring  *ring.Ring
	Mutex sync.RWMutex
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
