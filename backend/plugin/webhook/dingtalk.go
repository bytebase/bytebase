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

// DingTalkMention is the API message for DingTalk webhook to mention users in DingTalk.
// https://open.dingtalk.com/document/orgapp/custom-robots-send-group-messages
type DingTalkMention struct {
	Mobiles []string `json:"atMobiles"`
}

// DingTalkWebhook is the API message for DingTalk webhook.
type DingTalkWebhook struct {
	MessageType string                  `json:"msgtype"`
	Markdown    DingTalkWebhookMarkdown `json:"markdown"`
	Mention     DingTalkMention         `json:"at"`
}

func init() {
	register("bb.plugin.webhook.dingtalk", &DingTalkReceiver{})
}

// DingTalkReceiver is the receiver for DingTalk.
type DingTalkReceiver struct {
}

func (*DingTalkReceiver) post(context Context) error {
	metaStrList := []string{}
	for _, meta := range context.getMetaList() {
		metaStrList = append(metaStrList, fmt.Sprintf("##### **%s:** %s", meta.Name, meta.Value))
	}
	metaStrList = append(metaStrList, fmt.Sprintf("##### **By:** %s (%s)", context.CreatorName, context.CreatorEmail))

	text := fmt.Sprintf("# %s\n%s\n##### [View in Bytebase](%s)", context.Title, strings.Join(metaStrList, "\n"), context.Link)
	if context.Description != "" {
		text = fmt.Sprintf("# %s\n> %s\n%s\n##### [View in Bytebase](%s)", context.Title, context.Description, strings.Join(metaStrList, "\n"), context.Link)
	}
	if len(context.MentionUsersByPhone) > 0 {
		var ats []string
		for _, phone := range context.MentionUsersByPhone {
			ats = append(ats, fmt.Sprintf("@%s", phone))
		}
		text += "\n" + strings.Join(ats, " ")
	}

	post := DingTalkWebhook{
		MessageType: "markdown",
		Markdown: DingTalkWebhookMarkdown{
			Title: context.Title,
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
		return errors.Errorf("failed to POST webhook %s, status code: %d, response body: %s", context.URL, resp.StatusCode, b)
	}

	webhookResponse := &DingTalkWebhookResponse{}
	if err := json.Unmarshal(b, webhookResponse); err != nil {
		return errors.Wrapf(err, "malformed webhook response from %s", context.URL)
	}

	if webhookResponse.ErrorCode != 0 {
		return errors.Errorf("%s", webhookResponse.ErrorMessage)
	}

	return nil
}
