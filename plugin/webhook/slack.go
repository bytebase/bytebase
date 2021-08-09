package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SlackWebhookBlockMarkdown struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type SlackWebhookElementButton struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type SlackWebhookElement struct {
	Type   string                    `json:"type"`
	Button SlackWebhookElementButton `json:"text,omitempty"`
	URL    string                    `json:"url,omitempty"`
}

type SlackWebhookBlock struct {
	Type        string                     `json:"type"`
	Text        *SlackWebhookBlockMarkdown `json:"text,omitempty"`
	ElementList []SlackWebhookElement      `json:"elements,omitempty"`
}

type SlackWebhook struct {
	Text      string              `json:"text"`
	BlockList []SlackWebhookBlock `json:"blocks"`
}

func init() {
	register("bb.plugin.webhook.slack", &SlackReceiver{})
}

type SlackReceiver struct {
}

func (receiver *SlackReceiver) post(context WebhookContext) error {
	blockList := []SlackWebhookBlock{}
	blockList = append(blockList, SlackWebhookBlock{
		Type: "section",
		Text: &SlackWebhookBlockMarkdown{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*%s*", context.Title),
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
