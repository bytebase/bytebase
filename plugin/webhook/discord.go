package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type DiscordWebhookEmbedField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DiscordWebhookEmbed struct {
	Title       string                     `json:"title"`
	Type        string                     `json:"type"`
	Description string                     `json:"description,omitempty"`
	URL         string                     `json:"url,omitempty"`
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

func (receiver *DiscordReceiver) post(url string, title string, description string, metaList []WebHookMeta, link string) error {
	embedList := []DiscordWebhookEmbed{}

	fieldList := []DiscordWebhookEmbedField{}
	for _, meta := range metaList {
		fieldList = append(fieldList, DiscordWebhookEmbedField(meta))
	}

	embedList = append(embedList, DiscordWebhookEmbed{
		Title:       title,
		Type:        "rich",
		Description: description,
		URL:         link,
		FieldList:   fieldList,
	})

	post := DiscordWebhook{
		EmbedList: embedList,
	}
	body, err := json.Marshal(post)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook POST request: %v", url)
	}
	req, err := http.NewRequest("POST",
		url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to construct webhook POST request %v (%w)", url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: timeout,
	}
	_, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST webhook %+v (%w)", url, err)
	}

	return nil
}
