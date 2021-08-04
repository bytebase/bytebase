package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

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
	register("open.feishu.cn", &FeishuReceiver{})
}

type FeishuReceiver struct {
}

func (receiver *FeishuReceiver) post(url string, title string, description string, metaList []WebHookMeta, link string) error {
	contentList := [][]FeishuWebhookPostSection{}
	if description != "" {
		sectionList := []FeishuWebhookPostSection{}
		sectionList = append(sectionList, FeishuWebhookPostSection{
			Tag:  "text",
			Text: description,
		})
		contentList = append(contentList, sectionList)

		sectionList = []FeishuWebhookPostSection{}
		sectionList = append(sectionList, FeishuWebhookPostSection{
			Tag:  "text",
			Text: "",
		})
		contentList = append(contentList, sectionList)
	}

	for _, meta := range metaList {
		sectionList := []FeishuWebhookPostSection{}
		sectionList = append(sectionList, FeishuWebhookPostSection{
			Tag:  "text",
			Text: fmt.Sprintf("%s: %s", meta.Name, meta.Value),
		})
		contentList = append(contentList, sectionList)
	}

	sectionList := []FeishuWebhookPostSection{}
	sectionList = append(sectionList, FeishuWebhookPostSection{
		Tag:  "a",
		Text: "View in Bytebase",
		Href: link,
	})
	contentList = append(contentList, sectionList)

	post := FeishuWebhook{
		MessageType: "post",
		Content: FeishuWebhookContent{
			Post: FeishuWebhookPostLanguage{
				English: FeishuWebhookPost{
					Title:       title,
					ContentList: contentList,
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
