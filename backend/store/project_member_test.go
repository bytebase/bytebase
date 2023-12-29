package store

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

func TestGetIAMPolicyDiff(t *testing.T) {
	testCases := []struct {
		oldPolicy *IAMPolicyMessage
		newPolicy *IAMPolicyMessage
		remove    *IAMPolicyMessage
		add       *IAMPolicyMessage
	}{
		{
			oldPolicy: &IAMPolicyMessage{},
			newPolicy: &IAMPolicyMessage{},
			remove:    &IAMPolicyMessage{},
			add:       &IAMPolicyMessage{},
		},
		{
			oldPolicy: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 1}, {ID: 2}},
					},
					{
						Role:    api.ProjectDeveloper,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
			newPolicy: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 1}, {ID: 2}},
					},
				},
			},
			remove: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectDeveloper,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
			add: &IAMPolicyMessage{},
		},
		{
			oldPolicy: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 1}, {ID: 2}},
					},
					{
						Role:    api.ProjectDeveloper,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
			newPolicy: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 1}},
					},
					{
						Role:    api.ProjectDeveloper,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
			remove: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
			add: &IAMPolicyMessage{},
		},
		{
			oldPolicy: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 1}, {ID: 2}},
					},
				},
			},
			newPolicy: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 1}, {ID: 2}},
					},
					{
						Role:    api.ProjectDeveloper,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
			remove: &IAMPolicyMessage{},
			add: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectDeveloper,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
		},
		{
			oldPolicy: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 1}},
					},
					{
						Role:    api.ProjectDeveloper,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
			newPolicy: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 1}, {ID: 2}},
					},
					{
						Role:    api.ProjectDeveloper,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
			remove: &IAMPolicyMessage{},
			add: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
		},
		{
			oldPolicy: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 1}, {ID: 2}},
					},
				},
			},
			newPolicy: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 1}},
					},
					{
						Role:    api.ProjectDeveloper,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
			remove: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
			add: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectDeveloper,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
		},
		{
			oldPolicy: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 1}, {ID: 2}},
					},
				},
			},
			newPolicy: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 1}},
					},
					{
						Role:    api.ProjectDeveloper,
						Members: []*UserMessage{{ID: 1}},
					},
					{
						Role:    api.ProjectDeveloper,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
			remove: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectOwner,
						Members: []*UserMessage{{ID: 2}},
					},
				},
			},
			add: &IAMPolicyMessage{
				Bindings: []*PolicyBinding{
					{
						Role:    api.ProjectDeveloper,
						Members: []*UserMessage{{ID: 1}, {ID: 2}},
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		remove, add, err := GetIAMPolicyDiff(tc.oldPolicy, tc.newPolicy)
		require.NoError(t, err)
		require.Equal(t, tc.remove.String(), remove.String(), fmt.Sprintf("%d", i))
		require.Equal(t, tc.add.String(), add.String(), fmt.Sprintf("%d", i))
	}
}
