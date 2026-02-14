package masker

import (
	"crypto/md5"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// MaskData is the data to be masked.
type MaskData struct {
	// WantBytes indicates whether the Masker should return the masked data as
	// *v1pb.RowValue_BytesValue.
	WantBytes bool

	// Data is the data to be masked.
	Data *v1pb.RowValue
}

// Masker is the interface that masks the data.
type Masker interface {
	Mask(data *MaskData) *v1pb.RowValue
	Equal(other Masker) bool
}

// NoneMasker is the masker that does not mask the data.
type NoneMasker struct{}

// NewNoneMasker returns a new NoneMasker.
func NewNoneMasker() *NoneMasker {
	return &NoneMasker{}
}

// Mask implements Masker.Mask.
func (*NoneMasker) Mask(data *MaskData) *v1pb.RowValue {
	return noneMask(data)
}

func noneMask(data *MaskData) *v1pb.RowValue {
	return data.Data
}

// Equal implements Masker.Equal.
func (*NoneMasker) Equal(other Masker) bool {
	_, ok := other.(*NoneMasker)
	return ok
}

// FullMasker is the masker that masks the data with `substitution`.
type FullMasker struct {
	substitution string
}

// NewFullMasker returns a new FullMasker.
func NewFullMasker(substitution string) *FullMasker {
	return &FullMasker{
		substitution: substitution,
	}
}

// NewDefaultFullMasker returns a new FullMasker with default substitution(`******`).
func NewDefaultFullMasker() *FullMasker {
	return &FullMasker{
		substitution: "******",
	}
}

// Mask implements Masker.Mask.
func (m *FullMasker) Mask(*MaskData) *v1pb.RowValue {
	return &v1pb.RowValue{
		Kind: &v1pb.RowValue_StringValue{
			StringValue: m.substitution,
		},
	}
}

// Equal implements Masker.Equal.
func (m *FullMasker) Equal(other Masker) bool {
	if otherFullMasker, ok := other.(*FullMasker); ok {
		return m.substitution == otherFullMasker.substitution
	}
	return false
}

type MaskRangeSlice struct {
	// Start is the start index of the range.
	Start int32
	// End is the end index of the range.
	End int32
	// Substitution is the substitution string.
	Substitution string
}

// RangeMasker is the masker that masks the left and right quarters with "**".
type RangeMasker struct {
	// MaskRangeSlice is the slice of the range to be masked.
	MaskRangeSlice []*MaskRangeSlice
}

// NewRangeMasker returns a new RangeMasker.
func NewRangeMasker(maskRangeSlice []*MaskRangeSlice) *RangeMasker {
	return &RangeMasker{
		MaskRangeSlice: maskRangeSlice,
	}
}

func (m *RangeMasker) enableMask() bool {
	return len(m.MaskRangeSlice) > 0
}

func getValidRangeValue(length, val int32) int32 {
	if val >= 0 {
		return val
	}
	resp := length + val + 1
	if resp < 0 {
		return 0
	}
	return resp
}

func (m *RangeMasker) formatRanges(maxLength int32) []*MaskRangeSlice {
	// Merge the overlapping ranges.
	var mergedMaskRangeSlice []*MaskRangeSlice

	for _, maskRange := range m.MaskRangeSlice {
		start := getValidRangeValue(maxLength, maskRange.Start)
		end := getValidRangeValue(maxLength, maskRange.End)

		if start > end {
			end = start + 1
		}

		if len(mergedMaskRangeSlice) == 0 {
			mergedMaskRangeSlice = append(mergedMaskRangeSlice, &MaskRangeSlice{
				Start:        start,
				End:          end,
				Substitution: maskRange.Substitution,
			})
			continue
		}
		lastMaskRange := mergedMaskRangeSlice[len(mergedMaskRangeSlice)-1]
		if lastMaskRange.End >= start {
			mergedMaskRangeSlice[len(mergedMaskRangeSlice)-1].End = end
		} else {
			mergedMaskRangeSlice = append(mergedMaskRangeSlice, &MaskRangeSlice{
				Start:        start,
				End:          end,
				Substitution: maskRange.Substitution,
			})
		}
	}

	slices.SortStableFunc(mergedMaskRangeSlice, func(a, b *MaskRangeSlice) int {
		if a.Start < b.Start {
			return -1
		} else if a.Start > b.Start {
			return 1
		}
		return 0
	})

	return mergedMaskRangeSlice
}

// Mask implements Masker.Mask.
func (m *RangeMasker) Mask(data *MaskData) *v1pb.RowValue {
	if !m.enableMask() {
		return noneMask(data)
	}

	fRune := func(s []rune) []rune {
		maskRangeSlice := m.formatRanges(int32(len(s)))

		var ret []rune
		prevEnd := 0
		for _, maskRange := range maskRangeSlice {
			// First, append the unmasked part.
			begin, end := prevEnd, int(maskRange.Start)
			if begin >= len(s) {
				// If the begin index is out of range, we should stop the masking.
				break
			}
			// To avoid the panic of slice out of range when end is greater than len(s).
			if end > len(s) {
				end = len(s)
			}
			ret = append(ret, s[begin:end]...)
			// If the end index is out of range, we should stop the masking.
			if end == len(s) {
				prevEnd = end
				break
			}
			// Second, append the masked part.
			ret = append(ret, []rune(maskRange.Substitution)...)

			// Goto the next unmasked part start index.
			end = min(len(s), int(maskRange.End))
			prevEnd = end
		}
		if prevEnd < len(s) {
			ret = append(ret, s[prevEnd:]...)
		}
		return ret
	}

	fBytes := func(s []byte) []byte {
		maskRangeSlice := m.formatRanges(int32(len(s)))

		var ret []byte
		prevEnd := 0
		for _, maskRange := range maskRangeSlice {
			// First, append the unmasked part.
			begin, end := prevEnd, int(maskRange.Start)
			if begin >= len(s) {
				// If the begin index is out of range, we should stop the masking.
				break
			}
			// To avoid the panic of slice out of range when end is greater than len(s).
			if end > len(s) {
				end = len(s)
			}
			ret = append(ret, s[begin:end]...)
			// If the end index is out of range, we should stop the masking.
			if end == len(s) {
				prevEnd = end
				break
			}
			// Second, append the masked part.
			ret = append(ret, []byte(maskRange.Substitution)...)

			// Goto the next unmasked part start index.
			end = min(len(s), int(maskRange.End))
			prevEnd = end
		}
		if prevEnd < len(s) {
			ret = append(ret, s[prevEnd:]...)
		}
		return ret
	}

	switch kind := data.Data.Kind.(type) {
	case *v1pb.RowValue_NullValue:
		if kind.NullValue == structpb.NullValue_NULL_VALUE {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: "******",
				},
			}
		}
	case *v1pb.RowValue_BoolValue:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: "******",
			},
		}
	case *v1pb.RowValue_BytesValue:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_BytesValue{
				BytesValue: fBytes(kind.BytesValue),
			},
		}
	case *v1pb.RowValue_DoubleValue:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(fRune([]rune(strconv.FormatFloat(kind.DoubleValue, 'f', -1, 64)))),
			},
		}
	case *v1pb.RowValue_FloatValue:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(fRune([]rune(strconv.FormatFloat(float64(kind.FloatValue), 'f', -1, 64)))),
			},
		}
	case *v1pb.RowValue_Int32Value:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(fRune([]rune(strconv.FormatInt(int64(kind.Int32Value), 10)))),
			},
		}
	case *v1pb.RowValue_Int64Value:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(fRune([]rune(strconv.FormatInt(kind.Int64Value, 10)))),
			},
		}
	case *v1pb.RowValue_StringValue:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(fRune([]rune(kind.StringValue))),
			},
		}
	case *v1pb.RowValue_Uint32Value:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(fRune([]rune(strconv.FormatUint(uint64(kind.Uint32Value), 10)))),
			},
		}
	case *v1pb.RowValue_Uint64Value:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(fRune([]rune(strconv.FormatUint(kind.Uint64Value, 10)))),
			},
		}
	case *v1pb.RowValue_ValueValue:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_ValueValue{
				ValueValue: maskProtoValue(m, kind.ValueValue),
			},
		}
	case *v1pb.RowValue_TimestampValue:
		s := kind.TimestampValue.GoogleTimestamp.AsTime().Format("2006-01-02 15:04:05.000000")
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(fRune([]rune(s))),
			},
		}
	case *v1pb.RowValue_TimestampTzValue:
		t := kind.TimestampTzValue.GoogleTimestamp.AsTime()
		z := time.FixedZone(kind.TimestampTzValue.GetZone(), int(kind.TimestampTzValue.GetOffset()))
		s := t.In(z).Format(time.RFC3339Nano)
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(fRune([]rune(s))),
			},
		}
	default:
	}
	return nil
}

// Equal implements Masker.Equal.
func (m *RangeMasker) Equal(other Masker) bool {
	if otherRangeMasker, ok := other.(*RangeMasker); ok {
		if len(m.MaskRangeSlice) != len(otherRangeMasker.MaskRangeSlice) {
			return false
		}
		for i, maskRange := range m.MaskRangeSlice {
			if maskRange.Start != otherRangeMasker.MaskRangeSlice[i].Start ||
				maskRange.End != otherRangeMasker.MaskRangeSlice[i].End ||
				maskRange.Substitution != otherRangeMasker.MaskRangeSlice[i].Substitution {
				return false
			}
		}
		return true
	}
	return false
}

// DefaultRangeMasker is the masker that masks the left and right quarters with "**".
type DefaultRangeMasker struct{}

// NewDefaultRangeMasker returns a new DefaultRangeMasker.
func NewDefaultRangeMasker() *DefaultRangeMasker {
	return &DefaultRangeMasker{}
}

// Mask implements Masker.Mask.
func (*DefaultRangeMasker) Mask(data *MaskData) *v1pb.RowValue {
	paddingAsterisk := func(t string) string {
		return fmt.Sprintf("**%s**", t)
	}
	var stringValue string
	switch kind := data.Data.Kind.(type) {
	case *v1pb.RowValue_NullValue:
		if kind.NullValue == structpb.NullValue_NULL_VALUE {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: "******",
				},
			}
		}
	case *v1pb.RowValue_BoolValue:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: "******",
			},
		}
	case *v1pb.RowValue_BytesValue:
		bytesValue := append([]byte{'*', '*'}, middle([]byte(kind.BytesValue))...)
		bytesValue = append(bytesValue, []byte{'*', '*'}...)
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_BytesValue{
				BytesValue: bytesValue,
			},
		}
	case *v1pb.RowValue_DoubleValue:
		stringValue = string(middle([]byte(strconv.FormatFloat(kind.DoubleValue, 'f', -1, 64))))
	case *v1pb.RowValue_FloatValue:
		stringValue = string(middle([]byte(strconv.FormatFloat(float64(kind.FloatValue), 'f', -1, 64))))
	case *v1pb.RowValue_Int32Value:
		stringValue = string(middle([]byte(strconv.FormatInt(int64(kind.Int32Value), 10))))
	case *v1pb.RowValue_Int64Value:
		stringValue = string(middle([]byte(strconv.FormatInt(kind.Int64Value, 10))))
	case *v1pb.RowValue_StringValue:
		stringValue = string(middle([]rune(kind.StringValue)))
	case *v1pb.RowValue_Uint32Value:
		stringValue = string(middle([]byte(strconv.FormatUint(uint64(kind.Uint32Value), 10))))
	case *v1pb.RowValue_Uint64Value:
		stringValue = string(middle([]byte(strconv.FormatUint(kind.Uint64Value, 10))))
	case *v1pb.RowValue_ValueValue:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_ValueValue{
				ValueValue: maskProtoValue(&DefaultRangeMasker{}, kind.ValueValue),
			},
		}
	case *v1pb.RowValue_TimestampValue:
		s := kind.TimestampValue.GoogleTimestamp.AsTime().Format("2006-01-02 15:04:05.000000")
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string((middle([]byte(s)))),
			},
		}
	case *v1pb.RowValue_TimestampTzValue:
		t := kind.TimestampTzValue.GoogleTimestamp.AsTime()
		z := time.FixedZone(kind.TimestampTzValue.GetZone(), int(kind.TimestampTzValue.GetOffset()))
		s := t.In(z).Format(time.RFC3339Nano)
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(middle([]byte(s))),
			},
		}
	default:
	}

	return &v1pb.RowValue{
		Kind: &v1pb.RowValue_StringValue{
			StringValue: paddingAsterisk(stringValue),
		},
	}
}

// Equal implements Masker.Equal.
func (*DefaultRangeMasker) Equal(other Masker) bool {
	_, ok := other.(*DefaultRangeMasker)
	return ok
}

// middle will get the middle part of the given slice.
func middle[T ~byte | ~rune](str []T) []T {
	if len(str) == 0 || len(str) == 1 {
		return []T{}
	}
	if len(str) == 2 || len(str) == 3 {
		return str[len(str)/2 : len(str)/2+1]
	}

	if len(str)%4 != 0 {
		str = str[:len(str)/4*4]
	}

	var ret []T
	ret = append(ret, str[len(str)/4:len(str)/2]...)
	ret = append(ret, str[len(str)/2:len(str)/4*3]...)
	return ret
}

// MD5Masker is the masker that masks the data with their MD5 hash.
type MD5Masker struct {
	salt string
}

// NewMD5Masker returns a new MD5Masker.
func NewMD5Masker(salt string) *MD5Masker {
	return &MD5Masker{
		salt: salt,
	}
}

// Mask implements Masker.Mask.
func (m *MD5Masker) Mask(data *MaskData) *v1pb.RowValue {
	f := func(s string) string {
		h := md5.New()
		if _, err := h.Write([]byte(s + m.salt)); err != nil {
			slog.Error("Failed to write to md5 hash: %v", log.BBError(err))
		}
		return fmt.Sprintf("%x", h.Sum(nil))
	}

	var stringValue string
	switch kind := data.Data.Kind.(type) {
	case *v1pb.RowValue_NullValue:
		if kind.NullValue == structpb.NullValue_NULL_VALUE {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: "******",
				},
			}
		}
	case *v1pb.RowValue_BoolValue:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: "******",
			},
		}
	case *v1pb.RowValue_BytesValue:
		stringValue = f(string(kind.BytesValue))
	case *v1pb.RowValue_DoubleValue:
		stringValue = f(strconv.FormatFloat(kind.DoubleValue, 'f', -1, 64))
	case *v1pb.RowValue_FloatValue:
		stringValue = f(strconv.FormatFloat(float64(kind.FloatValue), 'f', -1, 64))
	case *v1pb.RowValue_Int32Value:
		stringValue = f(strconv.FormatInt(int64(kind.Int32Value), 10))
	case *v1pb.RowValue_Int64Value:
		stringValue = f(strconv.FormatInt(kind.Int64Value, 10))
	case *v1pb.RowValue_StringValue:
		stringValue = f(kind.StringValue)
	case *v1pb.RowValue_Uint32Value:
		stringValue = f(strconv.FormatUint(uint64(kind.Uint32Value), 10))
	case *v1pb.RowValue_Uint64Value:
		stringValue = f(strconv.FormatUint(kind.Uint64Value, 10))
	case *v1pb.RowValue_ValueValue:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_ValueValue{
				ValueValue: maskProtoValue(m, kind.ValueValue),
			},
		}
	case *v1pb.RowValue_TimestampValue:
		s := kind.TimestampValue.GoogleTimestamp.AsTime().Format("2006-01-02 15:04:05.000000")
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: f(s),
			},
		}
	case *v1pb.RowValue_TimestampTzValue:
		t := kind.TimestampTzValue.GoogleTimestamp.AsTime()
		z := time.FixedZone(kind.TimestampTzValue.GetZone(), int(kind.TimestampTzValue.GetOffset()))
		s := t.In(z).Format(time.RFC3339Nano)
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(middle([]byte(s))),
			},
		}
	default:
	}

	return &v1pb.RowValue{
		Kind: &v1pb.RowValue_StringValue{
			StringValue: stringValue,
		},
	}
}

// Equal implements Masker.Equal.
func (m *MD5Masker) Equal(other Masker) bool {
	if otherMD5Masker, ok := other.(*MD5Masker); ok {
		return m.salt == otherMD5Masker.salt
	}
	return false
}

func maskProtoValue(m Masker, value *structpb.Value) *structpb.Value {
	switch kindValue := value.Kind.(type) {
	case *structpb.Value_NullValue:
		nullValue := m.Mask(&MaskData{
			Data: &v1pb.RowValue{
				Kind: &v1pb.RowValue_NullValue{
					NullValue: kindValue.NullValue,
				},
			},
		}).GetNullValue()
		return &structpb.Value{
			Kind: &structpb.Value_NullValue{
				NullValue: nullValue,
			},
		}
	case *structpb.Value_NumberValue:
		numberValue := m.Mask(&MaskData{
			Data: &v1pb.RowValue{
				Kind: &v1pb.RowValue_DoubleValue{
					DoubleValue: kindValue.NumberValue,
				},
			},
		}).GetDoubleValue()
		return &structpb.Value{
			Kind: &structpb.Value_NumberValue{
				NumberValue: numberValue,
			},
		}
	case *structpb.Value_StringValue:
		stringValue := m.Mask(&MaskData{
			Data: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: kindValue.StringValue,
				},
			},
		}).GetStringValue()
		return &structpb.Value{
			Kind: &structpb.Value_StringValue{
				StringValue: stringValue,
			},
		}
	case *structpb.Value_BoolValue:
		boolValue := m.Mask(&MaskData{
			Data: &v1pb.RowValue{
				Kind: &v1pb.RowValue_BoolValue{
					BoolValue: kindValue.BoolValue,
				},
			},
		}).GetBoolValue()
		return &structpb.Value{
			Kind: &structpb.Value_BoolValue{
				BoolValue: boolValue,
			},
		}
	case *structpb.Value_StructValue:
		v := proto.CloneOf(value)
		//nolint
		f := v.Kind.(*structpb.Value_StructValue)
		for field, value := range f.StructValue.Fields {
			kindValue.StructValue.Fields[field] = maskProtoValue(m, value)
		}
		return v
	case *structpb.Value_ListValue:
		v := proto.CloneOf(value)
		//nolint
		l := v.Kind.(*structpb.Value_ListValue)
		for i, value := range l.ListValue.Values {
			kindValue.ListValue.Values[i] = maskProtoValue(m, value)
		}
		return v
	}
	return nil
}

func NewInnerOuterMasker(storeMaskerType storepb.Algorithm_InnerOuterMask_MaskType, prefixLen, suffixLen int32, substitution string) (*InnerOuterMasker, error) {
	var maskerType int32
	switch storeMaskerType {
	case storepb.Algorithm_InnerOuterMask_INNER:
		maskerType = InnerOuterMaskerTypeInner
	case storepb.Algorithm_InnerOuterMask_OUTER:
		maskerType = InnerOuterMaskerTypeOuter
	default:
		return nil, errors.Errorf("invalid masker type %s", storeMaskerType)
	}

	return &InnerOuterMasker{
		maskerType:   int32(maskerType),
		prefixLen:    prefixLen,
		suffixLen:    suffixLen,
		substitution: substitution,
	}, nil
}

type InnerOuterMaskerType = int32

const (
	InnerOuterMaskerTypeUndefined InnerOuterMaskerType = 0
	InnerOuterMaskerTypeInner     InnerOuterMaskerType = 1
	InnerOuterMaskerTypeOuter     InnerOuterMaskerType = 2
)

type InnerOuterMasker struct {
	prefixLen    int32
	suffixLen    int32
	maskerType   InnerOuterMaskerType
	substitution string
}

func (m *InnerOuterMasker) Equal(other Masker) bool {
	if otherInnerOuterMasker, ok := other.(*InnerOuterMasker); ok {
		return otherInnerOuterMasker.maskerType == m.maskerType && otherInnerOuterMasker.prefixLen == m.prefixLen && otherInnerOuterMasker.suffixLen == m.suffixLen
	}
	return false
}

// Mask implements Masker.
func (m *InnerOuterMasker) Mask(data *MaskData) *v1pb.RowValue {
	unmaskedData := ""
	isDataNullOrBool := false
	switch kind := data.Data.Kind.(type) {
	case *v1pb.RowValue_NullValue:
		isDataNullOrBool = true
	case *v1pb.RowValue_BoolValue:
		isDataNullOrBool = true
	case *v1pb.RowValue_BytesValue:
		unmaskedData = string(kind.BytesValue)
	case *v1pb.RowValue_DoubleValue:
		unmaskedData = strconv.FormatFloat(kind.DoubleValue, 'f', -1, 64)
	case *v1pb.RowValue_FloatValue:
		unmaskedData = strconv.FormatFloat(float64(kind.FloatValue), 'f', -1, 64)
	case *v1pb.RowValue_Int32Value:
		unmaskedData = strconv.FormatInt(int64(kind.Int32Value), 10)
	case *v1pb.RowValue_Int64Value:
		unmaskedData = strconv.FormatInt(kind.Int64Value, 10)
	case *v1pb.RowValue_StringValue:
		unmaskedData = kind.StringValue
	case *v1pb.RowValue_Uint32Value:
		unmaskedData = strconv.FormatUint(uint64(kind.Uint32Value), 10)
	case *v1pb.RowValue_Uint64Value:
		unmaskedData = strconv.FormatUint(kind.Uint64Value, 10)
	case *v1pb.RowValue_ValueValue:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_ValueValue{
				ValueValue: maskProtoValue(m, kind.ValueValue),
			},
		}
	case *v1pb.RowValue_TimestampValue:
		s := kind.TimestampValue.GoogleTimestamp.AsTime().Format("2006-01-02 15:04:05.000000")
		unmaskedData = s
	case *v1pb.RowValue_TimestampTzValue:
		t := kind.TimestampTzValue.GoogleTimestamp.AsTime()
		z := time.FixedZone(kind.TimestampTzValue.GetZone(), int(kind.TimestampTzValue.GetOffset()))
		s := t.In(z).Format(time.RFC3339Nano)
		unmaskedData = s
	}
	if isDataNullOrBool {
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: strings.Repeat(m.substitution, 6),
			},
		}
	}

	maskedData := ""
	// Do nothing if the sum of the margin excced the length of the data.
	// Reference: https://dev.mysql.com/doc/refman/5.7/en/data-masking-functions.html#function_mask-inner.
	if !m.isMarginSumWithinRange(len([]rune(unmaskedData))) {
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: unmaskedData,
			},
		}
	}
	// Return null value if either margin is negetive.
	if m.isMarginNegetive() {
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_NullValue{},
		}
	}

	switch m.maskerType {
	case InnerOuterMaskerTypeInner:
		maskedData = m.maskInner([]rune(unmaskedData))
	case InnerOuterMaskerTypeOuter:
		maskedData = m.maskOuter([]rune(unmaskedData))
	default:
		// For undefined or unknown masker types, return the original data unmasked
		maskedData = unmaskedData
	}

	return &v1pb.RowValue{
		Kind: &v1pb.RowValue_StringValue{
			StringValue: maskedData,
		},
	}
}

// Return true if either margin is negetive.
func (m *InnerOuterMasker) isMarginSumWithinRange(lenRange int) bool {
	return m.prefixLen+m.suffixLen <= int32(lenRange)
}

// Return true if either margin is negetive.
func (m *InnerOuterMasker) isMarginNegetive() bool {
	return int(m.prefixLen) < 0 || int(m.suffixLen) < 0
}

func (m *InnerOuterMasker) maskInner(data []rune) string {
	maskedData := ""
	maxSuffixLen := len(data) - int(m.prefixLen)
	if maskLen := maxSuffixLen - int(m.suffixLen); maskLen > 0 {
		builder := strings.Builder{}
		if _, err := builder.WriteString(string(data[:m.prefixLen])); err != nil {
			return ""
		}
		if _, err := builder.WriteString(strings.Repeat(m.substitution, maskLen)); err != nil {
			return ""
		}
		if _, err := builder.WriteString(string(data[m.prefixLen+int32(maskLen):])); err != nil {
			return ""
		}
		maskedData = builder.String()
	} else {
		maskedData = string(data)
	}
	return maskedData
}

func (m *InnerOuterMasker) maskOuter(data []rune) string {
	maskedData := ""
	maxSuffixLen := len(data) - int(m.prefixLen)
	if dataLen := maxSuffixLen - int(m.suffixLen); dataLen > 0 {
		builder := strings.Builder{}
		dataStartIdx := m.prefixLen
		dataEndIdx := m.prefixLen + int32(dataLen)
		if _, err := builder.WriteString(strings.Repeat(m.substitution, int(m.prefixLen))); err != nil {
			return ""
		}
		if _, err := builder.WriteString(string(data[dataStartIdx:dataEndIdx])); err != nil {
			return ""
		}
		if _, err := builder.WriteString(strings.Repeat(m.substitution, int(m.suffixLen))); err != nil {
			return ""
		}
		maskedData = builder.String()
	} else {
		maskedData = strings.Repeat(m.substitution, len(data))
	}
	return maskedData
}

var _ Masker = (*InnerOuterMasker)(nil)
