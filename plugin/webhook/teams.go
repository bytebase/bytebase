package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

var themeColor = "4f46e5"

// TeamsWebhookActionTarget is the API message for Teams webhook action target.
type TeamsWebhookActionTarget struct {
	OS  string `json:"os"`
	URI string `json:"uri"`
}

// TeamsWebhookAction is the API message for Teams webhook action.
type TeamsWebhookAction struct {
	Type       string                     `json:"@type"`
	Name       string                     `json:"name"`
	TargetList []TeamsWebhookActionTarget `json:"targets"`
}

// TeamsWebhookSectionFact is the API message for Teams webhook section fact.
type TeamsWebhookSectionFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TeamsWebhookSection is the API message for Teams webhook section.
type TeamsWebhookSection struct {
	ActivityTitle    string                    `json:"activityTitle"`
	ActivitySubtitle string                    `json:"activitySubtitle"`
	FactList         []TeamsWebhookSectionFact `json:"facts"`
	Text             string                    `json:"text"`
}

// TeamsWebhook is the API message for Teams webhook.
type TeamsWebhook struct {
	Type        string                `json:"@type"`
	Context     string                `json:"@context"`
	Summary     string                `json:"summary"`
	ThemeColor  string                `json:"themeColor"`
	Title       string                `json:"title"`
	SectionList []TeamsWebhookSection `json:"sections"`
	ActionList  []TeamsWebhookAction  `json:"potentialAction"`
}

func init() {
	register("bb.plugin.webhook.teams", &TeamsReceiver{})
}

// TeamsReceiver is the receiver for Teams.
type TeamsReceiver struct{}

func (*TeamsReceiver) post(context Context) error {
	// TODO(p0ny): handle context.Task
	factList := []TeamsWebhookSectionFact{}
	for _, meta := range context.getMetaList() {
		factList = append(factList, TeamsWebhookSectionFact(meta))
	}

	post := TeamsWebhook{
		Type:       "MessageCard",
		Context:    "https://schema.org/extensions",
		Summary:    context.Title,
		ThemeColor: themeColor,
		Title:      context.Title,
		SectionList: []TeamsWebhookSection{
			{
				ActivityTitle: fmt.Sprintf("%s (%s)", context.CreatorName, context.CreatorEmail),
				FactList:      factList,
				Text:          context.Description,
			},
		},
		ActionList: []TeamsWebhookAction{
			{
				Type: "OpenUri",
				Name: "View in Bytebase",
				TargetList: []TeamsWebhookActionTarget{
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

	if string(b) != "1" {
		return errors.Errorf("%.100s", string(b))
	}

	return nil
}
