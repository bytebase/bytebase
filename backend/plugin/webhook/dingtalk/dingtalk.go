package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/webhook"
)

// Response is the API message for DingTalk webhook response.
type Response struct {
	ErrorCode    int    `json:"errcode"`
	ErrorMessage string `json:"errmsg"`
}

// MessageMarkdown is the API message for DingTalk webhook markdown.
type MessageMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// Mention is the API message for DingTalk webhook to mention users in DingTalk.
// https://open.dingtalk.com/document/orgapp/custom-robots-send-group-messages
type Mention struct {
	Mobiles []string `json:"atMobiles"`
}

// Message is the API message for DingTalk webhook message.
type Message struct {
	MessageType string          `json:"msgtype"`
	Markdown    MessageMarkdown `json:"markdown"`
	Mention     Mention         `json:"at"`
}

func init() {
	webhook.Register("bb.plugin.webhook.dingtalk", &Receiver{})
}

// Receiver is the receiver for DingTalk.
type Receiver struct {
}

func (*Receiver) Post(context webhook.Context) error {
	if context.DirectMessage && len(context.MentionEndUsers) > 0 {
		if sendDirectMessage(context) {
			return nil
		}
	}
	return sendMessage(context)
}

// returns true if the message is sent successfully.
func sendDirectMessage(webhookCtx webhook.Context) bool {
	dingtalk := webhookCtx.IMSetting.GetDingtalk()
	if dingtalk == nil {
		return false
	}
	p := newProvider(dingtalk.ClientId, dingtalk.ClientSecret, dingtalk.RobotCode)
	ctx := context.Background()

	sent := map[string]bool{}
	if err := common.Retry(ctx, func() error {
		var errs error
		var userPhones, userIDs []string
		for _, u := range webhookCtx.MentionEndUsers {
			if u.Phone == "" {
				continue
			}
			if sent[u.Phone] {
				continue
			}

			userID, err := p.getIDByPhone(ctx, u.Phone)
			if err != nil {
				err = errors.Wrapf(err, "failed to get user id by phone %v", u.Phone)
				multierr.AppendInto(&errs, err)
				continue
			}
			if userID == "" {
				// user not found
				sent[u.Phone] = true
				continue
			}
			userIDs = append(userIDs, userID)
			userPhones = append(userPhones, u.Phone)
		}
		if len(userIDs) == 0 {
			err := errors.Errorf("dingtalk dm: got 0 user id, errs: %v", errs)
			return backoff.Permanent(err)
		}

		if err := p.sendMessage(ctx, userIDs, webhookCtx.TitleZh, getMarkdownText(webhookCtx)); err != nil {
			err = errors.Wrapf(err, "failed to send message")
			multierr.AppendInto(&errs, err)
		} else {
			for _, phone := range userPhones {
				sent[phone] = true
			}
		}
		return errs
	}); err != nil {
		slog.Warn("failed to send direct message to dingtalk users", log.BBError(err))
		return false
	}

	return true
}

func sendMessage(context webhook.Context) error {
	text := getMarkdownText(context)
	if len(context.MentionUsersByPhone) > 0 {
		var ats []string
		for _, phone := range context.MentionUsersByPhone {
			ats = append(ats, fmt.Sprintf("@%s", phone))
		}
		text += "\n" + strings.Join(ats, " ")
	}

	post := Message{
		MessageType: "markdown",
		Markdown: MessageMarkdown{
			Title: context.TitleZh,
			Text:  text,
		},
	}
	if len(context.MentionUsersByPhone) > 0 {
		post.Mention.Mobiles = append(post.Mention.Mobiles, context.MentionUsersByPhone...)
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
		return errors.Errorf("failed to POST webhook %s, status code: %d, response body: %s", context.URL, resp.StatusCode, b)
	}

	webhookResponse := &Response{}
	if err := json.Unmarshal(b, webhookResponse); err != nil {
		return errors.Wrapf(err, "malformed webhook response from %s", context.URL)
	}

	if webhookResponse.ErrorCode != 0 {
		return errors.Errorf("%s", webhookResponse.ErrorMessage)
	}

	return nil
}

func getMarkdownText(context webhook.Context) string {
	var metaStrList []string
	for _, meta := range context.GetMetaListZh() {
		metaStrList = append(metaStrList, fmt.Sprintf("##### **%s:** %s", meta.Name, meta.Value))
	}
	metaStrList = append(metaStrList, fmt.Sprintf("##### **由:** %s (%s)", context.ActorName, context.ActorEmail))

	text := fmt.Sprintf("# %s\n%s\n##### [在 Bytebase 中显示](%s)", context.TitleZh, strings.Join(metaStrList, "\n"), context.Link)
	if context.Description != "" {
		text = fmt.Sprintf("# %s\n> %s\n%s\n##### [在 Bytebase 中显示](%s)", context.TitleZh, context.Description, strings.Join(metaStrList, "\n"), context.Link)
	}
	return text
}
