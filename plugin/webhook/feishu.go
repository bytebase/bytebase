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

// FeishuWebhookResponse is the API message for Feishu webhook response.
type FeishuWebhookResponse struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// FeishuWebhookPostSection is the API message for Feishu webhook POST section.
type FeishuWebhookPostSection struct {
	Tag  string `json:"tag"`
	Text string `json:"text"`
	Href string `json:"href,omitempty"`
}

// FeishuWebhookPostLine is the API message for Feishu webhook POST line.
type FeishuWebhookPostLine struct {
	SectionList []FeishuWebhookPostSection `json:""`
}

// FeishuWebhookPost is the API message for Feishu webhook POST.
type FeishuWebhookPost struct {
	Title       string                       `json:"title"`
	ContentList [][]FeishuWebhookPostSection `json:"content"`
}

// FeishuWebhookPostLanguage is the API message for Feishu webhook POST language.
type FeishuWebhookPostLanguage struct {
	English FeishuWebhookPost `json:"en_us"`
}

// FeishuWebhookContent is the API message for Feishu webhook content.
type FeishuWebhookContent struct {
	Post FeishuWebhookPostLanguage `json:"post"`
}

// FeishuWebhookCardConfig is the API message for Feishu webhook card config.
type FeishuWebhookCardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode,omitempty"`
	EnableForward  bool `json:"enable_forward,omitempty"`
}

// FeishuWebhookMarkdownSection is the API message for Feishu webhook card i18n content markdown.
type FeishuWebhookMarkdownSection struct {
	Tag     string `json:"tag,omitempty"`
	Content string `json:"content,omitempty"`
}

// FeishuWebhookCardI18nElements is the API message for Feishu webhook card i18n content.
type FeishuWebhookCardI18nElements struct {
	English []FeishuWebhookMarkdownSection `json:"en_us"`
}

// FeishuWebhookCardHeaderTitle is the API message for Feishu webhook card header title.
type FeishuWebhookCardHeaderTitle struct {
	Content string `json:"content"`
	Tag     string `json:"tag"`
}

// FeishuWebhookCardHeader is the API message for Feishu webhook card header.
type FeishuWebhookCardHeader struct {
	Title FeishuWebhookCardHeaderTitle `json:"title"`
}

// FeishuWebhookCard is the API message for Feishu webhook card.
type FeishuWebhookCard struct {
	Config       FeishuWebhookCardConfig       `json:"config"`
	Header       FeishuWebhookCardHeader       `json:"header"`
	I18nElements FeishuWebhookCardI18nElements `json:"i18n_elements"`
}

// FeishuWebhook is the API message for Feishu webhook.
type FeishuWebhook struct {
	MessageType string                `json:"msg_type"`
	Content     *FeishuWebhookContent `json:"content,omitempty"`
	Card        *FeishuWebhookCard    `json:"card,omitempty"`
}

func init() {
	register("bb.plugin.webhook.feishu", &FeishuReceiver{})
}

// FeishuReceiver is the receiver for Feishu.
type FeishuReceiver struct{}

func (*FeishuReceiver) post(context Context) error {
	// TODO(p0ny): handle context.Task
	var markdownBuf strings.Builder

	if context.Description != "" {
		if _, err := markdownBuf.WriteString(fmt.Sprintf("%s\n", context.Description)); err != nil {
			return err
		}
	}

	for _, meta := range context.getMetaList() {
		if _, err := markdownBuf.WriteString(fmt.Sprintf("**%s**: %s\n", meta.Name, meta.Value)); err != nil {
			return err
		}
	}

	if _, err := markdownBuf.WriteString(fmt.Sprintf("**By**: %s (%s)\n[View in Bytebase](%s)", context.CreatorName, context.CreatorEmail, context.Link)); err != nil {
		return err
	}

	post := FeishuWebhook{
		MessageType: "interactive",
		Card: &FeishuWebhookCard{
			Config: FeishuWebhookCardConfig{
				WideScreenMode: true,
				EnableForward:  true,
			},
			Header: FeishuWebhookCardHeader{
				Title: FeishuWebhookCardHeaderTitle{
					Content: context.Title,
					Tag:     "plain_text",
				},
			},
			I18nElements: FeishuWebhookCardI18nElements{
				English: []FeishuWebhookMarkdownSection{
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

	webhookResponse := &FeishuWebhookResponse{}
	if err := json.Unmarshal(b, webhookResponse); err != nil {
		return errors.Wrapf(err, "malformed webhook response from %s", context.URL)
	}

	if webhookResponse.Code != 0 {
		return errors.Errorf("%s", webhookResponse.Message)
	}

	return nil
}
