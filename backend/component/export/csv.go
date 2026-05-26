package export

import (
	"encoding/hex"
	"io"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// CSV exports query results as CSV format (legacy wrapper).
func CSV(result *v1pb.QueryResult) ([]byte, error) {
	return exportToBytes(result, CSVToWriter)
}

// CSVToWriter streams query results as CSV directly to the writer.
// This minimizes memory usage by avoiding intermediate buffering.
func CSVToWriter(w io.Writer, result *v1pb.QueryResult) error {
	if _, err := w.Write([]byte(strings.Join(result.ColumnNames, ","))); err != nil {
		return err
	}
	if _, err := w.Write([]byte{'\n'}); err != nil {
		return err
	}
	for i, row := range result.Rows {
		for j, value := range row.Values {
			if j != 0 {
				if _, err := w.Write([]byte{','}); err != nil {
					return err
				}
			}
			if _, err := w.Write(convertValueToBytesInCSV(value)); err != nil {
				return err
			}
		}
		if i != len(result.Rows)-1 {
			if _, err := w.Write([]byte{'\n'}); err != nil {
				return err
			}
		}
	}
	return nil
}

func convertValueToBytesInCSV(value *v1pb.RowValue) []byte {
	if value == nil || value.Kind == nil {
		return []byte("")
	}
	switch value.Kind.(type) {
	case *v1pb.RowValue_StringValue:
		return quoteCSVString(value.GetStringValue())
	case *v1pb.RowValue_Int32Value:
		return []byte(strconv.FormatInt(int64(value.GetInt32Value()), 10))
	case *v1pb.RowValue_Int64Value:
		return []byte(strconv.FormatInt(value.GetInt64Value(), 10))
	case *v1pb.RowValue_Uint32Value:
		return []byte(strconv.FormatUint(uint64(value.GetUint32Value()), 10))
	case *v1pb.RowValue_Uint64Value:
		return []byte(strconv.FormatUint(value.GetUint64Value(), 10))
	case *v1pb.RowValue_FloatValue:
		return []byte(strconv.FormatFloat(float64(value.GetFloatValue()), 'f', -1, 32))
	case *v1pb.RowValue_DoubleValue:
		return []byte(strconv.FormatFloat(value.GetDoubleValue(), 'f', -1, 64))
	case *v1pb.RowValue_BoolValue:
		return []byte(strconv.FormatBool(value.GetBoolValue()))
	case *v1pb.RowValue_BytesValue:
		hexStr := "0x" + hex.EncodeToString(value.GetBytesValue())
		return quoteCSVString(hexStr)
	case *v1pb.RowValue_NullValue:
		return []byte("")
	case *v1pb.RowValue_TimestampValue:
		return quoteCSVString(formatTimestamp(value.GetTimestampValue()))
	case *v1pb.RowValue_TimestampTzValue:
		return quoteCSVString(formatTimestampTz(value.GetTimestampTzValue()))
	case *v1pb.RowValue_ValueValue:
		return convertValueValueToBytes(value.GetValueValue())
	default:
		return []byte("")
	}
}

// quoteCSVString wraps a string in double quotes and escapes internal quotes.
func quoteCSVString(s string) []byte {
	escaped := escapeCSVString(s)
	result := make([]byte, 0, len(escaped)+2)
	result = append(result, '"')
	result = append(result, []byte(escaped)...)
	result = append(result, '"')
	return result
}

func escapeCSVString(str string) string {
	return strings.ReplaceAll(str, `"`, `""`)
}

// convertValueValueToBytes renders a structpb.Value as a CSV-quoted JSON cell.
// Tier 2: inner content is the canonical compact JSON form (was a custom
// bracket-form prototext-style encoding under Tier 1); the CSV-style outer
// quoting + double-quote doubling stays so the cell remains a valid CSV
// (and incidentally a valid SQL literal-ish — sql.go reuses this same encoding
// because structpb cells in SQL exports historically went through CSV-style
// quoting, not engine-native SQL string quoting).
func convertValueValueToBytes(value *structpb.Value) []byte {
	s := structpbValueAsJSON(value)
	return []byte(`"` + strings.ReplaceAll(s, `"`, `""`) + `"`)
}
