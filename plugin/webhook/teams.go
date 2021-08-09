package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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
	FactList []TeamsWebhookSectionFact `json:"facts"`
	Text     string                    `json:"text"`
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

func (receiver *TeamsReceiver) post(url string, title string, description string, metaList []WebHookMeta, link string) error {
	factList := []TeamsWebhookSectionFact{}
	for _, meta := range metaList {
		factList = append(factList, TeamsWebhookSectionFact(meta))
	}

	post := TeamsWebhook{
		Type:       "MessageCard",
		Context:    "https://schema.org/extensions",
		Summary:    title,
		ThemeColor: THEME_COLOR,
		Title:      title,
		SectionList: []TeamsWebhookSection{
			{
				FactList: factList,
				Text:     description,
			},
		},
		ActionList: []TeamsWebhookAction{
			{
				Type: "OpenUri",
				Name: "View in Bytebase",
				TargetList: []TeamsWebhookActionTarget{
					{
						OS:  "default",
						URI: link,
					},
				},
			},
		},
	}
	body, err := json.Marshal(post)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook POST request: %v", url)
	}
	req, err := http.NewRequest("POST",
		url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to construct webhook POST request %v (%w)", url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: timeout,
	}
	_, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST webhook %+v (%w)", url, err)
	}

	return nil
}
