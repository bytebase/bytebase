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

// CustomWebhookRequest is the API message for Custom webhook request.
type CustomWebhookRequest struct {
	Level        Level   `json:"level"`
	Title        string  `json:"title"`
	ActivityType string  `json:"activity_type"`
	Issue        Issue   `json:"issue"`
	Project      Project `json:"project"`
	Description  string  `json:"description"`
	Link         string  `json:"link"`
	CreatorID    int     `json:"creator_id"`
	CreatorName  string  `json:"creator_name"`
	CreatorEmail string  `json:"creator_email"`
	CreatedTS    int64   `json:"created_ts"`
}

func init() {
	register("bb.plugin.webhook.custom", &CustomReceiver{})
}

// CustomReceiver is the receiver for custom.
type CustomReceiver struct {
}

func (receiver *CustomReceiver) post(context Context) error {
	payload := CustomWebhookRequest{
		Level:        context.Level,
		Title:        context.Title,
		ActivityType: context.ActivityType,
		Issue:        context.Issue,
		Project:      context.Project,
		Description:  context.Description,
		Link:         context.Link,
		CreatorID:    context.CreatorID,
		CreatorName:  context.CreatorName,
		CreatorEmail: context.CreatorEmail,
		CreatedTS:    context.CreatedTs,
	}

	body, err := json.Marshal(&payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook POST request: %v (%w)", context.URL, err)
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to POST webhook %+v, http status_code: %d", context.URL, resp.StatusCode)
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
