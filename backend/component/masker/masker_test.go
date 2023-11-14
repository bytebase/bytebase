package masker

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
				Data: &sql.NullString{String: "012345678", Valid: true},
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
			description: "Multiple slices",
			input: &MaskData{
				Data: &sql.NullString{String: "012345678", Valid: true},
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
			description: "Mask slices out of data range",
			input: &MaskData{
				Data: &sql.NullString{String: "0123", Valid: true},
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
				Data: &sql.NullString{String: "😂😠😡😊😂", Valid: true},
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
