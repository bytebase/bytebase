package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type DiscordWebhookEmbedField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DiscordWebhookEmbedAuthor struct {
	Name string `json:"name"`
}

type DiscordWebhookEmbed struct {
	Title       string                     `json:"title"`
	Type        string                     `json:"type"`
	Description string                     `json:"description,omitempty"`
	URL         string                     `json:"url,omitempty"`
	Timestamp   string                     `json:"timestamp"`
	Author      DiscordWebhookEmbedAuthor  `json:"author"`
	FieldList   []DiscordWebhookEmbedField `json:"fields,omitempty"`
}

type DiscordWebhook struct {
	EmbedList []DiscordWebhookEmbed `json:"embeds"`
}

func init() {
	register("bb.plugin.webhook.discord", &DiscordReceiver{})
}

type DiscordReceiver struct {
}

func (receiver *DiscordReceiver) post(context WebhookContext) error {
	embedList := []DiscordWebhookEmbed{}

	fieldList := []DiscordWebhookEmbedField{}
	for _, meta := range context.MetaList {
		fieldList = append(fieldList, DiscordWebhookEmbedField(meta))
	}

	embedList = append(embedList, DiscordWebhookEmbed{
		Title:       context.Title,
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
	_, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST webhook %+v (%w)", context.URL, err)
	}

	return nil
}
