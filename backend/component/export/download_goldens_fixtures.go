// backend/component/export/download_goldens_fixtures.go
package export

import (
	"math"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// downloadFixture identifies a curated QueryResult for golden generation.
type downloadFixture struct {
	id     string
	result *v1pb.QueryResult
}

func downloadFixtures() []downloadFixture {
	return []downloadFixture{
		{id: "empty_no_columns_no_rows", result: &v1pb.QueryResult{}},
		{id: "empty_columns_no_rows", result: &v1pb.QueryResult{
			ColumnNames:     []string{"a", "b"},
			ColumnTypeNames: []string{"INT", "TEXT"},
		}},
		{id: "ascii_basic", result: &v1pb.QueryResult{
			ColumnNames:     []string{"id", "name"},
			ColumnTypeNames: []string{"INT", "TEXT"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{intVal(1), strVal("Alice")}},
				{Values: []*v1pb.RowValue{intVal(2), strVal("Bob")}},
			},
		}},
		{id: "string_escapes", result: &v1pb.QueryResult{
			ColumnNames:     []string{"s"},
			ColumnTypeNames: []string{"TEXT"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{strVal(`it's`)}},
				{Values: []*v1pb.RowValue{strVal(`a"b`)}},
				{Values: []*v1pb.RowValue{strVal("a\\b")}},
				{Values: []*v1pb.RowValue{strVal("a\nb")}},
				{Values: []*v1pb.RowValue{strVal("a\r\nb")}},
				{Values: []*v1pb.RowValue{strVal("a\tb")}},
				{Values: []*v1pb.RowValue{strVal("a\x00b")}},
				{Values: []*v1pb.RowValue{strVal("a\x1bb")}},
				{Values: []*v1pb.RowValue{strVal("中文 👍")}},
			},
		}},
		{id: "bytes_full_range", result: &v1pb.QueryResult{
			ColumnNames:     []string{"b"},
			ColumnTypeNames: []string{"BLOB"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{bytesVal([]byte{})}},
				{Values: []*v1pb.RowValue{bytesVal([]byte{0x00})}},
				{Values: []*v1pb.RowValue{bytesVal(allBytes())}},
			},
		}},
		{id: "ints_edges", result: &v1pb.QueryResult{
			ColumnNames:     []string{"i32", "i64", "u32", "u64"},
			ColumnTypeNames: []string{"INT", "BIGINT", "INT UNSIGNED", "BIGINT UNSIGNED"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{i32Val(0), i64Val(0), u32Val(0), u64Val(0)}},
				{Values: []*v1pb.RowValue{i32Val(math.MinInt32), i64Val(math.MinInt64), u32Val(0), u64Val(0)}},
				{Values: []*v1pb.RowValue{i32Val(math.MaxInt32), i64Val(math.MaxInt64), u32Val(math.MaxUint32), u64Val(math.MaxUint64)}},
			},
		}},
		// B14: extreme float magnitudes — pinned via goldens because the exact
		// non-exponential decimal expansion of 1.7e308 (etc.) is fragile to
		// reproduce by hand. Catches drift in formatFloat64's expandExponential.
		{id: "floats_extreme_magnitudes", result: &v1pb.QueryResult{
			ColumnNames:     []string{"f64", "f32"},
			ColumnTypeNames: []string{"DOUBLE", "FLOAT"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{f64Val(math.MaxFloat64), f32Val(math.MaxFloat32)}},
				{Values: []*v1pb.RowValue{f64Val(math.SmallestNonzeroFloat64), f32Val(math.SmallestNonzeroFloat32)}},
				// Right at the JSON 'f' / 'e' threshold (1e21 and 1e-7).
				{Values: []*v1pb.RowValue{f64Val(1e21), f32Val(1e10)}},
				{Values: []*v1pb.RowValue{f64Val(1e-7), f32Val(1e-7)}},
				{Values: []*v1pb.RowValue{f64Val(9.999999999999998e20), f32Val(0)}},  // just below 1e21
				{Values: []*v1pb.RowValue{f64Val(1.0000000000000002e-6), f32Val(0)}}, // just above 1e-6
			},
		}},
		// Finite-only float fixtures (used for ALL formats, including JSON).
		{id: "floats_finite_edges", result: &v1pb.QueryResult{
			ColumnNames:     []string{"f32", "f64"},
			ColumnTypeNames: []string{"FLOAT", "DOUBLE"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{f32Val(0), f64Val(0)}},
				{Values: []*v1pb.RowValue{f32Val(float32(math.Copysign(0, -1))), f64Val(math.Copysign(0, -1))}},
				{Values: []*v1pb.RowValue{f32Val(1.5), f64Val(1.5)}},
				{Values: []*v1pb.RowValue{f32Val(1e10), f64Val(1e21)}},
				{Values: []*v1pb.RowValue{f32Val(1e-7), f64Val(1e-7)}},
			},
		}},
		// NaN/Inf are unrepresentable in Go's encoding/json — JSON goldens
		// are SKIPPED for this fixture; CSV/SQL/XLSX goldens still generate.
		// The TS-side JSON encoder must throw SerializationFailed on these
		// inputs; an explicit unit test in formats/json.test.ts covers that.
		{id: "floats_special_skip_json", result: &v1pb.QueryResult{
			ColumnNames:     []string{"f32", "f64"},
			ColumnTypeNames: []string{"FLOAT", "DOUBLE"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{f32Val(float32(math.NaN())), f64Val(math.NaN())}},
				{Values: []*v1pb.RowValue{f32Val(float32(math.Inf(1))), f64Val(math.Inf(1))}},
				{Values: []*v1pb.RowValue{f32Val(float32(math.Inf(-1))), f64Val(math.Inf(-1))}},
			},
		}},
		{id: "timestamps", result: &v1pb.QueryResult{
			ColumnNames:     []string{"ts", "tstz"},
			ColumnTypeNames: []string{"TIMESTAMP", "TIMESTAMPTZ"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{tsVal(0, 0), tstzVal(0, 0, "UTC", 0)}},
				{Values: []*v1pb.RowValue{tsVal(1700000000, 123456789), tstzVal(1700000000, 500000000, "JST", 9*3600)}},
				{Values: []*v1pb.RowValue{tsVal(1700000000, 123456000), tstzVal(1700000000, 0, "PDT", -7*3600)}},
			},
		}},
		{id: "structpb_kinds", result: &v1pb.QueryResult{
			ColumnNames:     []string{"v"},
			ColumnTypeNames: []string{"VARIANT"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{wrapStruct(structpb.NewNullValue())}},
				{Values: []*v1pb.RowValue{wrapStruct(structpb.NewStringValue("ab"))}},
				{Values: []*v1pb.RowValue{wrapStruct(structpb.NewNumberValue(1.5))}},
				{Values: []*v1pb.RowValue{wrapStruct(structpb.NewBoolValue(true))}},
				{Values: []*v1pb.RowValue{wrapStruct(mustList(structpb.NewNumberValue(1), structpb.NewStringValue("x")))}},
				{Values: []*v1pb.RowValue{wrapStruct(mustStruct(map[string]*structpb.Value{
					"b": structpb.NewNumberValue(2),
					"a": structpb.NewNumberValue(1),
				}))}},
			},
		}},
		// TODO(B2/B3): Backend currently does NOT escape column names embedded
		// in SQL identifier-quotes or CSV header rows. The goldens below
		// capture this CURRENT behavior so TS stays byte-equal. Fixing the
		// underlying gap requires a coordinated backend + TS change with
		// regenerated goldens — tracked as a follow-up.
		{id: "column_name_quotes_my", result: &v1pb.QueryResult{
			ColumnNames:     []string{"a`b", "c,d", "e\"f"},
			ColumnTypeNames: []string{"INT", "TEXT", "TEXT"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{intVal(1), strVal("x"), strVal("y")}},
			},
		}},
		{id: "column_name_quotes_pg", result: &v1pb.QueryResult{
			ColumnNames:     []string{"a\"b", "c,d", "e\nf"},
			ColumnTypeNames: []string{"INT", "TEXT", "TEXT"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{intVal(1), strVal("x"), strVal("y")}},
			},
		}},
		// Exercises B1 (prototext escaping in XLSX/JSON for structpb).
		// String values and struct keys carrying quote/backslash/control chars
		// must round-trip Go's proto.String() output byte-for-byte.
		{id: "structpb_string_with_quotes", result: &v1pb.QueryResult{
			ColumnNames:     []string{"v"},
			ColumnTypeNames: []string{"VARIANT"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{wrapStruct(structpb.NewStringValue(`a"b`))}},
				{Values: []*v1pb.RowValue{wrapStruct(structpb.NewStringValue("a\\b"))}},
				{Values: []*v1pb.RowValue{wrapStruct(structpb.NewStringValue("a\nb"))}},
				{Values: []*v1pb.RowValue{wrapStruct(structpb.NewStringValue("a\tb"))}},
				{Values: []*v1pb.RowValue{wrapStruct(structpb.NewStringValue("a\x00b"))}},
				// C1 control: Go prototext emits \uHHHH (4-digit), not \xHH.
				// See prototextEscape comment in frontend value.ts.
				{Values: []*v1pb.RowValue{wrapStruct(structpb.NewStringValue("a\u0080b"))}},
				{Values: []*v1pb.RowValue{wrapStruct(structpb.NewStringValue("a\u009fb"))}},
				{Values: []*v1pb.RowValue{wrapStruct(mustStruct(map[string]*structpb.Value{
					`k"y`: structpb.NewStringValue(`a"b`),
				}))}},
			},
		}},
		{id: "nulls_only", result: &v1pb.QueryResult{
			ColumnNames:     []string{"x", "y"},
			ColumnTypeNames: []string{"TEXT", "INT"},
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{nullVal(), nullVal()}},
			},
		}},
	}
}

func allBytes() []byte {
	out := make([]byte, 256)
	for i := 0; i < 256; i++ {
		out[i] = byte(i)
	}
	return out
}

func intVal(n int64) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: n}}
}
func i32Val(n int32) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: n}}
}
func i64Val(n int64) *v1pb.RowValue { return intVal(n) }
func u32Val(n uint32) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: n}}
}
func u64Val(n uint64) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: n}}
}
func f32Val(f float32) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: f}}
}
func f64Val(f float64) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: f}}
}
func strVal(s string) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: s}}
}
func bytesVal(b []byte) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: b}}
}
func nullVal() *v1pb.RowValue { return &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{}} }

func tsVal(seconds int64, nanos int32) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
		GoogleTimestamp: &timestamppb.Timestamp{Seconds: seconds, Nanos: nanos},
	}}}
}
func tstzVal(seconds int64, nanos int32, zone string, offset int32) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampTzValue{TimestampTzValue: &v1pb.RowValue_TimestampTZ{
		GoogleTimestamp: &timestamppb.Timestamp{Seconds: seconds, Nanos: nanos},
		Zone:            zone,
		Offset:          offset,
	}}}
}
func wrapStruct(v *structpb.Value) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_ValueValue{ValueValue: v}}
}
func mustList(vals ...*structpb.Value) *structpb.Value {
	return structpb.NewListValue(&structpb.ListValue{Values: vals})
}
func mustStruct(fields map[string]*structpb.Value) *structpb.Value {
	s := &structpb.Struct{Fields: fields}
	return structpb.NewStructValue(s)
}
