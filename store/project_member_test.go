package store

import (
	"context"
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

	opt := cmp.Comparer(func(x, y Result) bool {
		compareTwoList := func(l1, l2 []int) bool {
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
		}
		return compareTwoList(x.createIDList, y.createIDList) && compareTwoList(x.deleteIDList, y.deleteIDList) && compareTwoList(x.patchIDList, y.patchIDList)

	})

	ctx := context.Background()
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var got Result
			got.createIDList, got.patchIDList, got.deleteIDList, _ = getBatchUpdatePrincipalIDList(ctx, tc.input.oldIDList, tc.input.newIDList)
			if diff := cmp.Diff(tc.expect, got, opt); diff != "" {
				t.Fatalf(diff)
			}
		})
	}
}
