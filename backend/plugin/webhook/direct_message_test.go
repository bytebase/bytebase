package webhook

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestURLSupportsDirectMessage pins the fact the console can no longer work out
// for a saved webhook, since reads stopped returning the URL. Getting it wrong
// in the false direction offers an option that diverts the customer's
// notifications away from the Power Automate flow they built, because
// teams.Post sends the direct messages and returns before it ever posts.
func TestURLSupportsDirectMessage(t *testing.T) {
	for _, tc := range []struct {
		name        string
		webhookType storepb.WebhookType
		url         string
		want        bool
	}{
		{
			name:        "a Power Automate workflow endpoint",
			webhookType: storepb.WebhookType_TEAMS,
			url:         "https://default1234.aa.environment.api.powerplatform.com/powerautomate/automations/direct/workflows/abc/triggers/manual/paths/invoke",
			want:        false,
		},
		{
			name:        "and the legacy logic.azure.com spelling of one",
			webhookType: storepb.WebhookType_TEAMS,
			url:         "https://prod-03.westus.logic.azure.com:443/workflows/abc/triggers/manual/paths/invoke?api-version=2016-06-01",
			want:        false,
		},
		{
			// ValidateWebhookURL writes these domains with a leading dot and
			// accepts the apex as well as a subdomain, so a webhook on the
			// apex stores fine. It has to be recognised here too, or the
			// console offers direct messages on a URL that bypasses them.
			name:        "and the apex of one, which the validator also accepts",
			webhookType: storepb.WebhookType_TEAMS,
			url:         "https://powerplatform.com/powerautomate/automations/direct/workflows/abc",
			want:        false,
		},
		{
			name:        "and the apex of the legacy one",
			webhookType: storepb.WebhookType_TEAMS,
			url:         "https://logic.azure.com/workflows/abc/triggers/manual/paths/invoke",
			want:        false,
		},
		{
			name:        "an Office 365 connector URL is the Teams form that does",
			webhookType: storepb.WebhookType_TEAMS,
			url:         "https://example.webhook.office.com/webhookb2/abc/IncomingWebhook/def",
			want:        true,
		},
		{
			// A domain that merely ends with the same text is a different
			// domain: the leading dot is a boundary, not a substring match.
			name:        "a lookalike domain is not one of them",
			webhookType: storepb.WebhookType_TEAMS,
			url:         "https://notpowerplatform.com/webhookb2/abc",
			want:        true,
		},
		{
			// The restriction is Teams-specific. A Slack URL on a
			// powerplatform host is not a thing, and would not change what
			// slack.Post does anyway.
			name:        "no other type has a URL-dependent restriction",
			webhookType: storepb.WebhookType_SLACK,
			url:         "https://hooks.slack.com/services/fixture-not-a-real-token",
			want:        true,
		},
		{
			name:        "an unparseable URL is not assumed to be the special case",
			webhookType: storepb.WebhookType_TEAMS,
			url:         "://not a url",
			want:        true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, URLSupportsDirectMessage(tc.webhookType, tc.url))
		})
	}
}
