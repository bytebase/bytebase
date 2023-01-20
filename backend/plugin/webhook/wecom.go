package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// WeComWebhookResponse is the API message for WeCom webhook response.
type WeComWebhookResponse struct {
	ErrorCode    int    `json:"errcode"`
	ErrorMessage string `json:"errmsg"`
}

// WeComWebhookMarkdown is the API message for WeCom webhook markdown.
type WeComWebhookMarkdown struct {
	Content string `json:"content"`
}

// WeComWebhook is the API message for WeCom webhook.
type WeComWebhook struct {
	MessageType string               `json:"msgtype"`
	Markdown    WeComWebhookMarkdown `json:"markdown"`
}

func init() {
	register("bb.plugin.webhook.wecom", &WeComReceiver{})
}

// WeComReceiver is the receiver for WeCom.
type WeComReceiver struct {
}

func (*WeComReceiver) post(context Context) error {
	metaStrList := []string{}
	for _, meta := range context.getMetaList() {
		metaStrList = append(metaStrList, fmt.Sprintf("%s: <font color=\"comment\">%s</font>", meta.Name, meta.Value))
	}
	metaStrList = append(metaStrList, fmt.Sprintf("By: <font color=\"comment\">%s (%s)</font>", context.CreatorName, context.CreatorEmail))

	status := ""
	switch context.Level {
	case WebhookSuccess:
		status = "<font color=\"green\">Success</font> "
	case WebhookWarn:
		status = "<font color=\"yellow\">Warn</font> "
	case WebhookError:
		status = "<font color=\"red\">Error</font> "
	}
	content := fmt.Sprintf("# %s%s\n\n%s\n[View in Bytebase](%s)", status, context.Title, strings.Join(metaStrList, "\n"), context.Link)
	if context.Description != "" {
		content = fmt.Sprintf("# %s%s\n> %s\n\n%s\n[View in Bytebase](%s)", status, context.Title, context.Description, strings.Join(metaStrList, "\n"), context.Link)
	}

	post := WeComWebhook{
		MessageType: "markdown",
		Markdown: WeComWebhookMarkdown{
			Content: content,
		},
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
		Timeout: timeout,
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

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("failed to POST webhook to %s, status code: %d, response body: %s", context.URL, resp.StatusCode, b)
	}

	webhookResponse := &WeComWebhookResponse{}
	if err := json.Unmarshal(b, webhookResponse); err != nil {
		return errors.Wrapf(err, "malformed webhook response from %s", context.URL)
	}

	if webhookResponse.ErrorCode != 0 {
		return errors.Errorf("%s", webhookResponse.ErrorMessage)
	}

	return nil
}
