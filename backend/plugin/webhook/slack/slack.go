package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/webhook"
)

// getSlackToken extracts the Slack token from the AppIMSetting.
func getSlackToken(setting *storepb.AppIMSetting) string {
	if setting == nil {
		return ""
	}
	for _, s := range setting.Settings {
		if s.Type == storepb.WebhookType_SLACK {
			if slack := s.GetSlack(); slack != nil {
				return slack.Token
			}
		}
	}
	return ""
}

// BlockMarkdown is the API message for Slack webhook block markdown.
type BlockMarkdown struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ElementButton is the API message for Slack webhook element button.
type ElementButton struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Element is the API message for Slack webhook element.
type Element struct {
	Type   string        `json:"type"`
	Button ElementButton `json:"text,omitempty"`
	URL    string        `json:"url,omitempty"`
}

// Block is the API message for Slack webhook block.
type Block struct {
	Type        string         `json:"type"`
	Text        *BlockMarkdown `json:"text,omitempty"`
	ElementList []Element      `json:"elements,omitempty"`
}

// MessagePayload is the API message for Slack webhook.
type MessagePayload struct {
	Text      string  `json:"text"`
	BlockList []Block `json:"blocks"`
}

func init() {
	webhook.Register(storepb.WebhookType_SLACK, &Receiver{})
}

// Receiver is the receiver for Slack.
type Receiver struct {
}

func GetBlocks(context webhook.Context) []Block {
	blockList := []Block{}

	status := ""
	switch context.Level {
	case webhook.WebhookSuccess:
		status = ":white_check_mark: "
	case webhook.WebhookWarn:
		status = ":warning: "
	case webhook.WebhookError:
		status = ":exclamation: "
	default:
		status = ""
	}
	blockList = append(blockList, Block{
		Type: "section",
		Text: &BlockMarkdown{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*%s%s*", status, context.Title),
		},
	})

	if context.Description != "" {
		blockList = append(blockList, Block{
			Type: "section",
			Text: &BlockMarkdown{
				Type: "mrkdwn",
				Text: fmt.Sprintf("```%s```", context.Description),
			},
		})
	}

	for _, meta := range context.GetMetaList() {
		blockList = append(blockList, Block{
			Type: "section",
			Text: &BlockMarkdown{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*%s:* %s", meta.Name, meta.Value),
			},
		})
	}

	if context.ActorName != "" {
		blockList = append(blockList, Block{
			Type: "section",
			Text: &BlockMarkdown{
				Type: "mrkdwn",
				Text: fmt.Sprintf("Actor: %s (%s)", context.ActorName, context.ActorEmail),
			},
		})
	}

	blockList = append(blockList, Block{
		Type: "actions",
		ElementList: []Element{
			{
				Type: "button",
				Button: ElementButton{
					Type: "plain_text",
					Text: "View in Bytebase",
				},
				URL: context.Link,
			},
		},
	})

	return blockList
}

func (*Receiver) Post(context webhook.Context) error {
	if context.DirectMessage && len(context.MentionEndUsers) > 0 {
		if postDirectMessage(context) {
			return nil
		}
		slog.Warn("failed to send direct messages for slack users, fallback to normal webhook", slog.String("event", context.EventType))
	}
	return postMessage(context)
}

func postMessage(context webhook.Context) error {
	blockList := GetBlocks(context)

	post := MessagePayload{
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
		Timeout: webhook.Timeout,
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

func postDirectMessage(webhookCtx webhook.Context) bool {
	ctx := context.Background()
	t := getSlackToken(webhookCtx.IMSetting)
	if t == "" {
		slog.Warn("failed to found valid slack im token", slog.String("event", webhookCtx.EventType))
		return false
	}
	p := newProvider(t)
	sent := map[string]bool{}
	if err := common.Retry(ctx, func() error {
		var errs error
		for _, u := range webhookCtx.MentionEndUsers {
			if sent[u.Email] {
				continue
			}
			err := func() error {
				userID, err := p.lookupByEmail(ctx, u.Email)
				if err != nil {
					return errors.Wrapf(err, "failed to lookup user")
				}
				if userID == "" {
					return errors.Errorf("failed to find user id for %v", u.Email)
				}
				channelID, err := p.openConversation(ctx, userID)
				if err != nil {
					return errors.Wrapf(err, "failed to open conversation")
				}
				if err := p.chatPostMessage(ctx, channelID, webhookCtx); err != nil {
					return errors.Wrapf(err, "failed to post message")
				}
				sent[u.Email] = true
				return nil
			}()
			multierr.AppendInto(&errs, errors.Wrapf(err, "failed to send message to user %v", u.Email))
		}
		return errs
	}); err != nil {
		slog.Warn("failed to send direct message to slack user", log.BBError(err), slog.String("event", webhookCtx.EventType))
		return false
	}

	return true
}
