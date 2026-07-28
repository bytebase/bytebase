package server

import (
	"testing"

	"buf.build/go/protovalidate"

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
