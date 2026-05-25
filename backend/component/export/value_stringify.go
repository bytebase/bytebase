package export

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

// protoTextStringifyValue returns a deterministic prototext-like rendering of a
// structpb.Value. Behaves like (*structpb.Value).String() but sorts Struct
// fields alphabetically so output is stable across Go map iteration ordering.
// Frontend mirror lives at frontend/src/utils/sql-download/value.ts
// (xlsxStringFromStructpbValue) and must stay byte-equal.
func protoTextStringifyValue(v *structpb.Value) string {
	if v == nil || v.Kind == nil {
		return ""
	}
	switch k := v.Kind.(type) {
	case *structpb.Value_NullValue:
		return "null_value:NULL_VALUE"
	case *structpb.Value_StringValue:
		return `string_value:"` + prototextEscape(k.StringValue) + `"`
	case *structpb.Value_NumberValue:
		return "number_value:" + strconv.FormatFloat(k.NumberValue, 'g', -1, 64)
	case *structpb.Value_BoolValue:
		if k.BoolValue {
			return "bool_value:true"
		}
		return "bool_value:false"
	case *structpb.Value_ListValue:
		vals := k.ListValue.GetValues()
		parts := make([]string, 0, len(vals))
		for _, x := range vals {
			parts = append(parts, "values:{"+protoTextStringifyValue(x)+"}")
		}
		return "list_value:{" + strings.Join(parts, " ") + "}"
	case *structpb.Value_StructValue:
		fields := k.StructValue.GetFields()
		keys := make([]string, 0, len(fields))
		for kn := range fields {
			keys = append(keys, kn)
		}
		slices.Sort(keys)
		parts := make([]string, 0, len(keys))
		for _, kn := range keys {
			parts = append(parts, `fields:{key:"`+prototextEscape(kn)+`" value:{`+protoTextStringifyValue(fields[kn])+`}}`)
		}
		return "struct_value:{" + strings.Join(parts, " ") + "}"
	}
	return ""
}

// prototextEscape escapes a string the way Go's prototext renders a string
// literal inside (*structpb.Value).String(). Mirrors the frontend helper of
// the same name in frontend/src/utils/sql-download/value.ts.
func prototextEscape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\\':
			b.WriteString(`\\`)
		case r == '"':
			b.WriteString(`\"`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\x%02x`, r)
		case r >= 0x80 && r <= 0x9f:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
