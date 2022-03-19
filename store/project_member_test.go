package store

import (
	"testing"

	"github.com/stretchr/testify/require"
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
