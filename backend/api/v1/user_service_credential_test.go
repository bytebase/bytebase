package v1

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// TestEmailCodeEligibility pins the deployment split for the email_code proof
// channel: Cloud only — self-hosted keeps an administrator who can reset a
// password instead — and never against a live MFA factor. The SaaS side has
// no e2e harness, so the rule is pinned here.
func TestEmailCodeEligibility(t *testing.T) {
	userWith := func(otpSecret string, lastChangePassword bool) *store.UserMessage {
		profile := &storepb.UserProfile{}
		if lastChangePassword {
			profile.LastChangePasswordTime = timestamppb.Now()
		}
		return &store.UserMessage{
			MFAConfig: &storepb.MFAConfig{OtpSecret: otpSecret},
			Profile:   profile,
		}
	}

	saas := &UserService{profile: &config.Profile{SaaS: true}}
	selfHosted := &UserService{profile: &config.Profile{SaaS: false}}

	require.NoError(t, saas.checkEmailCodeEligible(userWith("", false)),
		"a Cloud account without MFA is exactly who the channel exists for")
	err := saas.checkEmailCodeEligible(userWith("live-secret", false))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err),
		"a mailbox code must never stand in for a live factor")

	for _, user := range []*store.UserMessage{
		userWith("", false),
		userWith("", true),
		userWith("live-secret", false),
	} {
		err := selfHosted.checkEmailCodeEligible(user)
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err),
			"self-hosted has no email capability, so the channel does not exist there for any account")
		require.ErrorContains(t, err, "Cloud")
	}
}

// TestFirstTimeEnrollmentCredentialShape pins which first-time enrollments
// must carry a password proof, now that the answer comes from the deployment
// rather than from a stored timestamp: self-hosted always, because the
// password is the only proof channel there, and Cloud never, because no
// account has a caller-chosen password and the emailed code is spent at
// promotion instead.
//
// The self-hosted refusal has to name the administrator-reset route: an
// SSO-provisioned account holds a random password nobody was told, and the
// server cannot tell it apart from one whose owner simply did not type it.
func TestFirstTimeEnrollmentCredentialShape(t *testing.T) {
	saas := &UserService{profile: &config.Profile{SaaS: true}}
	selfHosted := &UserService{profile: &config.Profile{SaaS: false}}
	enrolling := &store.UserMessage{
		MFAConfig: &storepb.MFAConfig{TempOtpSecret: "pending"},
		Profile:   &storepb.UserProfile{},
	}
	password := &v1pb.CredentialProof{
		Proof: &v1pb.CredentialProof_CurrentPassword{CurrentPassword: "hunter2"},
	}

	err := selfHosted.checkEnableMFACredentialShape(enrolling, nil)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.ErrorContains(t, err, "ask a workspace admin")
	require.NoError(t, selfHosted.checkEnableMFACredentialShape(enrolling, password))

	require.NoError(t, saas.checkEnableMFACredentialShape(enrolling, nil),
		"Cloud proves a first-time enrollment with the emailed code at promotion")

	// A live factor is proven with the factor on either deployment.
	rotating := &store.UserMessage{
		MFAConfig: &storepb.MFAConfig{OtpSecret: "live", TempOtpSecret: "pending"},
		Profile:   &storepb.UserProfile{},
	}
	for _, s := range []*UserService{saas, selfHosted} {
		require.Error(t, s.checkEnableMFACredentialShape(rotating, nil))
	}
}
