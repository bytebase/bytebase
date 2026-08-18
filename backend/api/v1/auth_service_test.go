package v1

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

const authTestEnterpriseLicense = "eyJhbGciOiJSUzI1NiIsImtpZCI6InYxIiwidHlwIjoiSldUIn0.eyJpbnN0YW5jZUNvdW50Ijo5OTksInRyaWFsaW5nIjpmYWxzZSwicGxhbiI6IkVOVEVSUFJJU0UiLCJvcmdOYW1lIjoiYmIiLCJhdWQiOiJiYi5saWNlbnNlIiwiZXhwIjo3OTc0OTc5MjAwLCJpYXQiOjE2NjM2Njc1NjEsImlzcyI6ImJ5dGViYXNlIiwic3ViIjoiMDAwMDEwMDAuIn0.JjYCMeAAMB9FlVeDFLdN3jvFcqtPsbEzaIm1YEDhUrfekthCbIOeX_DB2Bg2OUji3HSX5uDvG9AkK4Gtrc4gLMPI3D5mk3L-6wUKZ0L4REztS47LT4oxVhpqPQayYa9lKJB1YoHaqeMV4Z5FXeOXwuACoELznlwpT6pXo9xXm_I6QwQiO7-zD83XOTO4PRjByc-q3GKQu_64zJMIKiCW0I8a3GvrdSnO7jUuYU1KPmCuk0ZRq3I91m29LTo478BMST59HqCLj1GGuCKtR3SL_376XsZfUUM0iSAur5scg99zNGWRj-sUo05wbAadYx6V6TKaWrBUi_8_0RnJyP5gbA"

func TestCountRecentLoginFailures(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('ws-test')`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })
	service := &AuthService{store: stores}

	const email = "member@example.com"
	createFailure := func(workspace, method, resource, user, message, response string) {
		t.Helper()
		require.NoError(t, stores.CreateAuditLog(ctx, workspace, &storepb.AuditLog{
			Method:   method,
			Resource: resource,
			User:     user,
			Response: response,
			Status: &statuspb.Status{
				Code:    int32(connect.CodeUnauthenticated),
				Message: message,
			},
		}))
	}

	createFailure("ws-test", v1connect.AuthServiceLoginProcedure, email, "", errMsgInvalidCredentials, "")
	createFailure("ws-test", v1connect.AuthServiceLoginProcedure, email, "", errMsgInvalidCredentials, "")
	createFailure("ws-test", v1connect.AuthServiceLoginProcedure, email, "", errMsgInvalidCredentials, "old")
	createFailure("ws-test", v1connect.AuthServiceLoginProcedure, email, "", errMsgInvalidMFACode, "")
	createFailure("ws-test", v1connect.AuthServiceLoginProcedure, email, "", errMsgInvalidRecoveryCode, "")
	createFailure("ws-test", v1connect.AuthServiceSwitchWorkspaceProcedure, "workspaces/ws-test", common.FormatUserEmail(email), errMsgInvalidMFACode, "")
	createFailure("ws-test", v1connect.AuthServiceSwitchWorkspaceProcedure, "workspaces/ws-test", common.FormatUserEmail(email), errMsgInvalidRecoveryCode, "")
	require.NoError(t, stores.CreateAuditLog(ctx, "ws-test", &storepb.AuditLog{
		Method:   v1connect.AuthServiceLoginProcedure,
		Resource: email,
		Status:   &statuspb.Status{},
	}))

	result, err := db.ExecContext(ctx, `UPDATE audit_log SET created_at = $1 WHERE payload->>'response' = 'old'`, time.Now().Add(-2*time.Hour))
	require.NoError(t, err)
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	require.EqualValues(t, 1, rowsAffected)

	passwordFailures, err := service.countRecentLoginFailures(ctx, email, time.Hour, errMsgInvalidCredentials)
	require.NoError(t, err)
	require.Equal(t, 2, passwordFailures)

	mfaFailures, err := service.countRecentLoginFailures(ctx, email, time.Hour, errMsgInvalidMFACode, errMsgInvalidRecoveryCode)
	require.NoError(t, err)
	require.Equal(t, 2, mfaFailures)
}

func TestLoginAnnouncesPreAuthenticationWorkspace(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('ws-test')`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	t.Run("self-hosted singleton", func(t *testing.T) {
		service := &AuthService{store: stores, profile: &config.Profile{}}
		var auditWorkspaceID string
		ctx := common.WithSetAuditWorkspaceID(ctx, func(workspaceID string) {
			auditWorkspaceID = workspaceID
		})
		_, err := service.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
			Email:    "unknown@example.com",
			Password: "wrong-password",
		}))
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		require.Equal(t, "ws-test", auditWorkspaceID)
	})

	t.Run("SaaS requested workspace", func(t *testing.T) {
		service := &AuthService{store: stores, profile: &config.Profile{SaaS: true}}
		var auditWorkspaceID string
		ctx := common.WithSetAuditWorkspaceID(ctx, func(workspaceID string) {
			auditWorkspaceID = workspaceID
		})
		workspace := common.FormatWorkspace("ws-test")
		_, err := service.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
			Email:     "unknown@example.com",
			Password:  "wrong-password",
			Workspace: &workspace,
		}))
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		require.Equal(t, "ws-test", auditWorkspaceID)
	})

	t.Run("SaaS requested workspace without membership", func(t *testing.T) {
		user, err := stores.CreateUser(ctx, &store.UserMessage{
			Email:        "member@example.com",
			Name:         "Member",
			PasswordHash: "wrong-password-hash",
			Profile:      &storepb.UserProfile{},
		})
		require.NoError(t, err)
		_, err = stores.CreateWorkspace(ctx, &store.WorkspaceMessage{
			ResourceID: "ws-member",
			Payload:    &storepb.WorkspacePayload{Title: "Member workspace"},
		}, user.Email)
		require.NoError(t, err)

		service := &AuthService{store: stores, profile: &config.Profile{SaaS: true}}
		var auditWorkspaceIDs []string
		ctx := common.WithSetAuditWorkspaceID(ctx, func(workspaceID string) {
			auditWorkspaceIDs = append(auditWorkspaceIDs, workspaceID)
		})
		workspace := common.FormatWorkspace("ws-test")
		_, err = service.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
			Email:     user.Email,
			Password:  "wrong-password",
			Workspace: &workspace,
		}))
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		require.Equal(t, []string{"ws-test"}, auditWorkspaceIDs)
	})

	t.Run("rejects nonexistent requested workspace", func(t *testing.T) {
		service := &AuthService{store: stores, profile: &config.Profile{}}
		var auditWorkspaceID string
		ctx := common.WithSetAuditWorkspaceID(ctx, func(workspaceID string) {
			auditWorkspaceID = workspaceID
		})
		workspace := common.FormatWorkspace("missing")
		_, err := service.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
			Email:     "unknown@example.com",
			Password:  "wrong-password",
			Workspace: &workspace,
		}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.Empty(t, auditWorkspaceID)
	})
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{
			domain: "www.google.com",
			want:   "google.com",
		},
		{
			domain: "code.google.com",
			want:   "google.com",
		},
		{
			domain: "code.google.com.cn",
			want:   "google.com.cn",
		},
		{
			domain: "google.com",
			want:   "google.com",
		},
	}

	for _, test := range tests {
		got := extractDomain(test.domain)
		if got != test.want {
			t.Errorf("extractDomain %s, got %s, want %s", test.domain, got, test.want)
		}
	}
}

func TestPasswordResetEmailSkipsDeletedUser(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	require.NoError(t, migrator.MigrateSchema(ctx, container.GetDB()))

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	user, err := stores.CreateUser(ctx, &store.UserMessage{
		Email:        "deleted@example.com",
		Name:         "Deleted user",
		PasswordHash: "unused",
		Profile:      &storepb.UserProfile{},
	})
	require.NoError(t, err)
	deleted := true
	_, err = stores.UpdateUser(ctx, user, &store.UpdateUserMessage{Delete: &deleted})
	require.NoError(t, err)

	t.Setenv("EMAIL_CONFIG", "")
	service := &AuthService{store: stores, secret: "test-secret"}
	require.NoError(t, service.sendEmailVerificationCode(
		ctx,
		"",
		user.Email,
		storepb.EmailVerificationCodePurpose_PASSWORD_RESET,
		"subject",
		"code: %s, expires in %d minutes",
	))
}

func TestLoginEnforcesWorkspaceDomains(t *testing.T) {
	const (
		workspaceID = "email-code-domain-test"
		secret      = "test-secret"
		code        = "123456"
	)

	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	require.NoError(t, migrator.MigrateSchema(ctx, container.GetDB()))

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	_, err = stores.CreateWorkspace(ctx, &store.WorkspaceMessage{
		ResourceID: workspaceID,
		Payload:    &storepb.WorkspacePayload{Title: "Email code domain test"},
	}, "admin@allowed.example")
	require.NoError(t, err)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_WORKSPACE_PROFILE,
		Workspace: workspaceID,
		Value: &storepb.WorkspaceProfileSetting{
			Domains:                []string{"allowed.example"},
			EnforceIdentityDomain:  true,
			AllowEmailCodeSignin:   true,
			PasswordRestriction:    &storepb.WorkspaceProfileSetting_PasswordRestriction{MinLength: 8},
			EnableMetricCollection: true,
			DisallowSignup:         false,
			DisallowPasswordSignin: true,
		},
	})
	require.NoError(t, err)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_EMAIL,
		Workspace: workspaceID,
		Value:     &storepb.EmailSetting{},
	})
	require.NoError(t, err)

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	require.NoError(t, licenseService.StoreLicense(ctx, workspaceID, authTestEnterpriseLicense))
	service := NewAuthService(stores, secret, licenseService, &config.Profile{}, nil)
	workspaceName := common.FormatWorkspace(workspaceID)

	allowedAdmin, err := stores.CreateUser(ctx, &store.UserMessage{
		Email:        "admin@allowed.example",
		Name:         "Allowed admin",
		PasswordHash: "unused",
		Profile:      &storepb.UserProfile{},
	})
	require.NoError(t, err)
	blockedAdmin, err := stores.CreateUser(ctx, &store.UserMessage{
		Email:        "admin@blocked.example",
		Name:         "Blocked admin",
		PasswordHash: "unused",
		Profile:      &storepb.UserProfile{},
	})
	require.NoError(t, err)
	_, err = stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: workspaceID,
		Member:    common.FormatUserEmail(blockedAdmin.Email),
		Roles:     []string{common.FormatRole(store.WorkspaceAdminRole)},
	})
	require.NoError(t, err)

	t.Run("workspace admin", func(t *testing.T) {
		err := service.validateLoginPermissions(ctx, blockedAdmin, workspaceID, &v1pb.LoginRequest{
			Email:   blockedAdmin.Email,
			IdpName: "idps/test",
		})
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.ErrorContains(t, err, "does not belong to allowed domains")

		err = service.validateLoginPermissions(ctx, allowedAdmin, workspaceID, &v1pb.LoginRequest{
			Email:    allowedAdmin.Email,
			Password: "password",
		})
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
		require.ErrorContains(t, err, "password signin is disallowed")
	})

	t.Run("send code", func(t *testing.T) {
		for _, email := range []string{"new@blocked.example", blockedAdmin.Email} {
			_, err := service.SendEmailLoginCode(ctx, connect.NewRequest(&v1pb.SendEmailLoginCodeRequest{
				Email:     email,
				Workspace: &workspaceName,
			}))
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			require.ErrorContains(t, err, "does not belong to allowed domains")

			row, err := stores.GetEmailVerificationCode(ctx, email, storepb.EmailVerificationCodePurpose_LOGIN)
			require.NoError(t, err)
			require.Nil(t, row)
		}
	})

	t.Run("authenticate existing user", func(t *testing.T) {
		const email = "existing@blocked.example"
		_, err := stores.CreateUser(ctx, &store.UserMessage{
			Email:        email,
			Name:         "Existing user",
			PasswordHash: "unused",
			Profile:      &storepb.UserProfile{},
		})
		require.NoError(t, err)
		_, err = stores.UpsertEmailVerificationCodeIfCooldownExpired(ctx, &store.EmailVerificationCodeMessage{
			Email:      email,
			Purpose:    storepb.EmailVerificationCodePurpose_LOGIN,
			CodeHash:   service.hashEmailCode(code),
			ExpiresAt:  time.Now().Add(time.Minute),
			LastSentAt: time.Now(),
			Workspace:  workspaceID,
		}, 0)
		require.NoError(t, err)

		_, err = service.authenticateEmailCodeLogin(ctx, &v1pb.LoginRequest{
			Email:     email,
			EmailCode: ptr(code),
		})
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.ErrorContains(t, err, "does not belong to allowed domains")
	})

	t.Run("authenticate workspace admin", func(t *testing.T) {
		_, err := stores.UpsertEmailVerificationCodeIfCooldownExpired(ctx, &store.EmailVerificationCodeMessage{
			Email:      blockedAdmin.Email,
			Purpose:    storepb.EmailVerificationCodePurpose_LOGIN,
			CodeHash:   service.hashEmailCode(code),
			ExpiresAt:  time.Now().Add(time.Minute),
			LastSentAt: time.Now(),
			Workspace:  workspaceID,
		}, 0)
		require.NoError(t, err)

		_, err = service.authenticateEmailCodeLogin(ctx, &v1pb.LoginRequest{
			Email:     blockedAdmin.Email,
			EmailCode: ptr(code),
		})
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.ErrorContains(t, err, "does not belong to allowed domains")
	})

	t.Run("authenticate unknown user before provisioning", func(t *testing.T) {
		const email = "unknown@blocked.example"
		_, err := stores.UpsertEmailVerificationCodeIfCooldownExpired(ctx, &store.EmailVerificationCodeMessage{
			Email:      email,
			Purpose:    storepb.EmailVerificationCodePurpose_LOGIN,
			CodeHash:   service.hashEmailCode(code),
			ExpiresAt:  time.Now().Add(time.Minute),
			LastSentAt: time.Now(),
			Workspace:  workspaceID,
		}, 0)
		require.NoError(t, err)

		_, err = service.authenticateEmailCodeLogin(ctx, &v1pb.LoginRequest{
			Email:     email,
			EmailCode: ptr(code),
		})
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.ErrorContains(t, err, "does not belong to allowed domains")

		user, err := stores.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		require.Nil(t, user)

		policy, err := stores.GetWorkspaceIamPolicy(ctx, workspaceID)
		require.NoError(t, err)
		for _, binding := range policy.Policy.Bindings {
			require.NotContains(t, binding.Members, common.FormatUserEmail(email))
		}
	})
}

// TestSwitchWorkspaceMCPRecognition pins the SwitchWorkspace guard predicate
// across both MCP credential generations. An MCP session must never mint a
// plain user token: that token is not audience-bound to the MCP resource, does
// not die with the OAuth grant, and ignores the workspace MCP kill switch.
//
// The delegated-credential row is the P1a PR 4 half. Tool traffic now rides the
// internal transport, so the bearer this guard sees is the delegated
// credential, not the client's MCP token — and it is signed with a derived key,
// invisible to the raw-secret extraction the guard used to do (asserted below).
// Recognizing only extractable claims would fail OPEN for every MCP session.
func TestSwitchWorkspaceMCPRecognition(t *testing.T) {
	const secret = "test-secret"

	delegated, err := auth.GenerateInternalMCPToken(auth.DelegatedMCPCredential{
		Principal:   "demo@example.com",
		WorkspaceID: "ws-test",
		ClientID:    "client-A",
	}, secret)
	require.NoError(t, err)
	_, extractErr := auth.ExtractClaimsFromExpiredToken(delegated, secret)
	require.Error(t, extractErr, "the delegated credential must not verify under the raw secret")
	require.True(t, auth.IsMCPOriginatedToken(delegated, secret),
		"the delegated credential must be recognized as MCP-originated")

	// Current external MCP token: recognized by token_use, since its audience is
	// a per-deployment resource URI that cannot be matched by value.
	mcpToken, err := auth.GenerateOAuth2AccessToken("demo@example.com", "client-A", "ws-test", "https://bb.example.com/mcp", "", secret, time.Hour)
	require.NoError(t, err)
	require.True(t, auth.IsMCPOriginatedToken(mcpToken, secret))

	// Pre-3.23 external token: recognized by the fixed legacy audience.
	require.True(t, auth.IsMCPOriginatedToken(mustLegacyOAuth2Token(t, secret), secret))

	webToken, err := auth.GenerateAccessToken("demo@example.com", "ws-test", secret, time.Hour)
	require.NoError(t, err)
	require.False(t, auth.IsMCPOriginatedToken(webToken, secret),
		"a web session token must stay eligible to switch workspaces")
	require.False(t, auth.IsMCPOriginatedToken("", secret))
	require.False(t, auth.IsMCPOriginatedToken("not-a-jwt", secret))
}

// mustLegacyOAuth2Token mints a pre-PR-3 fixed-audience OAuth2 token.
func mustLegacyOAuth2Token(t *testing.T, secret string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss":          "bytebase",
		"sub":          "demo@example.com",
		"aud":          auth.OAuth2AccessTokenAudience,
		"workspace_id": "ws-test",
		"exp":          time.Now().Add(time.Hour).Unix(),
		"iat":          time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = "v1"
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenStr
}

func TestLoginAuthMethodRequiresPasswordReset(t *testing.T) {
	emailCode := "123456"
	tests := []struct {
		name    string
		request *v1pb.LoginRequest
		want    bool
	}{
		{
			name:    "password login enforces password reset",
			request: &v1pb.LoginRequest{Email: "user@example.com", Password: "password"},
			want:    true,
		},
		{
			name:    "idp login skips password reset",
			request: &v1pb.LoginRequest{IdpName: "idps/okta"},
			want:    false,
		},
		{
			name:    "email code login skips password reset",
			request: &v1pb.LoginRequest{Email: "user@example.com", EmailCode: &emailCode},
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := loginAuthMethodFromRequest(test.request).requiresPasswordReset()
			require.Equal(t, test.want, got)
		})
	}
}

func TestMFATempTokenPreservesLoginAuthMethod(t *testing.T) {
	const secret = "test-secret"

	tests := []struct {
		name       string
		method     loginAuthMethod
		wantReset  bool
		wantMethod loginAuthMethod
	}{
		{
			name:       "password mfa completion enforces password reset",
			method:     loginAuthMethodPassword,
			wantReset:  true,
			wantMethod: loginAuthMethodPassword,
		},
		{
			name:       "idp mfa completion skips password reset",
			method:     loginAuthMethodIDP,
			wantReset:  false,
			wantMethod: loginAuthMethodIDP,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := auth.GenerateMFATempTokenWithLoginMethod("user@example.com", string(test.method), secret, time.Minute)
			require.NoError(t, err)

			email, method, err := loginAuthMethodFromMFATempToken(token, secret)
			require.NoError(t, err)
			require.Equal(t, "user@example.com", email)
			require.Equal(t, test.wantMethod, method)
			require.Equal(t, test.wantReset, method.requiresPasswordReset())
		})
	}
}

func TestLegacyMFATempTokenDefaultsToPasswordLoginAuthMethod(t *testing.T) {
	const secret = "test-secret"

	token, err := auth.GenerateMFATempToken("user@example.com", secret, time.Minute)
	require.NoError(t, err)

	email, method, err := loginAuthMethodFromMFATempToken(token, secret)
	require.NoError(t, err)
	require.Equal(t, "user@example.com", email)
	require.Equal(t, loginAuthMethodPassword, method)
	require.True(t, method.requiresPasswordReset())
}

// TestSwitchWorkspaceInternalRefusesMCPCaller pins the refusal at the mint.
// The three handlers that reach switchWorkspaceInternal each refuse
// MCP-originated callers up front, where the refusal still precedes their
// mutations; the check inside switchWorkspaceInternal is the one that keeps a
// future caller from reopening the escape the way LeaveWorkspace and
// DeleteWorkspace did. Because the handlers refuse first, no e2e reaches it,
// so it needs its own pin.
//
// The AuthService here deliberately carries a nil store: the guard has to
// return before generateLoginToken touches it, so a regression panics rather
// than quietly minting a token.
func TestSwitchWorkspaceInternalRefusesMCPCaller(t *testing.T) {
	const secret = "test-secret"
	s := &AuthService{secret: secret}
	user := &store.UserMessage{Email: "demo@example.com"}

	delegated, err := auth.GenerateInternalMCPToken(auth.DelegatedMCPCredential{
		Principal:   "demo@example.com",
		WorkspaceID: "ws-test",
		ClientID:    "client-A",
	}, secret)
	require.NoError(t, err)
	mcpHeaders := http.Header{}
	mcpHeaders.Set("Authorization", "Bearer "+delegated)

	resp, err := s.switchWorkspaceInternal(context.Background(), user, "ws-test", false, mcpHeaders)
	require.Nil(t, resp, "an MCP-originated caller must never reach the mint")
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	// A caller that forwards no headers must be refused just the same: on the
	// internal transport the AuthContext carries the delegated grant, and its
	// presence — not any field value — marks the request MCP-originated.
	ctx := context.WithValue(context.Background(), common.AuthContextKey,
		&common.AuthContext{DelegatedGrant: &common.DelegatedGrant{}})
	resp, err = s.switchWorkspaceInternal(ctx, user, "ws-test", false, http.Header{})
	require.Nil(t, resp, "the grant in the AuthContext must refuse on its own")
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	// Control: a plain web session on the public chain carries no grant and is
	// not MCP-originated, so the guard lets it through to the mint.
	webToken, err := auth.GenerateAccessToken("demo@example.com", "ws-test", secret, time.Hour)
	require.NoError(t, err)
	webHeaders := http.Header{}
	webHeaders.Set("Authorization", "Bearer "+webToken)
	require.NoError(t, s.rejectMCPOriginatedTokenMint(context.Background(), webHeaders, "obtain a workspace token"))
}
