package masker

import (
	"database/sql"
	"fmt"
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
}

// NoneMasker is the masker that does not mask the data.
type NoneMasker struct{}

// NewNoneMasker returns a new NoneMasker.
func NewNoneMasker() *NoneMasker {
	return &NoneMasker{}
}

// Mask implements Masker.Mask.
func (*NoneMasker) Mask(data *MaskData) *v1pb.RowValue {
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

// FullMasker is the masker that masks the data with `substitution`.
type FullMasker struct {
	substitution string
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

// DefaultRangeMasker is the masker that masks the the left and right quarters with "**".
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
