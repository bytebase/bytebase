package webhook

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name string

		webhookType storepb.WebhookType
		webhookURL  string
		wantErr     bool
	}{
		// Slack tests
		{
			name:        "valid slack URL",
			webhookType: storepb.WebhookType_SLACK,
			webhookURL:  "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXX",
			wantErr:     false,
		},
		{
			name:        "valid slack-gov URL",
			webhookType: storepb.WebhookType_SLACK,
			webhookURL:  "https://hooks.slack-gov.com/services/T00000000/B00000000/XXXXXXXXXXXX",
			wantErr:     false,
		},
		{
			name:        "invalid slack domain",
			webhookType: storepb.WebhookType_SLACK,
			webhookURL:  "https://evil.com/hooks",
			wantErr:     true,
		},
		{
			name:        "slack SSRF attempt localhost",
			webhookType: storepb.WebhookType_SLACK,
			webhookURL:  "http://127.0.0.1:8080/",
			wantErr:     true,
		},
		{
			name:        "slack SSRF attempt private IP",
			webhookType: storepb.WebhookType_SLACK,
			webhookURL:  "http://192.168.1.1/",
			wantErr:     true,
		},
		// Discord tests
		{
			name:        "valid discord URL",
			webhookType: storepb.WebhookType_DISCORD,
			webhookURL:  "https://discord.com/api/webhooks/123456789/abcdefg",
			wantErr:     false,
		},
		{
			name:        "valid discordapp URL (legacy)",
			webhookType: storepb.WebhookType_DISCORD,
			webhookURL:  "https://discordapp.com/api/webhooks/123456789/abcdefg",
			wantErr:     false,
		},
		{
			name:        "invalid discord domain",
			webhookType: storepb.WebhookType_DISCORD,
			webhookURL:  "https://evil-discord.com/api/webhooks/123456789/abcdefg",
			wantErr:     true,
		},
		// Teams tests
		{
			name:        "valid teams office.com URL",
			webhookType: storepb.WebhookType_TEAMS,
			webhookURL:  "https://outlook.office.com/webhook/xxx",
			wantErr:     false,
		},
		{
			name:        "valid teams office365.com URL",
			webhookType: storepb.WebhookType_TEAMS,
			webhookURL:  "https://outlook.office365.com/webhook/xxx",
			wantErr:     false,
		},
		{
			name:        "valid teams subdomain",
			webhookType: storepb.WebhookType_TEAMS,
			webhookURL:  "https://example.office.com/webhook/xxx",
			wantErr:     false,
		},
		{
			name:        "invalid teams domain",
			webhookType: storepb.WebhookType_TEAMS,
			webhookURL:  "https://evil-office.com/webhook/xxx",
			wantErr:     true,
		},
		// DingTalk tests
		{
			name:        "valid dingtalk oapi URL",
			webhookType: storepb.WebhookType_DINGTALK,
			webhookURL:  "https://oapi.dingtalk.com/robot/send?access_token=xxx",
			wantErr:     false,
		},
		{
			name:        "valid dingtalk api URL",
			webhookType: storepb.WebhookType_DINGTALK,
			webhookURL:  "https://api.dingtalk.com/robot/send?access_token=xxx",
			wantErr:     false,
		},
		{
			name:        "invalid dingtalk domain",
			webhookType: storepb.WebhookType_DINGTALK,
			webhookURL:  "https://evil.dingtalk.com/robot/send",
			wantErr:     true,
		},
		// Feishu tests
		{
			name:        "valid feishu URL",
			webhookType: storepb.WebhookType_FEISHU,
			webhookURL:  "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
			wantErr:     false,
		},
		{
			name:        "invalid feishu domain",
			webhookType: storepb.WebhookType_FEISHU,
			webhookURL:  "https://evil.feishu.cn/open-apis/bot/v2/hook/xxx",
			wantErr:     true,
		},
		// Lark tests
		{
			name:        "valid lark URL",
			webhookType: storepb.WebhookType_LARK,
			webhookURL:  "https://open.larksuite.com/open-apis/bot/v2/hook/xxx",
			wantErr:     false,
		},
		{
			name:        "invalid lark domain",
			webhookType: storepb.WebhookType_LARK,
			webhookURL:  "https://evil.larksuite.com/open-apis/bot/v2/hook/xxx",
			wantErr:     true,
		},
		// WeCom tests
		{
			name:        "valid wecom URL",
			webhookType: storepb.WebhookType_WECOM,
			webhookURL:  "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx",
			wantErr:     false,
		},
		{
			name:        "invalid wecom domain",
			webhookType: storepb.WebhookType_WECOM,
			webhookURL:  "https://evil.weixin.qq.com/cgi-bin/webhook/send?key=xxx",
			wantErr:     true,
		},
		// URL format tests
		{
			name:        "invalid URL format",
			webhookType: storepb.WebhookType_SLACK,
			webhookURL:  "not-a-url",
			wantErr:     true,
		},
		{
			name:        "invalid scheme ftp",
			webhookType: storepb.WebhookType_SLACK,
			webhookURL:  "ftp://hooks.slack.com/services/xxx",
			wantErr:     true,
		},
		{
			name:        "invalid scheme file",
			webhookType: storepb.WebhookType_SLACK,
			webhookURL:  "file:///etc/passwd",
			wantErr:     true,
		},
		// Unknown webhook type
		{
			name:        "unknown webhook type",
			webhookType: storepb.WebhookType_WEBHOOK_TYPE_UNSPECIFIED,
			webhookURL:  "https://example.com/webhook",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.webhookType, tt.webhookURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWebhookURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateWebhookURL_CaseInsensitive(t *testing.T) {
	// Test that domain matching is case-insensitive
	tests := []struct {
		name        string
		webhookType storepb.WebhookType
		webhookURL  string
		wantErr     bool
	}{
		{
			name:        "slack uppercase domain",
			webhookType: storepb.WebhookType_SLACK,
			webhookURL:  "https://HOOKS.SLACK.COM/services/xxx",
			wantErr:     false,
		},
		{
			name:        "slack mixed case domain",
			webhookType: storepb.WebhookType_SLACK,
			webhookURL:  "https://Hooks.Slack.Com/services/xxx",
			wantErr:     false,
		},
		{
			name:        "discord uppercase",
			webhookType: storepb.WebhookType_DISCORD,
			webhookURL:  "https://DISCORD.COM/api/webhooks/123/abc",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.webhookType, tt.webhookURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWebhookURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
