package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/webhook"
)

// SlackWebhookBlockMarkdown is the API message for Slack webhook block markdown.
type SlackWebhookBlockMarkdown struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SlackWebhookElementButton is the API message for Slack webhook element button.
type SlackWebhookElementButton struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// SlackWebhookElement is the API message for Slack webhook element.
type SlackWebhookElement struct {
	Type   string                    `json:"type"`
	Button SlackWebhookElementButton `json:"text,omitempty"`
	URL    string                    `json:"url,omitempty"`
}

// SlackWebhookBlock is the API message for Slack webhook block.
type SlackWebhookBlock struct {
	Type        string                     `json:"type"`
	Text        *SlackWebhookBlockMarkdown `json:"text,omitempty"`
	ElementList []SlackWebhookElement      `json:"elements,omitempty"`
}

// SlackWebhook is the API message for Slack webhook.
type SlackWebhook struct {
	Text      string              `json:"text"`
	BlockList []SlackWebhookBlock `json:"blocks"`
}

func init() {
	webhook.Register("bb.plugin.webhook.slack", &SlackReceiver{})
}

// SlackReceiver is the receiver for Slack.
type SlackReceiver struct {
}

func GetBlocks(context webhook.Context) []SlackWebhookBlock {
	blockList := []SlackWebhookBlock{}

	status := ""
	switch context.Level {
	case webhook.WebhookSuccess:
		status = ":white_check_mark: "
	case webhook.WebhookWarn:
		status = ":warning: "
	case webhook.WebhookError:
		status = ":exclamation: "
	}
	blockList = append(blockList, SlackWebhookBlock{
		Type: "section",
		Text: &SlackWebhookBlockMarkdown{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*%s%s*", status, context.Title),
		},
	})

	if context.Description != "" {
		blockList = append(blockList, SlackWebhookBlock{
			Type: "section",
			Text: &SlackWebhookBlockMarkdown{
				Type: "mrkdwn",
				Text: fmt.Sprintf("```%s```", context.Description),
			},
		})
	}

	for _, meta := range context.GetMetaList() {
		blockList = append(blockList, SlackWebhookBlock{
			Type: "section",
			Text: &SlackWebhookBlockMarkdown{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*%s:* %s", meta.Name, meta.Value),
			},
		})
	}

	blockList = append(blockList, SlackWebhookBlock{
		Type: "section",
		Text: &SlackWebhookBlockMarkdown{
			Type: "mrkdwn",
			Text: fmt.Sprintf("By: %s (%s)", context.CreatorName, context.CreatorEmail),
		},
	})

	blockList = append(blockList, SlackWebhookBlock{
		Type: "actions",
		ElementList: []SlackWebhookElement{
			{
				Type: "button",
				Button: SlackWebhookElementButton{
					Type: "plain_text",
					Text: "View in Bytebase",
				},
				URL: context.Link,
			},
		},
	})

	return blockList
}

func (*SlackReceiver) Post(context webhook.Context) error {
	return postMessage(context)
}

func postMessage(context webhook.Context) error {
	blockList := GetBlocks(context)

	post := SlackWebhook{
		Text:      context.Title,
		BlockList: blockList,
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
		Timeout: 3 * time.Second,
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

	if string(b) != "ok" {
		return errors.Errorf("%.100s", string(b))
	}

	return nil
}

func postDirectMessage(context webhook.Context) error {
	return nil
}
