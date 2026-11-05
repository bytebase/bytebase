package masker

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// The built-in semantic types. Masking adds them to the workspace's types, and
// a stored type with a built-in ID is ignored.
const (
	DefaultSemanticTypeID        = "bb.default"
	DefaultPartialSemanticTypeID = "bb.default-partial"
)

// BuiltinSemanticTypes returns the built-in semantic types.
func BuiltinSemanticTypes() []*storepb.SemanticTypeSetting_SemanticType {
	return []*storepb.SemanticTypeSetting_SemanticType{
		{
			Id:    DefaultSemanticTypeID,
			Title: "Default",
			Algorithm: &storepb.Algorithm{
				Mask: &storepb.Algorithm_FullMask_{
					FullMask: &storepb.Algorithm_FullMask{},
				},
			},
		},
		{
			Id:    DefaultPartialSemanticTypeID,
			Title: "Default Partial",
			Algorithm: &storepb.Algorithm{
				Mask: &storepb.Algorithm_RangeMask_{
					RangeMask: &storepb.Algorithm_RangeMask{},
				},
			},
		},
	}
}

// IsBuiltinSemanticTypeID reports whether id names a built-in semantic type.
func IsBuiltinSemanticTypeID(id string) bool {
	return id == DefaultSemanticTypeID || id == DefaultPartialSemanticTypeID
}

// ForSemanticType returns the masker that semanticType applies to a column. A
// built-in type masks by its ID: Default Partial stores a range mask with no
// slices, and masks with the default ranges.
func ForSemanticType(semanticType *storepb.SemanticTypeSetting_SemanticType) (Masker, error) {
	switch semanticType.GetId() {
	case DefaultSemanticTypeID:
		return NewDefaultFullMasker(), nil
	case DefaultPartialSemanticTypeID:
		return NewDefaultRangeMasker(), nil
	default:
		return ForAlgorithm(semanticType.GetAlgorithm())
	}
}

// ForAlgorithm returns the masker that algorithm configures. No algorithm masks
// nothing.
func ForAlgorithm(algorithm *storepb.Algorithm) (Masker, error) {
	switch m := algorithm.GetMask().(type) {
	case *storepb.Algorithm_FullMask_:
		return NewFullMasker(m.FullMask.Substitution), nil
	case *storepb.Algorithm_RangeMask_:
		slices := make([]*MaskRangeSlice, 0, len(m.RangeMask.Slices))
		for _, slice := range m.RangeMask.Slices {
			slices = append(slices, &MaskRangeSlice{Start: slice.Start, End: slice.End, Substitution: slice.Substitution})
		}
		return NewRangeMasker(slices), nil
	case *storepb.Algorithm_Md5Mask:
		return NewMD5Masker(m.Md5Mask.Salt), nil
	case *storepb.Algorithm_InnerOuterMask_:
		return NewInnerOuterMasker(m.InnerOuterMask.Type, m.InnerOuterMask.PrefixLen, m.InnerOuterMask.SuffixLen, m.InnerOuterMask.Substitution)
	default:
		return NewNoneMasker(), nil
	}
}

// AlgorithmName names the kind of masking algorithm configures, as masking
// reasons show it.
func AlgorithmName(algorithm *storepb.Algorithm) string {
	if algorithm == nil {
		return ""
	}
	switch algorithm.Mask.(type) {
	case *storepb.Algorithm_FullMask_:
		return "Full mask"
	case *storepb.Algorithm_RangeMask_:
		return "Partial mask"
	case *storepb.Algorithm_Md5Mask:
		return "Hash (MD5)"
	case *storepb.Algorithm_InnerOuterMask_:
		return "Inner/Outer mask"
	default:
		return "Unknown"
	}
}
