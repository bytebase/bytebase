package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DingTalkWebhookResponse is the API message for DingTalk webhook response.
type DingTalkWebhookResponse struct {
	ErrorCode    int    `json:"errcode"`
	ErrorMessage string `json:"errmsg"`
}

// DingTalkWebhookMarkdown is the API message for DingTalk webhook markdown.
type DingTalkWebhookMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// DingTalkWebhook is the API message for DingTalk webhook.
type DingTalkWebhook struct {
	MessageType string                  `json:"msgtype"`
	Markdown    DingTalkWebhookMarkdown `json:"markdown"`
}

func init() {
	register("bb.plugin.webhook.dingtalk", &DingTalkReceiver{})
}

// DingTalkReceiver is the receiver for DingTalk.
type DingTalkReceiver struct {
}

func (receiver *DingTalkReceiver) post(context Context) error {
	metaStrList := []string{}
	for _, meta := range context.MetaList {
		metaStrList = append(metaStrList, fmt.Sprintf("##### **%s:** %s", meta.Name, meta.Value))
	}
	metaStrList = append(metaStrList, fmt.Sprintf("##### **By:** %s (%s)", context.CreatorName, context.CreatorEmail))
	metaStrList = append(metaStrList, fmt.Sprintf("##### **At:** %s", time.Unix(context.CreatedTs, 0).Format(timeFormat)))

	text := fmt.Sprintf("# %s\n%s\n##### [View in Bytebase](%s)", context.Title, strings.Join(metaStrList, "\n"), context.Link)
	if context.Description != "" {
		text = fmt.Sprintf("# %s\n> %s\n%s\n##### [View in Bytebase](%s)", context.Title, context.Description, strings.Join(metaStrList, "\n"), context.Link)
	}

	post := DingTalkWebhook{
		MessageType: "markdown",
		Markdown: DingTalkWebhookMarkdown{
			Title: context.Title,
			Text:  text,
		},
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
		return fmt.Errorf("failed to POST webhook %v (%w)", context.URL, err)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read POST webhook response %v (%w)", context.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to POST webhook %v, status code: %d, response body: %s", context.URL, resp.StatusCode, b)
	}

	webhookResponse := &DingTalkWebhookResponse{}
	if err := json.Unmarshal(b, webhookResponse); err != nil {
		return fmt.Errorf("malformatted webhook response %v (%w)", context.URL, err)
	}

	if webhookResponse.ErrorCode != 0 {
		return fmt.Errorf("%s", webhookResponse.ErrorMessage)
	}

	return nil
}
