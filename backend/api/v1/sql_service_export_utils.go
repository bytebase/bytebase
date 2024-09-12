package v1

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/xuri/excelize/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func exportCSV(result *v1pb.QueryResult) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := buf.WriteString(strings.Join(result.ColumnNames, ",")); err != nil {
		return nil, err
	}
	if err := buf.WriteByte('\n'); err != nil {
		return nil, err
	}
	for i, row := range result.Rows {
		for i, value := range row.Values {
			if i != 0 {
				if err := buf.WriteByte(','); err != nil {
					return nil, err
				}
			}
			if _, err := buf.Write(convertValueToBytesInCSV(value)); err != nil {
				return nil, err
			}
		}
		if i != len(result.Rows)-1 {
			if err := buf.WriteByte('\n'); err != nil {
				return nil, err
			}
		}
	}
	return buf.Bytes(), nil
}

func convertValueToBytesInCSV(value *v1pb.RowValue) []byte {
	if value == nil || value.Kind == nil {
		return []byte("")
	}
	switch value.Kind.(type) {
	case *v1pb.RowValue_StringValue:
		var result []byte
		result = append(result, '"')
		result = append(result, []byte(escapeCSVString(value.GetStringValue()))...)
		result = append(result, '"')
		return result
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
		var result []byte
		result = append(result, '"')
		result = append(result, []byte(escapeCSVString(string(value.GetBytesValue())))...)
		result = append(result, '"')
		return result
	case *v1pb.RowValue_NullValue:
		return []byte("")
	case *v1pb.RowValue_ValueValue:
		// This is used by ClickHouse and Spanner only.
		return convertValueValueToBytes(value.GetValueValue())
	default:
		return []byte("")
	}
}

func escapeCSVString(str string) string {
	escapedStr := strings.ReplaceAll(str, `"`, `""`)
	return escapedStr
}

func getSQLStatementPrefix(engine storepb.Engine, resourceList []base.SchemaResource, columnNames []string) (string, error) {
	var escapeQuote string
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_TIDB, storepb.Engine_OCEANBASE, storepb.Engine_SPANNER:
		escapeQuote = "`"
	case storepb.Engine_CLICKHOUSE, storepb.Engine_MSSQL, storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE, storepb.Engine_DM, storepb.Engine_POSTGRES, storepb.Engine_REDSHIFT, storepb.Engine_SQLITE, storepb.Engine_SNOWFLAKE:
		// ClickHouse takes both double-quotes or backticks.
		escapeQuote = "\""
	default:
		// storepb.Engine_MONGODB, storepb.Engine_REDIS
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

func exportSQL(engine storepb.Engine, statementPrefix string, result *v1pb.QueryResult) ([]byte, error) {
	var buf bytes.Buffer
	for i, row := range result.Rows {
		if _, err := buf.WriteString(statementPrefix); err != nil {
			return nil, err
		}
		for i, value := range row.Values {
			if i != 0 {
				if err := buf.WriteByte(','); err != nil {
					return nil, err
				}
			}
			if _, err := buf.Write(convertValueToBytesInSQL(engine, value)); err != nil {
				return nil, err
			}
		}
		if i != len(result.Rows)-1 {
			if _, err := buf.WriteString(");\n"); err != nil {
				return nil, err
			}
		} else {
			if _, err := buf.WriteString(");"); err != nil {
				return nil, err
			}
		}
	}
	return buf.Bytes(), nil
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
	case *v1pb.RowValue_ValueValue:
		// This is used by ClickHouse and Spanner only.
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

func escapeSQLBytes(engine storepb.Engine, v []byte) []byte {
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB:
		result := []byte("B'")
		s := fmt.Sprintf("%b", v)
		s = s[1 : len(s)-1]
		result = append(result, []byte(s)...)
		result = append(result, '\'')
		return result
	default:
		return escapeSQLString(engine, v)
	}
}

func convertValueValueToBytes(value *structpb.Value) []byte {
	if value == nil || value.Kind == nil {
		return []byte("")
	}
	switch value.Kind.(type) {
	case *structpb.Value_NullValue:
		return []byte("")
	case *structpb.Value_StringValue:
		var result []byte
		result = append(result, '"')
		result = append(result, []byte(value.GetStringValue())...)
		result = append(result, '"')
		return result
	case *structpb.Value_NumberValue:
		return []byte(strconv.FormatFloat(value.GetNumberValue(), 'f', -1, 64))
	case *structpb.Value_BoolValue:
		return []byte(strconv.FormatBool(value.GetBoolValue()))
	case *structpb.Value_ListValue:
		var buf [][]byte
		for _, v := range value.GetListValue().Values {
			buf = append(buf, convertValueValueToBytes(v))
		}
		var result []byte
		result = append(result, '"')
		result = append(result, '[')
		result = append(result, bytes.Join(buf, []byte(","))...)
		result = append(result, ']')
		result = append(result, '"')
		return result
	case *structpb.Value_StructValue:
		first := true
		var buf []byte
		buf = append(buf, '"')
		for k, v := range value.GetStructValue().Fields {
			if first {
				first = false
			} else {
				buf = append(buf, ',')
			}
			buf = append(buf, []byte(k)...)
			buf = append(buf, ':')
			buf = append(buf, convertValueValueToBytes(v)...)
		}
		buf = append(buf, '"')
		return buf
	default:
		return []byte("")
	}
}

// extractResourceList extracts the resource list from the statement for exporting results as SQL.
func extractResourceList(ctx context.Context, storeInstance *store.Store, engine storepb.Engine, databaseName string, statement string, instance *store.InstanceMessage) ([]base.SchemaResource, error) {
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		list, err := base.ExtractResourceList(engine, databaseName, "", statement)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to extract resource list: %s", err.Error())
		} else if databaseName == "" {
			return list, nil
		}

		databaseMessage, err := storeInstance.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instance.ResourceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			if httpErr, ok := err.(*echo.HTTPError); ok && httpErr.Code == echo.ErrNotFound.Code {
				// If database not found, skip.
				return nil, nil
			}
			return nil, status.Errorf(codes.Internal, "failed to fetch database: %v", err)
		}
		if databaseMessage == nil {
			return nil, nil
		}

		dbSchema, err := storeInstance.GetDBSchema(ctx, databaseMessage.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
		}

		var result []base.SchemaResource
		for _, resource := range list {
			if resource.Database != dbSchema.GetMetadata().Name {
				// MySQL allows cross-database query, we should check the corresponding database.
				resourceDB, err := storeInstance.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
					InstanceID:          &instance.ResourceID,
					DatabaseName:        &resource.Database,
					IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if resourceDB == nil {
					continue
				}
				resourceDBSchema, err := storeInstance.GetDBSchema(ctx, resourceDB.UID)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database schema %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if !resourceDBSchema.TableExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) &&
					!resourceDBSchema.ViewExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) {
					// If table not found, we regard it as a CTE/alias/... and skip.
					continue
				}
				result = append(result, resource)
				continue
			}
			if !dbSchema.TableExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) &&
				!dbSchema.ViewExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) {
				// If table not found, skip.
				continue
			}
			result = append(result, resource)
		}
		return result, nil
	case storepb.Engine_POSTGRES, storepb.Engine_REDSHIFT:
		list, err := base.ExtractResourceList(engine, databaseName, "public", statement)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to extract resource list: %s", err.Error())
		}

		databaseMessage, err := storeInstance.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instance.ResourceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database: %v", err)
		}
		if databaseMessage == nil {
			return nil, nil
		}

		dbSchema, err := storeInstance.GetDBSchema(ctx, databaseMessage.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
		}

		var result []base.SchemaResource
		for _, resource := range list {
			if resource.Database != dbSchema.GetMetadata().Name {
				// Should not happen.
				continue
			}

			if !dbSchema.TableExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) &&
				!dbSchema.ViewExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) {
				// If table not found, skip.
				continue
			}

			result = append(result, resource)
		}

		return result, nil
	case storepb.Engine_ORACLE, storepb.Engine_DM, storepb.Engine_OCEANBASE_ORACLE:
		list, err := base.ExtractResourceList(engine, databaseName, databaseName, statement)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to extract resource list: %s", err.Error())
		}

		databaseMessage, err := storeInstance.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instance.ResourceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database: %v", err)
		}
		if databaseMessage == nil {
			return nil, nil
		}

		dbSchema, err := storeInstance.GetDBSchema(ctx, databaseMessage.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
		}

		var result []base.SchemaResource
		for _, resource := range list {
			if resource.Database != dbSchema.GetMetadata().Name {
				resourceDB, err := storeInstance.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
					InstanceID:          &instance.ResourceID,
					DatabaseName:        &resource.Database,
					IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if resourceDB == nil {
					continue
				}
				resourceDBSchema, err := storeInstance.GetDBSchema(ctx, resourceDB.UID)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database schema %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if !resourceDBSchema.TableExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) &&
					!resourceDBSchema.ViewExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) {
					// If table not found, we regard it as a CTE/alias/... and skip.
					continue
				}
				result = append(result, resource)
				continue
			}

			if !dbSchema.TableExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) &&
				!dbSchema.ViewExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) {
				// If table not found, skip.
				continue
			}

			result = append(result, resource)
		}

		return result, nil
	case storepb.Engine_SNOWFLAKE:
		dataSource := utils.DataSourceFromInstanceWithType(instance, api.RO)
		adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
		// If there are no read-only data source, fall back to admin data source.
		if dataSource == nil {
			dataSource = adminDataSource
		}
		if dataSource == nil {
			return nil, status.Errorf(codes.Internal, "failed to find data source for instance: %s", instance.ResourceID)
		}
		list, err := base.ExtractResourceList(engine, databaseName, "PUBLIC", statement)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to extract resource list: %s", err.Error())
		}
		databaseMessage, err := storeInstance.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instance.ResourceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			if httpErr, ok := err.(*echo.HTTPError); ok && httpErr.Code == echo.ErrNotFound.Code {
				// If database not found, skip.
				return nil, nil
			}
			return nil, status.Errorf(codes.Internal, "failed to fetch database: %v", err)
		}
		if databaseMessage == nil {
			return nil, nil
		}

		dbSchema, err := storeInstance.GetDBSchema(ctx, databaseMessage.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
		}

		var result []base.SchemaResource
		for _, resource := range list {
			if resource.Database != dbSchema.GetMetadata().Name {
				// Snowflake allows cross-database query, we should check the corresponding database.
				resourceDB, err := storeInstance.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
					InstanceID:          &instance.ResourceID,
					DatabaseName:        &resource.Database,
					IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if resourceDB == nil {
					continue
				}
				resourceDBSchema, err := storeInstance.GetDBSchema(ctx, resourceDB.UID)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database schema %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if !resourceDBSchema.TableExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) &&
					!resourceDBSchema.ViewExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) {
					// If table not found, we regard it as a CTE/alias/... and skip.
					continue
				}
				result = append(result, resource)
				continue
			}
			if !dbSchema.TableExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) &&
				!dbSchema.ViewExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) {
				// If table not found, skip.
				continue
			}
			result = append(result, resource)
		}
		return result, nil
	case storepb.Engine_MSSQL:
		dataSource := utils.DataSourceFromInstanceWithType(instance, api.RO)
		adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
		// If there are no read-only data source, fall back to admin data source.
		if dataSource == nil {
			dataSource = adminDataSource
		}
		if dataSource == nil {
			return nil, status.Errorf(codes.Internal, "failed to find data source for instance: %s", instance.ResourceID)
		}
		list, err := base.ExtractResourceList(engine, databaseName, "dbo", statement)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to extract resource list: %s", err.Error())
		}
		databaseMessage, err := storeInstance.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instance.ResourceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			if httpErr, ok := err.(*echo.HTTPError); ok && httpErr.Code == echo.ErrNotFound.Code {
				// If database not found, skip.
				return nil, nil
			}
			return nil, status.Errorf(codes.Internal, "failed to fetch database: %v", err)
		}
		if databaseMessage == nil {
			return nil, nil
		}

		dbSchema, err := storeInstance.GetDBSchema(ctx, databaseMessage.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
		}

		var result []base.SchemaResource
		for _, resource := range list {
			if resource.LinkedServer != "" {
				continue
			}
			if resource.Database != dbSchema.GetMetadata().Name {
				// MSSQL allows cross-database query, we should check the corresponding database.
				resourceDB, err := storeInstance.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
					InstanceID:          &instance.ResourceID,
					DatabaseName:        &resource.Database,
					IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if resourceDB == nil {
					continue
				}
				resourceDBSchema, err := storeInstance.GetDBSchema(ctx, resourceDB.UID)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database schema %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if !resourceDBSchema.TableExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) &&
					!resourceDBSchema.ViewExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) {
					// If table not found, we regard it as a CTE/alias/... and skip.
					continue
				}
				result = append(result, resource)
				continue
			}
			if !dbSchema.TableExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) &&
				!dbSchema.ViewExists(resource.Schema, resource.Table, store.IgnoreDatabaseAndTableCaseSensitive(instance)) {
				// If table not found, skip.
				continue
			}
			result = append(result, resource)
		}
		return result, nil
	default:
		return base.ExtractResourceList(engine, databaseName, "", statement)
	}
}

func exportJSON(result *v1pb.QueryResult) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := buf.WriteString("["); err != nil {
		return nil, err
	}

	for rowIndex, row := range result.Rows {
		if _, err := buf.WriteString("{"); err != nil {
			return nil, err
		}
		for i, value := range row.Values {
			if _, err := buf.WriteString(fmt.Sprintf(`"%s":`, result.ColumnNames[i])); err != nil {
				return nil, err
			}
			if _, err := buf.WriteString(convertValueToStringInJSON(value)); err != nil {
				return nil, err
			}
			if i != len(row.Values)-1 {
				if _, err := buf.WriteString(","); err != nil {
					return nil, err
				}
			}
		}
		if _, err := buf.WriteString("}"); err != nil {
			return nil, err
		}

		if rowIndex != len(result.Rows)-1 {
			if _, err := buf.WriteString(","); err != nil {
				return nil, err
			}
		}
	}

	if _, err := buf.WriteString("]"); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func convertValueToStringInJSON(value *v1pb.RowValue) string {
	if value == nil || value.Kind == nil {
		return ""
	}

	switch value.Kind.(type) {
	case *v1pb.RowValue_StringValue:
		return `"` + escapeJSONString(value.GetStringValue()) + `"`
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
		value, err := convertBytesToBinaryString(value.GetBytesValue())
		if err != nil {
			return ""
		}
		return value
	case *v1pb.RowValue_NullValue:
		return "null"
	case *v1pb.RowValue_ValueValue:
		// This is used by ClickHouse and Spanner only.
		return value.GetValueValue().String()
	default:
		return ""
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

func escapeJSONString(str string) string {
	s := strconv.Quote(str)
	return s[1 : len(s)-1]
}

const (
	excelLetters   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	sheet1Name     = "Sheet1"
	excelMaxColumn = 18278
)

func exportXLSX(result *v1pb.QueryResult) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	index, err := f.NewSheet("Sheet1")
	if err != nil {
		return nil, err
	}
	var columnPrefixes []string
	for i, columnName := range result.ColumnNames {
		columnPrefix, err := getExcelColumnName(i)
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

func getExcelColumnName(index int) (string, error) {
	if index >= excelMaxColumn {
		return "", errors.Errorf("index cannot be greater than %v (column ZZZ)", excelMaxColumn)
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
	case *v1pb.RowValue_ValueValue:
		// This is used by ClickHouse and Spanner only.
		return value.GetValueValue().String()
	default:
		return ""
	}
}
