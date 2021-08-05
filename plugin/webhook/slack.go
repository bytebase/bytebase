package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type SlackWebhookBlockText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type SlackWebhookBlock struct {
	Type string                `json:"type"`
	Text SlackWebhookBlockText `json:"text"`
}

type SlackWebhook struct {
	Text      string              `json:"text"`
	BlockList []SlackWebhookBlock `json:"blocks"`
}

func init() {
	register("hooks.slack.com", &SlackReceiver{})
}

type SlackReceiver struct {
}

func (receiver *SlackReceiver) post(url string, title string, description string, metaList []WebHookMeta, link string) error {
	blockList := []SlackWebhookBlock{}
	blockList = append(blockList, SlackWebhookBlock{
		Type: "section",
		Text: SlackWebhookBlockText{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*%s*", title),
		},
	})

	if description != "" {
		blockList = append(blockList, SlackWebhookBlock{
			Type: "section",
			Text: SlackWebhookBlockText{
				Type: "mrkdwn",
				Text: fmt.Sprintf("```%s```", description),
			},
		})
	}

	for _, meta := range metaList {
		blockList = append(blockList, SlackWebhookBlock{
			Type: "section",
			Text: SlackWebhookBlockText{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*%s:* %s", meta.Name, meta.Value),
			},
		})
	}

	blockList = append(blockList, SlackWebhookBlock{
		Type: "section",
		Text: SlackWebhookBlockText{
			Type: "mrkdwn",
			Text: fmt.Sprintf("<%s|View in Bytebase>", link),
		},
	})

	post := SlackWebhook{
		Text:      title,
		BlockList: blockList,
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
