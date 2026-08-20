package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// slackWebhookURL is a well-formed Slack incoming-webhook URL on the domain the
// validator allows, so AddWebhook accepts it without the test having to widen
// the allowed-domain table other tests share. Nothing posts to it here; the
// delivery half of the story is in TestWebhookIntegration, which owns that
// table and a server that answers.
const slackWebhookURL = "https://hooks.slack.com/services/read-path-secret-probe"

// TestReadPathHidesTheWebhookURL is the leak as a caller meets it. An
// incoming-webhook URL is a bearer credential: whoever holds it posts into the
// customer's chat as Bytebase. Every project read shares one converter, so the
// four of them are checked together rather than trusting that they still do.
func TestReadPathHidesTheWebhookURL(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	project := ctl.createTestProject(ctx, t, "webhook-redaction")
	added, err := ctl.projectServiceClient.AddWebhook(ctx, connect.NewRequest(&v1pb.AddWebhookRequest{
		Project: project.Name,
		Webhook: &v1pb.Webhook{
			Type:              v1pb.WebhookType_SLACK,
			Title:             "release channel",
			Url:               slackWebhookURL,
			NotificationTypes: []v1pb.Activity_Type{v1pb.Activity_ISSUE_CREATED},
		},
	}))
	a.NoError(err)
	a.Len(added.Msg.Webhooks, 1)

	// The write RPC's own response goes through the same converter. The caller
	// supplied the URL, so losing it costs nothing, and keeping the rule in one
	// place is what stops a read from finding a way around it.
	a.Empty(added.Msg.Webhooks[0].Url)

	get, err := ctl.projectServiceClient.GetProject(ctx, connect.NewRequest(&v1pb.GetProjectRequest{Name: project.Name}))
	a.NoError(err)
	list, err := ctl.projectServiceClient.ListProjects(ctx, connect.NewRequest(&v1pb.ListProjectsRequest{}))
	a.NoError(err)
	batch, err := ctl.projectServiceClient.BatchGetProjects(ctx, connect.NewRequest(&v1pb.BatchGetProjectsRequest{
		Names: []string{project.Name},
	}))
	a.NoError(err)
	search, err := ctl.projectServiceClient.SearchProjects(ctx, connect.NewRequest(&v1pb.SearchProjectsRequest{}))
	a.NoError(err)

	// A client that reads a project and writes it straight back sends the URL it
	// never received, which has to be refused rather than saved over a working
	// webhook. The validator is what refuses it, and this is the RPC that has
	// to still be calling the validator.
	readBack, err := ctl.projectServiceClient.GetProject(ctx, connect.NewRequest(&v1pb.GetProjectRequest{Name: project.Name}))
	a.NoError(err)
	_, err = ctl.projectServiceClient.UpdateWebhook(ctx, connect.NewRequest(&v1pb.UpdateWebhookRequest{
		Webhook:    readBack.Msg.Webhooks[0],
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"url"}},
	}))
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err),
		"writing the read-back webhook back must be refused, not saved")

	read := map[string][]*v1pb.Project{
		"GetProject":       {get.Msg},
		"ListProjects":     list.Msg.Projects,
		"BatchGetProjects": batch.Msg.Projects,
		"SearchProjects":   search.Msg.Projects,
	}
	for rpc, projects := range read {
		var found bool
		for _, p := range projects {
			if p.Name != project.Name {
				continue
			}
			found = true
			a.Len(p.Webhooks, 1, "%s", rpc)
			a.Empty(p.Webhooks[0].Url,
				"%s returns the webhook URL, which is the whole credential", rpc)
			a.Equal("release channel", p.Webhooks[0].Title, "%s", rpc)
		}
		a.True(found, "%s did not return the project under test", rpc)
	}
}

// TestReadPathHidesTheMFAEnrollmentSecrets covers the third leak and the
// constraint that makes it awkward: one converter serves both the reads and the
// RPC that mints the enrollment. The mint has to keep the secrets — the console
// renders the QR code and the recovery codes out of that response and the
// server keeps no other copy to hand over — and every read has to lose them.
//
// GetUser is here beside GetCurrentUser because the leak was never
// GetCurrentUser's: the converter exposed the fields to any read whose subject
// was the caller, so reading your own profile through GetUser leaked the same
// TOTP seed. GetUser stays EXCLUDED for MCP on other grounds, which is exactly
// why the fix could not live in one RPC.
func TestReadPathHidesTheMFAEnrollmentSecrets(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	userName := common.FormatUserEmail("demo@example.com")
	minted, err := ctl.userServiceClient.UpdateUser(ctx, connect.NewRequest(&v1pb.UpdateUserRequest{
		User:                    &v1pb.User{Name: userName},
		UpdateMask:              &fieldmaskpb.FieldMask{},
		RegenerateTempMfaSecret: true,
	}))
	a.NoError(err)
	a.NotEmpty(minted.Msg.TempOtpSecret, "the minting response is the only place the console can read the TOTP seed")
	a.NotEmpty(minted.Msg.TempRecoveryCodes, "and the only place it can read the recovery codes")
	a.NotNil(minted.Msg.TempOtpSecretCreatedTime, "the countdown needs the moment the window opened")

	current, err := ctl.userServiceClient.GetCurrentUser(ctx, connect.NewRequest(&emptypb.Empty{}))
	a.NoError(err)
	self, err := ctl.userServiceClient.GetUser(ctx, connect.NewRequest(&v1pb.GetUserRequest{Name: userName}))
	a.NoError(err)

	for rpc, user := range map[string]*v1pb.User{"GetCurrentUser": current.Msg, "GetUser": self.Msg} {
		a.Empty(user.TempOtpSecret, "%s returns the TOTP seed of an open enrollment", rpc)
		a.Empty(user.TempRecoveryCodes, "%s returns the recovery codes of an open enrollment", rpc)
		a.NotNil(user.TempOtpSecretCreatedTime,
			"%s must still say an enrollment is open and when it expires; that is not a secret", rpc)
	}

	// An update that mints nothing does not answer with the enrollment either,
	// even though the window is open. UpdateUser is one RPC doing several jobs,
	// and only the two that drive the enrollment carry it back.
	renamed, err := ctl.userServiceClient.UpdateUser(ctx, connect.NewRequest(&v1pb.UpdateUserRequest{
		User:       &v1pb.User{Name: userName, Title: "Demo Renamed"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
	}))
	a.NoError(err)
	a.Equal("Demo Renamed", renamed.Msg.Title)
	a.Empty(renamed.Msg.TempOtpSecret, "a title update mints nothing and must carry nothing")
	a.Empty(renamed.Msg.TempRecoveryCodes)

	// The enrollment still completes, which is the whole point of leaving the
	// minting response alone: the seed the console was handed is the seed the
	// server validates against. The otp_code verification is the other request
	// that answers with the enrollment, because it is the step the console
	// moves to the recovery-code screen from.
	otp, err := totp.GenerateCode(minted.Msg.TempOtpSecret, time.Now())
	a.NoError(err)
	verified, err := ctl.userServiceClient.UpdateUser(ctx, connect.NewRequest(&v1pb.UpdateUserRequest{
		User:       &v1pb.User{Name: userName},
		UpdateMask: &fieldmaskpb.FieldMask{},
		OtpCode:    &otp,
	}))
	a.NoError(err)
	a.NotEmpty(verified.Msg.TempRecoveryCodes,
		"the console reads the recovery codes it asks the user to save out of this response")

	enabled, err := ctl.userServiceClient.UpdateUser(ctx, connect.NewRequest(&v1pb.UpdateUserRequest{
		User:       &v1pb.User{Name: userName, MfaEnabled: true},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"mfa_enabled"}},
		OtpCode:    &otp,
	}))
	a.NoError(err)
	a.True(enabled.Msg.MfaEnabled)
}

// TestMCPServesTheReadsTheRedactionFreed is the regression this batch exists to
// close, checked where it bit: a live MCP session under the default ceiling.
// Since the gate shipped, every EXCLUDED method refuses under every ceiling,
// and the eight reads excluded only for a leak were among them — so an agent
// could not name a project or an instance, and therefore could not reach a
// database at all.
//
// The other half is that the widening stopped where it was meant to. The reads
// that are out for what they do are still out, and the responses the freed
// reads return no longer carry the secret that put them there.
func TestMCPServesTheReadsTheRedactionFreed(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	project, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: "mcp-freed-reads",
		Project:   &v1pb.Project{Title: "MCP freed reads"},
	}))
	a.NoError(err)
	projectName := project.Msg.Name

	_, err = ctl.projectServiceClient.AddWebhook(ctx, connect.NewRequest(&v1pb.AddWebhookRequest{
		Project: projectName,
		Webhook: &v1pb.Webhook{
			Type:              v1pb.WebhookType_SLACK,
			Title:             "release channel",
			Url:               slackWebhookURL,
			NotificationTypes: []v1pb.Activity_Type{v1pb.Activity_ISSUE_CREATED},
		},
	}))
	a.NoError(err)

	_, err = ctl.userServiceClient.UpdateUser(ctx, connect.NewRequest(&v1pb.UpdateUserRequest{
		User:                    &v1pb.User{Name: common.FormatUserEmail("demo@example.com")},
		UpdateMask:              &fieldmaskpb.FieldMask{},
		RegenerateTempMfaSecret: true,
	}))
	a.NoError(err)

	// The two instance reads need something to read. A PostgreSQL instance is
	// what this suite can stand up cheaply, and PostgreSQL role attributes are
	// keyword lists, so nothing here can assert anything about the grant-text
	// mask: the credential it takes out is MariaDB's, and it is pinned against
	// real server output in the converter's own test. What these two rows are
	// for is the classification — the reads are served at all.
	pgContainer, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: "mcp-freed-instance",
		Instance: &v1pb.Instance{
			Title:       "MCP freed instance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.NoError(err)
	instanceName := instance.Msg.Name

	session := openMCPSession(ctx, t, ctl, ctl.authInterceptor.token)
	defer session.Close()

	// Each row carries what the response has to contain as well as what it must
	// not. Without the positive control a change that emptied or rescoped a
	// payload would leave every leak assertion green for the wrong reason.
	//
	// ListInstanceRoles has no control, on purpose. Its response is the role
	// list, which CreateInstance fills in on a best-effort sync it swallows on
	// failure, so an empty body is a legitimate outcome and asserting anything
	// about the payload would make a slow container look like a redaction bug.
	// What that row is here for is the classification: the method is served.
	for _, tc := range []struct {
		operation string
		body      map[string]any
		want      string
	}{
		{"ProjectService/ListProjects", map[string]any{}, projectName},
		{"ProjectService/GetProject", map[string]any{"name": projectName}, projectName},
		{"ProjectService/BatchGetProjects", map[string]any{"names": []string{projectName}}, projectName},
		{"ProjectService/SearchProjects", map[string]any{}, projectName},
		{"InstanceService/GetInstance", map[string]any{"name": instanceName}, instanceName},
		{"InstanceService/ListInstances", map[string]any{}, instanceName},
		{"InstanceRoleService/ListInstanceRoles", map[string]any{"parent": instanceName}, ""},
		{"UserService/GetCurrentUser", map[string]any{}, "demo@example.com"},
	} {
		got := callAPIOnSession(ctx, t, session, tc.operation, tc.body)
		a.Equal(200, got.Status, "%s must be served: %s", tc.operation, got.Error)
		if tc.want != "" {
			a.Contains(got.RawResponse, tc.want,
				"%s answered without the resource under test, so its leak assertions would pass vacuously", tc.operation)
		}
		a.NotContains(got.RawResponse, "read-path-secret-probe",
			"%s handed the agent the webhook credential", tc.operation)
		a.NotContains(got.RawResponse, "tempOtpSecret\":\"",
			"%s handed the agent an MFA enrollment secret", tc.operation)
	}

	// Excluded for what they do, not for a leak, so nothing here moved.
	for _, tc := range []struct {
		operation string
		reason    string
	}{
		{"UserService/ListUsers", "administers the workspace"},
		{"UserService/GetUser", "administers the workspace"},
		{"QueryHistoryService/ListQueryHistories", "SQL that other people wrote"},
		{"ProjectService/TestWebhook", "third party"},
		// The deprecated SQLService alias of a refused method, which the MCP
		// spec indexes by its own operation ID and would otherwise be a way
		// round the canonical method's class.
		{"SQLService/ListQueryHistories", "SQL that other people wrote"},
	} {
		got := callAPIOnSession(ctx, t, session, tc.operation, map[string]any{})
		a.Equal(403, got.Status, "%s must stay refused", tc.operation)
		a.True(strings.Contains(got.Error, tc.reason),
			"%s must still be refused for what it does; got %q", tc.operation, got.Error)
	}
}
