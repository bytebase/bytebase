package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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
	register("bb.plugin.webhook.slack", &SlackReceiver{})
}

// SlackReceiver is the receiver for Slack.
type SlackReceiver struct {
}

func (receiver *SlackReceiver) post(context Context) error {
	blockList := []SlackWebhookBlock{}

	status := ""
	switch context.Level {
	case WebhookSuccess:
		status = ":white_check_mark: "
	case WebhookWarn:
		status = ":warning: "
	case WebhookError:
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

	for _, meta := range context.MetaList {
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
		Type: "section",
		Text: &SlackWebhookBlockMarkdown{
			Type: "mrkdwn",
			Text: fmt.Sprintf("At: %s", time.Unix(context.CreatedTs, 0).Format(timeFormat)),
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

	post := SlackWebhook{
		Text:      context.Title,
		BlockList: blockList,
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

	if string(b) != "ok" {
		return fmt.Errorf("%s", fmt.Sprintf("%.100s", string(b)))
	}

	return nil
}
