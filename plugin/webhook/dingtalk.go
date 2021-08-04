package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type DingTalkWebhookMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type DingTalkWebhook struct {
	MessageType string                  `json:"msgtype"`
	Markdown    DingTalkWebhookMarkdown `json:"markdown"`
}

func init() {
	register("oapi.dingtalk.com", &DingTalkReceiver{})
}

type DingTalkReceiver struct {
}

func (receiver *DingTalkReceiver) post(url string, title string, description string, metaList []WebHookMeta, link string) error {
	metaStrList := []string{}
	for _, meta := range metaList {
		metaStrList = append(metaStrList, fmt.Sprintf("##### **%s:** %s", meta.Name, meta.Value))
	}
	text := fmt.Sprintf("# %s\n%s\n##### [View in Bytebase](%s)", title, strings.Join(metaStrList, "\n"), link)
	if description != "" {
		text = fmt.Sprintf("# %s\n> %s\n%s\n##### [View in Bytebase](%s)", title, description, strings.Join(metaStrList, "\n"), link)
	}

	post := DingTalkWebhook{
		MessageType: "markdown",
		Markdown: DingTalkWebhookMarkdown{
			Title: title,
			Text:  text,
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
