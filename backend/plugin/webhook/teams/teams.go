// Package teams implements Microsoft Teams webhook and direct message integration.
//
// Documentation:
// - Teams Webhooks (Incoming Webhook): https://learn.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook
// - MessageCard Format: https://learn.microsoft.com/en-us/outlook/actionable-messages/message-card-reference
// - Adaptive Cards: https://learn.microsoft.com/en-us/adaptive-cards/
// - Proactive Messaging: https://learn.microsoft.com/en-us/microsoftteams/platform/bots/how-to/conversations/send-proactive-messages
// - Graph API App Installation: https://learn.microsoft.com/en-us/microsoftteams/platform/graph-api/proactive-bots-and-messages/graph-proactive-bots-and-messages
package teams

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

var themeColor = "4f46e5"

// getTeamsConfig extracts the Teams configuration from the AppIMSetting.
func getTeamsConfig(setting *storepb.AppIMSetting) *storepb.AppIMSetting_Teams {
	if setting == nil {
		return nil
	}
	for _, s := range setting.Settings {
		if s.Type == storepb.WebhookType_TEAMS {
			return s.GetTeams()
		}
	}
	return nil
}

// WebhookActionTarget is the API message for Teams webhook action target.
type WebhookActionTarget struct {
	OS  string `json:"os"`
	URI string `json:"uri"`
}

// WebhookAction is the API message for Teams webhook action.
type WebhookAction struct {
	Type       string                `json:"@type"`
	Name       string                `json:"name"`
	TargetList []WebhookActionTarget `json:"targets"`
}

// WebhookSectionFact is the API message for Teams webhook section fact.
type WebhookSectionFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// WebhookSection is the API message for Teams webhook section.
type WebhookSection struct {
	ActivityTitle    string               `json:"activityTitle"`
	ActivitySubtitle string               `json:"activitySubtitle"`
	FactList         []WebhookSectionFact `json:"facts"`
	Text             string               `json:"text"`
}

// Webhook is the API message for Teams webhook (MessageCard format).
// MessageCard reference: https://learn.microsoft.com/en-us/outlook/actionable-messages/message-card-reference
// MessageCard designer: https://amdesigner.azurewebsites.net/
type Webhook struct {
	Type        string           `json:"@type"`
	Context     string           `json:"@context"`
	Summary     string           `json:"summary"`
	ThemeColor  string           `json:"themeColor"`
	Title       string           `json:"title"`
	SectionList []WebhookSection `json:"sections"`
	ActionList  []WebhookAction  `json:"potentialAction"`
}

func init() {
	webhook.Register(storepb.WebhookType_TEAMS, &Receiver{})
}

// Receiver is the receiver for Teams.
type Receiver struct {
}

func (*Receiver) Post(context webhook.Context) error {
	if context.DirectMessage && len(context.MentionEndUsers) > 0 {
		if postDirectMessage(context) {
			return nil
		}
	}
	return postMessage(context)
}

func postDirectMessage(webhookCtx webhook.Context) bool {
	teams := getTeamsConfig(webhookCtx.IMSetting)
	if teams == nil {
		return false
	}

	p := newProvider(teams.TenantId, teams.ClientId, teams.ClientSecret)
	ctx := context.Background()

	sent := map[string]bool{}

	if err := common.Retry(ctx, func() error {
		var errs error

		var emails []string
		for _, u := range webhookCtx.MentionEndUsers {
			if sent[u.Email] {
				continue
			}
			emails = append(emails, u.Email)
		}

		idByEmail, err := p.getIDByEmail(ctx, emails)
		if err != nil {
			return errors.Wrapf(err, "failed to get id by email")
		}

		for _, u := range webhookCtx.MentionEndUsers {
			if sent[u.Email] {
				continue
			}
			id, ok := idByEmail[u.Email]
			if !ok {
				continue
			}

			err := p.sendMessage(ctx, id, getAdaptiveCard(webhookCtx))
			if err != nil {
				slog.Error("Teams failed to send message",
					slog.String("email", u.Email),
					log.BBError(err))
				err = errors.Wrapf(err, "failed to send message to %s", u.Email)
				multierr.AppendInto(&errs, err)
			}
			sent[u.Email] = true
		}
		return errs
	}); err != nil {
		slog.Warn("failed to send direct message to Teams user", log.BBError(err))
		return false
	}

	return true
}

func postMessage(context webhook.Context) error {
	factList := []WebhookSectionFact{}
	for _, meta := range context.GetMetaList() {
		factList = append(factList, WebhookSectionFact(meta))
	}

	section := WebhookSection{
		FactList: factList,
		Text:     context.Description,
	}
	if context.ActorName != "" {
		section.ActivityTitle = fmt.Sprintf("%s (%s)", context.ActorName, context.ActorEmail)
	}

	post := Webhook{
		Type:       "MessageCard",
		Context:    "https://schema.org/extensions",
		Summary:    context.Title,
		ThemeColor: themeColor,
		Title:      context.Title,
		SectionList: []WebhookSection{
			section,
		},
		ActionList: []WebhookAction{
			{
				Type: "OpenUri",
				Name: "View in Bytebase",
				TargetList: []WebhookActionTarget{
					{
						OS:  "default",
						URI: context.Link,
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

	if string(b) != "1" {
		return errors.Errorf("%.100s", string(b))
	}

	return nil
}

// getAdaptiveCard creates an Adaptive Card for Teams direct messages.
func getAdaptiveCard(context webhook.Context) *AdaptiveCard {
	var body []any

	// Title
	body = append(body, textBlock{
		Type:   "TextBlock",
		Text:   context.Title,
		Size:   "Large",
		Weight: "Bolder",
		Wrap:   true,
	})

	// Description
	if context.Description != "" {
		body = append(body, textBlock{
			Type: "TextBlock",
			Text: context.Description,
			Wrap: true,
		})
	}

	// Facts (metadata)
	var facts []fact
	for _, meta := range context.GetMetaList() {
		facts = append(facts, fact{
			Title: meta.Name,
			Value: meta.Value,
		})
	}
	if context.ActorName != "" {
		facts = append(facts, fact{
			Title: "Actor",
			Value: fmt.Sprintf("%s (%s)", context.ActorName, context.ActorEmail),
		})
	}

	if len(facts) > 0 {
		body = append(body, factSet{
			Type:  "FactSet",
			Facts: facts,
		})
	}

	// Actions
	var actions []any
	actions = append(actions, actionOpenURL{
		Type:  "Action.OpenUrl",
		Title: "View in Bytebase",
		URL:   context.Link,
	})

	return &AdaptiveCard{
		Type:    "AdaptiveCard",
		Version: "1.4",
		Body:    body,
		Actions: actions,
	}
}
