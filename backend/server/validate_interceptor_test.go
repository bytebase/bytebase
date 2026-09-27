package server

import (
	"strings"
	"testing"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/proto"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestProtovalidateStandardRules guards the cel-go/protovalidate version pair.
// protovalidate compiles buf.validate's standard rules (string.max_len and
// friends) into CEL programs, so a cel-go bump ahead of what protovalidate
// supports still builds but fails at request time with
// "no such attribute(s): rules" — turning every validated RPC into
// invalid_argument. Keep the versions moving together.
func TestProtovalidateStandardRules(t *testing.T) {
	v, err := protovalidate.New()
	if err != nil {
		t.Fatalf("protovalidate.New: %v", err)
	}
	// Issue.title carries (buf.validate.field).string.max_len = 200.
	if err := v.Validate(&v1pb.Issue{Title: "hello"}); err != nil {
		t.Fatalf("validating a legal Issue.title failed: %v", err)
	}
	if err := v.Validate(&v1pb.Issue{Title: string(make([]byte, 201))}); err == nil {
		t.Fatal("expected string.max_len violation for a 201-byte title, got nil")
	}
}

// TestAuthEmailFieldsAreLengthBounded pins the edge half of the login-attempt
// identity bound (docs/design/login-attempt-lockout.md): every auth request
// field that becomes a lockout identity carries string.max_len = 254, so
// oversized identities are refused by the validate interceptor before any
// handler runs. The service still bounds server-composed identities (LDAP,
// MFA temp token) in claimLoginAttempt.
func TestAuthEmailFieldsAreLengthBounded(t *testing.T) {
	v, err := protovalidate.New()
	if err != nil {
		t.Fatalf("protovalidate.New: %v", err)
	}
	longEmail := strings.Repeat("a", 243) + "@example.com" // 255 chars
	okEmail := strings.Repeat("a", 242) + "@example.com"   // 254 chars
	for _, msg := range []interface {
		proto.Message
	}{
		&v1pb.LoginRequest{Email: longEmail},
		&v1pb.SignupRequest{Email: longEmail},
		&v1pb.RequestPasswordResetRequest{Email: longEmail},
		&v1pb.ResetPasswordRequest{Email: longEmail},
		&v1pb.SendEmailLoginCodeRequest{Email: longEmail},
	} {
		if err := v.Validate(msg); err == nil {
			t.Errorf("expected string.max_len violation for 255-char email on %T, got nil", msg)
		}
	}
	if err := v.Validate(&v1pb.LoginRequest{Email: okEmail}); err != nil {
		t.Errorf("a 254-char email must pass: %v", err)
	}
}

// TestAuthRequestFieldsAreSizeBounded pins the size bounds on the remaining
// auth request strings: passwords at bcrypt's 72-byte hard limit, submitted
// codes at 64 chars, server-minted and IdP-issued tokens at generous caps, and
// resource names at 256 — so no unauthenticated request can push an oversized
// value into hashing, comparison, or lockout bookkeeping.
func TestAuthRequestFieldsAreSizeBounded(t *testing.T) {
	v, err := protovalidate.New()
	if err != nil {
		t.Fatalf("protovalidate.New: %v", err)
	}
	long := func(n int) string { return strings.Repeat("a", n) }
	for _, tc := range []struct {
		name string
		msg  proto.Message
	}{
		{"login password over the LDAP-tolerant 512 bytes", &v1pb.LoginRequest{Password: long(513)}},
		{"signup password over bcrypt's 72 bytes", &v1pb.SignupRequest{Password: long(73)}},
		{"reset new_password over bcrypt's 72 bytes", &v1pb.ResetPasswordRequest{NewPassword: long(73)}},
		{"signup title over 200", &v1pb.SignupRequest{Title: long(201)}},
		{"otp code over 64", &v1pb.LoginRequest{OtpCode: ptrOf(long(65))}},
		{"email code over 64", &v1pb.LoginRequest{EmailCode: ptrOf(long(65))}},
		{"reset code over 64", &v1pb.ResetPasswordRequest{Code: long(65)}},
		{"switch recovery code over 64", &v1pb.SwitchWorkspaceRequest{RecoveryCode: ptrOf(long(65))}},
		{"mfa temp token over 4096", &v1pb.SwitchWorkspaceRequest{MfaTempToken: ptrOf(long(4097))}},
		{"login workspace over 256", &v1pb.LoginRequest{Workspace: ptrOf(long(257))}},
		{"idp name over 256", &v1pb.LoginRequest{IdpName: long(257)}},
		{"exchange token over 64KiB", &v1pb.ExchangeTokenRequest{Token: long(65537)}},
		{"exchange email over 254", &v1pb.ExchangeTokenRequest{Email: long(255)}},
		{"user email over 254", &v1pb.User{Email: long(255)}},
		{"user password over bcrypt's 72 bytes", &v1pb.User{Password: long(73)}},
		{"update email over 254", &v1pb.UpdateEmailRequest{Email: long(255)}},
		// The credential-change methods turn name into a T9 lockout identity,
		// so the interceptor must refuse an oversized one before any handler
		// claims a slot. 260 = "users/" + a 254-char email.
		{"reauth code name over 260", &v1pb.RequestReauthCodeRequest{Name: long(261)}},
		{"change password name over 260", &v1pb.ChangePasswordRequest{Name: long(261)}},
		{"start mfa enrollment name over 260", &v1pb.StartMFAEnrollmentRequest{Name: long(261)}},
		{"enable mfa name over 260", &v1pb.EnableMFARequest{Name: long(261)}},
		{"disable mfa name over 260", &v1pb.DisableMFARequest{Name: long(261)}},
		{"regenerate recovery codes name over 260", &v1pb.RegenerateRecoveryCodesRequest{Name: long(261)}},
		{"confirm recovery codes name over 260", &v1pb.ConfirmRecoveryCodesRequest{Name: long(261)}},
		{"change password new_password over bcrypt's 72 bytes", &v1pb.ChangePasswordRequest{NewPassword: long(73)}},
		{"proof current_password over bcrypt's 72 bytes", &v1pb.CredentialProof{Proof: &v1pb.CredentialProof_CurrentPassword{CurrentPassword: long(73)}}},
		{"proof otp code over 64", &v1pb.CredentialProof{Proof: &v1pb.CredentialProof_OtpCode{OtpCode: long(65)}}},
		{"proof recovery code over 64", &v1pb.CredentialProof{Proof: &v1pb.CredentialProof_RecoveryCode{RecoveryCode: long(65)}}},
		{"proof email code over 64", &v1pb.CredentialProof{Proof: &v1pb.CredentialProof_EmailCode{EmailCode: long(65)}}},
		{"enable mfa otp code over 64", &v1pb.EnableMFARequest{OtpCode: long(65)}},
		{"confirm recovery codes otp code over 64", &v1pb.ConfirmRecoveryCodesRequest{OtpCode: long(65)}},
	} {
		if err := v.Validate(tc.msg); err == nil {
			t.Errorf("expected size violation for %s, got nil", tc.name)
		}
	}
	// Login tolerates LDAP passwords beyond bcrypt's 72 bytes; the sinks that
	// bcrypt-hash (signup, reset) cap at exactly 72.
	if err := v.Validate(&v1pb.LoginRequest{Password: long(512)}); err != nil {
		t.Errorf("a 512-byte login password must pass: %v", err)
	}
	if err := v.Validate(&v1pb.SignupRequest{Password: long(72)}); err != nil {
		t.Errorf("a 72-byte signup password must pass: %v", err)
	}
}

// TestCredentialProofIsRequired pins the proof-carrying requests that must
// never reach a handler without a CredentialProof. The handler dereferences
// the proof to pick a verification path, so a missing one has to be an
// interceptor-level invalid_argument rather than a nil dereference deeper in
// (docs/design/reauthenticate-credential-changes.md).
func TestCredentialProofIsRequired(t *testing.T) {
	v, err := protovalidate.New()
	if err != nil {
		t.Fatalf("protovalidate.New: %v", err)
	}
	if err := v.Validate(&v1pb.ChangePasswordRequest{Name: "users/user@example.com", NewPassword: "new-password"}); err == nil {
		t.Error("expected a required violation for ChangePasswordRequest without a credential, got nil")
	}
	if err := v.Validate(&v1pb.ChangePasswordRequest{
		Name:        "users/user@example.com",
		NewPassword: "new-password",
		Credential:  &v1pb.CredentialProof{Proof: &v1pb.CredentialProof_CurrentPassword{CurrentPassword: "old-password"}},
	}); err != nil {
		t.Errorf("a ChangePasswordRequest carrying a credential must pass: %v", err)
	}
}

func ptrOf[T any](v T) *T { return &v }
