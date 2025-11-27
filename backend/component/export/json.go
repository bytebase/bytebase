package export

import (
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// JSON exports query results as JSON format (legacy wrapper).
func JSON(result *v1pb.QueryResult) ([]byte, error) {
	return exportToBytes(result, JSONToWriter)
}

// JSONToWriter streams query results as pretty-printed JSON directly to the writer.
func JSONToWriter(w io.Writer, result *v1pb.QueryResult) error {
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
		return base64.StdEncoding.EncodeToString(value.GetBytesValue())
	case *v1pb.RowValue_NullValue:
		return nil
	case *v1pb.RowValue_TimestampValue:
		return formatTimestamp(value.GetTimestampValue())
	case *v1pb.RowValue_TimestampTzValue:
		return formatTimestampTz(value.GetTimestampTzValue())
	case *v1pb.RowValue_ValueValue:
		return value.GetValueValue().String()
	default:
		return nil
	}
}
