package webhook

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

var (
	// allowedDomains maps webhook types to their allowed domains.
	allowedDomains = map[storepb.WebhookType][]string{
		storepb.WebhookType_SLACK: {
			"hooks.slack.com",
			"hooks.slack-gov.com",
		},
		storepb.WebhookType_DISCORD: {
			"discord.com",
			"discordapp.com",
		},
		storepb.WebhookType_TEAMS: {
			// Power Automate Workflows (recommended):
			// https://learn.microsoft.com/en-us/power-automate/how-tos-bulk-update-trigger-urls
			".powerplatform.com",
			// Legacy Power Automate URLs (deprecated, retired Nov 2025):
			// https://devblogs.microsoft.com/microsoft365dev/retirement-of-office-365-connectors-within-microsoft-teams/
			".logic.azure.com",
			// Legacy Office 365 Connectors (deprecated, retiring Mar 2026):
			// https://devblogs.microsoft.com/microsoft365dev/retirement-of-office-365-connectors-within-microsoft-teams/
			".office.com",
			".office365.com",
		},
		storepb.WebhookType_DINGTALK: {
			"oapi.dingtalk.com",
			"api.dingtalk.com",
		},
		storepb.WebhookType_FEISHU: {
			"open.feishu.cn",
		},
		storepb.WebhookType_LARK: {
			"open.larksuite.com",
		},
		storepb.WebhookType_WECOM: {
			"qyapi.weixin.qq.com",
		},
		storepb.WebhookType_GOOGLE_CHAT: {
			"chat.googleapis.com",
		},
	}

	// TestOnlyAllowedDomains contains additional domains allowed for testing purposes only.
	// This should only be modified in test files.
	TestOnlyAllowedDomains = map[storepb.WebhookType][]string{}
)

// ValidateWebhookURL validates that the webhook URL matches the allowed domains for the webhook type.
func ValidateWebhookURL(webhookType storepb.WebhookType, webhookURL string) error {
	// Parse URL
	u, err := url.Parse(webhookURL)
	if err != nil {
		// Do not include the parser error because it may contain the raw URL and be persisted in the audit status.
		return errors.New("invalid URL format")
	}

	// Only allow http/https
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.Errorf("invalid URL scheme: %s (only http and https are allowed)", u.Scheme)
	}

	// Get allowed domains for this webhook type
	allowedDomainsForType, ok := allowedDomains[webhookType]
	if !ok {
		return errors.Errorf("unknown webhook type: %s", webhookType)
	}

	// Merge with test-only allowed domains
	allAllowedDomains := append([]string{}, allowedDomainsForType...)
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
				if webhookType == storepb.WebhookType_GOOGLE_CHAT {
					return validateGoogleChatURL(u)
				}
				return nil
			}
		}
	}

	return errors.Errorf("webhook URL domain %q is not allowed for webhook type %s (allowed domains: %v)",
		hostname, webhookType, allowedDomainsForType)
}

func validateGoogleChatURL(u *url.URL) error {
	if u.Scheme != "https" {
		return errors.Errorf("invalid Google Chat URL scheme: %s (only https is allowed)", u.Scheme)
	}

	parts := strings.Split(u.Path, "/")
	if len(parts) != 5 || parts[1] != "v1" || parts[2] != "spaces" || parts[3] == "" || parts[4] != "messages" {
		return errors.Errorf("invalid Google Chat webhook path: %s", u.Path)
	}

	query := u.Query()
	if query.Get("key") == "" {
		return errors.Errorf("missing Google Chat webhook key")
	}
	if query.Get("token") == "" {
		return errors.Errorf("missing Google Chat webhook token")
	}

	return nil
}

// URLSupportsDirectMessage reports whether a webhook URL's endpoint form can
// carry a direct message to the users an event mentions, rather than only a
// post to the channel the URL names.
//
// The one form that cannot is a Microsoft Teams Power Automate workflow
// endpoint. teams.Post routes on the same fact at delivery time, and it decides
// the question before the URL does: a webhook with direct messages enabled and
// mentioned users sends them and returns, so the workflow post never happens.
// Enabling it on a Power Automate webhook therefore diverts the customer's
// notifications away from the flow they built, which is why the console hides
// the option for those URLs.
//
// The console cannot work this out for a saved webhook any more, because reads
// no longer return the URL. Webhook.url_supports_direct_message carries the
// answer instead, and this is where both callers get it from.
func URLSupportsDirectMessage(webhookType storepb.WebhookType, webhookURL string) bool {
	if webhookType != storepb.WebhookType_TEAMS {
		return true
	}
	return !IsPowerAutomateURL(webhookURL)
}

// powerAutomateDomains are the Teams destinations that take the Power Automate
// workflow payload rather than the Office 365 connector one. They are a subset
// of allowedDomains[TEAMS] above and are written in the same notation, so the
// same matching rule applies to both: a leading dot means the domain itself and
// any subdomain of it.
//
// Sharing the notation is the point. This list used to be a pair of
// strings.HasSuffix calls that accepted a subdomain and not the apex, while
// ValidateWebhookURL accepted both — so a webhook on powerplatform.com itself
// stored fine and then was not recognised as Power Automate.
var powerAutomateDomains = []string{
	".powerplatform.com",
	// Legacy, retired November 2025.
	".logic.azure.com",
}

// hostMatchesDomain applies the allowed-domain notation to a hostname: a
// leading dot matches the domain itself and any subdomain of it, and anything
// else matches exactly. Both the hostname and the domain are compared in lower
// case, which is the caller's job for the hostname.
func hostMatchesDomain(hostname, domain string) bool {
	domain = strings.ToLower(domain)
	if after, ok := strings.CutPrefix(domain, "."); ok {
		return hostname == after || strings.HasSuffix(hostname, domain)
	}
	return hostname == domain
}

// IsPowerAutomateURL reports whether the URL is a Power Automate workflow
// endpoint rather than an Office 365 connector URL. The two take different
// payload formats, and only the connector one carries a direct message.
//
// ProjectWebhookForm.tsx carries a copy of this rule, because the console has
// to answer it for a URL the server has never seen: everything on the create
// form, and anything typed over a saved URL. This is the authority, and the two
// have to agree.
func IsPowerAutomateURL(webhookURL string) bool {
	u, err := url.Parse(webhookURL)
	if err != nil {
		return false
	}
	hostname := strings.ToLower(u.Hostname())
	for _, domain := range powerAutomateDomains {
		if hostMatchesDomain(hostname, domain) {
			return true
		}
	}
	return false
}
