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
		{"login password over bcrypt's 72 bytes", &v1pb.LoginRequest{Password: long(73)}},
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
		{"exchange token over 8192", &v1pb.ExchangeTokenRequest{Token: long(8193)}},
		{"exchange email over 254", &v1pb.ExchangeTokenRequest{Email: long(255)}},
		{"user email over 254", &v1pb.User{Email: long(255)}},
		{"user password over bcrypt's 72 bytes", &v1pb.User{Password: long(73)}},
		{"update email over 254", &v1pb.UpdateEmailRequest{Email: long(255)}},
	} {
		if err := v.Validate(tc.msg); err == nil {
			t.Errorf("expected size violation for %s, got nil", tc.name)
		}
	}
	// A 72-byte password is bcrypt's maximum and must pass.
	if err := v.Validate(&v1pb.LoginRequest{Password: long(72)}); err != nil {
		t.Errorf("a 72-byte password must pass: %v", err)
	}
}

func ptrOf[T any](v T) *T { return &v }
