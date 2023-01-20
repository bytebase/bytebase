package webhook

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

// CustomWebhookResponse is the API message for Custom webhook response.
type CustomWebhookResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CustomWebhookRequest is the API message for Custom webhook request.
type CustomWebhookRequest struct {
	Level        Level    `json:"level"`
	ActivityType string   `json:"activity_type"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Link         string   `json:"link"`
	CreatorID    int      `json:"creator_id"`
	CreatorName  string   `json:"creator_name"`
	CreatedTS    int64    `json:"created_ts"`
	Issue        *Issue   `json:"issue"`
	Project      *Project `json:"project"`
}

func init() {
	register("bb.plugin.webhook.custom", &CustomReceiver{})
}

// CustomReceiver is the receiver for custom.
type CustomReceiver struct{}

func (*CustomReceiver) post(context Context) error {
	// TODO(p0ny): handle context.Task
	payload := CustomWebhookRequest{
		Level:        context.Level,
		ActivityType: context.ActivityType,
		Title:        context.Title,
		Description:  context.Description,
		Link:         context.Link,
		CreatorID:    context.CreatorID,
		CreatorName:  context.CreatorName,
		CreatedTS:    context.CreatedTs,
		Issue:        context.Issue,
		Project:      context.Project,
	}

	body, err := json.Marshal(&payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal webhook POST request to %s", context.URL)
	}
	req, err := http.NewRequest("POST",
		context.URL, bytes.NewBuffer(body))
	if err != nil {
		return errors.Wrapf(err, "failed to construct webhook POST request to %s", context.URL)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to POST webhook to %s", context.URL)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read POST webhook response from %s", context.URL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("failed to POST webhook %s, status code: %d, response body: %s", context.URL, resp.StatusCode, b)
	}

	webhookResponse := &CustomWebhookResponse{}
	if err := json.Unmarshal(b, webhookResponse); err != nil {
		return errors.Wrapf(err, "malformed webhook response from %s", context.URL)
	}

	if webhookResponse.Code != 0 {
		return errors.Errorf("receive error code sent by webhook server, code %d, msg: %s", webhookResponse.Code, webhookResponse.Message)
	}

	return nil
}
