package export

import (
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// SQL exports query results as SQL INSERT statements (legacy wrapper).
func SQL(engine storepb.Engine, statementPrefix string, result *v1pb.QueryResult) ([]byte, error) {
	return exportToBytes(result, func(w io.Writer, r *v1pb.QueryResult) error {
		return SQLToWriter(w, engine, statementPrefix, r)
	})
}

// SQLToWriter streams SQL INSERT statements directly to the writer.
func SQLToWriter(w io.Writer, engine storepb.Engine, statementPrefix string, result *v1pb.QueryResult) error {
	for i, row := range result.Rows {
		if _, err := w.Write([]byte(statementPrefix)); err != nil {
			return err
		}
		for j, value := range row.Values {
			if j != 0 {
				if _, err := w.Write([]byte{','}); err != nil {
					return err
				}
			}
			if _, err := w.Write(convertValueToBytesInSQL(engine, value)); err != nil {
				return err
			}
		}
		if i != len(result.Rows)-1 {
			if _, err := w.Write([]byte(");\n")); err != nil {
				return err
			}
		} else {
			if _, err := w.Write([]byte(");")); err != nil {
				return err
			}
		}
	}
	return nil
}

// SQLStatementPrefix generates the INSERT INTO statement prefix.
func SQLStatementPrefix(engine storepb.Engine, resourceList []base.SchemaResource, columnNames []string) (string, error) {
	var escapeQuote string
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_TIDB, storepb.Engine_OCEANBASE, storepb.Engine_SPANNER:
		escapeQuote = "`"
	case storepb.Engine_CLICKHOUSE, storepb.Engine_MSSQL, storepb.Engine_ORACLE, storepb.Engine_POSTGRES, storepb.Engine_REDSHIFT, storepb.Engine_SQLITE, storepb.Engine_SNOWFLAKE:
		escapeQuote = "\""
	default:
		return "", errors.Errorf("unsupported engine %v for exporting as SQL", engine)
	}

	s := "INSERT INTO "
	if len(resourceList) == 1 {
		resource := resourceList[0]
		if resource.Schema != "" {
			s = fmt.Sprintf("%s%s%s%s%s", s, escapeQuote, resource.Schema, escapeQuote, ".")
		}
		s = fmt.Sprintf("%s%s%s%s", s, escapeQuote, resource.Table, escapeQuote)
	} else {
		s = fmt.Sprintf("%s%s%s%s", s, escapeQuote, "<table_name>", escapeQuote)
	}
	var columnTokens []string
	for _, columnName := range columnNames {
		columnTokens = append(columnTokens, fmt.Sprintf("%s%s%s", escapeQuote, columnName, escapeQuote))
	}
	s = fmt.Sprintf("%s (%s) VALUES (", s, strings.Join(columnTokens, ","))
	return s, nil
}

func convertValueToBytesInSQL(engine storepb.Engine, value *v1pb.RowValue) []byte {
	if value == nil || value.Kind == nil {
		return []byte("")
	}
	switch value.Kind.(type) {
	case *v1pb.RowValue_StringValue:
		return escapeSQLString(engine, []byte(value.GetStringValue()))
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
		return escapeSQLBytes(engine, value.GetBytesValue())
	case *v1pb.RowValue_NullValue:
		return []byte("NULL")
	case *v1pb.RowValue_TimestampValue:
		return escapeSQLString(engine, []byte(formatTimestamp(value.GetTimestampValue())))
	case *v1pb.RowValue_TimestampTzValue:
		return escapeSQLString(engine, []byte(formatTimestampTz(value.GetTimestampTzValue())))
	case *v1pb.RowValue_ValueValue:
		return convertValueValueToBytes(value.GetValueValue())
	default:
		return []byte("")
	}
}

func escapeSQLString(engine storepb.Engine, v []byte) []byte {
	switch engine {
	case storepb.Engine_POSTGRES, storepb.Engine_REDSHIFT:
		escapedStr := pq.QuoteLiteral(string(v))
		return []byte(escapedStr)
	default:
		result := []byte("'")
		s := strconv.Quote(string(v))
		s = s[1 : len(s)-1]
		s = strings.ReplaceAll(s, `'`, `''`)
		result = append(result, []byte(s)...)
		result = append(result, '\'')
		return result
	}
}

func escapeSQLBytes(_ storepb.Engine, v []byte) []byte {
	// Use hex encoding for all engines (SQL standard)
	result := make([]byte, 2+hex.EncodedLen(len(v)))
	copy(result, "0x")
	hex.Encode(result[2:], v)
	return result
}
