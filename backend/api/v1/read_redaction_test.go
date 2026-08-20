package v1

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// TestRedactRoleAttributeMasksTheCredential walks the grant text a MySQL-family
// server actually returns. Every input here was read off a live server rather
// than written from the manual: MariaDB 11.4.12 for the leaking forms, MySQL
// 8.0.46 and 5.7.44 for the ones that carry no credential, and the remaining
// engines' attributes are what their own role.go builds.
//
// The forms matter more than the count. MariaDB has four ways of putting a
// credential on a grant line and one way of naming an auth plugin without one,
// and a mask that handles the first form and not the fourth reads as fixed
// while still handing over a password hash.
func TestRedactRoleAttributeMasksTheCredential(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{
			name: "mariadb native password hash, the common case",
			in:   "GRANT SELECT ON *.* TO `leaky`@`%` IDENTIFIED BY PASSWORD '*05A6C9A570CCFE4FA5048183ECB511DA6E6414C9'",
			want: "GRANT SELECT ON *.* TO `leaky`@`%` IDENTIFIED BY PASSWORD '<redacted>'",
		},
		{
			name: "the clauses after the credential survive it",
			in:   "GRANT ALL PRIVILEGES ON *.* TO `root`@`%` IDENTIFIED BY PASSWORD '*3D3B92F242033365AE5BC6A8E6FC3E1679F4140A' WITH GRANT OPTION",
			want: "GRANT ALL PRIVILEGES ON *.* TO `root`@`%` IDENTIFIED BY PASSWORD '<redacted>' WITH GRANT OPTION",
		},
		{
			name: "REQUIRE SSL survives too",
			in:   "GRANT USAGE ON *.* TO `sslguy`@`%` IDENTIFIED BY PASSWORD '*B69027D44F6E5EDC07F1AEAD1477967B16F28227' REQUIRE SSL",
			want: "GRANT USAGE ON *.* TO `sslguy`@`%` IDENTIFIED BY PASSWORD '<redacted>' REQUIRE SSL",
		},
		{
			name: "mariadb pluggable auth, a different keyword for the same secret",
			in:   "GRANT USAGE ON *.* TO `edguy`@`%` IDENTIFIED VIA ed25519 USING 'ZIgUREUg5PVgQ6LskhXmO+eZLS0nC8be6HPjYWR4YJY'",
			want: "GRANT USAGE ON *.* TO `edguy`@`%` IDENTIFIED VIA ed25519 USING '<redacted>'",
		},
		{
			name: "two auth methods on one line, and neither may survive",
			in:   "GRANT USAGE ON *.* TO `multi`@`%` IDENTIFIED VIA ed25519 USING 'GvRmi9ungFjJD9sKjaq/T3CL1LmO2CLpz5I42gnB7Eg' OR mysql_native_password USING '*F33AE6DD04EF4C7C1D3105568E7FB7C1EE16C937'",
			want: "GRANT USAGE ON *.* TO `multi`@`%` IDENTIFIED VIA ed25519 USING '<redacted>'",
		},
		{
			// MariaDB does not escape a quote inside the stored auth string:
			// an auth string of ab'cd\\ef comes back exactly like this, which
			// is not re-parseable SQL. Matching to the first inner quote would
			// return the rest of the secret.
			name: "an unescaped quote inside the secret does not end the mask",
			in:   "GRANT USAGE ON *.* TO `quoteguy`@`%` IDENTIFIED VIA ed25519 USING 'ab'cd\\\\ef'",
			want: "GRANT USAGE ON *.* TO `quoteguy`@`%` IDENTIFIED VIA ed25519 USING '<redacted>'",
		},
		{
			name: "an auth plugin that stores no secret keeps its name",
			in:   "GRANT USAGE ON *.* TO `sockguy`@`%` IDENTIFIED VIA unix_socket",
			want: "GRANT USAGE ON *.* TO `sockguy`@`%` IDENTIFIED VIA unix_socket",
		},
		{
			name: "the grant lines that carry no credential are untouched",
			in:   "GRANT `devrole` TO `leaky`@`%`\nGRANT SELECT ON *.* TO `leaky`@`%` IDENTIFIED BY PASSWORD '*05A6C9A570CCFE4FA5048183ECB511DA6E6414C9'",
			want: "GRANT `devrole` TO `leaky`@`%`\nGRANT SELECT ON *.* TO `leaky`@`%` IDENTIFIED BY PASSWORD '<redacted>'",
		},
		{
			name: "mysql 8.0 says nothing about the password",
			in:   "GRANT SELECT ON *.* TO `leaky`@`%`",
			want: "GRANT SELECT ON *.* TO `leaky`@`%`",
		},
		{
			name: "mysql 5.7.44 does not either",
			in:   "GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' WITH GRANT OPTION",
			want: "GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' WITH GRANT OPTION",
		},
		{
			name: "postgres stores role keywords, not grant text",
			in:   "Superuser Create role Create DB Password valid until 2027-01-01 00:00:00+00",
			want: "Superuser Create role Create DB Password valid until 2027-01-01 00:00:00+00",
		},
		{
			name: "snowflake stores grants with no auth clause",
			in:   "GRANT ROLE \"ANALYST\" TO USER \"ALICE\", GRANT ROLE \"LOADER\" TO USER \"ALICE\"",
			want: "GRANT ROLE \"ANALYST\" TO USER \"ALICE\", GRANT ROLE \"LOADER\" TO USER \"ALICE\"",
		},
		{
			name: "elasticsearch stores a role list",
			in:   "[superuser, kibana_admin]",
			want: "[superuser, kibana_admin]",
		},
		{
			name: "empty stays empty",
			in:   "",
			want: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, redactRoleAttribute(tc.in))
		})
	}
}

// TestConvertInstanceRolesRedactsOnTheReadPath holds the redaction at the
// converter both instance reads and ListInstanceRoles go through, rather than
// only at the function they call. It also pins that the stored role is left
// alone: the response is a copy, and a converter that masked in place would
// corrupt the cached instance for everything else reading it.
func TestConvertInstanceRolesRedactsOnTheReadPath(t *testing.T) {
	leaking := "GRANT ALL PRIVILEGES ON *.* TO `root`@`%` IDENTIFIED BY PASSWORD '*3D3B92F242033365AE5BC6A8E6FC3E1679F4140A' WITH GRANT OPTION"
	stored := []*storepb.InstanceRole{
		{Name: "root", Attribute: &leaking},
		{Name: "nameless"},
	}
	roles := convertInstanceRoles(&store.InstanceMessage{ResourceID: "mariadb"}, stored)

	require.Len(t, roles, 2)
	require.Equal(t, "instances/mariadb/roles/root", roles[0].Name)
	require.NotContains(t, roles[0].GetAttribute(), "3D3B92F242033365AE5BC6A8E6FC3E1679F4140A",
		"a read must not return the account's password hash")
	require.Contains(t, roles[0].GetAttribute(), "WITH GRANT OPTION",
		"the grants a DBA reads have to survive the mask")
	require.Nil(t, roles[1].Attribute, "a role with no attribute stays that way rather than gaining an empty one")
	require.Equal(t, leaking, *stored[0].Attribute, "the stored role must not be masked in place")
}

// TestConvertToProjectWithholdsTheWebhookURL pins the other converter. The four
// project reads share it, so one assertion covers GetProject, ListProjects,
// BatchGetProjects and SearchProjects; the write RPCs return a project through
// the same converter, which is correct, because the caller of AddWebhook is the
// one who supplied the URL and does not need it read back.
//
// url_set is what a client reads instead, so it can say a webhook is configured
// without being told what to. The rest of the webhook stays: type, title,
// events and the direct-message flag are configuration, not a credential, and a
// client still has to be able to list what is configured and edit it.
func TestConvertToProjectWithholdsTheWebhookURL(t *testing.T) {
	const storedURL = "https://hooks.slack.com/services/fixture-not-a-real-token"
	project := convertToProject(&store.ProjectMessage{
		ResourceID: "chat",
		Setting:    &storepb.Project{},
		Webhooks: []*store.ProjectWebhookMessage{
			{
				ResourceID: "1",
				Payload: &storepb.ProjectWebhook{
					Type:       storepb.WebhookType_SLACK,
					Title:      "release channel",
					Url:        storedURL,
					Activities: []storepb.Activity_Type{storepb.Activity_ISSUE_CREATED},
				},
			},
			{
				ResourceID: "2",
				Payload:    &storepb.ProjectWebhook{Type: storepb.WebhookType_SLACK, Title: "no url yet"},
			},
			{
				// The one URL form direct messages bypass. The console hides
				// the option for it and cannot see the URL to know that.
				ResourceID: "3",
				Payload: &storepb.ProjectWebhook{
					Type:  storepb.WebhookType_TEAMS,
					Title: "power automate flow",
					Url:   "https://default1234.aa.environment.api.powerplatform.com/powerautomate/automations/direct/workflows/abc/triggers/manual/paths/invoke",
				},
			},
		},
	})

	require.Len(t, project.Webhooks, 3)
	require.Empty(t, project.Webhooks[0].Url,
		"the incoming-webhook URL is the credential; a read must not return it")
	require.True(t, project.Webhooks[0].UrlSet,
		"a configured webhook still reads back as configured")
	require.True(t, project.Webhooks[0].UrlSupportsDirectMessage,
		"and the console still learns what it can no longer work out from the URL")
	require.Equal(t, "release channel", project.Webhooks[0].Title)
	require.Equal(t, []v1pb.Activity_Type{v1pb.Activity_ISSUE_CREATED}, project.Webhooks[0].NotificationTypes)
	require.False(t, project.Webhooks[1].UrlSet,
		"a webhook with no URL must not read back as one that has one")
	require.False(t, project.Webhooks[2].UrlSupportsDirectMessage,
		"a Power Automate workflow endpoint is the URL direct messages bypass")
}

// TestStoredWebhookDeliveryFailure pins the other exit a webhook URL could
// leave by. Every poster wraps its failure with the URL it posted to, and
// TestWebhook hands that message to a caller the read path withholds the URL
// from, so a deliberately failing test would otherwise read it straight back.
//
// The inputs below are what a real client produced against a real server, and
// they are why the summary is built from what is recognised rather than by
// removing the URL: the last two carry the URL in a form with no scheme and no
// host, which no amount of removing URL-shaped text would have caught.
func TestStoredWebhookDeliveryFailure(t *testing.T) {
	const storedURL = "https://hooks.example.com/services/T00/B00/SUPERSECRETTOKEN"
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{
			name: "a status code is what the caller can act on",
			err:  errors.Errorf("failed to POST webhook to %s, status code: 404, response body: no_service", storedURL),
			want: "the stored webhook URL answered with status code 404",
		},
		{
			// The vendor sometimes echoes the request path into the body, so
			// the body does not come back even when a status code does.
			name: "and the response body does not come with it",
			err:  errors.Errorf("failed to POST webhook to %s, status code: 404, response body: no_service for /services/T00/B00/SUPERSECRETTOKEN", storedURL),
			want: "the stored webhook URL answered with status code 404",
		},
		{
			name: "a transport failure has no status to report",
			err:  errors.Errorf("failed to POST webhook to %s: Post %q: dial tcp 127.0.0.1:1: connect: connection refused", storedURL, storedURL),
			want: "the test notification could not be delivered to the stored webhook URL",
		},
		{
			// net/http prints a scheme-less, host-less URL when the host
			// answers a redirect with a relative Location, and for Slack,
			// Discord and Feishu that path is the whole credential.
			name: "a redirect loop names the path with no scheme or host",
			err:  errors.Errorf(`failed to POST webhook to %s: Post "/services/T00/B00/SUPERSECRETTOKEN": stopped after 10 redirects`, storedURL),
			want: "the test notification could not be delivered to the stored webhook URL",
		},
		{
			// And so does a URL stored with a space in its path, which
			// url.Parse accepts and the poster wraps verbatim.
			name: "a stored URL with a space in its path",
			err:  errors.New(`failed to POST webhook to https://hooks.example.com/services/T00 SUPERSECRETTOKEN: Post "https://hooks.example.com/services/T00%20SUPERSECRETTOKEN": context deadline exceeded`),
			want: "the test notification could not be delivered to the stored webhook URL",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := storedWebhookDeliveryFailure(tc.err)
			require.Equal(t, tc.want, got)
			require.NotContains(t, got, "SUPERSECRETTOKEN",
				"the path is the credential for Slack, Discord and Feishu")
			require.NotContains(t, got, "hooks.example.com")
		})
	}
}
