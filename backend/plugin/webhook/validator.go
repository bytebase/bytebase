package webhook

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

var (
	// AllowedDomains maps webhook types to their allowed domains.
	AllowedDomains = map[string][]string{
		"bb.plugin.webhook.slack": {
			"hooks.slack.com",
			"hooks.slack-gov.com",
		},
		"bb.plugin.webhook.discord": {
			"discord.com",
			"discordapp.com",
		},
		"bb.plugin.webhook.teams": {
			".office.com",    // Matches *.office.com
			".office365.com", // Matches *.office365.com
		},
		"bb.plugin.webhook.dingtalk": {
			"oapi.dingtalk.com",
			"api.dingtalk.com",
		},
		"bb.plugin.webhook.feishu": {
			"open.feishu.cn",
		},
		"bb.plugin.webhook.lark": {
			"open.larksuite.com",
		},
		"bb.plugin.webhook.wecom": {
			"qyapi.weixin.qq.com",
		},
	}

	// TestOnlyAllowedDomains contains additional domains allowed for testing purposes only.
	// This should only be modified in test files.
	TestOnlyAllowedDomains = map[string][]string{}
)

// ValidateWebhookURL validates that the webhook URL matches the allowed domains for the webhook type.
func ValidateWebhookURL(webhookType, webhookURL string) error {
	// Parse URL
	u, err := url.Parse(webhookURL)
	if err != nil {
		return errors.Wrapf(err, "invalid URL format")
	}

	// Only allow http/https
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.Errorf("invalid URL scheme: %s (only http and https are allowed)", u.Scheme)
	}

	// Get allowed domains for this webhook type
	allowedDomains, ok := AllowedDomains[webhookType]
	if !ok {
		return errors.Errorf("unknown webhook type: %s", webhookType)
	}

	// Merge with test-only allowed domains
	allAllowedDomains := append([]string{}, allowedDomains...)
	if testDomains, exists := TestOnlyAllowedDomains[webhookType]; exists {
		allAllowedDomains = append(allAllowedDomains, testDomains...)
	}

	// Check if hostname matches any allowed domain
	hostname := strings.ToLower(u.Hostname())
	for _, domain := range allAllowedDomains {
		domain = strings.ToLower(domain)

		// Support wildcard subdomains (e.g., ".office.com" matches "outlook.office.com")
		if strings.HasPrefix(domain, ".") {
			if hostname == domain[1:] || strings.HasSuffix(hostname, domain) {
				return nil
			}
		} else {
			// Exact match
			if hostname == domain {
				return nil
			}
		}
	}

	return errors.Errorf("webhook URL domain %q is not allowed for webhook type %s (allowed domains: %v)",
		hostname, webhookType, allowedDomains)
}
