package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/webhook"
)

// WebhookResponse is the API message for Discord webhook response.
type WebhookResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// WebhookEmbedField is the API message for Discord webhook embed field.
type WebhookEmbedField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// WebhookEmbedAuthor is the API message for Discord webhook embed Author.
type WebhookEmbedAuthor struct {
	Name string `json:"name"`
}

// WebhookEmbed is the API message for Discord webhook embed.
type WebhookEmbed struct {
	Title       string              `json:"title"`
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	URL         string              `json:"url,omitempty"`
	Timestamp   string              `json:"timestamp"`
	Author      *WebhookEmbedAuthor `json:"author,omitempty"`
	FieldList   []WebhookEmbedField `json:"fields,omitempty"`
}

// Webhook is the API message for Discord webhook.
type Webhook struct {
	EmbedList []WebhookEmbed `json:"embeds"`
}

func init() {
	webhook.Register(storepb.WebhookType_DISCORD, &Receiver{})
}

// Receiver is the receiver for Discord.
type Receiver struct {
}

func (*Receiver) Post(context webhook.Context) error {
	embedList := []WebhookEmbed{}

	fieldList := []WebhookEmbedField{}
	for _, meta := range context.GetMetaList() {
		fieldList = append(fieldList, WebhookEmbedField(meta))
	}

	status := ""
	switch context.Level {
	case webhook.WebhookSuccess:
		status = ":white_check_mark: "
	case webhook.WebhookWarn:
		status = ":warning: "
	case webhook.WebhookError:
		status = ":exclamation: "
	default:
		// No status icon for other levels
		status = ""
	}
	embed := WebhookEmbed{
		Title:       fmt.Sprintf("%s%s", status, context.Title),
		Type:        "rich",
		Description: context.Description,
		URL:         context.Link,
		FieldList:   fieldList,
	}
	if context.ActorName != "" {
		embed.Author = &WebhookEmbedAuthor{
			Name: fmt.Sprintf("%s (%s)", context.ActorName, context.ActorEmail),
		}
	}
	embedList = append(embedList, embed)

	post := Webhook{
		EmbedList: embedList,
	}
	body, err := json.Marshal(post)
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
		Timeout: webhook.Timeout,
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return errors.Errorf("failed to POST webhook %s, status code: %d, response body: %s", context.URL, resp.StatusCode, b)
	}

	webhookResponse := &WebhookResponse{}
	if err := json.Unmarshal(b, webhookResponse); err != nil {
		return errors.Wrapf(err, "malformed webhook response from %s", context.URL)
	}

	if webhookResponse.Code != 0 {
		return errors.Errorf("%s", webhookResponse.Message)
	}

	return nil
}
