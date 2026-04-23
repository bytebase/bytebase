// Package googlechat implements Google Chat incoming webhook integration.
package googlechat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/webhook"
)

// MessagePayload is the API message for Google Chat webhook.
type MessagePayload struct {
	Text    string   `json:"text,omitempty"`
	CardsV2 []CardV2 `json:"cardsV2,omitempty"`
}

// CardV2 is the Google Chat card wrapper.
type CardV2 struct {
	Card Card `json:"card"`
}

// Card is the Google Chat card.
type Card struct {
	Header   *CardHeader `json:"header,omitempty"`
	Sections []Section   `json:"sections,omitempty"`
}

// CardHeader is the Google Chat card header.
type CardHeader struct {
	Title    string `json:"title,omitempty"`
	Subtitle string `json:"subtitle,omitempty"`
}

// Section is a Google Chat card section.
type Section struct {
	Widgets []Widget `json:"widgets,omitempty"`
}

// Widget is a Google Chat card widget.
type Widget struct {
	TextParagraph *TextParagraph `json:"textParagraph,omitempty"`
	ButtonList    *ButtonList    `json:"buttonList,omitempty"`
}

// TextParagraph is a Google Chat text paragraph widget.
type TextParagraph struct {
	Text string `json:"text"`
}

// ButtonList is a Google Chat button list widget.
type ButtonList struct {
	Buttons []Button `json:"buttons,omitempty"`
}

// Button is a Google Chat button.
type Button struct {
	Text    string   `json:"text"`
	OnClick *OnClick `json:"onClick,omitempty"`
}

// OnClick is a Google Chat button click action.
type OnClick struct {
	OpenLink *OpenLink `json:"openLink,omitempty"`
}

// OpenLink is a Google Chat open link action.
type OpenLink struct {
	URL string `json:"url"`
}

func init() {
	webhook.Register(storepb.WebhookType_GOOGLE_CHAT, &Receiver{})
}

// Receiver is the receiver for Google Chat.
type Receiver struct{}

// Post posts the message to Google Chat.
func (*Receiver) Post(context webhook.Context) error {
	return postMessage(context)
}

// BuildMessage constructs the Google Chat message payload.
func BuildMessage(ctx webhook.Context) MessagePayload {
	header := &CardHeader{
		Title: levelEmoji(ctx.Level) + escapeText(ctx.Title),
	}
	if ctx.Project != nil {
		header.Subtitle = escapeText(ctx.Project.Title)
	}

	widgets := []Widget{}
	if ctx.Description != "" {
		widgets = append(widgets, textWidget(ctx.Description))
	}
	if ctx.Issue != nil {
		if ctx.Issue.Name != "" {
			widgets = append(widgets, textWidgetHTML(fmt.Sprintf("<b>%s</b>", escapeText(ctx.Issue.Name))))
		}
		if ctx.Issue.Description != "" {
			widgets = append(widgets, textWidgetHTML(fmt.Sprintf("<b>%s</b>", escapeText(common.TruncateStringWithDescription(ctx.Issue.Description)))))
		}
	} else if ctx.Rollout != nil && ctx.Rollout.Title != "" {
		widgets = append(widgets, textWidgetHTML(fmt.Sprintf("<b>%s</b>", escapeText(ctx.Rollout.Title))))
	}

	if metadata := metadataText(ctx); metadata != "" {
		widgets = append(widgets, textWidgetHTML(metadata))
	}
	if ctx.Link != "" {
		widgets = append(widgets, Widget{
			ButtonList: &ButtonList{
				Buttons: []Button{
					{
						Text: "View in Bytebase",
						OnClick: &OnClick{
							OpenLink: &OpenLink{URL: ctx.Link},
						},
					},
				},
			},
		})
	}

	card := Card{Header: header}
	if len(widgets) > 0 {
		card.Sections = []Section{{Widgets: widgets}}
	}

	return MessagePayload{
		CardsV2: []CardV2{
			{
				Card: card,
			},
		},
	}
}

func metadataText(ctx webhook.Context) string {
	parts := []string{}
	if ctx.Project != nil && ctx.Project.Title != "" {
		parts = append(parts, fmt.Sprintf("<b>Project:</b> %s", escapeText(ctx.Project.Title)))
	}
	if ctx.Issue != nil && ctx.Issue.Creator.Name != "" {
		parts = append(parts, fmt.Sprintf("<b>Creator:</b> %s", escapeText(ctx.Issue.Creator.Name)))
	}
	if ctx.Environment != "" {
		parts = append(parts, fmt.Sprintf("<b>Environment:</b> %s", escapeText(ctx.Environment)))
	}
	if ctx.ActorName != "" {
		parts = append(parts, fmt.Sprintf("<b>By:</b> %s", escapeText(ctx.ActorName)))
	}
	return strings.Join(parts, "<br>")
}

func textWidget(text string) Widget {
	return textWidgetHTML(escapeText(text))
}

func textWidgetHTML(text string) Widget {
	return Widget{TextParagraph: &TextParagraph{Text: text}}
}

func escapeText(text string) string {
	return strings.ReplaceAll(html.EscapeString(text), "&#34;", "&quot;")
}

func levelEmoji(level webhook.Level) string {
	switch level {
	case webhook.WebhookSuccess:
		return "✅ "
	case webhook.WebhookWarn:
		return "⚠️ "
	case webhook.WebhookError:
		return "❗ "
	default:
		return ""
	}
}

func postMessage(context webhook.Context) error {
	redactedURL := redactWebhookURL(context.URL)
	post := BuildMessage(context)
	body, err := marshal(post)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal webhook POST request to %s", redactedURL)
	}
	req, err := http.NewRequest(http.MethodPost, context.URL, bytes.NewBuffer(body))
	if err != nil {
		return errors.Errorf("failed to construct webhook POST request to %s: %s", redactedURL, redactWebhookErrorMessage(err, context.URL, redactedURL))
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	client := &http.Client{
		Timeout: webhook.Timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Errorf("failed to POST webhook to %s: %s", redactedURL, redactWebhookErrorMessage(err, context.URL, redactedURL))
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Errorf("failed to read POST webhook response from %s: %s", redactedURL, redactWebhookErrorMessage(err, context.URL, redactedURL))
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return errors.Errorf("failed to POST webhook to %s, status code: %d, response body: %s", redactedURL, resp.StatusCode, b)
	}

	return nil
}

func redactWebhookErrorMessage(err error, rawURL, redactedURL string) string {
	message := err.Error()
	message = strings.ReplaceAll(message, rawURL, redactedURL)
	u, parseErr := url.Parse(rawURL)
	if parseErr != nil {
		return message
	}
	query := u.Query()
	for _, key := range []string{"key", "token"} {
		for _, value := range query[key] {
			if value != "" {
				message = strings.ReplaceAll(message, value, "REDACTED")
			}
		}
	}
	return message
}

func redactWebhookURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := u.Query()
	for _, key := range []string{"key", "token"} {
		if _, ok := query[key]; ok {
			query.Set(key, "REDACTED")
		}
	}
	u.RawQuery = query.Encode()
	return u.String()
}

func marshal(post MessagePayload) ([]byte, error) {
	var body bytes.Buffer
	encoder := json.NewEncoder(&body)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(post); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(body.Bytes()), nil
}
