package webhook

import (
	"fmt"
	"sync"
	"time"
)

var (
	receiverMu sync.RWMutex
	receivers  = make(map[string]WebhookReceiver)
	timeout    = 1 * time.Second
	timeFormat = "2006-01-02 15:04:05"
)

type WebhookMeta struct {
	Name  string
	Value string
}

type WebhookContext struct {
	URL          string
	Title        string
	Description  string
	Link         string
	CreatorName  string
	CreatorEmail string
	CreatedTs    int64
	MetaList     []WebhookMeta
}

type WebhookReceiver interface {
	post(context WebhookContext) error
}

// Register makes a receiver available by the url host
// If Register is called twice with the same url host or if receiver is nil,
// it panics.
func register(host string, r WebhookReceiver) {
	receiverMu.Lock()
	defer receiverMu.Unlock()
	if r == nil {
		panic("webhook: Register receiver is nil")
	}
	if _, dup := receivers[host]; dup {
		panic("webhook: Register called twice for host " + host)
	}
	receivers[host] = r
}

func Post(webhookType string, context WebhookContext) error {
	receiverMu.RLock()
	r, ok := receivers[webhookType]
	receiverMu.RUnlock()
	if !ok {
		return fmt.Errorf("webhook: no applicable receiver for webhook type: %v", webhookType)
	}

	return r.post(context)
}
