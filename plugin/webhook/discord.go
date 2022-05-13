package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DiscordWebhookResponse is the API message for Discord webhook response.
type DiscordWebhookResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// DiscordWebhookEmbedField is the API message for Discord webhook embed field.
type DiscordWebhookEmbedField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// DiscordWebhookEmbedAuthor is the API message for Discord webhook embed Author.
type DiscordWebhookEmbedAuthor struct {
	Name string `json:"name"`
}

// DiscordWebhookEmbed is the API message for Discord webhook embed.
type DiscordWebhookEmbed struct {
	Title       string                     `json:"title"`
	Type        string                     `json:"type"`
	Description string                     `json:"description,omitempty"`
	URL         string                     `json:"url,omitempty"`
	Timestamp   string                     `json:"timestamp"`
	Author      DiscordWebhookEmbedAuthor  `json:"author"`
	FieldList   []DiscordWebhookEmbedField `json:"fields,omitempty"`
}

// DiscordWebhook is the API message for Discord webhook.
type DiscordWebhook struct {
	EmbedList []DiscordWebhookEmbed `json:"embeds"`
}

func init() {
	register("bb.plugin.webhook.discord", &DiscordReceiver{})
}

// DiscordReceiver is the receiver for Discord.
type DiscordReceiver struct {
}

func (receiver *DiscordReceiver) post(context Context) error {
	embedList := []DiscordWebhookEmbed{}

	fieldList := []DiscordWebhookEmbedField{}
	for _, meta := range context.genMeta() {
		fieldList = append(fieldList, DiscordWebhookEmbedField(meta))
	}

	status := ""
	switch context.Level {
	case WebhookSuccess:
		status = ":white_check_mark: "
	case WebhookWarn:
		status = ":warning: "
	case WebhookError:
		status = ":exclamation: "
	}
	embedList = append(embedList, DiscordWebhookEmbed{
		Title:       fmt.Sprintf("%s%s", status, context.Title),
		Type:        "rich",
		Description: context.Description,
		URL:         context.Link,
		Timestamp:   time.Unix(context.CreatedTs, 0).Format(timeFormat),
		Author: DiscordWebhookEmbedAuthor{
			Name: fmt.Sprintf("%s (%s)", context.CreatorName, context.CreatorEmail),
		},
		FieldList: fieldList,
	})

	post := DiscordWebhook{
		EmbedList: embedList,
	}
	body, err := json.Marshal(post)
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

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read POST webhook response %v (%w)", context.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to POST webhook %v, status code: %d, response body: %s", context.URL, resp.StatusCode, b)
	}

	webhookResponse := &DiscordWebhookResponse{}
	if err := json.Unmarshal(b, webhookResponse); err != nil {
		return fmt.Errorf("malformatted webhook response %v (%w)", context.URL, err)
	}

	if webhookResponse.Code != 0 {
		return fmt.Errorf("%s", webhookResponse.Message)
	}

	return nil
}
