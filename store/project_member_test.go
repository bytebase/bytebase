package store

import (
	"testing"

	"github.com/kr/pretty"
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
				createIDList: []int{},
				patchIDList:  []int{1, 2, 3},
				deleteIDList: []int{},
			},
		},
		"single_delete": {
			input: Input{
				oldIDList: []int{1, 2, 3},
				newIDList: []int{},
			},
			expect: Result{
				createIDList: []int{},
				patchIDList:  []int{},
				deleteIDList: []int{1, 2, 3},
			},
		},
		"single_create": {
			input: Input{
				oldIDList: []int{},
				newIDList: []int{1, 2, 3},
			},
			expect: Result{
				createIDList: []int{1, 2, 3},
				patchIDList:  []int{},
				deleteIDList: []int{},
			},
		},
		"create_delete": {
			input: Input{
				oldIDList: []int{1, 2},
				newIDList: []int{3, 4},
			},
			expect: Result{
				createIDList: []int{3, 4},
				patchIDList:  []int{},
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
				deleteIDList: []int{},
			},
		},
		"delete_patch": {
			input: Input{
				oldIDList: []int{1, 2},
				newIDList: []int{2},
			},
			expect: Result{
				createIDList: []int{},
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
			if diff := pretty.Diff(tc.expect.createIDList, got.createIDList); len(diff) != 0 {
				t.Errorf("\ncreateIDList: %v", diff)
			}
			if diff := pretty.Diff(tc.expect.patchIDList, got.patchIDList); len(diff) != 0 {
				t.Errorf("\npatchIDList: %v", diff)
			}
			if diff := pretty.Diff(tc.expect.deleteIDList, got.deleteIDList); len(diff) != 0 {
				t.Errorf("\ndeleteIDList: %v", diff)
			}
		})
	}
}
