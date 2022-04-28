package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CustomWebhookResponse is the API message for Custom webhook response.
type CustomWebhookResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CustomWebhookRequest struct {
	Level            Level                  `json:"level"`
	Title            string                 `json:"title"`
	EventType        EventType              `json:"event_type"`
	ObjectKind       ObjectKind             `json:"object_kind"`
	ObjectAttributes map[string]interface{} `json:"object_attributes"`
	Description      string                 `json:"description"`
	Link             string                 `json:"link"`
	CreatorName      string                 `json:"creator_name"`
	CreatorEmail     string                 `json:"creator_email"`
	CreatedTS        int64                  `json:"created_ts"`
	IssueStatus      string                 `json:"issue_status"`
	Metadata         []Meta                 `json:"metadata"`
}

func init() {
	register("bb.plugin.webhook.custom", &CustomReceiver{})
}

// CustomReceiver is the receiver for custom.
type CustomReceiver struct {
}

func (receiver *CustomReceiver) post(context Context) error {
	payload := CustomWebhookRequest{
		Level:            context.Level,
		Title:            context.Title,
		EventType:        context.EventType,
		ObjectKind:       context.ObjectKind,
		ObjectAttributes: context.ObjectAttributes.Attributes(),
		Description:      context.Description,
		Link:             context.Link,
		CreatorName:      context.CreatorName,
		CreatorEmail:     context.CreatorEmail,
		CreatedTS:        context.CreatedTs,
		Metadata:         context.MetaList,
	}

	body, err := json.Marshal(&payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook POST request: %v", context.URL)
	}
	req, err := http.NewRequest("POST",
		context.URL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to construct webhook POST request %v (%w)", context.URL, err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST webhook %+v (%w)", context.URL, err)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read POST webhook response %v (%w)", context.URL, err)
	}
	defer resp.Body.Close()

	webhookResponse := &CustomWebhookResponse{}
	if err := json.Unmarshal(b, webhookResponse); err != nil {
		return fmt.Errorf("malformatted webhook response %v (%w)", context.URL, err)
	}

	if webhookResponse.Code != 0 {
		return fmt.Errorf("%s", webhookResponse.Message)
	}

	return nil
}
