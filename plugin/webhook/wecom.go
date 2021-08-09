package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

func (receiver *WeComReceiver) post(url string, title string, description string, metaList []WebHookMeta, link string) error {
	metaStrList := []string{}
	for _, meta := range metaList {
		metaStrList = append(metaStrList, fmt.Sprintf("%s: <font color=\"comment\">%s</font>", meta.Name, meta.Value))
	}

	content := fmt.Sprintf("# %s\n\n%s\n[View in Bytebase](%s)", title, strings.Join(metaStrList, "\n"), link)
	if description != "" {
		content = fmt.Sprintf("# %s\n> %s\n\n%s\n[View in Bytebase](%s)", title, description, strings.Join(metaStrList, "\n"), link)
	}

	post := WeComWebhook{
		MessageType: "markdown",
		Markdown: WeComWebhookMarkdown{
			Content: content,
		},
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
		return fmt.Errorf("failed to POST webhook %v (%w)", url, err)
	}

	return nil
}
