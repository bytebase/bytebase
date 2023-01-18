package store

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/api"
)

func TestGetBatchUpdatePrincipalIDList(t *testing.T) {
	type Input struct {
		oldIDList []int
		newIDList []int
	}
	type Result struct {
		createIDList []int
		patchIDList  []int
		deleteIDList []int
	}

	testCases := map[string]struct {
		input  Input
		expect Result
	}{
		"single_patch": {
			input: Input{
				oldIDList: []int{1, 2, 3},
				newIDList: []int{1, 2, 3},
			},
			expect: Result{
				createIDList: nil,
				patchIDList:  []int{1, 2, 3},
				deleteIDList: nil,
			},
		},
		"single_delete": {
			input: Input{
				oldIDList: []int{1, 2, 3},
				newIDList: nil,
			},
			expect: Result{
				createIDList: nil,
				patchIDList:  nil,
				deleteIDList: []int{1, 2, 3},
			},
		},
		"single_create": {
			input: Input{
				oldIDList: nil,
				newIDList: []int{1, 2, 3},
			},
			expect: Result{
				createIDList: []int{1, 2, 3},
				patchIDList:  nil,
				deleteIDList: nil,
			},
		},
		"create_delete": {
			input: Input{
				oldIDList: []int{1, 2},
				newIDList: []int{3, 4},
			},
			expect: Result{
				createIDList: []int{3, 4},
				patchIDList:  nil,
				deleteIDList: []int{1, 2},
			},
		},
		"create_patch": {
			input: Input{
				oldIDList: []int{1, 2},
				newIDList: []int{1, 2, 3},
			},
			expect: Result{
				createIDList: []int{3},
				patchIDList:  []int{1, 2},
				deleteIDList: nil,
			},
		},
		"delete_patch": {
			input: Input{
				oldIDList: []int{1, 2},
				newIDList: []int{2},
			},
			expect: Result{
				createIDList: nil,
				patchIDList:  []int{2},
				deleteIDList: []int{1},
			},
		},
		"mixed": {
			input: Input{
				oldIDList: []int{1, 2},
				newIDList: []int{2, 3},
			},
			expect: Result{
				createIDList: []int{3},
				patchIDList:  []int{2},
				deleteIDList: []int{1},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var got Result
			got.createIDList, got.patchIDList, got.deleteIDList = getBatchUpdatePrincipalIDList(tc.input.oldIDList, tc.input.newIDList)
			require.Equal(t, tc.expect.createIDList, got.createIDList)
			require.Equal(t, tc.expect.patchIDList, got.patchIDList)
			require.Equal(t, tc.expect.deleteIDList, got.deleteIDList)
		})
	}
}

func TestGetIAMPolicyDiff(t *testing.T) {
	type Input struct {
		oldPolicy map[api.Role][]int
		newPolicy map[api.Role][]int
	}
	type Result struct {
		deleteIDs     []int
		createsPolicy map[api.Role][]int
	}

	buildPolicyMessageFromInputPolicy := func(m map[api.Role][]int) *IAMPolicyMessage {
		var bindings []*PolicyBinding
		for role, memberIDs := range m {
			var users []*UserMessage
			for _, memberID := range memberIDs {
				users = append(users, &UserMessage{
					ID: memberID,
				})
			}
			bindings = append(bindings, &PolicyBinding{
				Role:    role,
				Members: users,
			})
		}
		return &IAMPolicyMessage{
			Bindings: bindings,
		}
	}
	extractCreatePolicyFromIAMPolicyMessage := func(m *IAMPolicyMessage) map[api.Role][]int {
		result := make(map[api.Role][]int)
		for _, binding := range m.Bindings {
			var memberIDs []int
			for _, member := range binding.Members {
				memberIDs = append(memberIDs, member.ID)
			}
			result[binding.Role] = memberIDs
		}
		return result
	}

	testCases := []struct {
		input  Input
		result Result
	}{
		// Only Delete Member
		{
			input: Input{
				oldPolicy: map[api.Role][]int{
					api.Owner:     {1, 2},
					api.Developer: {3, 4},
				},
				newPolicy: map[api.Role][]int{
					api.Owner:     {1, 2},
					api.Developer: {3},
				},
			},
			result: Result{
				deleteIDs:     []int{4},
				createsPolicy: map[api.Role][]int{},
			},
		},
		// Only Add Member
		{
			input: Input{
				oldPolicy: map[api.Role][]int{
					api.Owner:     {1, 2},
					api.Developer: {3, 4},
				},
				newPolicy: map[api.Role][]int{
					api.Owner:     {1, 2},
					api.Developer: {3, 4, 5},
				},
			},
			result: Result{
				createsPolicy: map[api.Role][]int{
					api.Developer: {5},
				},
			},
		},
		// Only Change Member Role
		{
			input: Input{
				oldPolicy: map[api.Role][]int{
					api.Owner:     {1, 2},
					api.Developer: {3, 4},
				},
				newPolicy: map[api.Role][]int{
					api.Owner:     {1, 2, 3},
					api.Developer: {4},
				},
			},
			result: Result{
				deleteIDs: []int{3},
				createsPolicy: map[api.Role][]int{
					api.Owner: {3},
				},
			},
		},

		// Complex Case
		{
			input: Input{
				oldPolicy: map[api.Role][]int{
					api.Owner:     {1, 2},
					api.Developer: {3, 4},
				},
				newPolicy: map[api.Role][]int{
					api.Owner:     {2, 4, 5},
					api.Developer: {3, 6},
				},
			},
			result: Result{
				deleteIDs: []int{1, 4},
				createsPolicy: map[api.Role][]int{
					api.Owner:     {4, 5},
					api.Developer: {6},
				},
			},
		},
	}
	for _, tc := range testCases {
		oldPolicyMessage := buildPolicyMessageFromInputPolicy(tc.input.oldPolicy)
		newPolicyMessage := buildPolicyMessageFromInputPolicy(tc.input.newPolicy)
		deleteIDs, creates := getIAMPolicyDiff(oldPolicyMessage, newPolicyMessage)
		sort.Slice(deleteIDs, func(i, j int) bool { return deleteIDs[i] < deleteIDs[j] })
		sort.Slice(tc.result.deleteIDs, func(i, j int) bool { return tc.result.deleteIDs[i] < tc.result.deleteIDs[j] })
		require.Equal(t, tc.result.deleteIDs, deleteIDs)
		createsPolicy := extractCreatePolicyFromIAMPolicyMessage(creates)
		for role, memberIDs := range createsPolicy {
			sort.Slice(memberIDs, func(i, j int) bool { return memberIDs[i] < memberIDs[j] })
			createsPolicy[role] = memberIDs
		}
		for role, memberIDs := range tc.result.createsPolicy {
			sort.Slice(memberIDs, func(i, j int) bool { return memberIDs[i] < memberIDs[j] })
			tc.result.createsPolicy[role] = memberIDs
		}
		require.Equal(t, tc.result.createsPolicy, createsPolicy)
	}
}
