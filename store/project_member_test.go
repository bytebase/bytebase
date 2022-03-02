package store

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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

	opt := cmp.Comparer(func(l1, l2 []int) bool {
		if len(l1) != len(l2) {
			return false
		}
		set1 := make(map[int]bool)
		for _, i := range l1 {
			set1[i] = true
		}
		for _, i := range l2 {
			if _, ok := set1[i]; !ok {
				return false
			}
		}
		return true
	})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var got Result
			got.createIDList, got.patchIDList, got.deleteIDList, _ = getBatchUpdatePrincipalIDList(tc.input.oldIDList, tc.input.newIDList)
			if diff := cmp.Diff(tc.expect.createIDList, got.createIDList, opt); diff != "" {
				t.Errorf("\ncreateIDList: %v", diff)
			}
			if diff := cmp.Diff(tc.expect.patchIDList, got.patchIDList, opt); diff != "" {
				t.Errorf("\npatchIDList: %v", diff)
			}
			if diff := cmp.Diff(tc.expect.deleteIDList, got.deleteIDList, opt); diff != "" {
				t.Errorf("\ndeleteIDList: %v", diff)
			}
		})
	}
}
