package masker

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestMiddle_Byte(t *testing.T) {
	testCases := []struct {
		input []byte
		want  []byte
	}{
		{
			input: []byte("12"),
			want:  []byte("2"),
		},
		{
			input: []byte("123"),
			want:  []byte("2"),
		},
		{
			input: []byte("12345678"),
			want:  []byte("3456"),
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		got := middle(tc.input)
		a.Equal(tc.want, got)
	}
}

func TestMiddle_Rune(t *testing.T) {
	testCases := []struct {
		input []rune
		want  []rune
	}{
		{
			input: []rune("🥲🤣🥲🤣"),
			want:  []rune("🤣🥲"),
		},
		{
			input: []rune("abcdefghijklmnopqrstuvwxyzabcdef"),
			want:  []rune("ijklmnopqrstuvwx"),
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		got := middle(tc.input)
		a.Equal(tc.want, got)
	}
}

func TestRangeMask(t *testing.T) {
	testCases := []struct {
		description string
		input       *MaskData
		slices      []*MaskRangeSlice
		want        *v1pb.RowValue
	}{
		{
			description: "Single slice",
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: "012345678",
					},
				},
			},
			slices: []*MaskRangeSlice{
				{
					Start:        1,
					End:          3,
					Substitution: "###",
				},
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: "0###345678",
				},
			},
		},
		{
			description: "Single slice - only keep 1st raw value",
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: "012345678",
					},
				},
			},
			slices: []*MaskRangeSlice{
				{
					Start:        1,
					End:          -1,
					Substitution: "***",
				},
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: "0***",
				},
			},
		},
		{
			description: "Single slice - mask last 3 value",
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: "012345678",
					},
				},
			},
			slices: []*MaskRangeSlice{
				{
					Start:        -4,
					End:          -1,
					Substitution: "***",
				},
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: "012345***",
				},
			},
		},
		{
			description: "Multiple slices",
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: "012345678",
					},
				},
			},
			slices: []*MaskRangeSlice{
				{
					Start:        1,
					End:          3,
					Substitution: "###",
				},
				{
					Start:        5,
					End:          7,
					Substitution: "***",
				},
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: "0###34***78",
				},
			},
		},
		{
			description: "Multiple slices",
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: "012345678",
					},
				},
			},
			slices: []*MaskRangeSlice{
				{
					Start:        0,
					End:          2,
					Substitution: "###",
				},
				{
					Start:        -2,
					End:          -1,
					Substitution: "***",
				},
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: "###234567***",
				},
			},
		},
		{
			description: "Mask slices out of data range",
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: "0123",
					},
				},
			},
			slices: []*MaskRangeSlice{
				{
					Start:        1,
					End:          2,
					Substitution: "#",
				},
				{
					Start:        4,
					End:          10,
					Substitution: "***",
				},
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: "0#23",
				},
			},
		},
		{
			description: "Emoji",
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: "😂😠😡😊😂",
					},
				},
			},
			slices: []*MaskRangeSlice{
				{
					Start:        1,
					End:          4,
					Substitution: "😂😂😂",
				},
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: "😂😂😂😂😂",
				},
			},
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		rangeMasker := NewRangeMasker(tc.slices)
		got := rangeMasker.Mask(tc.input)
		a.Equal(tc.want, got, "description: %s", tc.description)
	}
}

func TestInnerOuterMask(t *testing.T) {
	testCases := []struct {
		input  *MaskData
		masker InnerOuterMasker
		want   *v1pb.RowValue
	}{
		{
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: "012345678",
					},
				},
			},
			masker: InnerOuterMasker{
				maskerType:   InnerOuterMaskerTypeOuter,
				prefixLen:    1,
				suffixLen:    3,
				substitution: "#",
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{StringValue: "#12345###"},
			},
		},
		{
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_Int64Value{
						Int64Value: 1,
					},
				},
			},
			masker: InnerOuterMasker{
				maskerType:   InnerOuterMaskerTypeOuter,
				prefixLen:    2,
				suffixLen:    3,
				substitution: "*",
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{StringValue: "1"},
			},
		},
		{
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_DoubleValue{
						DoubleValue: 123.4567,
					},
				},
			},
			masker: InnerOuterMasker{
				maskerType:   InnerOuterMaskerTypeOuter,
				prefixLen:    1,
				suffixLen:    2,
				substitution: "*",
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{StringValue: "*23.45**"},
			},
		},
		{
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_Int64Value{
						Int64Value: 27865874362589245,
					},
				},
			},
			masker: InnerOuterMasker{
				maskerType:   InnerOuterMaskerTypeInner,
				prefixLen:    6,
				suffixLen:    3,
				substitution: "*",
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{StringValue: "278658********245"},
			},
		},
		{
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: "😂😠😡😊😂",
					},
				},
			},
			masker: InnerOuterMasker{
				maskerType:   InnerOuterMaskerTypeInner,
				prefixLen:    1,
				suffixLen:    1,
				substitution: "xx",
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{StringValue: "😂xxxxxx😂"},
			},
		},
		{
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_NullValue{
						NullValue: structpb.NullValue_NULL_VALUE,
					},
				},
			},
			masker: InnerOuterMasker{
				maskerType:   InnerOuterMaskerTypeInner,
				prefixLen:    1,
				suffixLen:    2,
				substitution: "*",
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{StringValue: "******"},
			},
		},
		{
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: "1234",
					},
				},
			},
			masker: InnerOuterMasker{
				maskerType:   InnerOuterMaskerTypeInner,
				prefixLen:    1000,
				suffixLen:    10000,
				substitution: "*",
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{StringValue: "1234"},
			},
		},
		{
			input: &MaskData{
				Data: &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: "Love this LMG",
					},
				},
			},
			masker: InnerOuterMasker{
				maskerType:   InnerOuterMaskerTypeInner,
				prefixLen:    -1,
				suffixLen:    1,
				substitution: "*",
			},
			want: &v1pb.RowValue{
				Kind: &v1pb.RowValue_NullValue{},
			},
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		got := tc.masker.Mask(tc.input)
		a.Equal(tc.want, got)
	}
}
