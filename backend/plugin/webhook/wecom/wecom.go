package wecom

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

// WebhookResponse is the API message for WeCom webhook response.
type WebhookResponse struct {
	ErrorCode    int    `json:"errcode"`
	ErrorMessage string `json:"errmsg"`
}

// WebhookMarkdown is the API message for WeCom webhook markdown.
type WebhookMarkdown struct {
	Content string `json:"content"`
}

// Webhook is the API message for WeCom webhook.
type Webhook struct {
	MessageType string           `json:"msgtype"`
	Markdown    *WebhookMarkdown `json:"markdown"`
}

func init() {
	webhook.Register("bb.plugin.webhook.wecom", &Receiver{})
}

// Receiver is the receiver for WeCom.
type Receiver struct {
}

func getMessageCard(context webhook.Context) *WebhookMarkdown {
	metaStrList := []string{}
	for _, meta := range context.GetMetaList() {
		metaStrList = append(metaStrList, fmt.Sprintf("%s: <font color=\"comment\">%s</font>", meta.Name, meta.Value))
	}
	metaStrList = append(metaStrList, fmt.Sprintf("By: <font color=\"comment\">%s (%s)</font>", context.CreatorName, context.CreatorEmail))

	status := ""
	switch context.Level {
	case webhook.WebhookSuccess:
		status = "<font color=\"green\">Success</font> "
	case webhook.WebhookWarn:
		status = "<font color=\"yellow\">Warn</font> "
	case webhook.WebhookError:
		status = "<font color=\"red\">Error</font> "
	}
	content := fmt.Sprintf("# %s%s\n\n%s\n[View in Bytebase](%s)", status, context.Title, strings.Join(metaStrList, "\n"), context.Link)
	if context.Description != "" {
		content = fmt.Sprintf("# %s%s\n> %s\n\n%s\n[View in Bytebase](%s)", status, context.Title, context.Description, strings.Join(metaStrList, "\n"), context.Link)
	}
	return &WebhookMarkdown{
		Content: content,
	}
}

func (r *Receiver) Post(context webhook.Context) error {
	if context.DirectMessage && len(context.MentionUsers) > 0 {
		r.sendDirectMessage(context)
		return nil
	}
	return r.sendMessage(context)
}

func (*Receiver) sendMessage(context webhook.Context) error {
	post := Webhook{
		MessageType: "markdown",
		Markdown:    getMessageCard(context),
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

	webhookResponse := &WebhookResponse{}
	if err := json.Unmarshal(b, webhookResponse); err != nil {
		return errors.Wrapf(err, "malformed webhook response from %s", context.URL)
	}

	if webhookResponse.ErrorCode != 0 {
		return errors.Errorf("%s", webhookResponse.ErrorMessage)
	}

	return nil
}

func (*Receiver) sendDirectMessage(webhookCtx webhook.Context) {
	wecom := webhookCtx.IMSetting.GetWecom()
	if wecom == nil {
		return
	}
	p, err := newProvider(wecom.GetCorpId(), wecom.GetAgentId(), wecom.GetSecret())
	if err != nil {
		slog.Error("failed to get wecom provider", log.BBError(err))
	}

	ctx := context.Background()

	sent := map[string]bool{}

	if err := common.Retry(ctx, func() error {
		var errs error
		var users, userEmails []string

		for _, u := range webhookCtx.MentionUsers {
			if sent[u.Email] {
				continue
			}

			err := func() error {
				userID, err := p.getUserIDByEmail(ctx, u.Email)
				if err != nil {
					if strings.Contains(err.Error(), "errcode 46004") {
						return nil
					}
					return errors.Wrapf(err, "failed to get user id by email %v", u.Email)
				}
				users = append(users, userID)
				userEmails = append(userEmails, u.Email)

				return nil
			}()

			multierr.AppendInto(&errs, err)
		}
		if len(users) == 0 {
			return nil
		}

		if err := p.sendMessage(ctx, users, getMessageCard(webhookCtx)); err != nil {
			err = errors.Wrapf(err, "failed to send message")
			multierr.AppendInto(&errs, err)
		} else {
			for _, email := range userEmails {
				sent[email] = true
			}
		}

		return errs
	}); err != nil {
		slog.Warn("failed to send direct message to wecom users", log.BBError(err))
	}
}
