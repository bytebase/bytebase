package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type FeishuWebhookResponse struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

type FeishuWebhookPostSection struct {
	Tag  string `json:"tag"`
	Text string `json:"text"`
	Href string `json:"href,omitempty"`
}

type FeishuWebhookPostLine struct {
	SectionList []FeishuWebhookPostSection `json:""`
}

type FeishuWebhookPost struct {
	Title       string                       `json:"title"`
	ContentList [][]FeishuWebhookPostSection `json:"content"`
}

type FeishuWebhookPostLanguage struct {
	English FeishuWebhookPost `json:"en_us"`
}

type FeishuWebhookContent struct {
	Post FeishuWebhookPostLanguage `json:"post"`
}

type FeishuWebhook struct {
	MessageType string               `json:"msg_type"`
	Content     FeishuWebhookContent `json:"content"`
}

func init() {
	register("bb.plugin.webhook.feishu", &FeishuReceiver{})
}

type FeishuReceiver struct {
}

func (receiver *FeishuReceiver) post(context WebhookContext) error {
	contentList := [][]FeishuWebhookPostSection{}
	if context.Description != "" {
		sectionList := []FeishuWebhookPostSection{}
		sectionList = append(sectionList, FeishuWebhookPostSection{
			Tag:  "text",
			Text: context.Description,
		})
		contentList = append(contentList, sectionList)

		sectionList = []FeishuWebhookPostSection{}
		sectionList = append(sectionList, FeishuWebhookPostSection{
			Tag:  "text",
			Text: "",
		})
		contentList = append(contentList, sectionList)
	}

	for _, meta := range context.MetaList {
		sectionList := []FeishuWebhookPostSection{}
		sectionList = append(sectionList, FeishuWebhookPostSection{
			Tag:  "text",
			Text: fmt.Sprintf("%s: %s", meta.Name, meta.Value),
		})
		contentList = append(contentList, sectionList)
	}

	{
		sectionList := []FeishuWebhookPostSection{}
		sectionList = append(sectionList, FeishuWebhookPostSection{
			Tag:  "text",
			Text: fmt.Sprintf("By: %s (%s)", context.CreatorName, context.CreatorEmail),
		})
		contentList = append(contentList, sectionList)
	}

	{
		sectionList := []FeishuWebhookPostSection{}
		sectionList = append(sectionList, FeishuWebhookPostSection{
			Tag:  "text",
			Text: fmt.Sprintf("At: %s", time.Unix(context.CreatedTs, 0).Format(timeFormat)),
		})
		contentList = append(contentList, sectionList)
	}

	{
		sectionList := []FeishuWebhookPostSection{}
		sectionList = append(sectionList, FeishuWebhookPostSection{
			Tag:  "a",
			Text: "View in Bytebase",
			Href: context.Link,
		})
		contentList = append(contentList, sectionList)
	}

	post := FeishuWebhook{
		MessageType: "post",
		Content: FeishuWebhookContent{
			Post: FeishuWebhookPostLanguage{
				English: FeishuWebhookPost{
					Title:       context.Title,
					ContentList: contentList,
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

	webhookResponse := &FeishuWebhookResponse{}
	if err := json.Unmarshal(b, webhookResponse); err != nil {
		return fmt.Errorf("malformatted webhook response %v (%w)", context.URL, err)
	}

	if webhookResponse.Code != 0 {
		return fmt.Errorf("%s", webhookResponse.Message)
	}

	return nil
}
