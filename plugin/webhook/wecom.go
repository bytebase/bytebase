package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type WeComWebhookMarkdown struct {
	Content string `json:"content"`
}

type WeComWebhook struct {
	MessageType string               `json:"msgtype"`
	Markdown    WeComWebhookMarkdown `json:"markdown"`
}

func init() {
	register("bb.plugin.webhook.wecom", &WeComReceiver{})
}

type WeComReceiver struct {
}

func (receiver *WeComReceiver) post(context WebhookContext) error {
	metaStrList := []string{}
	for _, meta := range context.MetaList {
		metaStrList = append(metaStrList, fmt.Sprintf("%s: <font color=\"comment\">%s</font>", meta.Name, meta.Value))
	}
	metaStrList = append(metaStrList, fmt.Sprintf("By: <font color=\"comment\">%s (%s)</font>", context.CreatorName, context.CreatorEmail))
	metaStrList = append(metaStrList, fmt.Sprintf("At: <font color=\"comment\">%s</font>", time.Unix(context.CreatedTs, 0).Format(timeFormat)))

	content := fmt.Sprintf("# %s\n\n%s\n[View in Bytebase](%s)", context.Title, strings.Join(metaStrList, "\n"), context.Link)
	if context.Description != "" {
		content = fmt.Sprintf("# %s\n> %s\n\n%s\n[View in Bytebase](%s)", context.Title, context.Description, strings.Join(metaStrList, "\n"), context.Link)
	}

	post := WeComWebhook{
		MessageType: "markdown",
		Markdown: WeComWebhookMarkdown{
			Content: content,
		},
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
		return fmt.Errorf("failed to POST webhook %v (%w)", context.URL, err)
	}

	return nil
}
