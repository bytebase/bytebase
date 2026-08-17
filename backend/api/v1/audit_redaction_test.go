package v1

import (
	"encoding/base64"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

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
			name:     "saved query response SQL",
			response: &v1pb.SavedQuery{Content: []byte(secretSentinel)},
		},
		{
			name: "purchase checkout credentials",
			response: &v1pb.PurchaseResponse{
				PaymentUrl: secretSentinel,
				SessionId:  secretSentinel,
			},
		},
		{
			name: "VCS provider user export",
			response: &v1pb.ExportVCSProviderUsersResponse{
				Content: []byte(secretSentinel),
			},
		},
		{
			name: "exported audit log content",
			response: &v1pb.ExportAuditLogsResponse{
				Content:       []byte(secretSentinel),
				NextPageToken: "next-page-token",
			},
		},
		{
			// The single-instance response is redacted as defense in depth even
			// though the converter blanks these on reads; its batch twin gets
			// the same treatment rather than the default arm.
			name: "batch instance update response",
			response: &v1pb.BatchUpdateInstancesResponse{Instances: []*v1pb.Instance{{
				Name:        "instances/instance-a",
				DataSources: []*v1pb.DataSource{dataSourceWithEverySecret()},
			}}},
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

// dataSourceWithEverySecret is the readable fixture: every secret redactDataSource
// masks today, on one data source, so a case reads as the request a caller sends.
// It is a hand-kept list and cannot see a field added later —
// TestAuditRedactsEveryInputOnlyDataSourceField is the net that can.
func dataSourceWithEverySecret() *v1pb.DataSource {
	return &v1pb.DataSource{
		Id:                                 "admin",
		Type:                               v1pb.DataSourceType_ADMIN,
		Username:                           "bytebase",
		Host:                               "db.example.com",
		Password:                           secretSentinel,
		SslCa:                              secretSentinel,
		SslCaPath:                          secretSentinel,
		SslCert:                            secretSentinel,
		SslCertPath:                        secretSentinel,
		SslKey:                             secretSentinel,
		SslKeyPath:                         secretSentinel,
		SshPassword:                        secretSentinel,
		SshPrivateKey:                      secretSentinel,
		AuthenticationPrivateKey:           secretSentinel,
		AuthenticationPrivateKeyPassphrase: secretSentinel,
		MasterPassword:                     secretSentinel,
		ExternalSecret: &v1pb.DataSourceExternalSecret{
			AuthOption: &v1pb.DataSourceExternalSecret_Token{Token: secretSentinel},
		},
		SaslConfig: &v1pb.SASLConfig{Mechanism: &v1pb.SASLConfig_KrbConfig{
			KrbConfig: &v1pb.KerberosConfig{Keytab: []byte(secretSentinel)},
		}},
		IamExtension: &v1pb.DataSource_GcpCredential{
			GcpCredential: &v1pb.DataSource_GCPCredential{Content: secretSentinel},
		},
	}
}

// assertDataSourceIntact checks the fields a redactor rewrites rather than
// clears, then returns the password for the caller's own comparison.
func assertDataSourceIntact(t *testing.T, d *v1pb.DataSource) string {
	t.Helper()
	require.Equal(t, secretSentinel, d.GetAuthenticationPrivateKeyPassphrase(), "redaction mutated the passphrase")
	require.Equal(t, secretSentinel, d.GetGcpCredential().GetContent(), "redaction mutated the IAM credential")
	require.Equal(t, secretSentinel, string(d.GetSaslConfig().GetKrbConfig().GetKeytab()), "redaction mutated the keytab")
	return d.GetPassword()
}

// fillInputOnlyFields sets every INPUT_ONLY string and bytes field reachable
// from m to the sentinel, singular or repeated. Each oneof takes its arm-th
// variant, so calling this for every arm up to maxOneofArms covers every branch
// of every oneof declared alongside its siblings.
//
// This reads the same annotation the read path is pinned against
// (assertNoInputOnlyValues), so a credential added to the proto lands in the
// fixture without anyone remembering to add it here. Two shapes it cannot build:
// a map, and a oneof nested inside another oneof's arm, which one global arm
// counter only ever constructs on one value. Neither is silent — inputOnlyLeaves
// counts both, so a fill that cannot reach one fails the test rather than
// quietly asking less of the redactor.
func fillInputOnlyFields(m protoreflect.Message, arm int, onPath map[protoreflect.FullName]bool) {
	descriptor := m.Descriptor()
	if onPath[descriptor.FullName()] {
		return
	}
	onPath[descriptor.FullName()] = true
	defer delete(onPath, descriptor.FullName())

	skip := map[protoreflect.FieldNumber]bool{}
	for i := 0; i < descriptor.Oneofs().Len(); i++ {
		oneof := descriptor.Oneofs().Get(i)
		if oneof.IsSynthetic() {
			continue
		}
		for j := 0; j < oneof.Fields().Len(); j++ {
			skip[oneof.Fields().Get(j).Number()] = true
		}
		fillOneField(m, oneof.Fields().Get(arm%oneof.Fields().Len()), arm, onPath)
	}
	for i := 0; i < descriptor.Fields().Len(); i++ {
		if field := descriptor.Fields().Get(i); !skip[field.Number()] {
			fillOneField(m, field, arm, onPath)
		}
	}
}

func fillOneField(m protoreflect.Message, field protoreflect.FieldDescriptor, arm int, onPath map[protoreflect.FullName]bool) {
	if field.IsMap() {
		return
	}
	if field.Kind() == protoreflect.MessageKind {
		if field.IsList() {
			list := m.Mutable(field).List()
			element := list.NewElement()
			fillInputOnlyFields(element.Message(), arm, onPath)
			list.Append(element)
			return
		}
		fillInputOnlyFields(m.Mutable(field).Message(), arm, onPath)
		return
	}
	if !isInputOnly(field) {
		return
	}
	var value protoreflect.Value
	switch field.Kind() {
	case protoreflect.StringKind:
		value = protoreflect.ValueOfString(secretSentinel)
	case protoreflect.BytesKind:
		value = protoreflect.ValueOfBytes([]byte(secretSentinel))
	default:
		return
	}
	if field.IsList() {
		m.Mutable(field).List().Append(value)
		return
	}
	m.Set(field, value)
}

// maxOneofArms returns the widest non-synthetic oneof reachable from the
// descriptor. Driving the arm loop off this rather than off a literal is the
// point: a fourth iam_extension variant raises the bound by declaring itself.
func maxOneofArms(descriptor protoreflect.MessageDescriptor, onPath map[protoreflect.FullName]bool) int {
	if onPath[descriptor.FullName()] {
		return 0
	}
	onPath[descriptor.FullName()] = true
	defer delete(onPath, descriptor.FullName())

	widest := 1
	for i := 0; i < descriptor.Oneofs().Len(); i++ {
		if oneof := descriptor.Oneofs().Get(i); !oneof.IsSynthetic() {
			widest = max(widest, oneof.Fields().Len())
		}
	}
	for i := 0; i < descriptor.Fields().Len(); i++ {
		if field := descriptor.Fields().Get(i); field.Kind() == protoreflect.MessageKind && !field.IsMap() {
			widest = max(widest, maxOneofArms(field.Message(), onPath))
		}
	}
	return widest
}

// inputOnlyLeaves is the yardstick: every INPUT_ONLY scalar reachable from the
// descriptor, down every arm of every oneof at once and through map values. The
// fill can only take one arm per pass and cannot build a map at all, so the
// union over passes has to equal this — which is what makes a fill that quietly
// reaches less than it claims fail instead of pass. Every other assertion in the
// test is negative, and a field nobody populated passes them all.
//
// Counting what the fill cannot yet build is deliberate. DataSource already
// carries a map (extra_connection_parameters); the day one like it is declared
// INPUT_ONLY, this fails and names it, rather than the gap staying a comment.
func inputOnlyLeaves(descriptor protoreflect.MessageDescriptor, prefix string, onPath map[protoreflect.FullName]bool) []string {
	if onPath[descriptor.FullName()] {
		return nil
	}
	onPath[descriptor.FullName()] = true
	defer delete(onPath, descriptor.FullName())

	var paths []string
	for i := 0; i < descriptor.Fields().Len(); i++ {
		field := descriptor.Fields().Get(i)
		path := fmt.Sprintf("%s.%s", prefix, field.Name())
		switch {
		case field.IsMap():
			if isInputOnly(field) {
				paths = append(paths, path)
			}
			if field.MapValue().Kind() == protoreflect.MessageKind {
				paths = append(paths, inputOnlyLeaves(field.MapValue().Message(), path, onPath)...)
			}
		case field.Kind() == protoreflect.MessageKind:
			paths = append(paths, inputOnlyLeaves(field.Message(), path, onPath)...)
		case isInputOnly(field):
			paths = append(paths, path)
		default:
		}
	}
	return paths
}

// sentinelPaths reports which fields the fill actually reached.
func sentinelPaths(m protoreflect.Message, prefix string) []string {
	var paths []string
	m.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		path := fmt.Sprintf("%s.%s", prefix, field.Name())
		switch {
		case field.IsMap():
		case field.Kind() == protoreflect.MessageKind && field.IsList():
			for i := range value.List().Len() {
				paths = append(paths, sentinelPaths(value.List().Get(i).Message(), path)...)
			}
		case field.Kind() == protoreflect.MessageKind:
			paths = append(paths, sentinelPaths(value.Message(), path)...)
		case field.IsList():
			for i := range value.List().Len() {
				if isSentinel(field, value.List().Get(i)) {
					paths = append(paths, path)
					break
				}
			}
		default:
			if isSentinel(field, value) {
				paths = append(paths, path)
			}
		}
		return true
	})
	return paths
}

func isSentinel(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
	switch field.Kind() {
	case protoreflect.StringKind:
		return value.String() == secretSentinel
	case protoreflect.BytesKind:
		return string(value.Bytes()) == secretSentinel
	default:
		return false
	}
}

func isInputOnly(field protoreflect.FieldDescriptor) bool {
	options, ok := field.Options().(*descriptorpb.FieldOptions)
	if !ok || options == nil {
		return false
	}
	behaviors, ok := proto.GetExtension(options, annotations.E_FieldBehavior).([]annotations.FieldBehavior)
	return ok && slices.Contains(behaviors, annotations.FieldBehavior_INPUT_ONLY)
}

// A hand-kept fixture cannot fail for a credential nobody added to it. This one
// is derived from the proto instead: every INPUT_ONLY field on a DataSource,
// every arm of every oneof under it, on each request that carries one. A new
// credential field, or a fourth iam_extension variant, is covered on the day it
// is declared rather than on the day someone notices the audit row.
func TestAuditRedactsEveryInputOnlyDataSourceField(t *testing.T) {
	dataSource := (&v1pb.DataSource{}).ProtoReflect().Descriptor()
	reached := map[string]bool{}
	for arm := range maxOneofArms(dataSource, map[protoreflect.FullName]bool{}) {
		filled := &v1pb.DataSource{}
		fillInputOnlyFields(filled.ProtoReflect(), arm, map[protoreflect.FullName]bool{})
		for _, path := range sentinelPaths(filled.ProtoReflect(), "dataSource") {
			reached[path] = true
		}

		for _, tt := range []struct {
			name    string
			request any
		}{
			{"create instance", &v1pb.CreateInstanceRequest{Instance: &v1pb.Instance{
				DataSources: []*v1pb.DataSource{filled},
			}}},
			{"update instance", &v1pb.UpdateInstanceRequest{Instance: &v1pb.Instance{
				DataSources: []*v1pb.DataSource{filled},
			}}},
			{"batch update instances", &v1pb.BatchUpdateInstancesRequest{
				Requests: []*v1pb.UpdateInstanceRequest{{Instance: &v1pb.Instance{
					DataSources: []*v1pb.DataSource{filled},
				}}},
			}},
			{"add data source", &v1pb.AddDataSourceRequest{DataSource: filled}},
			{"update data source", &v1pb.UpdateDataSourceRequest{DataSource: filled}},
			{"remove data source", &v1pb.RemoveDataSourceRequest{DataSource: filled}},
		} {
			t.Run(fmt.Sprintf("%s/oneof arm %d", tt.name, arm), func(t *testing.T) {
				got, err := getRequestString(tt.request)
				require.NoError(t, err)
				require.NotContains(t, got, secretSentinel, "credential written to the audit log")
				require.NotContains(t, got, encodedSecretSentinel, "encoded credential written to the audit log")
			})
		}

		t.Run(fmt.Sprintf("redacted message/oneof arm %d", arm), func(t *testing.T) {
			// The string check above only catches what the fixture happened to
			// populate. This one holds the whole annotation: nothing declared
			// INPUT_ONLY may carry a value after redaction.
			assertNoInputOnlyValues(t, redactDataSource(filled).ProtoReflect(), "dataSource")
		})
	}

	// The positive half. Everything above is an assertion that something is
	// absent, which a field the fill never populated passes for free — so under-
	// coverage would read as success. This one fails instead.
	t.Run("the fill reaches every INPUT_ONLY field", func(t *testing.T) {
		var missed []string
		for _, path := range inputOnlyLeaves(dataSource, "dataSource", map[protoreflect.FullName]bool{}) {
			if !reached[path] {
				missed = append(missed, path)
			}
		}
		require.Empty(t, missed, "never populated, so the redaction assertions above say nothing about them")
	})
}

// A rejected request is audited too, so every redactor runs on messages the
// handler was about to refuse. redactDataSource used to read through the nil
// data source in one of these and panic the interceptor.
func TestAuditRedactionSurvivesAnIncompleteRequest(t *testing.T) {
	for _, tt := range []struct {
		name    string
		request any
	}{
		{"add data source without one", &v1pb.AddDataSourceRequest{Name: "instances/instance-a"}},
		{"update data source without one", &v1pb.UpdateDataSourceRequest{Name: "instances/instance-a"}},
		{"remove data source without one", &v1pb.RemoveDataSourceRequest{Name: "instances/instance-a"}},
		{"create instance without one", &v1pb.CreateInstanceRequest{InstanceId: "instance-a"}},
		{"update instance without one", &v1pb.UpdateInstanceRequest{}},
		{"batch update instances without any", &v1pb.BatchUpdateInstancesRequest{
			Requests: []*v1pb.UpdateInstanceRequest{{}},
		}},
		{"instance carrying an empty data source", &v1pb.UpdateInstanceRequest{Instance: &v1pb.Instance{
			DataSources: []*v1pb.DataSource{{}},
		}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRequestString(tt.request)
			require.NoError(t, err)
			require.NotEmpty(t, got, "a rejected request still gets a row")
		})
	}
}

func TestPrepareSampleProjectInstanceAuditRequestContainsOnlyParent(t *testing.T) {
	request := &v1pb.PrepareSampleProjectInstanceRequest{Parent: "projects/project-a"}

	got, err := getRequestString(request)
	require.NoError(t, err)
	require.Contains(t, got, "projects/project-a")
	require.NotContains(t, got, secretSentinel)
	require.Equal(
		t,
		"projects/project-a",
		getRequestResource(request, "/bytebase.v1.InstanceService/PrepareSampleProjectInstance"),
	)
}

func TestAuditRequestRedactsCredentials(t *testing.T) {
	batchParent := "projects/project-a"
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
		{"saved query SQL", &v1pb.CreateSavedQueryRequest{SavedQuery: &v1pb.SavedQuery{
			Content: []byte(secretSentinel),
		}}},
		// The data-source secrets are INPUT_ONLY, so a write request is the only
		// place they appear — which is exactly what an audit row records.
		{"instance data source secrets", &v1pb.UpdateInstanceRequest{Instance: &v1pb.Instance{
			Name:        "instances/instance-a",
			DataSources: []*v1pb.DataSource{dataSourceWithEverySecret()},
		}}},
		{"batch instance data source secrets", &v1pb.BatchUpdateInstancesRequest{
			Parent: &batchParent,
			Requests: []*v1pb.UpdateInstanceRequest{nil, {Instance: &v1pb.Instance{
				Name:        "instances/instance-a",
				DataSources: []*v1pb.DataSource{dataSourceWithEverySecret()},
			}}},
		}},
		{"created instance data source secrets", &v1pb.CreateInstanceRequest{
			InstanceId: "instance-a",
			Instance:   &v1pb.Instance{DataSources: []*v1pb.DataSource{dataSourceWithEverySecret()}},
		}},
		{"added data source secrets", &v1pb.AddDataSourceRequest{
			Name:       "instances/instance-a",
			DataSource: dataSourceWithEverySecret(),
		}},
		{"updated data source secrets", &v1pb.UpdateDataSourceRequest{
			Name:       "instances/instance-a",
			DataSource: dataSourceWithEverySecret(),
		}},
		{"removed data source secrets", &v1pb.RemoveDataSourceRequest{
			Name:       "instances/instance-a",
			DataSource: dataSourceWithEverySecret(),
		}},
		{"azure iam client secret", &v1pb.AddDataSourceRequest{DataSource: &v1pb.DataSource{
			IamExtension: &v1pb.DataSource_AzureCredential_{AzureCredential: &v1pb.DataSource_AzureCredential{
				TenantId: "tenant-a", ClientId: "client-a", ClientSecret: secretSentinel,
			}},
		}}},
		{"aws iam access keys", &v1pb.AddDataSourceRequest{DataSource: &v1pb.DataSource{
			IamExtension: &v1pb.DataSource_AwsCredential{AwsCredential: &v1pb.DataSource_AWSCredential{
				AccessKeyId: secretSentinel, SecretAccessKey: secretSentinel, SessionToken: secretSentinel,
				RoleArn: secretSentinel, ExternalId: secretSentinel,
			}},
		}}},
		{"password reset code and password", &v1pb.ResetPasswordRequest{
			Email:       "user@example.com",
			Code:        secretSentinel,
			NewPassword: secretSentinel,
		}},
		{"enterprise license", &v1pb.UploadLicenseRequest{License: secretSentinel}},
		{"add webhook URL", &v1pb.AddWebhookRequest{
			Project: "projects/project-a",
			Webhook: &v1pb.Webhook{Name: "projects/project-a/webhooks/webhook-a", Title: "Webhook A", Url: secretSentinel},
		}},
		{"update webhook URL", &v1pb.UpdateWebhookRequest{
			Webhook: &v1pb.Webhook{Name: "projects/project-a/webhooks/webhook-a", Title: "Webhook A", Url: secretSentinel},
		}},
		{"remove webhook URL", &v1pb.RemoveWebhookRequest{
			Webhook: &v1pb.Webhook{Name: "projects/project-a/webhooks/webhook-a", Title: "Webhook A", Url: secretSentinel},
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRequestString(tt.request)
			require.NoError(t, err)
			require.NotContains(t, got, secretSentinel, "credential written to the audit log")
			require.NotContains(t, got, encodedSecretSentinel, "encoded content written to the audit log")
			if _, ok := tt.request.(*v1pb.ResetPasswordRequest); ok {
				require.Contains(t, got, "user@example.com", "email is required for audit attribution")
			}
		})
	}
}

func TestSensitiveOutboundAuditMetadata(t *testing.T) {
	batchParent := "projects/project-a"
	for _, tt := range []struct {
		name     string
		value    any
		response bool
		want     []string
	}{
		{
			name:  "webhook request retains identifiers",
			value: &v1pb.AddWebhookRequest{Project: "projects/project-a", Webhook: &v1pb.Webhook{Name: "projects/project-a/webhooks/webhook-a", Title: "Webhook A", Url: secretSentinel}},
			want:  []string{"projects/project-a", "projects/project-a/webhooks/webhook-a", "Webhook A"},
		},
		{
			// The row has to stay useful: which instances were retargeted, and
			// which kind of IAM credential was supplied.
			name: "batch instance update retains targets and credential type",
			value: &v1pb.BatchUpdateInstancesRequest{
				Parent: &batchParent,
				Requests: []*v1pb.UpdateInstanceRequest{{Instance: &v1pb.Instance{
					Name:        "instances/instance-a",
					DataSources: []*v1pb.DataSource{dataSourceWithEverySecret()},
				}}},
			},
			want: []string{"projects/project-a", "instances/instance-a", "db.example.com", "gcpCredential"},
		},
		{
			name: "azure iam credential retains the principal it names",
			value: &v1pb.AddDataSourceRequest{DataSource: &v1pb.DataSource{
				IamExtension: &v1pb.DataSource_AzureCredential_{AzureCredential: &v1pb.DataSource_AzureCredential{
					TenantId: "tenant-a", ClientId: "client-a", ClientSecret: secretSentinel,
				}},
			}},
			want: []string{"azureCredential", "tenant-a", "client-a"},
		},
		{
			name: "aws iam credential records that one was supplied",
			value: &v1pb.AddDataSourceRequest{DataSource: &v1pb.DataSource{
				IamExtension: &v1pb.DataSource_AwsCredential{AwsCredential: &v1pb.DataSource_AWSCredential{
					AccessKeyId: secretSentinel, SecretAccessKey: secretSentinel,
				}},
			}},
			want: []string{"awsCredential"},
		},
		{
			name:     "audit export retains page token",
			value:    &v1pb.ExportAuditLogsResponse{Content: []byte(secretSentinel), NextPageToken: "next-page-token"},
			response: true,
			want:     []string{"next-page-token"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var (
				got string
				err error
			)
			if tt.response {
				got, err = getResponseString(tt.value)
			} else {
				got, err = getRequestString(tt.value)
			}
			require.NoError(t, err)
			for _, want := range tt.want {
				require.Contains(t, got, want)
			}
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

	savedQuery := &v1pb.SavedQuery{Content: []byte(secretSentinel)}
	_, err = getResponseString(savedQuery)
	require.NoError(t, err)
	require.Equal(t, secretSentinel, string(savedQuery.GetContent()), "redaction mutated the saved query response")

	auditExport := &v1pb.ExportAuditLogsResponse{Content: []byte(secretSentinel), NextPageToken: "next-page-token"}
	_, err = getResponseString(auditExport)
	require.NoError(t, err)
	require.Equal(t, secretSentinel, string(auditExport.GetContent()), "redaction mutated the audit export response")

	for _, tt := range []struct {
		name    string
		request any
	}{
		{
			name: "create instance request",
			request: &v1pb.CreateInstanceRequest{Instance: &v1pb.Instance{
				DataSources: []*v1pb.DataSource{dataSourceWithEverySecret()},
			}},
		},
		{
			name: "update instance request",
			request: &v1pb.UpdateInstanceRequest{Instance: &v1pb.Instance{
				DataSources: []*v1pb.DataSource{dataSourceWithEverySecret()},
			}},
		},
		{
			name: "batch update instances request",
			request: &v1pb.BatchUpdateInstancesRequest{Requests: []*v1pb.UpdateInstanceRequest{{
				Instance: &v1pb.Instance{DataSources: []*v1pb.DataSource{dataSourceWithEverySecret()}},
			}}},
		},
		{name: "add data source request", request: &v1pb.AddDataSourceRequest{DataSource: dataSourceWithEverySecret()}},
		{name: "update data source request", request: &v1pb.UpdateDataSourceRequest{DataSource: dataSourceWithEverySecret()}},
		{name: "remove data source request", request: &v1pb.RemoveDataSourceRequest{DataSource: dataSourceWithEverySecret()}},
		{name: "create release request", request: &v1pb.CreateReleaseRequest{Release: &v1pb.Release{Files: []*v1pb.Release_File{{Statement: []byte(secretSentinel)}}}}},
		{name: "update release request", request: &v1pb.UpdateReleaseRequest{Release: &v1pb.Release{Files: []*v1pb.Release_File{{Statement: []byte(secretSentinel)}}}}},
		{name: "create saved query request", request: &v1pb.CreateSavedQueryRequest{SavedQuery: &v1pb.SavedQuery{Content: []byte(secretSentinel)}}},
		{name: "reset password request", request: &v1pb.ResetPasswordRequest{Email: "user@example.com", Code: secretSentinel, NewPassword: secretSentinel}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getRequestString(tt.request)
			require.NoError(t, err)
			var sensitiveValue string
			switch request := tt.request.(type) {
			case *v1pb.CreateInstanceRequest:
				sensitiveValue = assertDataSourceIntact(t, request.GetInstance().GetDataSources()[0])
			case *v1pb.UpdateInstanceRequest:
				sensitiveValue = assertDataSourceIntact(t, request.GetInstance().GetDataSources()[0])
			case *v1pb.BatchUpdateInstancesRequest:
				sensitiveValue = assertDataSourceIntact(t, request.GetRequests()[0].GetInstance().GetDataSources()[0])
			case *v1pb.AddDataSourceRequest:
				sensitiveValue = assertDataSourceIntact(t, request.GetDataSource())
			case *v1pb.UpdateDataSourceRequest:
				sensitiveValue = assertDataSourceIntact(t, request.GetDataSource())
			case *v1pb.RemoveDataSourceRequest:
				sensitiveValue = assertDataSourceIntact(t, request.GetDataSource())
			case *v1pb.CreateReleaseRequest:
				sensitiveValue = string(request.GetRelease().GetFiles()[0].GetStatement())
			case *v1pb.UpdateReleaseRequest:
				sensitiveValue = string(request.GetRelease().GetFiles()[0].GetStatement())
			case *v1pb.CreateSavedQueryRequest:
				sensitiveValue = string(request.GetSavedQuery().GetContent())
			case *v1pb.ResetPasswordRequest:
				sensitiveValue = request.GetCode()
				require.Equal(t, secretSentinel, request.GetNewPassword())
			default:
				t.Fatalf("unexpected request type %T", request)
			}
			require.Equal(t, secretSentinel, sensitiveValue, "redaction mutated the request")
		})
	}
}
