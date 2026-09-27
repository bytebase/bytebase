package mongodb

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestFormatDoubleJS(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "0"},
		// Deviates from ECMAScript String(-0) == "0" to keep the sign
		// actually stored in the BSON double.
		{math.Copysign(0, -1), "-0"},
		{1, "1"},
		{-1, "-1"},
		{42, "42"},
		{999999, "999999"},
		{1000000, "1000000"},
		{1234567.5, "1234567.5"},
		{-1234567.5, "-1234567.5"},
		{123456.789, "123456.789"},
		{0.5, "0.5"},
		{0.1, "0.1"},
		{123.25, "123.25"},
		{0.000001, "0.000001"},
		{0.0000001, "1e-7"},
		{-0.0000001, "-1e-7"},
		{1e-10, "1e-10"},
		{2.5e-10, "2.5e-10"},
		{5e-324, "5e-324"},
		{0.30000000000000004, "0.30000000000000004"},
		{1779696815227, "1779696815227"},
		{585723378473606, "585723378473606"},
		{9007199254740992, "9007199254740992"},
		{123456789123456789, "123456789123456780"},
		{36893488147419103232, "36893488147419103000"},
		{1e20, "100000000000000000000"},
		{999999999999999900000, "999999999999999900000"},
		{1e21, "1e+21"},
		{-1e21, "-1e+21"},
		{1.5e21, "1.5e+21"},
		{math.MaxFloat64, "1.7976931348623157e+308"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, formatDoubleJS(tt.input))
		})
	}
}

func TestNormalizeExtJSONNumbers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "issue 20895 document",
			input: `{"_id":1129063441,"name":"repro user 1","created_at":1.779696815227E+12,"updated_at":1.779803079861E+12,"current_workspace_id":5.85723378473606E+14}`,
			want:  `{"_id":1129063441,"name":"repro user 1","created_at":1779696815227,"updated_at":1779803079861,"current_workspace_id":585723378473606}`,
		},
		{
			name:  "integral double drops .0",
			input: `{"score":42.0}`,
			want:  `{"score":42}`,
		},
		{
			name:  "integer tokens unchanged",
			input: `{"a":42,"b":9223372036854775807,"c":-15}`,
			want:  `{"a":42,"b":9223372036854775807,"c":-15}`,
		},
		{
			name:  "small scientific uses unpadded exponent",
			input: `{"x":1E-07}`,
			want:  `{"x":1e-7}`,
		},
		{
			name:  "huge doubles stay scientific",
			input: `{"x":1.5E+21}`,
			want:  `{"x":1.5e+21}`,
		},
		{
			name:  "numeric-looking strings and $numberDecimal untouched",
			input: `{"s":"1.5E+10","d":{"$numberDecimal":"1.5E+100"}}`,
			want:  `{"s":"1.5E+10","d":{"$numberDecimal":"1.5E+100"}}`,
		},
		{
			name:  "$numberDouble special values untouched",
			input: `{"nan":{"$numberDouble":"NaN"},"inf":{"$numberDouble":"Infinity"}}`,
			want:  `{"nan":{"$numberDouble":"NaN"},"inf":{"$numberDouble":"Infinity"}}`,
		},
		{
			name:  "nested arrays and objects",
			input: `{"arr":[1.0E+07,2,"x",{"n":9.9E-08}],"b":true,"z":null}`,
			want:  `{"arr":[10000000,2,"x",{"n":9.9e-8}],"b":true,"z":null}`,
		},
		{
			name:  "negative zero keeps sign",
			input: `{"z":-0.0}`,
			want:  `{"z":-0}`,
		},
		{
			name:  "string escapes survive",
			input: `{"s":"a\"b\\c\né"}`,
			want:  `{"s":"a\"b\\c\né"}`,
		},
		{
			name:  "html characters not escaped",
			input: `{"s":"<a>&"}`,
			want:  `{"s":"<a>&"}`,
		},
		{
			name:  "top-level array",
			input: `[1.0E+06,{"k":2.5}]`,
			want:  `[1000000,{"k":2.5}]`,
		},
		{
			name:  "invalid json returned unchanged",
			input: `{"a":`,
			want:  `{"a":`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, string(normalizeExtJSONNumbers([]byte(tt.input))))
		})
	}
}

func TestMarshalValueToExtJSONNumberNotation(t *testing.T) {
	decimal, err := bson.ParseDecimal128("1.5E+100")
	require.NoError(t, err)

	tests := []struct {
		name  string
		input any
		want  string
	}{
		{
			name: "doubles render like mongosh",
			input: bson.D{
				{Key: "_id", Value: int32(1129063441)},
				{Key: "created_at", Value: float64(1779696815227)},
				{Key: "current_workspace_id", Value: float64(585723378473606)},
				{Key: "score", Value: 42.5},
				{Key: "whole", Value: float64(42)},
			},
			want: `{"_id":1129063441,"created_at":1779696815227,"current_workspace_id":585723378473606,"score":42.5,"whole":42}`,
		},
		{
			name:  "int64 precision preserved",
			input: bson.D{{Key: "big", Value: int64(9223372036854775807)}},
			want:  `{"big":9223372036854775807}`,
		},
		{
			name: "special doubles and decimal128 keep extended json form",
			input: bson.D{
				{Key: "nan", Value: math.NaN()},
				{Key: "dec", Value: decimal},
			},
			want: `{"nan":{"$numberDouble":"NaN"},"dec":{"$numberDecimal":"1.5E+100"}}`,
		},
		{
			name:  "primitive value wrapped",
			input: int64(5),
			want:  `{"value":5}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := marshalValueToExtJSON(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
