package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

type CustomWebhookMessage struct {
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Link        string            `json:"link,omitempty"`
	Actor       string            `json:"actor"`
	Meta        map[string]string `json:"meta,omitempty"`
}

func init() {
	Register("bb.plugin.webhook.custom", &CustomReceiver{})
}

// CustomReceiver is the receiver for Custom Webhook.
type CustomReceiver struct{}

func (*CustomReceiver) Post(context Context) error {
	metaMap := make(map[string]string)
	for _, meta := range context.GetMetaList() {
		metaMap[meta.Name] = meta.Value
	}

	message := CustomWebhookMessage{
		Title:       context.Title,
		Description: context.Description,
		Link:        context.Link,
		Actor:       fmt.Sprintf("%s (%s)", context.ActorName, context.ActorEmail),
		Meta:        metaMap,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal webhook request to %s", context.URL)
	}

	req, err := http.NewRequest("POST", context.URL, bytes.NewBuffer(body))
	if err != nil {
		return errors.Wrapf(err, "failed to create webhook POST request to %s", context.URL)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to POST webhook to %s", context.URL)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return errors.Errorf("custom webhook returned status %d, body: %s", resp.StatusCode, respBody)
	}

	return nil
}
