package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/webhook"
)

// WebhookResponse is the API message for Feishu webhook response.
type WebhookResponse struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// WebhookPostSection is the API message for Feishu webhook POST section.
type WebhookPostSection struct {
	Tag  string `json:"tag"`
	Text string `json:"text"`
	Href string `json:"href,omitempty"`
}

// WebhookPostLine is the API message for Feishu webhook POST line.
type WebhookPostLine struct {
	SectionList []WebhookPostSection `json:""`
}

// WebhookPost is the API message for Feishu webhook POST.
type WebhookPost struct {
	Title       string                 `json:"title"`
	ContentList [][]WebhookPostSection `json:"content"`
}

// WebhookPostLanguage is the API message for Feishu webhook POST language.
type WebhookPostLanguage struct {
	English WebhookPost `json:"en_us"`
}

// WebhookContent is the API message for Feishu webhook content.
type WebhookContent struct {
	Post WebhookPostLanguage `json:"post"`
}

// WebhookCardConfig is the API message for Feishu webhook card config.
type WebhookCardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode,omitempty"`
	EnableForward  bool `json:"enable_forward,omitempty"`
}

// WebhookMarkdownSection is the API message for Feishu webhook card i18n content markdown.
type WebhookMarkdownSection struct {
	Tag     string `json:"tag,omitempty"`
	Content string `json:"content,omitempty"`
}

// WebhookCardI18nElements is the API message for Feishu webhook card i18n content.
type WebhookCardI18nElements struct {
	English []WebhookMarkdownSection `json:"en_us"`
}

// WebhookCardHeaderTitle is the API message for Feishu webhook card header title.
type WebhookCardHeaderTitle struct {
	Content string `json:"content"`
	Tag     string `json:"tag"`
}

// WebhookCardHeader is the API message for Feishu webhook card header.
type WebhookCardHeader struct {
	Title WebhookCardHeaderTitle `json:"title"`
}

// WebhookCard is the API message for Feishu webhook card.
type WebhookCard struct {
	Config       WebhookCardConfig       `json:"config"`
	Header       WebhookCardHeader       `json:"header"`
	I18nElements WebhookCardI18nElements `json:"i18n_elements"`
}

// Webhook is the API message for Feishu webhook.
type Webhook struct {
	MessageType string          `json:"msg_type"`
	Content     *WebhookContent `json:"content,omitempty"`
	Card        *WebhookCard    `json:"card,omitempty"`
}

func init() {
	webhook.Register("bb.plugin.webhook.feishu", &feishuReceiver{})
}

// feishuReceiver is the receiver for Feishu.
type feishuReceiver struct {
}

func (*feishuReceiver) Post(context webhook.Context) error {
	if len(context.MentionUsers) > 0 {
		postDirectMessage(context)
		return nil
	}
	return postMessage(context)
}

func postDirectMessage(webhookCtx webhook.Context) {
	feishu := webhookCtx.IMSetting.GetFeishu()
	if feishu == nil {
		return
	}
	p := newProvider(feishu.AppId, feishu.AppSecret)

	ctx := context.Background()

	sent := map[string]bool{}

	if err := common.Retry(ctx, func() error {
		var errs error

		var emails []string
		for _, u := range webhookCtx.MentionUsers {
			if sent[u.Email] {
				continue
			}
			emails = append(emails, u.Email)
		}

		idByEmail, err := p.getIDByEmail(ctx, emails)
		if err != nil {
			return errors.Wrapf(err, "failed to get id by email")
		}

		for _, u := range webhookCtx.MentionUsers {
			if sent[u.Email] {
				continue
			}
			id, ok := idByEmail[u.Email]
			if !ok {
				continue
			}
			err := p.sendMessage(ctx, id, getMessageCard(webhookCtx))
			if err != nil {
				err = errors.Wrapf(err, "failed to send message")
				multierr.AppendInto(&errs, err)
			}
			sent[u.Email] = true
		}
		return errs
	}); err != nil {
		slog.Warn("failed to send direct message to feishu user", log.BBError(err))
	}
}

func postMessage(context webhook.Context) error {
	post := Webhook{
		MessageType: "interactive",
		Card:        getMessageCard(context),
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

	webhookResponse := &WebhookResponse{}
	if err := json.Unmarshal(b, webhookResponse); err != nil {
		return errors.Wrapf(err, "malformed webhook response from %s", context.URL)
	}

	if webhookResponse.Code != 0 {
		return errors.Errorf("%s", webhookResponse.Message)
	}

	return nil
}

func getMessageCard(context webhook.Context) *WebhookCard {
	var markdownBuf strings.Builder

	if context.Description != "" {
		_, _ = markdownBuf.WriteString(fmt.Sprintf("%s\n", context.Description))
	}

	for _, meta := range context.GetMetaList() {
		_, _ = markdownBuf.WriteString(fmt.Sprintf("**%s**: %s\n", meta.Name, meta.Value))
	}

	_, _ = markdownBuf.WriteString(fmt.Sprintf("**By**: %s (%s)\n[View in Bytebase](%s)", context.CreatorName, context.CreatorEmail, context.Link))

	return &WebhookCard{
		Config: WebhookCardConfig{
			WideScreenMode: true,
			EnableForward:  true,
		},
		Header: WebhookCardHeader{
			Title: WebhookCardHeaderTitle{
				Content: context.Title,
				Tag:     "plain_text",
			},
		},
		I18nElements: WebhookCardI18nElements{
			English: []WebhookMarkdownSection{
				{
					Tag:     "markdown",
					Content: markdownBuf.String(),
				},
			},
		},
	}
}
