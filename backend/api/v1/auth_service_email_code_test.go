package v1

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// spendSendBudget drives the deployment bucket to its cap without sending mail.
// Claims until refused rather than a fixed count, since a send earlier in the
// test may already have taken a slot.
func spendSendBudget(ctx context.Context, t *testing.T, stores *store.Store) {
	t.Helper()
	claim := func() bool {
		granted, err := stores.ClaimAttempt(ctx, signinCodeSendBudgetKey, storepb.LoginAttemptKind_EMAIL_CODE_SEND, emailCodeSendPerWindow, emailCodeSendWindow)
		require.NoError(t, err)
		return granted
	}
	for range emailCodeSendPerWindow {
		if !claim() {
			return
		}
	}
	require.False(t, claim(), "the bucket must be spent")
}

// newBudgetTestService builds a SaaS service whose mail host refuses on connect,
// so Internal means a send reached delivery and ResourceExhausted means it was
// refused a budget slot.
func newBudgetTestService(t *testing.T, stores *store.Store) *AuthService {
	t.Helper()
	t.Setenv("EMAIL_CONFIG", `{"from":"bytebase@example.com","type":"SMTP","smtp":{"host":"127.0.0.1","port":1,"encryption":"ENCRYPTION_NONE","authentication":"AUTHENTICATION_NONE"}}`)
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	return NewAuthService(stores, "test-secret", licenseService, &config.Profile{SaaS: true}, nil)
}

// TestSendEmailLoginCodeBudgetsEverySender pins that naming a workspace does not
// buy a separate budget. SaaS copies EMAIL_CONFIG into every workspace it
// creates, so a named send leaves over the same deployment sender; giving each
// workspace its own bucket would meter a shared reputation per tenant, and the
// workspace is caller-supplied besides.
func TestSendEmailLoginCodeBudgetsEverySender(t *testing.T) {
	const workspaceID = "email-code-shared-sender"

	ctx := context.Background()
	stores := newAuthTestStore(t)
	_, err := stores.CreateWorkspace(ctx, &store.WorkspaceMessage{
		ResourceID: workspaceID,
		Payload:    &storepb.WorkspacePayload{Title: "Shared sender"},
	}, "admin@corp.example")
	require.NoError(t, err)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_WORKSPACE_PROFILE,
		Workspace: workspaceID,
		Value: &storepb.WorkspaceProfileSetting{
			AllowEmailCodeSignin: true,
			PasswordRestriction:  &storepb.WorkspaceProfileSetting_PasswordRestriction{MinLength: 8},
		},
	})
	require.NoError(t, err)
	// The workspace carries the deployment's own mail config, as SaaS signup writes it.
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

	service := newBudgetTestService(t, stores)
	workspaceName := common.FormatWorkspace(workspaceID)
	spendSendBudget(ctx, t, stores)

	_, err = service.SendEmailLoginCode(ctx, connect.NewRequest(&v1pb.SendEmailLoginCodeRequest{
		Email:     "stranger@elsewhere.example",
		Workspace: &workspaceName,
	}))
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err),
		"naming a workspace must not escape the deployment budget")
}

// TestSendEmailLoginCodeBudgetsWorkspacelessSends pins the cap on the path a
// brand-new signup takes, where the 60s per-recipient cooldown never fires
// because a campaign writes each address exactly once.
func TestSendEmailLoginCodeBudgetsWorkspacelessSends(t *testing.T) {
	ctx := context.Background()
	stores := newAuthTestStore(t)
	service := newBudgetTestService(t, stores)

	send := func(email string) error {
		_, err := service.SendEmailLoginCode(ctx, connect.NewRequest(&v1pb.SendEmailLoginCodeRequest{Email: email}))
		return err
	}

	require.Equal(t, connect.CodeInternal, connect.CodeOf(send("first@target.example")),
		"a send within budget must fail only on delivery")
	spendSendBudget(ctx, t, stores)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(send("over@target.example")))
	// The bucket is the sender, so a spent budget stops the next send whatever
	// address it names.
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(send("someone@other.example")))
}

// TestSendEmailLoginCodeBudgetKeepsExistingCode pins that a refused budget is
// inert: the upsert replaces the stored hash, so claiming after it would let an
// exhausted bucket destroy a pending code and send no replacement. It also pins
// that the refusal does not depend on the recipient — an address mid-cooldown is
// answered exactly as any other, or an exhausted bucket would let a caller probe
// who had just been sent a code.
func TestSendEmailLoginCodeBudgetKeepsExistingCode(t *testing.T) {
	ctx := context.Background()
	stores := newAuthTestStore(t)
	service := newBudgetTestService(t, stores)

	send := func(email string) error {
		_, err := service.SendEmailLoginCode(ctx, connect.NewRequest(&v1pb.SendEmailLoginCodeRequest{Email: email}))
		return err
	}

	const victim = "victim@victim.example"
	const heldHash = "held-code-hash"
	seed := func(email, hash string, lastSent time.Time) {
		t.Helper()
		sent, err := stores.UpsertEmailVerificationCodeIfCooldownExpired(ctx, &store.EmailVerificationCodeMessage{
			Email:      email,
			Purpose:    storepb.EmailVerificationCodePurpose_LOGIN,
			CodeHash:   hash,
			ExpiresAt:  time.Now().Add(emailCodeExpiry),
			LastSentAt: lastSent,
		}, emailCodeResendCooldown)
		require.NoError(t, err)
		require.True(t, sent)
	}
	// A held code the upsert would replace once the cooldown passes.
	seed(victim, heldHash, time.Now().Add(-5*time.Minute))
	seed("recent@victim.example", "recent-code-hash", time.Now())

	spendSendBudget(ctx, t, stores)

	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(send(victim)))
	row, err := stores.GetEmailVerificationCode(ctx, victim, storepb.EmailVerificationCodePurpose_LOGIN)
	require.NoError(t, err)
	require.NotNil(t, row, "a refused budget must not delete the code the recipient is holding")
	require.Equal(t, heldHash, row.CodeHash, "a refused budget must not replace it either")

	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(send("recent@victim.example")),
		"an address mid-cooldown must be answered as any other is")
}
