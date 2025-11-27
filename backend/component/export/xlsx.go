package export

import (
	"encoding/base64"
	"fmt"
	"io"
	"strconv"

	"github.com/pkg/errors"
	"github.com/xuri/excelize/v2"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const (
	excelLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	sheet1Name   = "Sheet1"
	// ExcelMaxColumn is the maximum number of columns supported by Excel.
	ExcelMaxColumn = 18278
)

// XLSX exports query results as XLSX format.
// Note: XLSX still materializes in memory due to excelize library limitations.
func XLSX(result *v1pb.QueryResult) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	index, err := f.NewSheet("Sheet1")
	if err != nil {
		return nil, err
	}
	var columnPrefixes []string
	for i, columnName := range result.ColumnNames {
		columnPrefix, err := ExcelColumnName(i)
		if err != nil {
			return nil, err
		}
		columnPrefixes = append(columnPrefixes, columnPrefix)
		if err := f.SetCellValue(sheet1Name, fmt.Sprintf("%s1", columnPrefix), columnName); err != nil {
			return nil, err
		}
	}
	for i, row := range result.Rows {
		for j, value := range row.Values {
			columnName := fmt.Sprintf("%s%d", columnPrefixes[j], i+2)
			if err := f.SetCellValue("Sheet1", columnName, convertValueToStringInXLSX(value)); err != nil {
				return nil, err
			}
		}
	}
	f.SetActiveSheet(index)
	excelBytes, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return excelBytes.Bytes(), nil
}

// XLSXToWriter exports XLSX format to a writer.
// Note: XLSX still materializes in memory due to excelize library limitations.
func XLSXToWriter(w io.Writer, result *v1pb.QueryResult) error {
	content, err := XLSX(result)
	if err != nil {
		return err
	}
	_, err = w.Write(content)
	return err
}

// ExcelColumnName converts a column index to Excel column name (A, B, ..., Z, AA, AB, ..., ZZZ).
func ExcelColumnName(index int) (string, error) {
	if index >= ExcelMaxColumn {
		return "", errors.Errorf("index cannot be greater than %v (column ZZZ)", ExcelMaxColumn)
	}

	var s string
	for {
		remain := index % 26
		s = string(excelLetters[remain]) + s
		index = index/26 - 1
		if index < 0 {
			break
		}
	}
	return s, nil
}

func convertValueToStringInXLSX(value *v1pb.RowValue) string {
	if value == nil || value.Kind == nil {
		return ""
	}
	switch value.Kind.(type) {
	case *v1pb.RowValue_StringValue:
		return value.GetStringValue()
	case *v1pb.RowValue_Int32Value:
		return strconv.FormatInt(int64(value.GetInt32Value()), 10)
	case *v1pb.RowValue_Int64Value:
		return strconv.FormatInt(value.GetInt64Value(), 10)
	case *v1pb.RowValue_Uint32Value:
		return strconv.FormatUint(uint64(value.GetUint32Value()), 10)
	case *v1pb.RowValue_Uint64Value:
		return strconv.FormatUint(value.GetUint64Value(), 10)
	case *v1pb.RowValue_FloatValue:
		return strconv.FormatFloat(float64(value.GetFloatValue()), 'f', -1, 32)
	case *v1pb.RowValue_DoubleValue:
		return strconv.FormatFloat(value.GetDoubleValue(), 'f', -1, 64)
	case *v1pb.RowValue_BoolValue:
		return strconv.FormatBool(value.GetBoolValue())
	case *v1pb.RowValue_BytesValue:
		return base64.StdEncoding.EncodeToString(value.GetBytesValue())
	case *v1pb.RowValue_NullValue:
		return ""
	case *v1pb.RowValue_TimestampValue:
		return formatTimestamp(value.GetTimestampValue())
	case *v1pb.RowValue_TimestampTzValue:
		return formatTimestampTz(value.GetTimestampTzValue())
	case *v1pb.RowValue_ValueValue:
		return value.GetValueValue().String()
	default:
		return ""
	}
}
