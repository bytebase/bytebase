package tests

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"regexp"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestAuditLogFormat is both a regression test for the 3.17.0 bug where
// AuthService/Login (and Signup/ExchangeToken) silently dropped audit entries,
// AND a contract test for the shape of audit log entries returned by
// AuditLogService/SearchAuditLogs. Downstream consumers (SIEMs, compliance
// tooling, `docker logs | grep log_type:audit`) depend on this shape being
// stable across releases — changes here are user-visible breaking changes.
func TestAuditLogFormat(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// --- Part 1: Login (workspace-scoped, allow_without_credential) ---
	//
	// Clear the token so the Login call runs without credentials — the exact
	// path that regressed in 3.17.0 (Resources is empty, so the audit loop
	// never fires unless the handler hands us the workspace).
	adminToken := ctl.authInterceptor.token
	ctl.authInterceptor.token = ""

	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    "demo@example.com",
		Password: "1024bytebase",
	}))
	a.NoError(err)
	workspace := loginResp.Msg.GetUser().GetWorkspace()
	a.NotEmpty(workspace, "login response should carry the user's workspace")

	// Restore the token so the SearchAuditLogs call below authenticates.
	ctl.authInterceptor.token = adminToken

	loginAuditLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent:  workspace,
		Filter:  `method == "/bytebase.v1.AuthService/Login"`,
		OrderBy: "create_time desc",
	}))
	a.NoError(err)
	a.NotEmpty(loginAuditLogs.Msg.AuditLogs, "Login must produce an audit entry under the caller's workspace (regression guard for 3.17.0)")

	entry := loginAuditLogs.Msg.AuditLogs[0]
	// Name: "workspaces/{id}/auditLogs/{uid}" — the resource name format is
	// part of the API contract and also the parent users filter on.
	a.Regexp(regexp.MustCompile(`^workspaces/[^/]+/auditLogs/[^/]+$`), entry.Name,
		"audit log name must match workspaces/{id}/auditLogs/{uid}")
	a.True(strings.HasPrefix(entry.Name, workspace+"/auditLogs/"),
		"audit log must be parented under the login workspace")
	a.NotNil(entry.CreateTime, "CreateTime must be set")
	a.Equal("users/demo@example.com", entry.User, "User must be users/{email}")
	a.Equal("/bytebase.v1.AuthService/Login", entry.Method,
		"Method is part of the filter API contract and must be the full procedure name")
	a.Equal(v1pb.AuditLog_INFO, entry.Severity, "successful Login is INFO severity")
	a.Equal("demo@example.com", entry.Resource, "Login's Resource is the login email")
	a.Nil(entry.Status, "successful Login has nil Status (code 0)")
	a.NotNil(entry.Latency, "Latency must be recorded")

	// Request JSON must round-trip back to LoginRequest, have the caller's
	// email, and must NOT contain the plaintext password.
	gotReq := &v1pb.LoginRequest{}
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(entry.Request), gotReq),
		"Request JSON must be valid protojson for LoginRequest")
	a.Equal("demo@example.com", gotReq.Email)
	a.Empty(gotReq.Password, "password must be redacted in the Request payload")
	a.NotContains(entry.Request, "1024bytebase", "plaintext password must never appear in audit Request")

	// Response JSON must round-trip back to LoginResponse with the user info
	// but NO token (tokens are intentionally dropped from the audit payload).
	gotResp := &v1pb.LoginResponse{}
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(entry.Response), gotResp),
		"Response JSON must be valid protojson for LoginResponse")
	a.Equal("users/demo@example.com", gotResp.GetUser().GetName())
	a.Equal("demo@example.com", gotResp.GetUser().GetEmail())
	a.Empty(gotResp.Token, "token must be redacted from the Response payload")
	a.NotContains(entry.Response, loginResp.Msg.Token,
		"actual access token must never appear in audit Response")

	// Project creation happens during fixture setup. Select by resource because
	// setup and future fixtures may create more than one project.
	createProjectLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: workspace,
		Filter: `method == "/bytebase.v1.ProjectService/CreateProject"`,
	}))
	a.NoError(err)
	createProjectMatches := 0
	for _, auditLog := range createProjectLogs.Msg.AuditLogs {
		if auditLog.Resource == ctl.project.Name {
			createProjectMatches++
			a.True(strings.HasPrefix(auditLog.Name, workspace+"/auditLogs/"))
		}
	}
	a.Equal(1, createProjectMatches, "fixture project creation must produce exactly one matching workspace audit row")

	updatedTitle := ctl.project.Title + " updated"
	updatedProject, err := ctl.projectServiceClient.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
		Project: &v1pb.Project{
			Name:  ctl.project.Name,
			Title: updatedTitle,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
	}))
	a.NoError(err)
	ctl.project = updatedProject.Msg

	updateProjectLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: ctl.project.Name,
		Filter: `method == "/bytebase.v1.ProjectService/UpdateProject"`,
	}))
	a.NoError(err)
	a.Len(updateProjectLogs.Msg.AuditLogs, 1)
	a.Equal(ctl.project.Name, updateProjectLogs.Msg.AuditLogs[0].Resource)

	savedQueryContent := "SELECT secret_audit_saved_query_content;"
	savedQuery, err := ctl.savedQueryServiceClient.CreateSavedQuery(ctx, connect.NewRequest(&v1pb.CreateSavedQueryRequest{
		Parent: ctl.project.Name,
		SavedQuery: &v1pb.SavedQuery{
			Title:   "Audit saved query",
			Content: []byte(savedQueryContent),
		},
	}))
	a.NoError(err)

	createSavedQueryLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: ctl.project.Name,
		Filter: `method == "/bytebase.v1.SavedQueryService/CreateSavedQuery"`,
	}))
	a.NoError(err)
	a.Len(createSavedQueryLogs.Msg.AuditLogs, 1)
	a.Equal(ctl.project.Name, createSavedQueryLogs.Msg.AuditLogs[0].Resource)
	a.True(strings.HasPrefix(createSavedQueryLogs.Msg.AuditLogs[0].Name, ctl.project.Name+"/auditLogs/"))
	auditedCreateSavedQueryRequest := &v1pb.CreateSavedQueryRequest{}
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(createSavedQueryLogs.Msg.AuditLogs[0].Request), auditedCreateSavedQueryRequest))
	a.Empty(auditedCreateSavedQueryRequest.GetSavedQuery().GetContent())
	auditedSavedQuery := &v1pb.SavedQuery{}
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(createSavedQueryLogs.Msg.AuditLogs[0].Response), auditedSavedQuery))
	a.Empty(auditedSavedQuery.Content)

	_, err = ctl.savedQueryServiceClient.DeleteSavedQuery(ctx, connect.NewRequest(&v1pb.DeleteSavedQueryRequest{
		Name: savedQuery.Msg.Name,
	}))
	a.NoError(err)

	deleteSavedQueryLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: ctl.project.Name,
		Filter: `method == "/bytebase.v1.SavedQueryService/DeleteSavedQuery"`,
	}))
	a.NoError(err)
	a.Len(deleteSavedQueryLogs.Msg.AuditLogs, 1)
	a.Equal(savedQuery.Msg.Name, deleteSavedQueryLogs.Msg.AuditLogs[0].Resource)
	a.True(strings.HasPrefix(deleteSavedQueryLogs.Msg.AuditLogs[0].Name, ctl.project.Name+"/auditLogs/"))

	// --- Part 2: Signup (workspace-scoped, allow_without_credential) ---
	//
	// The initial `signupAndLogin` in setup already produced a Signup audit
	// entry; just assert it landed with the expected parent and method.
	signupAuditLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: workspace,
		Filter: `method == "/bytebase.v1.AuthService/Signup"`,
	}))
	a.NoError(err)
	a.NotEmpty(signupAuditLogs.Msg.AuditLogs, "Signup must produce an audit entry under the caller's workspace")
	signupEntry := signupAuditLogs.Msg.AuditLogs[0]
	a.True(strings.HasPrefix(signupEntry.Name, workspace+"/auditLogs/"))
	a.Equal("/bytebase.v1.AuthService/Signup", signupEntry.Method)
	a.Equal(v1pb.AuditLog_INFO, signupEntry.Severity)
	a.NotContains(signupEntry.Request, "1024bytebase",
		"plaintext password must never appear in Signup audit Request")

	// --- Part 2.25: Email-code login audit events ---
	// SendEmailLoginCode is attributed only for an existing active workspace
	// member. Signup-capable targets are audited by Login after authentication.
	workspaceName := workspace
	invitedEmail := "invited-email-code@example.com"
	_, err = ctl.addMemberToWorkspaceIAM(ctx, workspace, "user:"+invitedEmail, "roles/workspaceMember")
	a.NoError(err)

	ctl.authInterceptor.token = ""
	_, err = ctl.authServiceClient.SendEmailLoginCode(ctx, connect.NewRequest(&v1pb.SendEmailLoginCodeRequest{
		Email:     " Demo@Example.com ",
		Workspace: &workspaceName,
	}))
	a.Error(err)
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
	_, err = ctl.authServiceClient.SendEmailLoginCode(ctx, connect.NewRequest(&v1pb.SendEmailLoginCodeRequest{
		Email:     invitedEmail,
		Workspace: &workspaceName,
	}))
	a.Error(err)
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))

	ctl.authInterceptor.token = adminToken
	_, err = ctl.authServiceClient.SendEmailLoginCode(ctx, connect.NewRequest(&v1pb.SendEmailLoginCodeRequest{
		Email:     "unknown-authenticated-email-code@example.com",
		Workspace: &workspaceName,
	}))
	a.Error(err)
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))

	emailCodeLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: workspace,
		Filter: `method == "/bytebase.v1.AuthService/SendEmailLoginCode"`,
	}))
	a.NoError(err)
	a.Len(emailCodeLogs.Msg.AuditLogs, 1)
	emailCodeByResource := make(map[string]*v1pb.AuditLog)
	for _, auditLog := range emailCodeLogs.Msg.AuditLogs {
		emailCodeByResource[auditLog.Resource] = auditLog
	}
	a.Contains(emailCodeByResource, "demo@example.com")
	a.Equal(v1pb.AuditLog_INFO, emailCodeByResource["demo@example.com"].Severity)
	a.Equal(int32(connect.CodeFailedPrecondition), emailCodeByResource["demo@example.com"].GetStatus().GetCode())
	a.Empty(emailCodeByResource[invitedEmail], "an invite without an account has no SendEmailLoginCode audit row")
	a.Empty(emailCodeByResource["unknown-authenticated-email-code@example.com"])

	// --- Part 2.5: Password reset security events ---
	// Only active end users with a validated workspace are auditable. The
	// endpoint still returns success for every target to prevent enumeration.
	deletedUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{Email: "deleted-reset@example.com", Title: "Deleted reset user", Password: "1024bytebase"},
	}))
	a.NoError(err)
	_, err = ctl.userServiceClient.DeleteUser(ctx, connect.NewRequest(&v1pb.DeleteUserRequest{Name: deletedUser.Msg.Name}))
	a.NoError(err)

	serviceAccount, err := ctl.serviceAccountServiceClient.CreateServiceAccount(ctx, connect.NewRequest(&v1pb.CreateServiceAccountRequest{
		Parent:           workspace,
		ServiceAccountId: "audit-reset",
		ServiceAccount:   &v1pb.ServiceAccount{Title: "Audit reset service account"},
	}))
	a.NoError(err)
	workloadIdentity, err := ctl.workloadIdentityServiceClient.CreateWorkloadIdentity(ctx, connect.NewRequest(&v1pb.CreateWorkloadIdentityRequest{
		Parent:             workspace,
		WorkloadIdentityId: "audit-reset",
		WorkloadIdentity: &v1pb.WorkloadIdentity{
			Title: "Audit reset workload identity",
			WorkloadIdentityConfig: &v1pb.WorkloadIdentityConfig{
				ProviderType:     v1pb.WorkloadIdentityConfig_GITHUB,
				AllowedAudiences: []string{"audit-reset"},
				SubjectPattern:   "repo:bytebase/bytebase:*",
			},
		},
	}))
	a.NoError(err)

	_, err = ctl.addMemberToWorkspaceIAM(ctx, workspace, common.AllUsers, "roles/workspaceMember")
	a.NoError(err)

	passwordResetTargets := []string{
		"demo@example.com",
		"unknown-reset@example.com",
		deletedUser.Msg.Email,
		serviceAccount.Msg.Email,
		workloadIdentity.Msg.Email,
	}
	ctl.authInterceptor.token = ""
	for _, email := range passwordResetTargets {
		_, err := ctl.authServiceClient.RequestPasswordReset(ctx, connect.NewRequest(&v1pb.RequestPasswordResetRequest{
			Email:     email,
			Workspace: &workspaceName,
		}))
		a.NoError(err, "password-reset request must not reveal target validity")
	}
	_, err = ctl.authServiceClient.RequestPasswordReset(ctx, connect.NewRequest(&v1pb.RequestPasswordResetRequest{
		Email: "demo@example.com",
	}))
	a.NoError(err, "no-workspace password-reset request must preserve existing behavior")
	ctl.authInterceptor.token = adminToken
	for _, email := range passwordResetTargets[1:] {
		_, err := ctl.authServiceClient.RequestPasswordReset(ctx, connect.NewRequest(&v1pb.RequestPasswordResetRequest{
			Email:     email,
			Workspace: &workspaceName,
		}))
		a.NoError(err, "authenticated password-reset request must not reveal target validity")
	}
	_, err = ctl.authServiceClient.RequestPasswordReset(ctx, connect.NewRequest(&v1pb.RequestPasswordResetRequest{
		Email: "demo@example.com",
	}))
	a.NoError(err, "authenticated no-workspace request must preserve existing behavior")

	requestResetLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: workspace,
		Filter: `method == "/bytebase.v1.AuthService/RequestPasswordReset"`,
	}))
	a.NoError(err)
	requestResetByEmail := make(map[string][]*v1pb.AuditLog)
	for _, auditLog := range requestResetLogs.Msg.AuditLogs {
		requestResetByEmail[auditLog.Resource] = append(requestResetByEmail[auditLog.Resource], auditLog)
	}
	a.Len(requestResetByEmail["demo@example.com"], 1,
		"only the workspace-bound request for the active end user is auditable")
	for _, email := range passwordResetTargets[1:] {
		a.Empty(requestResetByEmail[email], "non-end-user target %q must not produce an audit row", email)
	}
	requestResetPayload := &v1pb.RequestPasswordResetRequest{}
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(requestResetByEmail["demo@example.com"][0].Request), requestResetPayload))
	a.Equal("demo@example.com", requestResetPayload.Email)
	a.Equal(workspace, requestResetPayload.GetWorkspace())

	metadataDB, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer metadataDB.Close()
	var authSecret string
	a.NoError(metadataDB.QueryRowContext(ctx, `SELECT payload->>'authSecret' FROM server_config LIMIT 1`).Scan(&authSecret))
	insertVerificationCode := func(email, purpose, code string, workspaceID sql.NullString) {
		mac := hmac.New(sha256.New, []byte(authSecret))
		_, err := mac.Write([]byte(code))
		a.NoError(err)
		_, err = metadataDB.ExecContext(ctx, `
			INSERT INTO email_verification_code (email, purpose, code_hash, expires_at, last_sent_at, workspace)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (email, purpose) DO UPDATE SET code_hash = EXCLUDED.code_hash, attempts = 0,
				expires_at = EXCLUDED.expires_at, last_sent_at = EXCLUDED.last_sent_at, workspace = EXCLUDED.workspace
		`, email, purpose, hex.EncodeToString(mac.Sum(nil)), time.Now().Add(time.Hour), time.Now(), workspaceID)
		a.NoError(err)
	}

	workspaceID, err := common.GetWorkspaceID(workspace)
	a.NoError(err)
	var invitedUserCount int
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_EMAIL.String(),
			Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_Email{Email: &v1pb.EmailSetting{
				From: "noreply@example.com",
				Type: v1pb.EmailSetting_SMTP,
				Config: &v1pb.EmailSetting_Smtp{Smtp: &v1pb.EmailSetting_SMTPConfig{
					Host:           "localhost",
					Port:           1,
					Encryption:     v1pb.EmailSetting_SMTPConfig_ENCRYPTION_NONE,
					Authentication: v1pb.EmailSetting_SMTPConfig_AUTHENTICATION_NONE,
				}},
			}}},
		},
		AllowMissing: true,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{
			"value.email.from",
			"value.email.type",
			"value.email.smtp",
		}},
	}))
	a.NoError(err)
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_WORKSPACE_PROFILE.String(),
			Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_WorkspaceProfile{WorkspaceProfile: &v1pb.WorkspaceProfileSetting{
				AllowEmailCodeSignin: true,
			}}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{
			"value.workspace_profile.allow_email_code_signin",
		}},
	}))
	a.NoError(err)
	loginCode := "765432"
	insertVerificationCode(invitedEmail, "LOGIN", loginCode, sql.NullString{String: workspaceID, Valid: true})
	ctl.authInterceptor.token = ""
	_, err = ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:     invitedEmail,
		EmailCode: &loginCode,
		Workspace: &workspaceName,
	}))
	a.NoError(err)
	ctl.authInterceptor.token = adminToken
	a.NoError(metadataDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM principal WHERE email = $1`, invitedEmail).Scan(&invitedUserCount))
	a.Equal(1, invitedUserCount)
	loginAuditLogs, err = ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent:  workspace,
		Filter:  `method == "/bytebase.v1.AuthService/Login"`,
		OrderBy: "create_time desc",
	}))
	a.NoError(err)
	var invitedLoginAudit *v1pb.AuditLog
	for _, auditLog := range loginAuditLogs.Msg.AuditLogs {
		if auditLog.Resource == invitedEmail {
			invitedLoginAudit = auditLog
			break
		}
	}
	a.NotNil(invitedLoginAudit, "successful email-code signup must be audited by Login")
	invitedLoginRequest := &v1pb.LoginRequest{}
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(invitedLoginAudit.Request), invitedLoginRequest))
	a.Equal(invitedEmail, invitedLoginRequest.Email)
	a.NotEqual(loginCode, invitedLoginRequest.GetEmailCode())
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		Setting: &v1pb.Setting{
			Name:  "settings/" + v1pb.Setting_WORKSPACE_PROFILE.String(),
			Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_WorkspaceProfile{WorkspaceProfile: &v1pb.WorkspaceProfileSetting{}}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{
			"value.workspace_profile.allow_email_code_signin",
		}},
	}))
	a.NoError(err)

	resetCode := "123456"
	newPassword := "new-password-1024"
	insertVerificationCode("demo@example.com", "PASSWORD_RESET", resetCode, sql.NullString{String: workspaceID, Valid: true})
	ctl.authInterceptor.token = ""
	_, err = ctl.authServiceClient.ResetPassword(ctx, connect.NewRequest(&v1pb.ResetPasswordRequest{
		Email:       "demo@example.com",
		Code:        resetCode,
		NewPassword: newPassword,
	}))
	a.NoError(err)
	ctl.authInterceptor.token = adminToken

	resetLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: workspace,
		Filter: `method == "/bytebase.v1.AuthService/ResetPassword"`,
	}))
	a.NoError(err)
	a.Len(resetLogs.Msg.AuditLogs, 1)
	a.Equal("demo@example.com", resetLogs.Msg.AuditLogs[0].Resource)
	resetPayload := &v1pb.ResetPasswordRequest{}
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(resetLogs.Msg.AuditLogs[0].Request), resetPayload))
	a.Equal("demo@example.com", resetPayload.Email)
	a.Empty(resetPayload.Code)
	a.Empty(resetPayload.NewPassword)
	a.NotContains(resetLogs.Msg.AuditLogs[0].Request, resetCode)
	a.NotContains(resetLogs.Msg.AuditLogs[0].Request, newPassword)

	failedResetCode := "654321"
	failedPassword := "short"
	insertVerificationCode("demo@example.com", "PASSWORD_RESET", failedResetCode, sql.NullString{String: workspaceID, Valid: true})
	ctl.authInterceptor.token = ""
	_, resetErr := ctl.authServiceClient.ResetPassword(ctx, connect.NewRequest(&v1pb.ResetPasswordRequest{
		Email:       "demo@example.com",
		Code:        failedResetCode,
		NewPassword: failedPassword,
	}))
	a.Error(resetErr, "password policy must reject a short password")
	ctl.authInterceptor.token = adminToken
	resetLogs, err = ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: workspace,
		Filter: `method == "/bytebase.v1.AuthService/ResetPassword"`,
	}))
	a.NoError(err)
	a.Len(resetLogs.Msg.AuditLogs, 2, "post-verification failures retain the validated workspace audit parent")
	var failedResetLog *v1pb.AuditLog
	for _, auditLog := range resetLogs.Msg.AuditLogs {
		if auditLog.Status != nil {
			failedResetLog = auditLog
			break
		}
	}
	a.NotNil(failedResetLog)
	a.Equal(int32(connect.CodeInvalidArgument), failedResetLog.Status.Code)
	a.NotContains(failedResetLog.Request, failedResetCode)
	a.NotContains(failedResetLog.Request, failedPassword)

	noWorkspaceUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{Email: "no-workspace-reset@example.com", Title: "No workspace reset", Password: "1024bytebase"},
	}))
	a.NoError(err)
	insertVerificationCode(noWorkspaceUser.Msg.Email, "PASSWORD_RESET", resetCode, sql.NullString{})
	ctl.authInterceptor.token = ""
	_, err = ctl.authServiceClient.ResetPassword(ctx, connect.NewRequest(&v1pb.ResetPasswordRequest{
		Email:       noWorkspaceUser.Msg.Email,
		Code:        resetCode,
		NewPassword: newPassword,
	}))
	a.NoError(err, "no-workspace reset must preserve existing API behavior")
	ctl.authInterceptor.token = adminToken
	insertVerificationCode(noWorkspaceUser.Msg.Email, "PASSWORD_RESET", resetCode, sql.NullString{})
	_, err = ctl.authServiceClient.ResetPassword(ctx, connect.NewRequest(&v1pb.ResetPasswordRequest{
		Email:       noWorkspaceUser.Msg.Email,
		Code:        resetCode,
		NewPassword: "another-new-password-1024",
	}))
	a.NoError(err, "authenticated no-workspace reset must preserve existing API behavior")
	resetLogs, err = ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: workspace,
		Filter: `method == "/bytebase.v1.AuthService/ResetPassword"`,
	}))
	a.NoError(err)
	a.Len(resetLogs.Msg.AuditLogs, 2, "a reset code without workspace context has no valid audit parent")

	// --- Part 2.5: Denied Signup is still audited ---
	//
	// Regression guard for #20024 (discussion_r3089978499): Signup failures
	// that return *before* the original SetAuditWorkspaceID call site (e.g.
	// DisallowSignup denial on invited/self-hosted signups) must still land
	// in audit. Signup uses a defer'd SetAuditWorkspaceID so every exit
	// path announces the resolved workspace.
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_WORKSPACE_PROFILE.String(),
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceProfile{
					WorkspaceProfile: &v1pb.WorkspaceProfileSetting{
						DisallowSignup: true,
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"value.workspace_profile.disallow_signup"},
		},
	}))
	a.NoError(err)

	// Signup is allow_without_credential; clear token for cleanliness.
	savedToken := ctl.authInterceptor.token
	ctl.authInterceptor.token = ""
	_, signupErr := ctl.authServiceClient.Signup(ctx, connect.NewRequest(&v1pb.SignupRequest{
		Email:    "denied@example.com",
		Password: "1024bytebase",
		Title:    "denied",
	}))
	a.Error(signupErr, "DisallowSignup must reject the new-user signup")
	ctl.authInterceptor.token = savedToken

	deniedSignupLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent:  workspace,
		Filter:  `method == "/bytebase.v1.AuthService/Signup"`,
		OrderBy: "create_time desc",
	}))
	a.NoError(err)

	var deniedEntry *v1pb.AuditLog
	for _, e := range deniedSignupLogs.Msg.AuditLogs {
		if e.Resource == "denied@example.com" {
			deniedEntry = e
			break
		}
	}
	a.NotNil(deniedEntry,
		"denied Signup must produce an audit entry under the workspace "+
			"(without the defer'd SetAuditWorkspaceID, the handler returns "+
			"before the workspace is announced and the entry is silently dropped)")
	a.NotNil(deniedEntry.Status, "denied Signup must carry a non-nil Status")
	a.NotEqual(int32(0), deniedEntry.Status.Code,
		"denied Signup must carry a non-zero status code")

	// --- Part 3: SetIamPolicy (project-scoped, authenticated) ---
	//
	// Project-scoped audit entries must land under projects/{id} (not under
	// the workspace). This is what compliance tooling filters on.
	projectResource := ctl.project.Name // "projects/test-project"

	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: projectResource,
	}))
	a.NoError(err)
	policy := policyResp.Msg
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{
		Role:    "roles/projectDeveloper",
		Members: []string{"user:demo@example.com"},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Etag:     policy.Etag,
		Policy:   policy,
		Resource: projectResource,
	}))
	a.NoError(err)

	projectAuditLogs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent:  projectResource,
		Filter:  `method == "/bytebase.v1.ProjectService/SetIamPolicy"`,
		OrderBy: "create_time desc",
	}))
	a.NoError(err)
	a.NotEmpty(projectAuditLogs.Msg.AuditLogs,
		"SetIamPolicy must produce an audit entry under projects/{id}")

	projEntry := projectAuditLogs.Msg.AuditLogs[0]
	a.Regexp(regexp.MustCompile(`^projects/[^/]+/auditLogs/[^/]+$`), projEntry.Name,
		"project audit log name must match projects/{id}/auditLogs/{uid}")
	a.True(strings.HasPrefix(projEntry.Name, projectResource+"/auditLogs/"),
		"audit entry must be parented under the target project, not the workspace")
	a.Equal("/bytebase.v1.ProjectService/SetIamPolicy", projEntry.Method)
	a.Equal("users/demo@example.com", projEntry.User)
	a.Equal(v1pb.AuditLog_INFO, projEntry.Severity)
	a.Equal(projectResource, projEntry.Resource,
		"SetIamPolicy's Resource is the target project name")
	a.Nil(projEntry.Status)
	a.NotNil(projEntry.Latency)

	gotSetReq := &v1pb.SetIamPolicyRequest{}
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(projEntry.Request), gotSetReq),
		"Request JSON must be valid protojson for SetIamPolicyRequest")
	a.Equal(projectResource, gotSetReq.Resource)
	a.NotNil(gotSetReq.Policy, "Policy must round-trip through the audit Request")

	gotIamPolicy := &v1pb.IamPolicy{}
	a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(projEntry.Response), gotIamPolicy),
		"Response JSON must be valid protojson for IamPolicy")
	// The updated binding must be visible in the recorded response.
	foundBinding := false
	for _, b := range gotIamPolicy.Bindings {
		if b.Role == "roles/projectDeveloper" {
			for _, m := range b.Members {
				if m == "user:demo@example.com" {
					foundBinding = true
				}
			}
		}
	}
	a.True(foundBinding, "Response JSON must reflect the new IAM binding")

	// A project audit entry must NOT appear when searching under the
	// workspace — verifies parent scoping is strict.
	workspaceSearch, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent: workspace,
		Filter: `method == "/bytebase.v1.ProjectService/SetIamPolicy"`,
	}))
	a.NoError(err)
	a.Empty(workspaceSearch.Msg.AuditLogs,
		"project-scoped SetIamPolicy audit must not leak into the workspace-scoped log stream")
}
