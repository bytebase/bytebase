package masker

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestWithBuiltinSemanticTypesDropsAStoredTypeWithABuiltinID(t *testing.T) {
	stored := []*storepb.SemanticTypeSetting_SemanticType{
		{Id: "email", Title: "Email"},
		{Id: DefaultSemanticTypeID, Title: "Stored default"},
		{Id: DefaultPartialSemanticTypeID, Title: "Stored partial"},
	}

	got := WithBuiltinSemanticTypes(stored)

	require.Equal(t, []string{"email", DefaultSemanticTypeID, DefaultPartialSemanticTypeID}, semanticTypeIDs(got))
	require.Equal(t, "Default", got[1].GetTitle())
	require.Equal(t, "Default Partial", got[2].GetTitle())
	require.Equal(t, []string{"email", DefaultSemanticTypeID, DefaultPartialSemanticTypeID}, semanticTypeIDs(stored))
	require.Equal(t, "Stored default", stored[1].GetTitle())
}

// A built-in type masks by its ID: Default Partial stores a range mask with no
// slices, which forAlgorithm alone would read as masking nothing.
func TestForSemanticTypeMasksABuiltinTypeByItsID(t *testing.T) {
	for _, semanticType := range builtinSemanticTypes() {
		m, err := ForSemanticType(semanticType)
		require.NoError(t, err)
		require.True(t, MasksValues(m), semanticType.GetId())
	}
	stored, err := forAlgorithm(builtinSemanticTypes()[1].GetAlgorithm())
	require.NoError(t, err)
	require.False(t, MasksValues(stored))
}

func TestForAlgorithmBuildsEachKindOfMask(t *testing.T) {
	inner, err := NewInnerOuterMasker(storepb.Algorithm_InnerOuterMask_INNER, 2, 3, "#")
	require.NoError(t, err)
	for name, tc := range map[string]struct {
		algorithm *storepb.Algorithm
		want      Masker
		masks     bool
		name      string
	}{
		"none":  {nil, NewNoneMasker(), false, ""},
		"empty": {&storepb.Algorithm{}, NewNoneMasker(), false, "Unknown"},
		"full": {&storepb.Algorithm{Mask: &storepb.Algorithm_FullMask_{FullMask: &storepb.Algorithm_FullMask{Substitution: "#"}}},
			NewFullMasker("#"), true, "Full mask"},
		"range": {&storepb.Algorithm{Mask: &storepb.Algorithm_RangeMask_{RangeMask: &storepb.Algorithm_RangeMask{
			Slices: []*storepb.Algorithm_RangeMask_Slice{{Start: 0, End: 2, Substitution: "*"}}}}},
			NewRangeMasker([]*MaskRangeSlice{{Start: 0, End: 2, Substitution: "*"}}), true, "Partial mask"},
		"range with no slice": {&storepb.Algorithm{Mask: &storepb.Algorithm_RangeMask_{RangeMask: &storepb.Algorithm_RangeMask{}}},
			NewRangeMasker(nil), false, "Partial mask"},
		"md5": {&storepb.Algorithm{Mask: &storepb.Algorithm_Md5Mask{Md5Mask: &storepb.Algorithm_MD5Mask{Salt: "s"}}},
			NewMD5Masker("s"), true, "Hash (MD5)"},
		"inner": {&storepb.Algorithm{Mask: &storepb.Algorithm_InnerOuterMask_{InnerOuterMask: &storepb.Algorithm_InnerOuterMask{
			Type: storepb.Algorithm_InnerOuterMask_INNER, PrefixLen: 2, SuffixLen: 3, Substitution: "#"}}},
			inner, true, "Inner/Outer mask"},
	} {
		m, err := forAlgorithm(tc.algorithm)
		require.NoError(t, err, name)
		require.True(t, tc.want.Equal(m), name)
		// Equal leaves out a masker's substitution, and the masked value has it.
		sample := &MaskData{Data: &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "0123456789abcdefghij"}}}
		require.Equal(t, tc.want.Mask(sample).GetStringValue(), m.Mask(sample).GetStringValue(), name)
		require.Equal(t, tc.masks, MasksValues(m), name)
		require.Equal(t, tc.name, AlgorithmName(tc.algorithm), name)
	}
}

func TestMasksValuesMatchesWhatEachMaskerReturns(t *testing.T) {
	inner := func(maskType storepb.Algorithm_InnerOuterMask_MaskType, prefix, suffix int32) Masker {
		m, err := NewInnerOuterMasker(maskType, prefix, suffix, "*")
		require.NoError(t, err)
		return m
	}
	for name, tc := range map[string]struct {
		masker Masker
		want   bool
	}{
		"none":                 {NewNoneMasker(), false},
		"range with no slice":  {NewRangeMasker(nil), false},
		"outer with no margin": {inner(storepb.Algorithm_InnerOuterMask_OUTER, 0, 0), false},
		"range with a slice":   {NewRangeMasker([]*MaskRangeSlice{{Start: 0, End: 1, Substitution: "*"}}), true},
		"outer with a prefix":  {inner(storepb.Algorithm_InnerOuterMask_OUTER, 1, 0), true},
		"inner with no margin": {inner(storepb.Algorithm_InnerOuterMask_INNER, 0, 0), true},
		"full":                 {NewFullMasker("***"), true},
		"default full":         {NewDefaultFullMasker(), true},
		"default range":        {NewDefaultRangeMasker(), true},
		"md5":                  {NewMD5Masker("salt"), true},
	} {
		sample := &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "0123456789abcdefghij"}}
		masked := tc.masker.Mask(&MaskData{Data: sample})
		require.Equal(t, tc.want, MasksValues(tc.masker), name)
		require.Equal(t, tc.want, masked.GetStringValue() != sample.GetStringValue(), name)
	}
}

func TestMasksEveryValueOnlyForAFullMask(t *testing.T) {
	inner, err := NewInnerOuterMasker(storepb.Algorithm_InnerOuterMask_INNER, 0, 0, "*")
	require.NoError(t, err)
	for name, tc := range map[string]struct {
		masker Masker
		want   bool
	}{
		"full":          {NewFullMasker("#"), true},
		"default full":  {NewDefaultFullMasker(), true},
		"default range": {NewDefaultRangeMasker(), false},
		"range":         {NewRangeMasker([]*MaskRangeSlice{{Start: 0, End: 2, Substitution: "*"}}), false},
		"md5":           {NewMD5Masker("salt"), false},
		"inner":         {inner, false},
		"none":          {NewNoneMasker(), false},
		"no masker":     {nil, false},
	} {
		require.Equal(t, tc.want, MasksEveryValue(tc.masker), name)
	}
}

func semanticTypeIDs(types []*storepb.SemanticTypeSetting_SemanticType) []string {
	ids := make([]string, 0, len(types))
	for _, semanticType := range types {
		ids = append(ids, semanticType.GetId())
	}
	return ids
}
