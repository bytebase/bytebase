package webhook

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	receiverMu sync.RWMutex
	receivers  = make(map[string]WebhookReceiver)
	timeout    = 1 * time.Second
)

type WebHookMeta struct {
	Name  string
	Value string
}

type WebhookReceiver interface {
	post(url string, title string, description string, metaList []WebHookMeta, link string) error
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

func Post(urlstr string, title string, description string, metaList []WebHookMeta, link string) error {
	u, err := url.Parse(urlstr)
	if err != nil {
		return fmt.Errorf("webhook: invalid url: %v", urlstr)
	}

	receiverMu.RLock()
	var r WebhookReceiver
	for key, value := range receivers {
		// Microsfot Teams webhook host is like https://xxx.webhook.office.com where xxx is the team.
		// So we use contains instead of exact match
		if strings.Contains(u.Host, key) {
			r = value
			break
		}
	}
	receiverMu.RUnlock()
	if r == nil {
		return fmt.Errorf("webhook: no applicable receiver for host: %v", u.Host)
	}

	return r.post(urlstr, title, description, metaList, link)
}
