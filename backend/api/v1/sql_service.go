package v1

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/google/cel-go/cel"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/xuri/excelize/v2"

	tidbast "github.com/pingcap/tidb/parser/ast"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/backend/plugin/advisor/db"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const (
	// The maximum number of bytes for sql results in response body.
	// 10 MB.
	maximumSQLResultSize = 10 * 1024 * 1024
	// defaultTimeout is the default timeout for query and admin execution.
	defaultTimeout = 1 * time.Minute
)

// SQLService is the service for SQL.
type SQLService struct {
	v1pb.UnimplementedSQLServiceServer
	store           *store.Store
	schemaSyncer    *schemasync.Syncer
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	licenseService  enterpriseAPI.LicenseService
}

// NewSQLService creates a SQLService.
func NewSQLService(
	store *store.Store,
	schemaSyncer *schemasync.Syncer,
	dbFactory *dbfactory.DBFactory,
	activityManager *activity.Manager,
	licenseService enterpriseAPI.LicenseService,
) *SQLService {
	return &SQLService{
		store:           store,
		schemaSyncer:    schemaSyncer,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		licenseService:  licenseService,
	}
}

// Pretty returns pretty format SDL.
func (*SQLService) Pretty(_ context.Context, request *v1pb.PrettyRequest) (*v1pb.PrettyResponse, error) {
	engine := parser.EngineType(convertEngine(request.Engine))
	if _, err := transform.CheckFormat(engine, request.ExpectedSchema); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "User SDL is not SDL format: %s", err.Error())
	}
	if _, err := transform.CheckFormat(engine, request.CurrentSchema); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Dumped SDL is not SDL format: %s", err.Error())
	}

	prettyExpectedSchema, err := transform.SchemaTransform(engine, request.ExpectedSchema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to transform user SDL: %s", err.Error())
	}
	prettyCurrentSchema, err := transform.Normalize(engine, request.CurrentSchema, prettyExpectedSchema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to normalize dumped SDL: %s", err.Error())
	}
	return &v1pb.PrettyResponse{
		CurrentSchema:  prettyCurrentSchema,
		ExpectedSchema: prettyExpectedSchema,
	}, nil
}

// AdminExecute executes the SQL statement.
func (s *SQLService) AdminExecute(server v1pb.SQLService_AdminExecuteServer) error {
	ctx := server.Context()
	var driver db.Driver
	var conn *sql.Conn
	defer func() {
		if conn != nil {
			if err := conn.Close(); err != nil {
				log.Warn("failed to close connection", zap.Error(err))
			}
		}
		if driver != nil {
			driver.Close(ctx)
		}
	}()
	for {
		request, err := server.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return status.Errorf(codes.Internal, "failed to receive request: %v", err)
		}

		instance, database, activity, err := s.preAdminExecute(ctx, request)
		if err != nil {
			return err
		}

		// We only need to get the driver and connection once.
		if driver == nil {
			driver, err = s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to get database driver: %v", err)
			}

			sqlDB := driver.GetDB()
			if sqlDB != nil {
				conn, err = sqlDB.Conn(ctx)
				if err != nil {
					return status.Errorf(codes.Internal, "failed to get database connection: %v", err)
				}
			}
		}

		result, durationNs, queryErr := s.doAdminExecute(ctx, driver, conn, request)

		if err := s.postAdminExecute(ctx, activity, durationNs, queryErr); err != nil {
			log.Error("failed to post admin execute activity", zap.Error(err))
		}

		response := &v1pb.AdminExecuteResponse{}
		if queryErr != nil {
			response.Results = []*v1pb.QueryResult{
				{
					Error: queryErr.Error(),
				},
			}
		} else {
			response.Results = result
		}

		if proto.Size(response) > maximumSQLResultSize {
			response.Results = []*v1pb.QueryResult{
				{
					Error: fmt.Sprintf("Output of query exceeds max allowed output size of %dMB", maximumSQLResultSize/1024/1024),
				},
			}
		}

		if err := server.Send(response); err != nil {
			return status.Errorf(codes.Internal, "failed to send response: %v", err)
		}
	}
}

func (s *SQLService) postAdminExecute(ctx context.Context, activity *store.ActivityMessage, durationNs int64, queryErr error) error {
	var payload api.ActivitySQLEditorQueryPayload
	if err := json.Unmarshal([]byte(activity.Payload), &payload); err != nil {
		return status.Errorf(codes.Internal, "failed to unmarshal activity payload: %v", err)
	}

	var newLevel *api.ActivityLevel
	payload.DurationNs = durationNs
	if queryErr != nil {
		payload.Error = queryErr.Error()
		errorLevel := api.ActivityError
		newLevel = &errorLevel
	}

	// TODO: update the advice list
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Warn("Failed to marshal activity after executing sql statement",
			zap.String("database_name", payload.DatabaseName),
			zap.Int("instance_id", payload.InstanceID),
			zap.String("statement", payload.Statement),
			zap.Error(err))
		return status.Errorf(codes.Internal, "Failed to marshal activity after executing sql statement: %v", err)
	}

	payloadString := string(payloadBytes)
	if _, err := s.store.UpdateActivityV2(ctx, &store.UpdateActivityMessage{
		UID:        activity.UID,
		UpdaterUID: activity.CreatorUID,
		Level:      newLevel,
		Payload:    &payloadString,
	}); err != nil {
		return status.Errorf(codes.Internal, "Failed to update activity after executing sql statement: %v", err)
	}

	return nil
}

func (*SQLService) doAdminExecute(ctx context.Context, driver db.Driver, conn *sql.Conn, request *v1pb.AdminExecuteRequest) ([]*v1pb.QueryResult, int64, error) {
	start := time.Now().UnixNano()
	timeout := defaultTimeout
	if request.Timeout != nil {
		timeout = request.Timeout.AsDuration()
	}
	ctx, cancelCtx := context.WithTimeout(ctx, timeout)
	defer cancelCtx()
	result, err := driver.RunStatement(ctx, conn, request.Statement)
	select {
	case <-ctx.Done():
		// canceled or timed out
		return nil, time.Now().UnixNano() - start, errors.Errorf("timeout reached: %v", timeout)
	default:
		// So the select will not block
	}
	return result, time.Now().UnixNano() - start, err
}

func (s *SQLService) preAdminExecute(ctx context.Context, request *v1pb.AdminExecuteRequest) (*store.InstanceMessage, *store.DatabaseMessage, *store.ActivityMessage, error) {
	user, _, instance, database, err := s.prepareRelatedMessage(ctx, request.Name, request.ConnectionDatabase)
	if err != nil {
		return nil, nil, nil, err
	}

	activity, err := s.createQueryActivity(ctx, user, api.ActivityInfo, instance.UID, api.ActivitySQLEditorQueryPayload{
		Statement:              request.Statement,
		InstanceID:             instance.UID,
		DeprecatedInstanceName: instance.Title,
		DatabaseID:             database.UID,
		DatabaseName:           request.ConnectionDatabase,
	})
	if err != nil {
		return nil, nil, nil, err
	}

	return instance, database, activity, nil
}

// Export exports the SQL query result.
func (s *SQLService) Export(ctx context.Context, request *v1pb.ExportRequest) (*v1pb.ExportResponse, error) {
	instance, database, sensitiveSchemaInfo, activity, err := s.preExport(ctx, request)
	if err != nil {
		return nil, err
	}

	bytes, durationNs, exportErr := s.doExport(ctx, request, instance, database, sensitiveSchemaInfo)

	if err := s.postExport(ctx, activity, durationNs, exportErr); err != nil {
		return nil, err
	}

	if exportErr != nil {
		return nil, exportErr
	}

	return &v1pb.ExportResponse{
		Content: bytes,
	}, nil
}

func (s *SQLService) postExport(ctx context.Context, activity *store.ActivityMessage, durationNs int64, queryErr error) error {
	// Update the activity
	var payload api.ActivitySQLExportPayload
	if err := json.Unmarshal([]byte(activity.Payload), &payload); err != nil {
		return status.Errorf(codes.Internal, "failed to unmarshal activity payload: %v", err)
	}

	var newLevel *api.ActivityLevel
	payload.DurationNs = durationNs
	if queryErr != nil {
		payload.Error = queryErr.Error()
		errorLevel := api.ActivityError
		newLevel = &errorLevel
	}

	// TODO: update the advice list

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Warn("Failed to marshal activity after exporting sql statement",
			zap.String("database_name", payload.DatabaseName),
			zap.Int("instance_id", payload.InstanceID),
			zap.String("statement", payload.Statement),
			zap.Error(err))
		return status.Errorf(codes.Internal, "Failed to marshal activity after exporting sql statement: %v", err)
	}

	payloadString := string(payloadBytes)
	if _, err := s.store.UpdateActivityV2(ctx, &store.UpdateActivityMessage{
		UID:        activity.UID,
		UpdaterUID: activity.CreatorUID,
		Level:      newLevel,
		Payload:    &payloadString,
	}); err != nil {
		return status.Errorf(codes.Internal, "Failed to update activity after exporting sql statement: %v", err)
	}

	return nil
}

func (s *SQLService) doExport(ctx context.Context, request *v1pb.ExportRequest, instance *store.InstanceMessage, database *store.DatabaseMessage, sensitiveSchemaInfo *db.SensitiveSchemaInfo) ([]byte, int64, error) {
	// Don't anonymize data for exporting data using admin mode.
	if request.Admin {
		sensitiveSchemaInfo = nil
	}

	driver, err := s.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, database)
	if err != nil {
		return nil, 0, err
	}
	defer driver.Close(ctx)

	sqlDB := driver.GetDB()
	var conn *sql.Conn
	if sqlDB != nil {
		conn, err = sqlDB.Conn(ctx)
		if err != nil {
			return nil, 0, err
		}
		defer conn.Close()
	}

	start := time.Now().UnixNano()
	result, err := driver.QueryConn2(ctx, conn, request.Statement, &db.QueryContext{
		Limit:           int(request.Limit),
		ReadOnly:        true,
		CurrentDatabase: request.ConnectionDatabase,
		// TODO(rebelice): we cannot deal with multi-SensitiveDataMaskType now. Fix it.
		SensitiveDataMaskType: db.SensitiveDataMaskTypeDefault,
		SensitiveSchemaInfo:   sensitiveSchemaInfo,
		EnableSensitive:       s.licenseService.IsFeatureEnabledForInstance(api.FeatureSensitiveData, instance) == nil,
	})
	durationNs := time.Now().UnixNano() - start
	if err != nil {
		return nil, durationNs, err
	}
	if len(result) != 1 {
		return nil, durationNs, errors.Errorf("expecting 1 result, but got %d", len(result))
	}

	var content []byte
	switch request.Format {
	case v1pb.ExportRequest_CSV:
		if content, err = s.exportCSV(result[0]); err != nil {
			return nil, durationNs, err
		}
	case v1pb.ExportRequest_JSON:
		if content, err = s.exportJSON(result[0]); err != nil {
			return nil, durationNs, err
		}
	case v1pb.ExportRequest_SQL:
		resourceList, err := s.extractResourceList(ctx, convertToParserEngine(instance.Engine), request.ConnectionDatabase, request.Statement, instance)
		if err != nil {
			return nil, 0, status.Errorf(codes.InvalidArgument, "failed to extract resource list: %v", err)
		}
		statementPrefix, err := getSQLStatementPrefix(instance.Engine, resourceList, result[0].ColumnNames)
		if err != nil {
			return nil, 0, err
		}
		if content, err = exportSQL(instance.Engine, statementPrefix, result[0]); err != nil {
			return nil, durationNs, err
		}
	case v1pb.ExportRequest_XLSX:
		if content, err = s.exportXLSX(result[0]); err != nil {
			return nil, durationNs, err
		}
	default:
		return nil, durationNs, status.Errorf(codes.InvalidArgument, "unsupported export format: %s", request.Format.String())
	}
	return content, durationNs, nil
}

func (*SQLService) exportCSV(result *v1pb.QueryResult) ([]byte, error) {
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

func getSQLStatementPrefix(engine db.Type, resourceList []parser.SchemaResource, columnNames []string) (string, error) {
	var escapeQuote string
	switch engine {
	case db.MySQL, db.MariaDB, db.TiDB, db.OceanBase, db.Spanner:
		escapeQuote = "`"
	case db.ClickHouse, db.MSSQL, db.Oracle, db.DM, db.Postgres, db.Redshift, db.SQLite, db.Snowflake:
		// ClickHouse takes both double-quotes or backticks.
		escapeQuote = "\""
	default:
		// db.MongoDB, db.Redis
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

func exportSQL(engine db.Type, statementPrefix string, result *v1pb.QueryResult) ([]byte, error) {
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

func convertValueToBytesInSQL(engine db.Type, value *v1pb.RowValue) []byte {
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
		return escapeSQLString(engine, value.GetBytesValue())
	case *v1pb.RowValue_NullValue:
		return []byte("NULL")
	case *v1pb.RowValue_ValueValue:
		// This is used by ClickHouse and Spanner only.
		return convertValueValueToBytes(value.GetValueValue())
	default:
		return []byte("")
	}
}

func escapeSQLString(engine db.Type, v []byte) []byte {
	switch engine {
	case db.Postgres, db.Redshift:
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

func convertValueValueToBytes(value *structpb.Value) []byte {
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

func (*SQLService) exportJSON(result *v1pb.QueryResult) ([]byte, error) {
	var results []map[string]any
	for _, row := range result.Rows {
		m := make(map[string]any)
		for i, value := range row.Values {
			m[result.ColumnNames[i]] = convertValueToStringInJSON(value)
		}
		results = append(results, m)
	}
	return json.MarshalIndent(results, "", "  ")
}

func convertValueToStringInJSON(value *v1pb.RowValue) string {
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
		return "null"
	case *v1pb.RowValue_ValueValue:
		// This is used by ClickHouse and Spanner only.
		return value.GetValueValue().String()
	default:
		return ""
	}
}

const (
	excelLetters   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	sheet1Name     = "Sheet1"
	excelMaxColumn = 18278
)

func (*SQLService) exportXLSX(result *v1pb.QueryResult) ([]byte, error) {
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

func (s *SQLService) preExport(ctx context.Context, request *v1pb.ExportRequest) (*store.InstanceMessage, *store.DatabaseMessage, *db.SensitiveSchemaInfo, *store.ActivityMessage, error) {
	// Prepare related message.
	user, environment, instance, database, err := s.prepareRelatedMessage(ctx, request.Name, request.ConnectionDatabase)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Validate the request.
	if err := s.validateQueryRequest(instance, request.ConnectionDatabase, request.Statement); err != nil {
		return nil, nil, nil, nil, err
	}

	if s.licenseService.IsFeatureEnabled(api.FeatureAccessControl) == nil {
		// Check if the caller is admin for exporting with admin mode.
		if request.Admin && (user.Role != api.Owner && user.Role != api.DBA) {
			return nil, nil, nil, nil, status.Errorf(codes.PermissionDenied, "only workspace owner and DBA can export data using admin mode")
		}

		// Check if the environment is open for export privileges.
		result, err := s.checkWorkspaceIAMPolicy(ctx, common.ProjectExporter, environment)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if !result {
			// Check if the user has permission to execute the export.
			if err := s.checkQueryRights(ctx, request.ConnectionDatabase, database.DataShare, request.Statement, request.Limit, user, instance, request.Format); err != nil {
				return nil, nil, nil, nil, err
			}
		}
	}

	// Get sensitive schema info.
	var sensitiveSchemaInfo *db.SensitiveSchemaInfo
	switch instance.Engine {
	case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
		databaseList, err := parser.ExtractDatabaseList(parser.MySQL, request.Statement, "")
		if err != nil {
			return nil, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get database list: %s with error %v", request.Statement, err)
		}

		sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, databaseList, request.ConnectionDatabase)
		if err != nil {
			return nil, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get sensitive schema info: %s", request.Statement)
		}
	case db.Postgres:
		sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, []string{request.ConnectionDatabase}, request.ConnectionDatabase)
		if err != nil {
			return nil, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get sensitive schema info: %s", request.Statement)
		}
	case db.Oracle, db.DM:
		if instance.Options == nil || !instance.Options.SchemaTenantMode {
			sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, []string{request.ConnectionDatabase}, request.ConnectionDatabase)
			if err != nil {
				return nil, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get sensitive schema info for statement: %s, error: %v", request.Statement, err.Error())
			}
		} else {
			list, err := parser.ExtractResourceList(parser.Oracle, request.ConnectionDatabase, request.ConnectionDatabase, request.Statement)
			if err != nil {
				return nil, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get resource list: %s", request.Statement)
			}
			databaseMap := make(map[string]bool)
			for _, resource := range list {
				databaseMap[resource.Database] = true
			}
			var databaseList []string
			databaseList = append(databaseList, request.ConnectionDatabase)
			for database := range databaseMap {
				databaseList = append(databaseList, database)
			}
			sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, databaseList, request.ConnectionDatabase)
			if err != nil {
				return nil, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get sensitive schema info for statement: %s, error: %v", request.Statement, err.Error())
			}
		}
	case db.Snowflake:
		databaseList, err := parser.ExtractDatabaseList(parser.Snowflake, request.Statement, request.ConnectionDatabase)
		if err != nil {
			return nil, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get database list: %s with error %v", request.Statement, err)
		}

		sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, databaseList, request.ConnectionDatabase)
		if err != nil {
			return nil, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get sensitive schema info: %s", request.Statement)
		}
	}

	// Create export activity.
	level := api.ActivityInfo
	activity, err := s.createExportActivity(ctx, user, level, instance.UID, api.ActivitySQLExportPayload{
		Statement:    request.Statement,
		InstanceID:   instance.UID,
		DatabaseID:   database.UID,
		DatabaseName: request.ConnectionDatabase,
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return instance, database, sensitiveSchemaInfo, activity, nil
}

func (s *SQLService) createExportActivity(ctx context.Context, user *store.UserMessage, level api.ActivityLevel, containerID int, payload api.ActivitySQLExportPayload) (*store.ActivityMessage, error) {
	// TODO: use v1 activity API instead of
	activityBytes, err := json.Marshal(payload)
	if err != nil {
		log.Warn("Failed to marshal activity before exporting sql statement",
			zap.String("database_name", payload.DatabaseName),
			zap.Int("instance_id", payload.InstanceID),
			zap.String("statement", payload.Statement),
			zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Failed to construct activity payload: %v", err)
	}

	activityCreate := &store.ActivityMessage{
		CreatorUID:   user.ID,
		Type:         api.ActivitySQLExport,
		ContainerUID: containerID,
		Level:        level,
		Comment: fmt.Sprintf("Export `%q` in database %q of instance %d.",
			payload.Statement, payload.DatabaseName, payload.InstanceID),
		Payload: string(activityBytes),
	}

	activity, err := s.store.CreateActivityV2(ctx, activityCreate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create activity: %v", err)
	}
	return activity, nil
}

// Query executes a SQL query.
// We have the following stages:
//  1. pre-query
//  2. do query
//  3. post-query
func (s *SQLService) Query(ctx context.Context, request *v1pb.QueryRequest) (*v1pb.QueryResponse, error) {
	instance, database, adviceStatus, adviceList, sensitiveSchemaInfo, activity, err := s.preQuery(ctx, request)
	if err != nil {
		return nil, err
	}

	var results []*v1pb.QueryResult
	var queryErr error
	var durationNs int64
	if adviceStatus != advisor.Error {
		results, durationNs, queryErr = s.doQuery(ctx, request, instance, database, sensitiveSchemaInfo)
	}

	adviceList, err = s.postQuery(ctx, request, adviceStatus, adviceList, instance, activity, durationNs, queryErr)
	if err != nil {
		return nil, err
	}

	if queryErr != nil {
		return nil, queryErr
	}

	response := &v1pb.QueryResponse{
		Results: results,
	}

	if proto.Size(response) > maximumSQLResultSize {
		response.Results = []*v1pb.QueryResult{
			{
				Error: fmt.Sprintf("Output of query exceeds max allowed output size of %dMB", maximumSQLResultSize/1024/1024),
			},
		}
	}

	response.Advices = adviceList

	return response, nil
}

// postQuery does the following:
//  1. Check index hit Explain statements
//  2. Update SQL query activity
func (s *SQLService) postQuery(ctx context.Context, _ *v1pb.QueryRequest, adviceStatus advisor.Status, adviceList []*v1pb.Advice, _ *store.InstanceMessage, activity *store.ActivityMessage, durationNs int64, queryErr error) ([]*v1pb.Advice, error) {
	indexHitAdvices, err := s.checkIndexHit()
	if err != nil {
		return nil, err
	}

	var finalAdviceList []*v1pb.Advice
	newLevel := activity.Level
	if len(indexHitAdvices) == 0 {
		finalAdviceList = append(finalAdviceList, adviceList...)
	} else {
		if adviceStatus != advisor.Success {
			finalAdviceList = append(finalAdviceList, adviceList...)
		}
		finalAdviceList = append(finalAdviceList, indexHitAdvices...)
		newLevel = api.ActivityError
	}

	// Update the activity
	var payload api.ActivitySQLEditorQueryPayload
	if err := json.Unmarshal([]byte(activity.Payload), &payload); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal activity payload: %v", err)
	}

	payload.DurationNs = durationNs
	if queryErr != nil {
		payload.Error = queryErr.Error()
		newLevel = api.ActivityError
	}

	// TODO: update the advice list

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Warn("Failed to marshal activity after executing sql statement",
			zap.String("database_name", payload.DatabaseName),
			zap.Int("instance_id", payload.InstanceID),
			zap.String("statement", payload.Statement),
			zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Failed to marshal activity after executing sql statement: %v", err)
	}

	payloadString := string(payloadBytes)
	if _, err := s.store.UpdateActivityV2(ctx, &store.UpdateActivityMessage{
		UID:        activity.UID,
		UpdaterUID: activity.CreatorUID,
		Level:      &newLevel,
		Payload:    &payloadString,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to update activity after executing sql statement: %v", err)
	}

	return finalAdviceList, nil
}

func (*SQLService) checkIndexHit() ([]*v1pb.Advice, error) {
	// TODO(rebelice): implement checkIndexHit
	return nil, nil
}

func (s *SQLService) doQuery(ctx context.Context, request *v1pb.QueryRequest, instance *store.InstanceMessage, database *store.DatabaseMessage, sensitiveSchemaInfo *db.SensitiveSchemaInfo) ([]*v1pb.QueryResult, int64, error) {
	driver, err := s.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, database)
	if err != nil {
		return nil, 0, err
	}
	defer driver.Close(ctx)

	sqlDB := driver.GetDB()
	var conn *sql.Conn
	if sqlDB != nil {
		conn, err = sqlDB.Conn(ctx)
		if err != nil {
			return nil, 0, err
		}
		defer conn.Close()
	}

	timeout := defaultTimeout
	if request.Timeout != nil {
		timeout = request.Timeout.AsDuration()
	}
	ctx, cancelCtx := context.WithTimeout(ctx, timeout)
	defer cancelCtx()

	start := time.Now().UnixNano()
	result, err := driver.QueryConn2(ctx, conn, request.Statement, &db.QueryContext{
		Limit:           int(request.Limit),
		ReadOnly:        true,
		CurrentDatabase: request.ConnectionDatabase,
		// TODO(rebelice): we cannot deal with multi-SensitiveDataMaskType now. Fix it.
		SensitiveDataMaskType: db.SensitiveDataMaskTypeDefault,
		SensitiveSchemaInfo:   sensitiveSchemaInfo,
		EnableSensitive:       s.licenseService.IsFeatureEnabledForInstance(api.FeatureSensitiveData, instance) == nil,
	})
	select {
	case <-ctx.Done():
		// canceled or timed out
		return nil, time.Now().UnixNano() - start, errors.Errorf("timeout reached: %v", timeout)
	default:
		// So the select will not block
	}

	return result, time.Now().UnixNano() - start, err
}

// preQuery does the following:
//  1. Validate the request.
//     i. Check if the instance exists.
//     ii. Check if the database exists.
//     iii. Check if the query is valid.
//  2. Check if the user has permission to execute the query.
//  3. Run SQL review.
//  4. Get sensitive schema info.
//  5. Create query activity.
//
// Due to the performance consideration, we DO NOT get the sensitive schema info if there are advice error in SQL review.
func (s *SQLService) preQuery(ctx context.Context, request *v1pb.QueryRequest) (*store.InstanceMessage, *store.DatabaseMessage, advisor.Status, []*v1pb.Advice, *db.SensitiveSchemaInfo, *store.ActivityMessage, error) {
	// Prepare related message.
	user, environment, instance, database, err := s.prepareRelatedMessage(ctx, request.Name, request.ConnectionDatabase)
	if err != nil {
		return nil, nil, advisor.Success, nil, nil, nil, err
	}

	// Validate the request.
	if err := s.validateQueryRequest(instance, request.ConnectionDatabase, request.Statement); err != nil {
		return nil, nil, advisor.Success, nil, nil, nil, err
	}

	if s.licenseService.IsFeatureEnabled(api.FeatureAccessControl) == nil {
		// Check if the environment is open for query privileges.
		result, err := s.checkWorkspaceIAMPolicy(ctx, common.ProjectQuerier, environment)
		if err != nil {
			return nil, nil, advisor.Success, nil, nil, nil, err
		}
		if !result {
			// Check if the user has permission to execute the query.
			if err := s.checkQueryRights(ctx, request.ConnectionDatabase, database.DataShare, request.Statement, request.Limit, user, instance, v1pb.ExportRequest_FORMAT_UNSPECIFIED); err != nil {
				return nil, nil, advisor.Success, nil, nil, nil, err
			}
		}
	}

	// Run SQL review.
	adviceStatus, adviceList, err := s.sqlReviewCheck(ctx, request, environment, instance, database)
	if err != nil {
		return nil, nil, adviceStatus, adviceList, nil, nil, err
	}

	// Get sensitive schema info.
	var sensitiveSchemaInfo *db.SensitiveSchemaInfo
	if adviceStatus != advisor.Error {
		switch instance.Engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			databaseList, err := parser.ExtractDatabaseList(parser.MySQL, request.Statement, "")
			if err != nil {
				return nil, nil, advisor.Success, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get database list: %s with error %v", request.Statement, err)
			}

			sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, databaseList, request.ConnectionDatabase)
			if err != nil {
				return nil, nil, advisor.Success, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get sensitive schema info for statement: %s, error: %v", request.Statement, err.Error())
			}
		case db.Postgres, db.Redshift:
			sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, []string{request.ConnectionDatabase}, request.ConnectionDatabase)
			if err != nil {
				return nil, nil, advisor.Success, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get sensitive schema info for statement: %s, error: %v", request.Statement, err.Error())
			}
		case db.Oracle, db.DM:
			if instance.Options == nil || !instance.Options.SchemaTenantMode {
				sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, []string{request.ConnectionDatabase}, request.ConnectionDatabase)
				if err != nil {
					return nil, nil, advisor.Success, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get sensitive schema info for statement: %s, error: %v", request.Statement, err.Error())
				}
			} else {
				list, err := parser.ExtractResourceList(parser.Oracle, request.ConnectionDatabase, request.ConnectionDatabase, request.Statement)
				if err != nil {
					return nil, nil, advisor.Success, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get resource list: %s", request.Statement)
				}
				databaseMap := make(map[string]bool)
				for _, resource := range list {
					databaseMap[resource.Database] = true
				}
				var databaseList []string
				databaseList = append(databaseList, request.ConnectionDatabase)
				for database := range databaseMap {
					databaseList = append(databaseList, database)
				}
				sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, databaseList, request.ConnectionDatabase)
				if err != nil {
					return nil, nil, advisor.Success, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get sensitive schema info for statement: %s, error: %v", request.Statement, err.Error())
				}
			}
		case db.Snowflake:
			databaseList, err := parser.ExtractDatabaseList(parser.Snowflake, request.Statement, request.ConnectionDatabase)
			if err != nil {
				return nil, nil, advisor.Success, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get database list: %s with error %v", request.Statement, err)
			}

			sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, databaseList, request.ConnectionDatabase)
			if err != nil {
				return nil, nil, advisor.Success, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get sensitive schema info for statement: %s, error: %v", request.Statement, err.Error())
			}
		}
	}

	// Create query activity.
	level := api.ActivityInfo
	switch adviceStatus {
	case advisor.Error:
		level = api.ActivityError
	case advisor.Warn:
		level = api.ActivityWarn
	}
	activity, err := s.createQueryActivity(ctx, user, level, instance.UID, api.ActivitySQLEditorQueryPayload{
		Statement:              request.Statement,
		InstanceID:             instance.UID,
		DeprecatedInstanceName: instance.Title,
		DatabaseID:             database.UID,
		DatabaseName:           request.ConnectionDatabase,
		// TODO: here we should use []*v1pb.Advice instead of []advisor.Advice
		// This should fix when we migrate to v1 activity API
		// AdviceList:             adviceList,
	})
	if err != nil {
		return nil, nil, advisor.Success, nil, nil, nil, err
	}

	return instance, database, adviceStatus, adviceList, sensitiveSchemaInfo, activity, nil
}

func (s *SQLService) createQueryActivity(ctx context.Context, user *store.UserMessage, level api.ActivityLevel, containerID int, payload api.ActivitySQLEditorQueryPayload) (*store.ActivityMessage, error) {
	// TODO: use v1 activity API instead of
	activityBytes, err := json.Marshal(payload)
	if err != nil {
		log.Warn("Failed to marshal activity before executing sql statement",
			zap.String("database_name", payload.DatabaseName),
			zap.Int("instance_id", payload.InstanceID),
			zap.String("statement", payload.Statement),
			zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Failed to construct activity payload: %v", err)
	}

	activityCreate := &store.ActivityMessage{
		CreatorUID:   user.ID,
		Type:         api.ActivitySQLEditorQuery,
		ContainerUID: containerID,
		Level:        level,
		Comment: fmt.Sprintf("Executed `%q` in database %q of instance %d.",
			payload.Statement, payload.DatabaseName, payload.InstanceID),
		Payload: string(activityBytes),
	}

	activity, err := s.store.CreateActivityV2(ctx, activityCreate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create activity: %v", err)
	}
	return activity, nil
}

func (s *SQLService) getSensitiveSchemaInfo(ctx context.Context, instance *store.InstanceMessage, databaseList []string, currentDatabase string) (*db.SensitiveSchemaInfo, error) {
	type sensitiveDataMap map[api.SensitiveData]api.SensitiveDataMaskType
	isEmpty := true
	result := &db.SensitiveSchemaInfo{
		DatabaseList: []db.DatabaseSchema{},
	}
	for _, name := range databaseList {
		databaseName := name
		if name == "" {
			if currentDatabase == "" {
				continue
			}
			databaseName = currentDatabase
		}

		if isExcludeDatabase(instance.Engine, databaseName) {
			continue
		}

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
		if err != nil {
			return nil, err
		}
		if database == nil {
			return nil, errors.Errorf("database %q not found", databaseName)
		}

		policy, err := s.store.GetSensitiveDataPolicy(ctx, database.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to find sensitive data policy for database %q in instance %q: %v", databaseName, instance.Title, err)
		}
		if len(policy.SensitiveDataList) == 0 {
			// If there is no sensitive data policy, return nil to skip mask sensitive data.
			return nil, nil
		}

		columnMap := make(sensitiveDataMap)
		for _, data := range policy.SensitiveDataList {
			columnMap[api.SensitiveData{
				Schema: data.Schema,
				Table:  data.Table,
				Column: data.Column,
			}] = data.Type
		}

		dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to find schema for database %q in instance %q: %v", databaseName, instance.Title, err)
		}

		if instance.Engine == db.Oracle || instance.Engine == db.DM{
			for _, schema := range dbSchema.Metadata.Schemas {
				databaseSchema := db.DatabaseSchema{
					Name:      schema.Name,
					TableList: []db.TableSchema{},
				}
				for _, table := range schema.Tables {
					tableSchema := db.TableSchema{
						Name:       table.Name,
						ColumnList: []db.ColumnInfo{},
					}
					for _, column := range table.Columns {
						_, sensitive := columnMap[api.SensitiveData{
							Schema: schema.Name,
							Table:  table.Name,
							Column: column.Name,
						}]
						tableSchema.ColumnList = append(tableSchema.ColumnList, db.ColumnInfo{
							Name:      column.Name,
							Sensitive: sensitive,
						})
					}
					databaseSchema.TableList = append(databaseSchema.TableList, tableSchema)
				}
				if len(databaseSchema.TableList) > 0 {
					isEmpty = false
				}
				result.DatabaseList = append(result.DatabaseList, databaseSchema)
			}
			continue
		}

		databaseSchema := db.DatabaseSchema{
			Name:       databaseName,
			SchemaList: []db.SchemaSchema{},
			TableList:  []db.TableSchema{},
		}
		for _, schema := range dbSchema.Metadata.Schemas {
			schemaSchema := db.SchemaSchema{
				Name:      schema.Name,
				TableList: []db.TableSchema{},
			}
			for _, table := range schema.Tables {
				tableSchema := db.TableSchema{
					Name:       table.Name,
					ColumnList: []db.ColumnInfo{},
				}
				if instance.Engine == db.Postgres || instance.Engine == db.Redshift {
					tableSchema.Name = fmt.Sprintf("%s.%s", schema.Name, table.Name)
				}
				for _, column := range table.Columns {
					_, sensitive := columnMap[api.SensitiveData{
						Schema: schema.Name,
						Table:  table.Name,
						Column: column.Name,
					}]
					tableSchema.ColumnList = append(tableSchema.ColumnList, db.ColumnInfo{
						Name:      column.Name,
						Sensitive: sensitive,
					})
				}
				if instance.Engine == db.Snowflake {
					schemaSchema.TableList = append(schemaSchema.TableList, tableSchema)
				} else {
					databaseSchema.TableList = append(databaseSchema.TableList, tableSchema)
				}
			}
			if instance.Engine == db.Snowflake {
				databaseSchema.SchemaList = append(databaseSchema.SchemaList, schemaSchema)
			}
		}
		if instance.Engine == db.Snowflake {
			if len(databaseSchema.SchemaList) > 0 {
				isEmpty = false
			}
		} else {
			if len(databaseSchema.TableList) > 0 {
				isEmpty = false
			}
		}
		result.DatabaseList = append(result.DatabaseList, databaseSchema)
	}

	if isEmpty {
		// If there is no tables, this query may access system databases, such as INFORMATION_SCHEMA.
		// Skip to extract sensitive column for this query.
		result = nil
	}
	return result, nil
}

func isExcludeDatabase(dbType db.Type, database string) bool {
	switch dbType {
	case db.MySQL, db.MariaDB:
		return isMySQLExcludeDatabase(database)
	case db.TiDB:
		if isMySQLExcludeDatabase(database) {
			return true
		}
		return database == "metrics_schema"
	case db.Snowflake:
		return database == "SNOWFLAKE"
	default:
		return false
	}
}

func isMySQLExcludeDatabase(database string) bool {
	if strings.ToLower(database) == "information_schema" {
		return true
	}

	switch database {
	case "mysql":
	case "sys":
	case "performance_schema":
	default:
		return false
	}
	return true
}

// getReadOnlyDataSource returns the read-only data source for the instance.
// If the read-only data source is not defined, we will fallback to admin data source.
func getReadOnlyDataSource(instance *store.InstanceMessage) *store.DataSourceMessage {
	dataSource := utils.DataSourceFromInstanceWithType(instance, api.RO)
	adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
	if dataSource == nil {
		dataSource = adminDataSource
	}
	return dataSource
}

func (s *SQLService) sqlReviewCheck(ctx context.Context, request *v1pb.QueryRequest, environment *store.EnvironmentMessage, instance *store.InstanceMessage, database *store.DatabaseMessage) (advisor.Status, []*v1pb.Advice, error) {
	if !IsSQLReviewSupported(instance.Engine) || request.ConnectionDatabase == "" || database == nil {
		return advisor.Success, nil, nil
	}

	dbType, err := advisorDB.ConvertToAdvisorDBType(string(instance.Engine))
	if err != nil {
		return advisor.Error, nil, status.Errorf(codes.Internal, "failed to convert engine to advisor db type: %v", err)
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return advisor.Error, nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
	}
	if dbSchema == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
			return advisor.Error, nil, status.Errorf(codes.Internal, "failed to sync database schema: %v", err)
		}
	}
	dbSchema, err = s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return advisor.Error, nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
	}
	if dbSchema == nil {
		return advisor.Error, nil, status.Errorf(codes.NotFound, "database schema not found: %v", database.UID)
	}

	catalog, err := s.store.NewCatalog(ctx, database.UID, instance.Engine, advisor.SyntaxModeNormal)
	if err != nil {
		return advisor.Error, nil, status.Errorf(codes.Internal, "Failed to create a catalog: %v", err)
	}

	currentSchema := ""
	if instance.Engine == db.Oracle || instance.Engine == db.DM{
		if instance.Options == nil || !instance.Options.SchemaTenantMode {
			currentSchema = getReadOnlyDataSource(instance).Username
		} else {
			currentSchema = database.DatabaseName
		}
	}

	driver, err := s.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, database)
	if err != nil {
		return advisor.Error, nil, status.Errorf(codes.Internal, "Failed to get database driver: %v", err)
	}
	defer driver.Close(ctx)
	connection := driver.GetDB()
	adviceLevel, adviceList, err := s.sqlCheck(
		ctx,
		dbType,
		dbSchema.Metadata.CharacterSet,
		dbSchema.Metadata.Collation,
		environment.UID,
		request.Statement,
		catalog,
		connection,
		currentSchema,
		database.DatabaseName,
	)
	if err != nil {
		return advisor.Error, nil, status.Errorf(codes.Internal, "Failed to check SQL review policy: %v", err)
	}

	return adviceLevel, convertAdviceList(adviceList), nil
}

func convertAdviceList(list []advisor.Advice) []*v1pb.Advice {
	var result []*v1pb.Advice
	for _, advice := range list {
		result = append(result, &v1pb.Advice{
			Status:  convertAdviceStatus(advice.Status),
			Code:    int32(advice.Code),
			Title:   advice.Title,
			Content: advice.Content,
			Line:    int32(advice.Line),
			Detail:  advice.Details,
		})
	}
	return result
}

func convertAdviceStatus(status advisor.Status) v1pb.Advice_Status {
	switch status {
	case advisor.Success:
		return v1pb.Advice_SUCCESS
	case advisor.Warn:
		return v1pb.Advice_WARNING
	case advisor.Error:
		return v1pb.Advice_ERROR
	default:
		return v1pb.Advice_STATUS_UNSPECIFIED
	}
}

func (s *SQLService) sqlCheck(
	ctx context.Context,
	dbType advisorDB.Type,
	dbCharacterSet string,
	dbCollation string,
	environmentID int,
	statement string,
	catalog catalog.Catalog,
	driver *sql.DB,
	currentSchema string,
	currentDatabase string,
) (advisor.Status, []advisor.Advice, error) {
	var adviceList []advisor.Advice
	policy, err := s.store.GetSQLReviewPolicy(ctx, environmentID)
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
			return advisor.Success, nil, nil
		}
		return advisor.Error, nil, err
	}

	res, err := advisor.SQLReviewCheck(statement, policy.RuleList, advisor.SQLReviewCheckContext{
		Charset:         dbCharacterSet,
		Collation:       dbCollation,
		DbType:          dbType,
		Catalog:         catalog,
		Driver:          driver,
		Context:         ctx,
		CurrentSchema:   currentSchema,
		CurrentDatabase: currentDatabase,
	})
	if err != nil {
		return advisor.Error, nil, err
	}

	adviceLevel := advisor.Success
	for _, advice := range res {
		switch advice.Status {
		case advisor.Warn:
			if adviceLevel != advisor.Error {
				adviceLevel = advisor.Warn
			}
		case advisor.Error:
			adviceLevel = advisor.Error
		case advisor.Success:
			continue
		}

		adviceList = append(adviceList, advice)
	}

	return adviceLevel, adviceList, nil
}

func (s *SQLService) prepareRelatedMessage(ctx context.Context, instanceToken string, databaseName string) (*store.UserMessage, *store.EnvironmentMessage, *store.InstanceMessage, *store.DatabaseMessage, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	instance, err := s.getInstanceMessage(ctx, instanceToken)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &instance.EnvironmentID})
	if err != nil {
		return nil, nil, nil, nil, status.Errorf(codes.Internal, "failed to fetch environment: %v", err)
	}
	if environment == nil {
		return nil, nil, nil, nil, status.Errorf(codes.NotFound, "environment ID not found: %s", instance.EnvironmentID)
	}

	var database *store.DatabaseMessage
	if databaseName != "" {
		database, err = s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
		if err != nil {
			return nil, nil, nil, nil, status.Errorf(codes.Internal, "failed to fetch database: %v", err)
		}
	}

	return user, environment, instance, database, nil
}

// validateQueryRequest validates the query request.
// 1. Check if the instance exists.
// 2. Check connection_database if the instance is postgres.
// 3. Parse statement for Postgres, MySQL, TiDB, Oracle.
// 4. Check if all statements are (EXPLAIN) SELECT statements.
func (*SQLService) validateQueryRequest(instance *store.InstanceMessage, databaseName string, statement string) error {
	if instance.Engine == db.Postgres {
		if databaseName == "" {
			return status.Error(codes.InvalidArgument, "connection_database is required for postgres instance")
		}
	}

	switch instance.Engine {
	case db.Postgres:
		stmtList, err := parser.Parse(parser.Postgres, parser.ParseContext{}, statement)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "failed to parse query: %s", err.Error())
		}
		for _, stmt := range stmtList {
			switch stmt.(type) {
			case *ast.SelectStmt, *ast.ExplainStmt:
			default:
				return status.Errorf(codes.InvalidArgument, "Malformed sql execute request, only support SELECT sql statement")
			}
		}
	case db.MySQL:
		trees, err := parser.ParseMySQL(statement)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "failed to parse query: %s", err.Error())
		}
		for _, item := range trees {
			tree := item.Tree
			if err := parser.MySQLValidateForEditor(tree); err != nil {
				return status.Errorf(codes.InvalidArgument, err.Error())
			}
		}
	case db.TiDB:
		stmtList, err := parser.ParseTiDB(statement, "", "")
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "failed to parse query: %s", err.Error())
		}
		for _, stmt := range stmtList {
			switch stmt.(type) {
			case *tidbast.SelectStmt, *tidbast.ExplainStmt:
			default:
				return status.Errorf(codes.InvalidArgument, "Malformed sql execute request, only support SELECT sql statement")
			}
		}
	case db.Oracle, db.DM:
		if instance.Options != nil && instance.Options.SchemaTenantMode && databaseName == "" {
			return status.Error(codes.InvalidArgument, "connection_database is required for oracle schema tenant mode instance")
		}
		tree, _, err := parser.ParsePLSQL(statement)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "failed to parse query: %s", err.Error())
		}
		if err := parser.PLSQLValidateForEditor(tree); err != nil {
			return status.Errorf(codes.InvalidArgument, err.Error())
		}
	case db.MongoDB, db.Redis:
		// Do nothing.
	default:
		// TODO(rebelice): support multiple statements here.
		if !parser.ValidateSQLForEditor(convertToParserEngine(instance.Engine), statement) {
			return status.Errorf(codes.InvalidArgument, "Malformed sql execute request, only support SELECT sql statement")
		}
	}

	return nil
}

func (s *SQLService) extractResourceList(ctx context.Context, engine parser.EngineType, databaseName string, statement string, instance *store.InstanceMessage) ([]parser.SchemaResource, error) {
	switch engine {
	case parser.MySQL, parser.MariaDB, parser.OceanBase:
		list, err := parser.ExtractResourceList(engine, databaseName, "", statement)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to extract resource list: %s", err.Error())
		}

		databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
		if err != nil {
			if httpErr, ok := err.(*echo.HTTPError); ok && httpErr.Code == echo.ErrNotFound.Code {
				// If database not found, skip.
				return nil, nil
			}
			return nil, status.Errorf(codes.Internal, "failed to fetch database: %v", err)
		}

		dbSchema, err := s.store.GetDBSchema(ctx, databaseMessage.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
		}

		var result []parser.SchemaResource
		for _, resource := range list {
			if resource.Database != dbSchema.Metadata.Name {
				// MySQL allows cross-database query, we should check the corresponding database.
				resourceDB, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &resource.Database})
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if resourceDB == nil {
					continue
				}
				resourceDBSchema, err := s.store.GetDBSchema(ctx, resourceDB.UID)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database schema %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if !resourceDBSchema.TableExists(resource.Schema, resource.Table) && !resourceDBSchema.ViewExists(resource.Schema, resource.Table) {
					// If table not found, we regard it as a CTE/alias/... and skip.
					continue
				}
				result = append(result, resource)
				continue
			}
			if !dbSchema.TableExists(resource.Schema, resource.Table) && !dbSchema.ViewExists(resource.Schema, resource.Table) {
				// If table not found, skip.
				continue
			}
			result = append(result, resource)
		}
		return result, nil
	case parser.Postgres, parser.Redshift:
		list, err := parser.ExtractResourceList(engine, databaseName, "public", statement)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to extract resource list: %s", err.Error())
		}

		databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database: %v", err)
		}
		if databaseMessage == nil {
			return nil, nil
		}

		dbSchema, err := s.store.GetDBSchema(ctx, databaseMessage.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
		}

		var result []parser.SchemaResource
		for _, resource := range list {
			if resource.Database != dbSchema.Metadata.Name {
				// Should not happen.
				continue
			}

			if !dbSchema.TableExists(resource.Schema, resource.Table) && !dbSchema.ViewExists(resource.Schema, resource.Table) {
				// If table not found, skip.
				continue
			}

			result = append(result, resource)
		}

		return result, nil
	case parser.Oracle:
		dataSource := utils.DataSourceFromInstanceWithType(instance, api.RO)
		adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
		// If there are no read-only data source, fall back to admin data source.
		if dataSource == nil {
			dataSource = adminDataSource
		}
		if dataSource == nil {
			return nil, status.Errorf(codes.Internal, "failed to find data source for instance: %s", instance.ResourceID)
		}
		currentSchema := dataSource.Username
		if instance.Options != nil && instance.Options.SchemaTenantMode {
			currentSchema = databaseName
		}
		list, err := parser.ExtractResourceList(engine, databaseName, currentSchema, statement)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to extract resource list: %s", err.Error())
		}

		databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database: %v", err)
		}
		if databaseMessage == nil {
			return nil, nil
		}

		dbSchema, err := s.store.GetDBSchema(ctx, databaseMessage.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
		}

		var result []parser.SchemaResource
		for _, resource := range list {
			if resource.Database != dbSchema.Metadata.Name {
				if instance.Options == nil || !instance.Options.SchemaTenantMode {
					continue
				}
				// Schema tenant mode allows cross-database query, we should check the corresponding database.
				resourceDB, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &resource.Database})
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if resourceDB == nil {
					continue
				}
				resourceDBSchema, err := s.store.GetDBSchema(ctx, resourceDB.UID)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database schema %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if !resourceDBSchema.TableExists(resource.Schema, resource.Table) && !resourceDBSchema.ViewExists(resource.Schema, resource.Table) {
					// If table not found, we regard it as a CTE/alias/... and skip.
					continue
				}
				result = append(result, resource)
				continue
			}

			if !dbSchema.TableExists(resource.Schema, resource.Table) && !dbSchema.ViewExists(resource.Schema, resource.Table) {
				// If table not found, skip.
				continue
			}

			result = append(result, resource)
		}

		return result, nil
	case parser.Snowflake:
		dataSource := utils.DataSourceFromInstanceWithType(instance, api.RO)
		adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
		// If there are no read-only data source, fall back to admin data source.
		if dataSource == nil {
			dataSource = adminDataSource
		}
		if dataSource == nil {
			return nil, status.Errorf(codes.Internal, "failed to find data source for instance: %s", instance.ResourceID)
		}
		list, err := parser.ExtractResourceList(engine, databaseName, dataSource.Username, statement)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to extract resource list: %s", err.Error())
		}
		databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
		if err != nil {
			if httpErr, ok := err.(*echo.HTTPError); ok && httpErr.Code == echo.ErrNotFound.Code {
				// If database not found, skip.
				return nil, nil
			}
			return nil, status.Errorf(codes.Internal, "failed to fetch database: %v", err)
		}

		dbSchema, err := s.store.GetDBSchema(ctx, databaseMessage.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
		}

		var result []parser.SchemaResource
		for _, resource := range list {
			if resource.Database != dbSchema.Metadata.Name {
				// Snowflake allows cross-database query, we should check the corresponding database.
				resourceDB, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &resource.Database})
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if resourceDB == nil {
					continue
				}
				resourceDBSchema, err := s.store.GetDBSchema(ctx, resourceDB.UID)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get database schema %v in instance %v, err: %v", resource.Database, instance.ResourceID, err)
				}
				if !resourceDBSchema.TableExists(resource.Schema, resource.Table) && !resourceDBSchema.ViewExists(resource.Schema, resource.Table) {
					// If table not found, we regard it as a CTE/alias/... and skip.
					continue
				}
				result = append(result, resource)
				continue
			}
			if !dbSchema.TableExists(resource.Schema, resource.Table) && !dbSchema.ViewExists(resource.Schema, resource.Table) {
				// If table not found, skip.
				continue
			}
			result = append(result, resource)
		}
		return result, nil
	default:
		return parser.ExtractResourceList(engine, databaseName, "", statement)
	}
}

func (s *SQLService) checkWorkspaceIAMPolicy(
	ctx context.Context,
	role common.ProjectRole,
	environment *store.EnvironmentMessage,
) (bool, error) {
	workspacePolicyResourceType := api.PolicyResourceTypeWorkspace
	workspaceIAMPolicyType := api.PolicyTypeWorkspaceIAM
	policy, err := s.store.GetPolicyV2(ctx, &store.FindPolicyMessage{
		ResourceType: &workspacePolicyResourceType,
		Type:         &workspaceIAMPolicyType,
		ResourceUID:  &defaultWorkspaceResourceID,
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to get workspace IAM policy")
	}
	if policy == nil {
		return false, nil
	}

	v1pbPolicy, err := convertToPolicy("", policy)
	if err != nil {
		return false, errors.Wrap(err, "failed to convert policy")
	}

	attributes := map[string]any{
		"resource.environment_name": fmt.Sprintf("%s%s", environmentNamePrefix, environment.ResourceID),
	}
	formattedRole := fmt.Sprintf("roles/%s", role)
	bindings := v1pbPolicy.GetWorkspaceIamPolicy().Bindings
	for _, binding := range bindings {
		if binding.Role != formattedRole {
			continue
		}

		ok, err := evaluateCondition(binding.Condition.Expression, attributes)
		if err != nil {
			return false, errors.Wrap(err, "failed to evaluate condition")
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func (s *SQLService) checkQueryRights(
	ctx context.Context,
	databaseName string,
	datashare bool,
	statement string,
	limit int32,
	user *store.UserMessage,
	instance *store.InstanceMessage,
	exportFormat v1pb.ExportRequest_Format,
) error {
	// Owner and DBA have all rights.
	if user.Role == api.Owner || user.Role == api.DBA {
		return nil
	}

	// TODO(d): use a Redshift extraction for shared database.
	extractingStatement := statement
	if datashare {
		extractingStatement = strings.ReplaceAll(statement, fmt.Sprintf("%s.", databaseName), "")
	}
	resourceList, err := s.extractResourceList(ctx, convertToParserEngine(instance.Engine), databaseName, extractingStatement, instance)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to extract resource list: %v", err)
	}

	databaseMap := make(map[string]bool)
	for _, resource := range resourceList {
		databaseMap[resource.Database] = true
	}

	if databaseName != "" {
		databaseMap[databaseName] = true
	}

	var project *store.ProjectMessage

	databaseMessageMap := make(map[string]*store.DatabaseMessage)
	for database := range databaseMap {
		projectMessage, databaseMessage, err := s.getProjectAndDatabaseMessage(ctx, instance, database)
		if err != nil {
			return err
		}
		if projectMessage == nil && databaseMessage == nil {
			// If database not found, skip.
			continue
		}
		if projectMessage == nil {
			// Never happen
			return status.Errorf(codes.Internal, "project not found for database: %s", databaseMessage.DatabaseName)
		}
		if project == nil {
			project = projectMessage
		}
		if project.UID != projectMessage.UID {
			return status.Errorf(codes.InvalidArgument, "allow querying databases within the same project only")
		}
		databaseMessageMap[database] = databaseMessage
	}

	if len(databaseMessageMap) == 0 && project == nil {
		project, _, err = s.getProjectAndDatabaseMessage(ctx, instance, databaseName)
		if err != nil {
			return err
		}
	}

	if project == nil {
		// Never happen
		return status.Error(codes.Internal, "project not found")
	}

	projectPolicy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return err
	}

	var conditionExpression string
	isExport := exportFormat != v1pb.ExportRequest_FORMAT_UNSPECIFIED
	for _, resource := range resourceList {
		databaseResourceURL := fmt.Sprintf("instances/%s/databases/%s", instance.ResourceID, resource.Database)
		attributes := map[string]any{
			"request.time":      time.Now(),
			"resource.database": databaseResourceURL,
			"resource.schema":   resource.Schema,
			"resource.table":    resource.Table,
			"request.statement": encodeToBase64String(statement),
			"request.row_limit": limit,
		}

		switch exportFormat {
		case v1pb.ExportRequest_FORMAT_UNSPECIFIED:
			attributes["request.export_format"] = "QUERY"
		case v1pb.ExportRequest_CSV:
			attributes["request.export_format"] = "CSV"
		case v1pb.ExportRequest_JSON:
			attributes["request.export_format"] = "JSON"
		case v1pb.ExportRequest_SQL:
			attributes["request.export_format"] = "SQL"
		case v1pb.ExportRequest_XLSX:
			attributes["request.export_format"] = "XLSX"
		default:
			return status.Errorf(codes.InvalidArgument, "invalid export format: %v", exportFormat)
		}

		ok, expression, err := hasDatabaseAccessRights(user.ID, projectPolicy, attributes, isExport)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check access control for database: %q", resource.Database)
		}
		if !ok {
			return status.Errorf(codes.PermissionDenied, "permission denied to access resource: %q", resource.Pretty())
		}
		conditionExpression = expression
	}

	if isExport {
		newPolicy := removeExportBinding(user.ID, conditionExpression, projectPolicy)
		if _, err := s.store.SetProjectIAMPolicy(ctx, newPolicy, api.SystemBotID, project.UID); err != nil {
			return err
		}
		// Post project IAM policy update activity.
		if _, err := s.activityManager.CreateActivity(ctx, &store.ActivityMessage{
			CreatorUID:   api.SystemBotID,
			ContainerUID: project.UID,
			Type:         api.ActivityProjectMemberCreate,
			Level:        api.ActivityInfo,
			Comment:      fmt.Sprintf("Granted %s to %s (%s).", user.Name, user.Email, api.Role(common.ProjectExporter)),
		}, &activity.Metadata{}); err != nil {
			log.Warn("Failed to create project activity", zap.Error(err))
		}
	}
	return nil
}

func removeExportBinding(principalID int, usedExpression string, projectPolicy *store.IAMPolicyMessage) *store.IAMPolicyMessage {
	var newPolicy store.IAMPolicyMessage
	for _, binding := range projectPolicy.Bindings {
		if binding.Role != api.Role(common.ProjectExporter) || binding.Condition.Expression != usedExpression {
			newPolicy.Bindings = append(newPolicy.Bindings, binding)
			continue
		}

		var newMembers []*store.UserMessage
		for _, member := range binding.Members {
			if member.ID != principalID {
				newMembers = append(newMembers, member)
			}
		}
		if len(newMembers) == 0 {
			continue
		}
		newBinding := *binding
		newBinding.Members = newMembers
		newPolicy.Bindings = append(newPolicy.Bindings, &newBinding)
	}
	return &newPolicy
}

func hasDatabaseAccessRights(principalID int, projectPolicy *store.IAMPolicyMessage, attributes map[string]any, isExport bool) (bool, string, error) {
	// TODO(rebelice): implement table-level query permission check and refactor this function.
	// Project IAM policy evaluation.
	pass := false
	usedExpression := ""
	for _, binding := range projectPolicy.Bindings {
		// Project owner has all permissions.
		if binding.Role == api.Role(common.ProjectOwner) {
			for _, member := range binding.Members {
				if member.ID == principalID {
					pass = true
					break
				}
			}
		}
		if !((isExport && binding.Role == api.Role(common.ProjectExporter)) || (!isExport && binding.Role == api.Role(common.ProjectQuerier))) {
			continue
		}
		for _, member := range binding.Members {
			if member.ID != principalID {
				continue
			}
			ok, err := evaluateCondition(binding.Condition.Expression, attributes)
			if err != nil {
				log.Error("failed to evaluate condition", zap.Error(err), zap.String("condition", binding.Condition.Expression))
				break
			}
			if ok {
				pass = true
				usedExpression = binding.Condition.Expression
				break
			}
		}
		if pass {
			break
		}
	}
	return pass, usedExpression, nil
}

func evaluateCondition(expression string, attributes map[string]any) (bool, error) {
	if expression == "" {
		return true, nil
	}
	env, err := cel.NewEnv(common.QueryExportPolicyCELAttributes...)
	if err != nil {
		return false, err
	}
	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, issues.Err()
	}
	prg, err := env.Program(ast)
	if err != nil {
		return false, err
	}

	out, _, err := prg.Eval(attributes)
	if err != nil {
		return false, err
	}
	val, err := out.ConvertToNative(reflect.TypeOf(false))
	if err != nil {
		return false, errors.Wrap(err, "expect bool result")
	}
	boolVal, ok := val.(bool)
	if !ok {
		return false, errors.Wrap(err, "failed to convert to bool")
	}
	return boolVal, nil
}

func (s *SQLService) getProjectAndDatabaseMessage(ctx context.Context, instance *store.InstanceMessage, database string) (*store.ProjectMessage, *store.DatabaseMessage, error) {
	databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &database})
	if err != nil {
		return nil, nil, err
	}
	if databaseMessage == nil {
		return nil, nil, nil
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &databaseMessage.ProjectID})
	if err != nil {
		return nil, nil, err
	}
	return project, databaseMessage, nil
}

func (s *SQLService) getUser(ctx context.Context) (*store.UserMessage, error) {
	principalPtr := ctx.Value(common.PrincipalIDContextKey)
	if principalPtr == nil {
		return nil, nil
	}
	principalID := principalPtr.(int)
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "failed to get member for user %v in processing authorize request.", principalID)
	}
	if user == nil {
		return nil, status.Errorf(codes.PermissionDenied, "member not found for user %v in processing authorize request.", principalID)
	}
	if user.MemberDeleted {
		return nil, status.Errorf(codes.PermissionDenied, "the user %v has been deactivated by the admin.", principalID)
	}

	return user, nil
}

func (s *SQLService) getInstanceMessage(ctx context.Context, name string) (*store.InstanceMessage, error) {
	instanceID, err := getInstanceID(name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	find := &store.FindInstanceMessage{}
	instanceUID, isNumber := isNumber(instanceID)
	if isNumber {
		find.UID = &instanceUID
	} else {
		find.ResourceID = &instanceID
	}

	instance, err := s.store.GetInstanceV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", name)
	}

	return instance, nil
}

func convertToParserEngine(engine db.Type) parser.EngineType {
	// convert to parser engine
	switch engine {
	case db.Postgres:
		return parser.Postgres
	case db.Redshift:
		return parser.Redshift
	case db.MySQL:
		return parser.MySQL
	case db.TiDB:
		return parser.TiDB
	case db.MariaDB:
		return parser.MariaDB
	case db.Oracle:
		return parser.Oracle
	case db.MSSQL:
		return parser.MSSQL
	case db.OceanBase:
		return parser.OceanBase
	case db.Snowflake:
		return parser.Snowflake
	case db.DM:
		return parser.Oracle
	}
	return parser.Standard
}

// IsSQLReviewSupported checks the engine type if SQL review supports it.
func IsSQLReviewSupported(dbType db.Type) bool {
	switch dbType {
	case db.Postgres, db.MySQL, db.TiDB, db.MariaDB, db.Oracle, db.OceanBase, db.Snowflake,db.DM:
		advisorDB, err := advisorDB.ConvertToAdvisorDBType(string(dbType))
		if err != nil {
			return false
		}

		return advisor.IsSQLReviewSupported(advisorDB)
	default:
		return false
	}
}

// encodeToBase64String encodes the statement to base64 string.
func encodeToBase64String(statement string) string {
	base64Encoded := base64.StdEncoding.EncodeToString([]byte(statement))
	return base64Encoded
}
