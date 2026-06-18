//nolint:revive
package utils

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestUserMessageFromAccountPreservesPhone(t *testing.T) {
	a := require.New(t)

	got := userMessageFromAccount(&store.AccountMessage{
		Email:         "alice@example.com",
		Name:          "Alice",
		Type:          storepb.PrincipalType_END_USER,
		Phone:         "+8613800000000",
		MemberDeleted: true,
	})

	a.Equal("alice@example.com", got.Email)
	a.Equal("Alice", got.Name)
	a.Equal(storepb.PrincipalType_END_USER, got.Type)
	a.Equal("+8613800000000", got.Phone)
	a.True(got.MemberDeleted)
}
