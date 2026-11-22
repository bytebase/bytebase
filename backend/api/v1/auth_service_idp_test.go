package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestGroupPatchesFromClaim(t *testing.T) {
	t.Parallel()
	const entraObjectID = "3f2a9c1e-7b4d-4e8a-9f61-2c5d8e0b1a74"
	alice := &store.UserMessage{Email: "alice@example.com"}
	aliceMember := common.FormatUserEmail(alice.Email)
	bobMember := common.FormatUserEmail("bob@example.com")

	newGroup := func(id, email, title, source string, members ...string) *store.GroupMessage {
		payload := &storepb.GroupPayload{Source: source}
		for _, member := range members {
			payload.Members = append(payload.Members, &storepb.GroupMember{Role: storepb.GroupMember_MEMBER, Member: member})
		}
		return &store.GroupMessage{ID: id, Workspace: "ws", Email: email, Title: title, Payload: payload}
	}

	tests := []struct {
		name  string
		group *store.GroupMessage
		claim []string
		// wantMembers is nil when the group must not change.
		wantMembers []string
	}{
		{
			name:        "claim names a manual group by title",
			group:       newGroup("dba", "", "dba", "", bobMember),
			claim:       []string{"dba"},
			wantMembers: []string{bobMember, aliceMember},
		},
		{
			name:        "claim names a manual group by email",
			group:       newGroup("dev", "dev@example.com", "Developers", ""),
			claim:       []string{"dev@example.com"},
			wantMembers: []string{aliceMember},
		},
		{
			name:        "claim omits a manual group the user is in",
			group:       newGroup("qa", "", "qa", "", aliceMember, bobMember),
			claim:       []string{"dba"},
			wantMembers: []string{bobMember},
		},
		{
			name:  "claim names a manual group the user is already in",
			group: newGroup("ops", "", "ops", "", aliceMember),
			claim: []string{"ops"},
		},
		{
			name:  "object-ID claim keeps the user in a SCIM group",
			group: newGroup(entraObjectID, "", "data-platform", "Entra ID", aliceMember),
			claim: []string{entraObjectID},
		},
		{
			name:  "claim omitting a SCIM group keeps the user in it",
			group: newGroup("okta-group", "", "analysts", "Okta", aliceMember),
			claim: nil,
		},
		{
			name:  "claim naming a SCIM group does not add the user",
			group: newGroup(entraObjectID, "", "data-platform", "Entra ID", bobMember),
			claim: []string{"data-platform"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			membersBefore := memberNames(tc.group.Payload)

			patches := groupPatchesFromClaim(alice, []*store.GroupMessage{tc.group}, tc.claim)

			require.Equal(t, membersBefore, memberNames(tc.group.Payload), "input group must not be mutated")
			if tc.wantMembers == nil {
				require.Empty(t, patches)
				return
			}
			require.Len(t, patches, 1)
			require.Equal(t, tc.group.ID, patches[0].ID)
			require.Equal(t, tc.group.Workspace, patches[0].Workspace)
			require.Equal(t, tc.wantMembers, memberNames(patches[0].Payload))
			require.Equal(t, tc.group.Payload.GetSource(), patches[0].Payload.GetSource())
		})
	}
}

func memberNames(payload *storepb.GroupPayload) []string {
	var names []string
	for _, member := range payload.GetMembers() {
		names = append(names, member.Member)
	}
	return names
}
