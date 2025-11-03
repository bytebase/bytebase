package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// ExportJSON exports query results as JSON format (legacy wrapper).
func ExportJSON(result *v1pb.QueryResult) ([]byte, error) {
	return exportToBytes(result, ExportJSONToWriter)
}

// ExportJSONToWriter streams query results as pretty-printed JSON directly to the writer.
func ExportJSONToWriter(w io.Writer, result *v1pb.QueryResult) error {
	records := make([]map[string]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		record := make(map[string]any, len(result.ColumnNames))
		for i, value := range row.Values {
			record[result.ColumnNames[i]] = convertValueToJSONValue(value)
		}
		records = append(records, record)
	}

	jsonBytes, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return errors.Errorf("failed to encode JSON: %v", err)
	}
	if _, err := w.Write(jsonBytes); err != nil {
		return err
	}
	return nil
}

func convertValueToJSONValue(value *v1pb.RowValue) any {
	if value == nil || value.Kind == nil {
		return nil
	}

	switch value.Kind.(type) {
	case *v1pb.RowValue_StringValue:
		return value.GetStringValue()
	case *v1pb.RowValue_Int32Value:
		return value.GetInt32Value()
	case *v1pb.RowValue_Int64Value:
		return value.GetInt64Value()
	case *v1pb.RowValue_Uint32Value:
		return value.GetUint32Value()
	case *v1pb.RowValue_Uint64Value:
		return value.GetUint64Value()
	case *v1pb.RowValue_FloatValue:
		return value.GetFloatValue()
	case *v1pb.RowValue_DoubleValue:
		return value.GetDoubleValue()
	case *v1pb.RowValue_BoolValue:
		return value.GetBoolValue()
	case *v1pb.RowValue_BytesValue:
		binaryStr, err := convertBytesToBinaryString(value.GetBytesValue())
		if err != nil {
			return nil
		}
		return binaryStr
	case *v1pb.RowValue_NullValue:
		return nil
	case *v1pb.RowValue_TimestampValue:
		return value.GetTimestampValue().GoogleTimestamp.AsTime().Format("2006-01-02 15:04:05.000000")
	case *v1pb.RowValue_TimestampTzValue:
		t := value.GetTimestampTzValue().GoogleTimestamp.AsTime()
		z := time.FixedZone(value.GetTimestampTzValue().GetZone(), int(value.GetTimestampTzValue().GetOffset()))
		return t.In(z).Format(time.RFC3339Nano)
	case *v1pb.RowValue_ValueValue:
		return value.GetValueValue().String()
	default:
		return nil
	}
}

func convertBytesToBinaryString(bs []byte) (string, error) {
	var buf bytes.Buffer
	if _, err := buf.WriteString("0b"); err != nil {
		return "", err
	}
	for _, b := range bs {
		if _, err := buf.WriteString(fmt.Sprintf("%08b", b)); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}
