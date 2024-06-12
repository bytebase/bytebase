package feishu

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
	var markdownBuf strings.Builder

	if context.Description != "" {
		if _, err := markdownBuf.WriteString(fmt.Sprintf("%s\n", context.Description)); err != nil {
			return err
		}
	}

	for _, meta := range context.GetMetaList() {
		if _, err := markdownBuf.WriteString(fmt.Sprintf("**%s**: %s\n", meta.Name, meta.Value)); err != nil {
			return err
		}
	}

	if _, err := markdownBuf.WriteString(fmt.Sprintf("**By**: %s (%s)\n[View in Bytebase](%s)", context.CreatorName, context.CreatorEmail, context.Link)); err != nil {
		return err
	}

	post := Webhook{
		MessageType: "interactive",
		Card: &WebhookCard{
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
