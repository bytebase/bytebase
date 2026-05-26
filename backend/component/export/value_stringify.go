package export

import (
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

// structpbValueAsJSON renders a structpb.Value as compact, deterministic JSON.
// The encoding is used for VARIANT-style cell content across CSV / JSON / SQL / XLSX
// exports. Frontend mirror lives at frontend/src/utils/sql-download/value.ts
// (structpbValueAsJSON). Tier 2 dropped the cross-side byte-equality requirement,
// so the two encodings agree for ASCII content but may differ on HTML-unsafe
// characters (Go's json.Marshal escapes <, >, & by default; JS JSON.stringify
// emits them verbatim) and on struct field order (Go sorts, JS preserves
// insertion order).
func structpbValueAsJSON(v *structpb.Value) string {
	if v == nil || v.Kind == nil {
		return "null"
	}
	switch k := v.Kind.(type) {
	case *structpb.Value_NullValue:
		return "null"
	case *structpb.Value_BoolValue:
		if k.BoolValue {
			return "true"
		}
		return "false"
	case *structpb.Value_NumberValue:
		n := k.NumberValue
		if math.IsNaN(n) || math.IsInf(n, 0) {
			// JSON spec forbids NaN / ±Inf — mirror JSON.stringify and emit null.
			return "null"
		}
		return strconv.FormatFloat(n, 'g', -1, 64)
	case *structpb.Value_StringValue:
		b, _ := json.Marshal(k.StringValue)
		return string(b)
	case *structpb.Value_ListValue:
		vals := k.ListValue.GetValues()
		var sb strings.Builder
		sb.WriteByte('[')
		for i, x := range vals {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(structpbValueAsJSON(x))
		}
		sb.WriteByte(']')
		return sb.String()
	case *structpb.Value_StructValue:
		fields := k.StructValue.GetFields()
		keys := make([]string, 0, len(fields))
		for kn := range fields {
			keys = append(keys, kn)
		}
		slices.Sort(keys)
		var sb strings.Builder
		sb.WriteByte('{')
		for i, kn := range keys {
			if i > 0 {
				sb.WriteByte(',')
			}
			b, _ := json.Marshal(kn)
			sb.Write(b)
			sb.WriteByte(':')
			sb.WriteString(structpbValueAsJSON(fields[kn]))
		}
		sb.WriteByte('}')
		return sb.String()
	}
	return "null"
}
