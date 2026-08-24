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
