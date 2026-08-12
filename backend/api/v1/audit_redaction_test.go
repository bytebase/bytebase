package v1

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Audited RPCs write their request and response payloads to audit_log, and to
// stdout when RuntimeEnableAuditLogStdout is set. Anything with
// bb.auditLogs.search/export, or read access to the log pipeline, can read them.
// So no live credential may survive into either payload.
//
// These are behavioral: each case populates a secret and asserts the marshaled
// audit string does not contain it. A redactor that stops covering a field fails
// here rather than silently starting to log it.

const secretSentinel = "s3cr3t-sentinel-value"

var encodedSecretSentinel = base64.StdEncoding.EncodeToString([]byte(secretSentinel))

func TestAuditResponseRedactsCredentials(t *testing.T) {
	for _, tt := range []struct {
		name     string
		response any
	}{
		{
			// Only create and key rotation populate this; the read path never
			// does. It is a live API key.
			name:     "service account key",
			response: &v1pb.ServiceAccount{Name: "serviceAccounts/a@b.com", ServiceKey: secretSentinel},
		},
		{
			name: "smtp password in a returned setting",
			response: &v1pb.Setting{Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_Email{
				Email: &v1pb.EmailSetting{Config: &v1pb.EmailSetting_Smtp{
					Smtp: &v1pb.EmailSetting_SMTPConfig{Password: secretSentinel},
				}},
			}}},
		},
		{
			name: "idp client secret in a returned provider",
			response: &v1pb.IdentityProvider{Config: &v1pb.IdentityProviderConfig{
				Config: &v1pb.IdentityProviderConfig_Oauth2Config{
					Oauth2Config: &v1pb.OAuth2IdentityProviderConfig{ClientSecret: secretSentinel},
				},
			}},
		},
		{
			name: "project webhook URL",
			response: &v1pb.Project{Webhooks: []*v1pb.Webhook{{
				Name: "projects/project-a/webhooks/webhook-a",
				Url:  secretSentinel,
			}}},
		},
		{
			name: "release response SQL",
			response: &v1pb.Release{Files: []*v1pb.Release_File{nil, {
				Statement: []byte(secretSentinel),
			}}},
		},
		{
			name:     "worksheet response SQL",
			response: &v1pb.Worksheet{Content: []byte(secretSentinel)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getResponseString(tt.response)
			require.NoError(t, err)
			require.NotContains(t, got, secretSentinel, "credential written to the audit log")
			require.NotContains(t, got, encodedSecretSentinel, "encoded content written to the audit log")
		})
	}
}

func TestAuditRequestRedactsCredentials(t *testing.T) {
	settingRequest := func(v *v1pb.SettingValue) *v1pb.UpdateSettingRequest {
		return &v1pb.UpdateSettingRequest{Setting: &v1pb.Setting{Value: v}}
	}
	imSetting := func(s *v1pb.AppIMSetting_IMSetting) *v1pb.UpdateSettingRequest {
		return settingRequest(&v1pb.SettingValue{Value: &v1pb.SettingValue_AppIm{
			AppIm: &v1pb.AppIMSetting{Settings: []*v1pb.AppIMSetting_IMSetting{s}},
		}})
	}

	for _, tt := range []struct {
		name    string
		request any
	}{
		{"smtp password", settingRequest(&v1pb.SettingValue{Value: &v1pb.SettingValue_Email{
			Email: &v1pb.EmailSetting{Config: &v1pb.EmailSetting_Smtp{
				Smtp: &v1pb.EmailSetting_SMTPConfig{Password: secretSentinel},
			}},
		}})},
		{"ai api key", settingRequest(&v1pb.SettingValue{Value: &v1pb.SettingValue_Ai{
			Ai: &v1pb.AISetting{ApiKey: secretSentinel},
		}})},
		{"slack token", imSetting(&v1pb.AppIMSetting_IMSetting{Payload: &v1pb.AppIMSetting_IMSetting_Slack{Slack: &v1pb.AppIMSetting_Slack{Token: secretSentinel}}})},
		{"feishu app secret", imSetting(&v1pb.AppIMSetting_IMSetting{Payload: &v1pb.AppIMSetting_IMSetting_Feishu{Feishu: &v1pb.AppIMSetting_Feishu{AppSecret: secretSentinel}}})},
		{"wecom secret", imSetting(&v1pb.AppIMSetting_IMSetting{Payload: &v1pb.AppIMSetting_IMSetting_Wecom{Wecom: &v1pb.AppIMSetting_Wecom{Secret: secretSentinel}}})},
		{"lark app secret", imSetting(&v1pb.AppIMSetting_IMSetting{Payload: &v1pb.AppIMSetting_IMSetting_Lark{Lark: &v1pb.AppIMSetting_Lark{AppSecret: secretSentinel}}})},
		{"dingtalk client secret", imSetting(&v1pb.AppIMSetting_IMSetting{Payload: &v1pb.AppIMSetting_IMSetting_Dingtalk{Dingtalk: &v1pb.AppIMSetting_DingTalk{ClientSecret: secretSentinel}}})},
		{"teams client secret", imSetting(&v1pb.AppIMSetting_IMSetting{Payload: &v1pb.AppIMSetting_IMSetting_Teams{Teams: &v1pb.AppIMSetting_Teams{ClientSecret: secretSentinel}}})},
		{"oauth2 client secret", &v1pb.CreateIdentityProviderRequest{IdentityProvider: &v1pb.IdentityProvider{
			Config: &v1pb.IdentityProviderConfig{Config: &v1pb.IdentityProviderConfig_Oauth2Config{
				Oauth2Config: &v1pb.OAuth2IdentityProviderConfig{ClientSecret: secretSentinel},
			}},
		}}},
		{"oidc client secret", &v1pb.CreateIdentityProviderRequest{IdentityProvider: &v1pb.IdentityProvider{
			Config: &v1pb.IdentityProviderConfig{Config: &v1pb.IdentityProviderConfig_OidcConfig{
				OidcConfig: &v1pb.OIDCIdentityProviderConfig{ClientSecret: secretSentinel},
			}},
		}}},
		{"ldap bind password", &v1pb.UpdateIdentityProviderRequest{IdentityProvider: &v1pb.IdentityProvider{
			Config: &v1pb.IdentityProviderConfig{Config: &v1pb.IdentityProviderConfig_LdapConfig{
				LdapConfig: &v1pb.LDAPIdentityProviderConfig{BindPassword: secretSentinel},
			}},
		}}},
		{"create project webhook URL", &v1pb.CreateProjectRequest{Project: &v1pb.Project{
			Webhooks: []*v1pb.Webhook{{Name: "projects/project-a/webhooks/webhook-a", Url: secretSentinel}},
		}}},
		{"update project webhook URL", &v1pb.UpdateProjectRequest{Project: &v1pb.Project{
			Name:     "projects/project-a",
			Webhooks: []*v1pb.Webhook{{Name: "projects/project-a/webhooks/webhook-a", Url: secretSentinel}},
		}}},
		{"release SQL", &v1pb.CreateReleaseRequest{Release: &v1pb.Release{
			Files: []*v1pb.Release_File{nil, {Statement: []byte(secretSentinel)}},
		}}},
		{"updated release SQL", &v1pb.UpdateReleaseRequest{Release: &v1pb.Release{
			Files: []*v1pb.Release_File{{Statement: []byte(secretSentinel)}},
		}}},
		{"worksheet SQL", &v1pb.CreateWorksheetRequest{Worksheet: &v1pb.Worksheet{
			Content: []byte(secretSentinel),
		}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRequestString(tt.request)
			require.NoError(t, err)
			require.NotContains(t, got, secretSentinel, "credential written to the audit log")
			require.NotContains(t, got, encodedSecretSentinel, "encoded content written to the audit log")
		})
	}
}

// Redaction must not mutate the caller's message: these run on the live request
// and response objects, so masking in place would corrupt what the handler
// returns to the client or what a later interceptor reads.
func TestAuditRedactionDoesNotMutateInput(t *testing.T) {
	account := &v1pb.ServiceAccount{ServiceKey: secretSentinel}
	_, err := getResponseString(account)
	require.NoError(t, err)
	require.Equal(t, secretSentinel, account.GetServiceKey(), "redaction mutated the response")

	request := &v1pb.UpdateSettingRequest{Setting: &v1pb.Setting{Value: &v1pb.SettingValue{
		Value: &v1pb.SettingValue_Ai{Ai: &v1pb.AISetting{ApiKey: secretSentinel}},
	}}}
	_, err = getRequestString(request)
	require.NoError(t, err)
	require.Equal(t, secretSentinel, request.GetSetting().GetValue().GetAi().GetApiKey(),
		"redaction mutated the request")

	project := &v1pb.Project{Webhooks: []*v1pb.Webhook{{Url: secretSentinel}}}
	_, err = getResponseString(project)
	require.NoError(t, err)
	require.Equal(t, secretSentinel, project.GetWebhooks()[0].GetUrl(), "redaction mutated the project response")

	release := &v1pb.Release{Files: []*v1pb.Release_File{{Statement: []byte(secretSentinel)}}}
	_, err = getResponseString(release)
	require.NoError(t, err)
	require.Equal(t, secretSentinel, string(release.GetFiles()[0].GetStatement()), "redaction mutated the release response")

	worksheet := &v1pb.Worksheet{Content: []byte(secretSentinel)}
	_, err = getResponseString(worksheet)
	require.NoError(t, err)
	require.Equal(t, secretSentinel, string(worksheet.GetContent()), "redaction mutated the worksheet response")

	for _, tt := range []struct {
		name    string
		request any
	}{
		{
			name: "create instance request",
			request: &v1pb.CreateInstanceRequest{Instance: &v1pb.Instance{DataSources: []*v1pb.DataSource{{
				Password: secretSentinel,
			}}}},
		},
		{
			name: "update instance request",
			request: &v1pb.UpdateInstanceRequest{Instance: &v1pb.Instance{DataSources: []*v1pb.DataSource{{
				Password: secretSentinel,
			}}}},
		},
		{name: "add data source request", request: &v1pb.AddDataSourceRequest{DataSource: &v1pb.DataSource{Password: secretSentinel}}},
		{name: "update data source request", request: &v1pb.UpdateDataSourceRequest{DataSource: &v1pb.DataSource{Password: secretSentinel}}},
		{name: "remove data source request", request: &v1pb.RemoveDataSourceRequest{DataSource: &v1pb.DataSource{Password: secretSentinel}}},
		{name: "create release request", request: &v1pb.CreateReleaseRequest{Release: &v1pb.Release{Files: []*v1pb.Release_File{{Statement: []byte(secretSentinel)}}}}},
		{name: "update release request", request: &v1pb.UpdateReleaseRequest{Release: &v1pb.Release{Files: []*v1pb.Release_File{{Statement: []byte(secretSentinel)}}}}},
		{name: "create worksheet request", request: &v1pb.CreateWorksheetRequest{Worksheet: &v1pb.Worksheet{Content: []byte(secretSentinel)}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getRequestString(tt.request)
			require.NoError(t, err)
			var sensitiveValue string
			switch request := tt.request.(type) {
			case *v1pb.CreateInstanceRequest:
				sensitiveValue = request.GetInstance().GetDataSources()[0].GetPassword()
			case *v1pb.UpdateInstanceRequest:
				sensitiveValue = request.GetInstance().GetDataSources()[0].GetPassword()
			case *v1pb.AddDataSourceRequest:
				sensitiveValue = request.GetDataSource().GetPassword()
			case *v1pb.UpdateDataSourceRequest:
				sensitiveValue = request.GetDataSource().GetPassword()
			case *v1pb.RemoveDataSourceRequest:
				sensitiveValue = request.GetDataSource().GetPassword()
			case *v1pb.CreateReleaseRequest:
				sensitiveValue = string(request.GetRelease().GetFiles()[0].GetStatement())
			case *v1pb.UpdateReleaseRequest:
				sensitiveValue = string(request.GetRelease().GetFiles()[0].GetStatement())
			case *v1pb.CreateWorksheetRequest:
				sensitiveValue = string(request.GetWorksheet().GetContent())
			default:
				t.Fatalf("unexpected request type %T", request)
			}
			require.Equal(t, secretSentinel, sensitiveValue, "redaction mutated the request")
		})
	}
}
