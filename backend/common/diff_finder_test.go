package common

import (
	"testing"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestDiff(t *testing.T) {
	type SubStruct struct {
		FieldD string
	}

	type Struct struct {
		FieldA string
		FieldB int
		FieldC SubStruct
	}

	type CycleStruct struct {
		FieldE *CycleStruct
		FieldF complex64
	}
	c1 := CycleStruct{FieldF: complex64(2)}
	c1.FieldE = &c1
	c2 := CycleStruct{FieldF: complex64(1)}
	c2.FieldE = &c2

	tests := []struct {
		oldVal any
		newVal any
		want   []*v1pb.Diff
	}{
		// Test Struct.
		{
			oldVal: Struct{"", 0, SubStruct{"HelloWorld"}},
			newVal: Struct{"", 2, SubStruct{""}},
			want: []*v1pb.Diff{
				{
					Action: string(ActionModify),
					Name:   "FieldB",
					Value:  "0 -> 2",
				},
				{
					Action: string(ActionModify),
					Name:   "FieldC.FieldD",
					Value:  "HelloWorld -> ",
				},
			},
		},
		// Test Slice.
		{
			oldVal: []any{"Hello", []string{"HelloWorld"}},
			newVal: []any{"Hello", "World"},
			want: []*v1pb.Diff{
				{
					Action: string(ActionAdd),
					Name:   "",
					Value:  "World",
				},
				{
					Action: string(ActionRemove),
					Name:   "",
					Value:  "[HelloWorld]",
				},
			},
		},
		// // Test Pointer.
		{
			oldVal: &c1,
			newVal: &c2,
			want: []*v1pb.Diff{
				{
					Action: string(ActionModify),
					Name:   "FieldF",
					Value:  "(2+0i) -> (1+0i)",
				},
			},
		},
		// Test Map.
		{
			oldVal: map[string]any{
				"FieldG": map[string]string{
					"FieldH": "ValueH",
				},
			},
			newVal: map[string]any{
				"FieldG": nil,
				"FieldI": map[string]string{
					"FieldJ": "ValueJ",
				},
			},
			want: []*v1pb.Diff{
				{
					Action: string(ActionAdd),
					Name:   "FieldI",
					Value:  "map[FieldJ:ValueJ]",
				},
				{
					Action: string(ActionRemove),
					Name:   "FieldG",
					Value:  "map[FieldH:ValueH]",
				},
			},
		},
	}

	for _, test := range tests {
		diffs, err := FindDiff(test.oldVal, test.newVal)
		if err != nil {
			t.Fatal(err)
		}
		v1pbDiffs := ConvertToV1pbDiffs(diffs, nil, nil)
		if len(v1pbDiffs) != len(test.want) {
			t.Fatalf("want len(test.want): %v, but get %v", len(v1pbDiffs), len(test.want))
		}
		for i, wantDiff := range test.want {
			if wantDiff.Action != v1pbDiffs[i].Action {
				t.Fatalf("want Action: %v but get %v", wantDiff.Action, v1pbDiffs[i].Action)
			}
			if wantDiff.Name != v1pbDiffs[i].Name {
				t.Fatalf("want Name: %v but get %v", wantDiff.Name, v1pbDiffs[i].Name)
			}
			if wantDiff.Value != v1pbDiffs[i].Value {
				t.Fatalf("want Value: %v but get %v", wantDiff.Value, v1pbDiffs[i].Value)
			}
		}
	}
}
