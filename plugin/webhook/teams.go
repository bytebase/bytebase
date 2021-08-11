package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

var THEME_COLOR = "4f46e5"

type TeamsWebhookActionTarget struct {
	OS  string `json:"os"`
	URI string `json:"uri"`
}

type TeamsWebhookAction struct {
	Type       string                     `json:"@type"`
	Name       string                     `json:"name"`
	TargetList []TeamsWebhookActionTarget `json:"targets"`
}

type TeamsWebhookSectionFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TeamsWebhookSection struct {
	ActivityTitle    string                    `json:"activityTitle"`
	ActivitySubtitle string                    `json:"activitySubtitle"`
	FactList         []TeamsWebhookSectionFact `json:"facts"`
	Text             string                    `json:"text"`
}

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

type TeamsReceiver struct {
}

func (receiver *TeamsReceiver) post(context WebhookContext) error {
	factList := []TeamsWebhookSectionFact{}
	for _, meta := range context.MetaList {
		factList = append(factList, TeamsWebhookSectionFact(meta))
	}

	post := TeamsWebhook{
		Type:       "MessageCard",
		Context:    "https://schema.org/extensions",
		Summary:    context.Title,
		ThemeColor: THEME_COLOR,
		Title:      context.Title,
		SectionList: []TeamsWebhookSection{
			{
				ActivityTitle:    fmt.Sprintf("%s (%s)", context.CreatorName, context.CreatorEmail),
				ActivitySubtitle: time.Unix(context.CreatedTs, 0).Format(timeFormat),
				FactList:         factList,
				Text:             context.Description,
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
		return fmt.Errorf("failed to marshal webhook POST request: %v", context.URL)
	}
	req, err := http.NewRequest("POST",
		context.URL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to construct webhook POST request %v (%w)", context.URL, err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST webhook %+v (%w)", context.URL, err)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read POST webhook response %v (%w)", context.URL, err)
	}
	defer resp.Body.Close()

	if string(b) != "1" {
		return fmt.Errorf("%s", string(b))
	}

	return nil
}
