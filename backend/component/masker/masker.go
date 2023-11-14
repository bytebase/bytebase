package masker

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strconv"

	"google.golang.org/protobuf/types/known/structpb"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// MaskData is the data to be masked.
type MaskData struct {
	// any is the data to be masked, it can be assigned with the following types:
	// *sql.NullString
	// *sql.NullBool
	// *sql.NullInt32
	// *sql.NullInt64
	// *sql.NullFloat64
	Data any
	// WantBytes indicates whether the Masker should return the masked data as
	// *v1pb.RowValue_BytesValue.
	WantBytes bool
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
	switch raw := data.Data.(type) {
	case *sql.NullBool:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_BoolValue{
					BoolValue: raw.Bool,
				},
			}
		}
	case *sql.NullString:
		if raw.Valid {
			if data.WantBytes {
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_BytesValue{
						BytesValue: []byte(raw.String),
					},
				}
			}
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: raw.String,
				},
			}
		}
	case *sql.NullInt32:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_Int32Value{
					Int32Value: raw.Int32,
				},
			}
		}
	case *sql.NullInt64:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_Int64Value{
					Int64Value: raw.Int64,
				},
			}
		}
	case *sql.NullFloat64:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_DoubleValue{
					DoubleValue: raw.Float64,
				},
			}
		}
	}
	return &v1pb.RowValue{
		Kind: &v1pb.RowValue_NullValue{
			NullValue: structpb.NullValue_NULL_VALUE,
		},
	}
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
	sort.SliceStable(maskRangeSlice, func(i, j int) bool {
		return maskRangeSlice[i].Start < maskRangeSlice[j].Start
	})
	// Merge the overlapping ranges.
	var mergedMaskRangeSlice []*MaskRangeSlice
	for _, maskRange := range maskRangeSlice {
		if maskRange.Start > maskRange.End {
			maskRange.End = maskRange.Start + 1
		}
		if len(mergedMaskRangeSlice) == 0 {
			mergedMaskRangeSlice = append(mergedMaskRangeSlice, maskRange)
			continue
		}
		lastMaskRange := mergedMaskRangeSlice[len(mergedMaskRangeSlice)-1]
		if lastMaskRange.End >= maskRange.Start {
			mergedMaskRangeSlice[len(mergedMaskRangeSlice)-1].End = maskRange.End
		} else {
			mergedMaskRangeSlice = append(mergedMaskRangeSlice, maskRange)
		}
	}
	return &RangeMasker{
		MaskRangeSlice: mergedMaskRangeSlice,
	}
}

func (m *RangeMasker) enableMask() bool {
	return len(m.MaskRangeSlice) > 0
}

// Mask implements Masker.Mask.
func (m *RangeMasker) Mask(data *MaskData) *v1pb.RowValue {
	fRune := func(s []rune) []rune {
		if !m.enableMask() {
			return s
		}

		var ret []rune
		prevEnd := 0
		for _, maskRange := range m.MaskRangeSlice {
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
		if !m.enableMask() {
			return s
		}

		var ret []byte
		prevEnd := 0
		for _, maskRange := range m.MaskRangeSlice {
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
	if !m.enableMask() {
		return noneMask(data)
	}
	var stringValue string
	var valid bool
	switch raw := data.Data.(type) {
	case *sql.NullBool:
		if raw.Valid {
			stringValue = "******"
			valid = true
		}
	case *sql.NullString:
		if raw.Valid {
			if data.WantBytes {
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_BytesValue{
						BytesValue: fBytes([]byte(raw.String)),
					},
				}
			}
			stringValue = string(fRune([]rune(raw.String)))
			valid = true
		}
	case *sql.NullInt32:
		if raw.Valid {
			stringValue = string(fBytes([]byte(strconv.FormatInt(int64(raw.Int32), 10))))
			valid = true
		}
	case *sql.NullInt64:
		if raw.Valid {
			stringValue = string(fBytes([]byte(strconv.FormatInt(raw.Int64, 10))))
			valid = true
		}
	case *sql.NullFloat64:
		if raw.Valid {
			stringValue = string(fBytes([]byte(strconv.FormatFloat(raw.Float64, 'f', -1, 64))))
			valid = true
		}
	}
	if !valid {
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_NullValue{
				NullValue: structpb.NullValue_NULL_VALUE,
			},
		}
	}
	return &v1pb.RowValue{
		Kind: &v1pb.RowValue_StringValue{
			StringValue: stringValue,
		},
	}
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
	stringValue := ""
	switch raw := data.Data.(type) {
	case *sql.NullBool:
		stringValue = ""
	case *sql.NullString:
		if data.WantBytes {
			bytesValue := append([]byte{'*', '*'}, middle([]byte(raw.String))...)
			bytesValue = append(bytesValue, []byte{'*', '*'}...)
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_BytesValue{
					BytesValue: bytesValue,
				},
			}
		}
		stringValue = string(middle([]rune(raw.String)))
	case *sql.NullInt64:
		s := strconv.FormatInt(raw.Int64, 10)
		stringValue = string(middle([]byte(s)))
	case *sql.NullInt32:
		s := strconv.FormatInt(int64(raw.Int32), 10)
		stringValue = string(middle([]byte(s)))
	case *sql.NullFloat64:
		s := strconv.FormatFloat(raw.Float64, 'f', -1, 64)
		stringValue = string(middle([]byte(s)))
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
			slog.Error("Failed to write to md5 hash: %v", err)
		}
		return fmt.Sprintf("%x", h.Sum(nil))
	}

	var stringValue string
	switch raw := data.Data.(type) {
	case *sql.NullBool:
		if raw.Valid {
			stringValue = f(strconv.FormatBool(raw.Bool))
		}
	case *sql.NullString:
		if raw.Valid {
			stringValue = f(raw.String)
		}
	case *sql.NullInt32:
		if raw.Valid {
			stringValue = f(strconv.FormatInt(int64(raw.Int32), 10))
		}
	case *sql.NullInt64:
		if raw.Valid {
			stringValue = f(strconv.FormatInt(raw.Int64, 10))
		}
	case *sql.NullFloat64:
		if raw.Valid {
			stringValue = f(strconv.FormatFloat(raw.Float64, 'f', -1, 64))
		}
	}
	if stringValue == "" {
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_NullValue{
				NullValue: structpb.NullValue_NULL_VALUE,
			},
		}
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
