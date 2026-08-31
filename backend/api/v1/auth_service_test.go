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

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

const authTestEnterpriseLicense = "eyJhbGciOiJSUzI1NiIsImtpZCI6InYxIiwidHlwIjoiSldUIn0.eyJpbnN0YW5jZUNvdW50Ijo5OTksInRyaWFsaW5nIjpmYWxzZSwicGxhbiI6IkVOVEVSUFJJU0UiLCJvcmdOYW1lIjoiYmIiLCJhdWQiOiJiYi5saWNlbnNlIiwiZXhwIjo3OTc0OTc5MjAwLCJpYXQiOjE2NjM2Njc1NjEsImlzcyI6ImJ5dGViYXNlIiwic3ViIjoiMDAwMDEwMDAuIn0.JjYCMeAAMB9FlVeDFLdN3jvFcqtPsbEzaIm1YEDhUrfekthCbIOeX_DB2Bg2OUji3HSX5uDvG9AkK4Gtrc4gLMPI3D5mk3L-6wUKZ0L4REztS47LT4oxVhpqPQayYa9lKJB1YoHaqeMV4Z5FXeOXwuACoELznlwpT6pXo9xXm_I6QwQiO7-zD83XOTO4PRjByc-q3GKQu_64zJMIKiCW0I8a3GvrdSnO7jUuYU1KPmCuk0ZRq3I91m29LTo478BMST59HqCLj1GGuCKtR3SL_376XsZfUUM0iSAur5scg99zNGWRj-sUo05wbAadYx6V6TKaWrBUi_8_0RnJyP5gbA"

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
	stores := newAuthTestStore(t)

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
	stores := newAuthTestStore(t)

	_, err := stores.CreateWorkspace(ctx, &store.WorkspaceMessage{
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
		_, err := service.validateLoginPermissions(ctx, blockedAdmin, workspaceID, &v1pb.LoginRequest{
			Email:   blockedAdmin.Email,
			IdpName: "idps/test",
		})
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.ErrorContains(t, err, "does not belong to allowed domains")

		_, err = service.validateLoginPermissions(ctx, allowedAdmin, workspaceID, &v1pb.LoginRequest{
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
			CodeHash:   hashEmailCode(service.secret, code),
			ExpiresAt:  time.Now().Add(time.Minute),
			LastSentAt: time.Now(),
		}, 0)
		require.NoError(t, err)

		// The domain gate for existing users runs downstream in
		// validateLoginPermissions against the server-resolved workspace, so
		// exercise the full Login pipeline.
		_, err = service.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
			Email:     email,
			EmailCode: ptr(code),
		}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.ErrorContains(t, err, "does not belong to allowed domains")
	})

	t.Run("authenticate workspace admin", func(t *testing.T) {
		_, err := stores.UpsertEmailVerificationCodeIfCooldownExpired(ctx, &store.EmailVerificationCodeMessage{
			Email:      blockedAdmin.Email,
			Purpose:    storepb.EmailVerificationCodePurpose_LOGIN,
			CodeHash:   hashEmailCode(service.secret, code),
			ExpiresAt:  time.Now().Add(time.Minute),
			LastSentAt: time.Now(),
		}, 0)
		require.NoError(t, err)

		_, err = service.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
			Email:     blockedAdmin.Email,
			EmailCode: ptr(code),
		}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.ErrorContains(t, err, "does not belong to allowed domains")
	})

	t.Run("authenticate unknown user before provisioning", func(t *testing.T) {
		const email = "unknown@blocked.example"
		_, err := stores.UpsertEmailVerificationCodeIfCooldownExpired(ctx, &store.EmailVerificationCodeMessage{
			Email:      email,
			Purpose:    storepb.EmailVerificationCodePurpose_LOGIN,
			CodeHash:   hashEmailCode(service.secret, code),
			ExpiresAt:  time.Now().Add(time.Minute),
			LastSentAt: time.Now(),
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

func TestSignupChecksRestrictionBeforeExistence(t *testing.T) {
	const secret = "test-secret"

	ctx := context.Background()
	stores := newAuthTestStore(t)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)

	// The account must hold a workspace binding, not merely exist: signup
	// resolves the target workspace through FindWorkspace(email), so a member
	// resolves to a real workspace ID while an unknown address resolves to none.
	// A bare CreateUser resolves to none either way and would let a denial that
	// names the workspace pass unnoticed.
	takenUser, err := stores.CreateUser(ctx, &store.UserMessage{
		Email:        "taken@example.com",
		Name:         "Taken",
		PasswordHash: "unused",
		Profile:      &storepb.UserProfile{},
	})
	require.NoError(t, err)
	_, err = stores.CreateWorkspace(ctx, &store.WorkspaceMessage{
		ResourceID: "signup-order-test",
		Payload:    &storepb.WorkspacePayload{Title: "Signup order test"},
	}, takenUser.Email)
	require.NoError(t, err)

	signup := func(service *AuthService, email string) error {
		_, err := service.Signup(ctx, connect.NewRequest(&v1pb.SignupRequest{
			Email:    email,
			Title:    "Signup test",
			Password: "password-long-enough",
		}))
		return err
	}

	// SaaS forces DisallowSignup for every workspace, so the RPC can never
	// succeed there — and must not answer differently for an address that has
	// an account than for one that does not.
	t.Run("denied signup reveals nothing", func(t *testing.T) {
		service := NewAuthService(stores, secret, licenseService, &config.Profile{SaaS: true}, nil)
		takenErr := signup(service, "taken@example.com")
		unknownErr := signup(service, "unknown@example.com")
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(takenErr))
		require.Equal(t, connect.CodeOf(unknownErr), connect.CodeOf(takenErr))
		require.Equal(t, unknownErr.Error(), takenErr.Error())
	})

	// Where signup is allowed the duplicate is still reported: the reorder moved
	// the check behind the gates, it did not remove it.
	t.Run("allowed signup still reports a duplicate", func(t *testing.T) {
		service := NewAuthService(stores, secret, licenseService, &config.Profile{}, nil)
		require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(signup(service, "taken@example.com")))
	})
}

func TestSendEmailLoginCodeBudgetsWorkspacelessSends(t *testing.T) {
	const secret = "test-secret"

	ctx := context.Background()
	stores := newAuthTestStore(t)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)

	// Port 1 refuses immediately, so every send fails after its budget is spent
	// — the shape a real campaign hits once the SMTP host starts rejecting.
	t.Setenv("EMAIL_CONFIG", `{"from":"bytebase@example.com","type":"SMTP","smtp":{"host":"127.0.0.1","port":1,"encryption":"ENCRYPTION_NONE","authentication":"AUTHENTICATION_NONE"}}`)
	service := NewAuthService(stores, secret, licenseService, &config.Profile{SaaS: true}, nil)

	send := func(email string) error {
		_, err := service.SendEmailLoginCode(ctx, connect.NewRequest(&v1pb.SendEmailLoginCodeRequest{Email: email}))
		return err
	}

	// Distinct addresses so the per-recipient cooldown never masks the budget:
	// this is the attack the cooldown does not bound.
	for i := range emailCodeSendPerDomain {
		require.Equal(t, connect.CodeInternal, connect.CodeOf(send(fmt.Sprintf("victim%d@target.example", i))),
			"send %d is within budget and must fail only on delivery", i)
	}
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(send("victim-over@target.example")))

	// The bucket is the recipient domain, so an exhausted campaign does not
	// stop sign-in for anyone else.
	require.Equal(t, connect.CodeInternal, connect.CodeOf(send("someone@other.example")))
}

// TestSendEmailLoginCodeAnswersUniformlyWhenSignupDisallowed pins the other half
// of the redeemability gate: skipping the send for an address that has no
// account is only safe while skipping it is unobservable. A workspace that
// disallows signup cannot onboard an unknown address through an email code, so
// no code is sent to one — and the known address must not be told anything the
// unknown address isn't, including that delivery failed.
func TestSendEmailLoginCodeAnswersUniformlyWhenSignupDisallowed(t *testing.T) {
	const (
		workspaceID = "email-code-uniform-test"
		secret      = "test-secret"
	)

	ctx := context.Background()
	stores := newAuthTestStore(t)

	member, err := stores.CreateUser(ctx, &store.UserMessage{
		Email:        "member@example.com",
		Name:         "Member",
		PasswordHash: "unused",
		Profile:      &storepb.UserProfile{},
	})
	require.NoError(t, err)
	_, err = stores.CreateWorkspace(ctx, &store.WorkspaceMessage{
		ResourceID: workspaceID,
		Payload:    &storepb.WorkspacePayload{Title: "Email code uniform test"},
	}, member.Email)
	require.NoError(t, err)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_WORKSPACE_PROFILE,
		Workspace: workspaceID,
		Value: &storepb.WorkspaceProfileSetting{
			AllowEmailCodeSignin: true,
			DisallowSignup:       true,
			PasswordRestriction:  &storepb.WorkspaceProfileSetting_PasswordRestriction{MinLength: 8},
		},
	})
	require.NoError(t, err)
	// Port 1 refuses immediately, so the known address takes the delivery-failure
	// path — the branch that used to answer Internal while the unknown address
	// answered success.
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_EMAIL,
		Workspace: workspaceID,
		Value: &storepb.EmailSetting{
			From: "bytebase@example.com",
			Type: storepb.EmailSetting_SMTP,
			Config: &storepb.EmailSetting_Smtp{Smtp: &storepb.EmailSetting_SMTPConfig{
				Host:           "127.0.0.1",
				Port:           1,
				Encryption:     storepb.EmailSetting_SMTPConfig_ENCRYPTION_NONE,
				Authentication: storepb.EmailSetting_SMTPConfig_AUTHENTICATION_NONE,
			}},
		},
	})
	require.NoError(t, err)

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	require.NoError(t, licenseService.StoreLicense(ctx, workspaceID, authTestEnterpriseLicense))
	service := NewAuthService(stores, secret, licenseService, &config.Profile{}, nil)
	workspaceName := common.FormatWorkspace(workspaceID)

	for _, email := range []string{member.Email, "stranger@example.com"} {
		_, err := service.SendEmailLoginCode(ctx, connect.NewRequest(&v1pb.SendEmailLoginCodeRequest{
			Email:     email,
			Workspace: &workspaceName,
		}))
		require.NoError(t, err, "%s must not learn whether it has an account here", email)
	}

	// The stranger is not merely answered the same — no code was written for
	// them, so nothing was mailed anywhere on their behalf.
	row, err := stores.GetEmailVerificationCode(ctx, "stranger@example.com", storepb.EmailVerificationCodePurpose_LOGIN)
	require.NoError(t, err)
	require.Nil(t, row, "an address that could never redeem a code must not be sent one")
}

// TestSendEmailLoginCodeBudgetsWorkspaceStrangers covers the relay a named
// workspace still had: its SMTP will address anyone, since the domain
// restriction needs a license, EnforceIdentityDomain and a domain list before it
// narrows recipients at all. Mail to an address the workspace has no
// relationship with is budgeted; members and invitees are exempt, so the limit
// cannot lock a customer's own workforce out of sign-in.
func TestSendEmailLoginCodeBudgetsWorkspaceStrangers(t *testing.T) {
	const (
		workspaceID = "email-code-stranger-test"
		secret      = "test-secret"
	)

	ctx := context.Background()
	stores := newAuthTestStore(t)

	member, err := stores.CreateUser(ctx, &store.UserMessage{
		Email:        "member@corp.example",
		Name:         "Member",
		PasswordHash: "unused",
		Profile:      &storepb.UserProfile{},
	})
	require.NoError(t, err)
	_, err = stores.CreateWorkspace(ctx, &store.WorkspaceMessage{
		ResourceID: workspaceID,
		Payload:    &storepb.WorkspacePayload{Title: "Email code stranger test"},
	}, member.Email)
	require.NoError(t, err)
	// Signup stays allowed, so the redeemability gate does not apply and every
	// address below reaches the send path — this is the default self-hosted shape.
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_WORKSPACE_PROFILE,
		Workspace: workspaceID,
		Value: &storepb.WorkspaceProfileSetting{
			AllowEmailCodeSignin: true,
			PasswordRestriction:  &storepb.WorkspaceProfileSetting_PasswordRestriction{MinLength: 8},
		},
	})
	require.NoError(t, err)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_EMAIL,
		Workspace: workspaceID,
		Value: &storepb.EmailSetting{
			From: "bytebase@corp.example",
			Type: storepb.EmailSetting_SMTP,
			Config: &storepb.EmailSetting_Smtp{Smtp: &storepb.EmailSetting_SMTPConfig{
				Host:           "127.0.0.1",
				Port:           1,
				Encryption:     storepb.EmailSetting_SMTPConfig_ENCRYPTION_NONE,
				Authentication: storepb.EmailSetting_SMTPConfig_AUTHENTICATION_NONE,
			}},
		},
	})
	require.NoError(t, err)

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	service := NewAuthService(stores, secret, licenseService, &config.Profile{}, nil)
	workspaceName := common.FormatWorkspace(workspaceID)

	// A named workspace answers success for every address, so the response cannot
	// say who was charged. The bucket counter can: it is the observable that
	// distinguishes a send charged to the stranger budget from one exempted.
	charged := func(t *testing.T) int {
		t.Helper()
		var attempts int
		err := stores.GetDB().QueryRowContext(ctx, `
			SELECT COALESCE((SELECT attempts FROM login_attempt WHERE identity = $1 AND kind = $2), 0)
		`, "signup-code-workspace:"+workspaceID, storepb.LoginAttemptKind_EMAIL_CODE_SEND.String()).Scan(&attempts)
		require.NoError(t, err)
		return attempts
	}
	send := func(t *testing.T, email string) {
		t.Helper()
		_, err := service.SendEmailLoginCode(ctx, connect.NewRequest(&v1pb.SendEmailLoginCodeRequest{
			Email:     email,
			Workspace: &workspaceName,
		}))
		// SMTP refuses on port 1, so a send that reached delivery failed — and the
		// caller is told none of that, which is the contract under test.
		require.NoError(t, err, "%s must not learn what happened to the mail", email)
	}

	for i := range emailCodeSendPerWorkspace {
		send(t, fmt.Sprintf("stranger%d@elsewhere.example", i))
		require.Equal(t, i+1, charged(t), "each stranger spends one slot")
	}

	// The bucket is spent. Further strangers are refused, and the refusal is
	// indistinguishable from a send: no code row, and no distinct error.
	send(t, "stranger-over@elsewhere.example")
	row, err := stores.GetEmailVerificationCode(ctx, "stranger-over@elsewhere.example", storepb.EmailVerificationCodePurpose_LOGIN)
	require.NoError(t, err)
	require.Nil(t, row, "a refused send must leave no code behind")

	// The workspace's own member is never charged, so sign-in still works while
	// the stranger budget is empty — and the response is the same one a refused
	// stranger got, so the exemption is not observable.
	before := charged(t)
	send(t, member.Email)
	require.Equal(t, before, charged(t), "a member must not spend the stranger budget")

	// A deactivated principal keeps its workspace IAM binding, so a membership
	// check alone would exempt it forever — while Login refuses it, making every
	// code mailed there unusable. It is charged like any other stranger.
	deactivated, err := stores.CreateUser(ctx, &store.UserMessage{
		Email:        "former@corp.example",
		Name:         "Former",
		PasswordHash: "unused",
		Profile:      &storepb.UserProfile{},
	})
	require.NoError(t, err)
	_, err = stores.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: workspaceID,
		Member:    common.FormatUserEmail(deactivated.Email),
		Roles:     []string{common.FormatRole(store.WorkspaceMemberRole)},
	})
	require.NoError(t, err)
	del := true
	_, err = stores.UpdateUser(ctx, deactivated, &store.UpdateUserMessage{Delete: &del})
	require.NoError(t, err)

	// Refill the bucket so the charge is observable rather than hidden by the
	// exhausted-bucket path, then confirm the deactivated address is charged.
	require.NoError(t, stores.ClearLoginAttempt(ctx, "signup-code-workspace:"+workspaceID, storepb.LoginAttemptKind_EMAIL_CODE_SEND))
	send(t, deactivated.Email)
	require.Equal(t, 1, charged(t), "a deactivated address must be budgeted like a stranger, not exempted as a member")
}

// TestSendEmailLoginCodeBudgetKeepsExistingCode pins that a refused budget is
// inert. The upsert that stores a new code replaces whatever the recipient was
// holding, so claiming the budget after it would let an anonymous caller with an
// exhausted bucket destroy a pending code and send no replacement — denying code
// sign-in to everyone on that domain for the window.
func TestSendEmailLoginCodeBudgetKeepsExistingCode(t *testing.T) {
	const secret = "test-secret"

	ctx := context.Background()
	stores := newAuthTestStore(t)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)

	// Workspaceless sends run on the deployment mail identity; port 1 refuses, so
	// each call spends budget and reports Internal.
	t.Setenv("EMAIL_CONFIG", `{"from":"bytebase@example.com","type":"SMTP","smtp":{"host":"127.0.0.1","port":1,"encryption":"ENCRYPTION_NONE","authentication":"AUTHENTICATION_NONE"}}`)
	service := NewAuthService(stores, secret, licenseService, &config.Profile{SaaS: true}, nil)

	send := func(email string) error {
		_, err := service.SendEmailLoginCode(ctx, connect.NewRequest(&v1pb.SendEmailLoginCodeRequest{Email: email}))
		return err
	}

	// A code the victim is still holding, last sent long enough ago that the
	// resend cooldown has elapsed — the state in which the upsert would replace it.
	const victim = "victim@victim.example"
	const heldHash = "held-code-hash"
	sent, err := stores.UpsertEmailVerificationCodeIfCooldownExpired(ctx, &store.EmailVerificationCodeMessage{
		Email:      victim,
		Purpose:    storepb.EmailVerificationCodePurpose_LOGIN,
		CodeHash:   heldHash,
		ExpiresAt:  time.Now().Add(emailCodeExpiry),
		LastSentAt: time.Now().Add(-5 * time.Minute),
	}, emailCodeResendCooldown)
	require.NoError(t, err)
	require.True(t, sent)

	// Exhaust the victim's domain bucket from other addresses on that domain.
	for i := range emailCodeSendPerDomain {
		require.Equal(t, connect.CodeInternal, connect.CodeOf(send(fmt.Sprintf("filler%d@victim.example", i))))
	}
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(send("another@victim.example")))

	// Now the attack: a request for the victim, with the bucket empty.
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(send(victim)))
	row, err := stores.GetEmailVerificationCode(ctx, victim, storepb.EmailVerificationCodePurpose_LOGIN)
	require.NoError(t, err)
	require.NotNil(t, row, "a refused budget must not delete the code the recipient is holding")
	require.Equal(t, heldHash, row.CodeHash, "a refused budget must not replace the code the recipient is holding")
}
