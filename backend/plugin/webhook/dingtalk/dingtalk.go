package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"

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
	webhook.Register("bb.plugin.webhook.dingtalk", &DingTalkReceiver{})
}

// DingTalkReceiver is the receiver for DingTalk.
type DingTalkReceiver struct {
}

func (*DingTalkReceiver) Post(context webhook.Context) error {
	metaStrList := []string{}
	for _, meta := range context.GetMetaListZh() {
		metaStrList = append(metaStrList, fmt.Sprintf("##### **%s:** %s", meta.Name, meta.Value))
	}
	metaStrList = append(metaStrList, fmt.Sprintf("##### **由:** %s (%s)", context.ActorName, context.ActorEmail))

	text := fmt.Sprintf("# %s\n%s\n##### [在 Bytebase 中显示](%s)", context.TitleZh, strings.Join(metaStrList, "\n"), context.Link)
	if context.Description != "" {
		text = fmt.Sprintf("# %s\n> %s\n%s\n##### [在 Bytebase 中显示](%s)", context.TitleZh, context.Description, strings.Join(metaStrList, "\n"), context.Link)
	}
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
