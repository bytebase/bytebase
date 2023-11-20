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

	"log/slog"

	"github.com/google/cel-go/cel"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/xuri/excelize/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/masker"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const (
	// The maximum number of bytes for sql results in response body.
	// 10 MB.
	maximumSQLResultSize = 10 * 1024 * 1024
	// defaultTimeout is the default timeout for query and admin execution.
	defaultTimeout = 10 * time.Minute
)

// SQLService is the service for SQL.
type SQLService struct {
	v1pb.UnimplementedSQLServiceServer
	store           *store.Store
	schemaSyncer    *schemasync.Syncer
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	licenseService  enterprise.LicenseService
}

// NewSQLService creates a SQLService.
func NewSQLService(
	store *store.Store,
	schemaSyncer *schemasync.Syncer,
	dbFactory *dbfactory.DBFactory,
	activityManager *activity.Manager,
	licenseService enterprise.LicenseService,
) *SQLService {
	return &SQLService{
		store:           store,
		schemaSyncer:    schemaSyncer,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		licenseService:  licenseService,
	}
}

type maskingPolicyKey struct {
	schema string
	table  string
	column string
}

// Pretty returns pretty format SDL.
func (*SQLService) Pretty(_ context.Context, request *v1pb.PrettyRequest) (*v1pb.PrettyResponse, error) {
	engine := convertEngine(request.Engine)
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
				slog.Warn("failed to close connection", log.BBError(err))
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
		sanitizeResults(result)

		if err := s.postAdminExecute(ctx, activity, durationNs, queryErr); err != nil {
			slog.Error("failed to post admin execute activity", log.BBError(err))
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
		slog.Warn("Failed to marshal activity after executing sql statement",
			slog.String("database_name", payload.DatabaseName),
			slog.Int("instance_id", payload.InstanceID),
			slog.String("statement", payload.Statement),
			log.BBError(err))
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
	databaseID := 0
	if database != nil {
		databaseID = database.UID
	}
	activity, err := s.createQueryActivity(ctx, user, api.ActivityInfo, instance.UID, api.ActivitySQLEditorQueryPayload{
		Statement:              request.Statement,
		InstanceID:             instance.UID,
		DeprecatedInstanceName: instance.Title,
		DatabaseID:             databaseID,
		DatabaseName:           request.ConnectionDatabase,
	})
	if err != nil {
		return nil, nil, nil, err
	}

	return instance, database, activity, nil
}

// Export exports the SQL query result.
func (s *SQLService) Export(ctx context.Context, request *v1pb.ExportRequest) (*v1pb.ExportResponse, error) {
	// TODO(zp): Remove this hack after switching all engines to use query span.
	_, _, instance, _, err := s.prepareRelatedMessage(ctx, request.Name, request.ConnectionDatabase)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to prepare related message")
	}
	if instance.Engine == storepb.Engine_POSTGRES {
		return s.ExportV2(ctx, request)
	}
	user, instance, database, _, _, sensitiveSchemaInfo, err := s.preCheck(ctx, request.Name, request.ConnectionDatabase, request.Statement, request.Limit, false /* isAdmin */, false /* isExport */)
	if err != nil {
		return nil, err
	}

	databaseID := 0
	if database != nil {
		databaseID = database.UID
	}
	// Create export activity.
	level := api.ActivityInfo
	activity, err := s.createExportActivity(ctx, user, level, instance.UID, api.ActivitySQLExportPayload{
		Statement:    request.Statement,
		InstanceID:   instance.UID,
		DatabaseID:   databaseID,
		DatabaseName: request.ConnectionDatabase,
	})
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
		slog.Warn("Failed to marshal activity after exporting sql statement",
			slog.String("database_name", payload.DatabaseName),
			slog.Int("instance_id", payload.InstanceID),
			slog.String("statement", payload.Statement),
			log.BBError(err))
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

func (s *SQLService) doExport(ctx context.Context, request *v1pb.ExportRequest, instance *store.InstanceMessage, database *store.DatabaseMessage, sensitiveSchemaInfo *base.SensitiveSchemaInfo) ([]byte, int64, error) {
	// Don't anonymize data for exporting data using admin mode.
	if request.Admin {
		sensitiveSchemaInfo = nil
	}

	driver, err := s.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, database, "" /* dataSourceID */)
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
	result, err := driver.QueryConn(ctx, conn, request.Statement, &db.QueryContext{
		Limit:               int(request.Limit),
		ReadOnly:            true,
		CurrentDatabase:     request.ConnectionDatabase,
		SensitiveSchemaInfo: sensitiveSchemaInfo,
		EnableSensitive:     s.licenseService.IsFeatureEnabledForInstance(api.FeatureSensitiveData, instance) == nil,
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
	case v1pb.ExportFormat_CSV:
		if content, err = exportCSV(result[0]); err != nil {
			return nil, durationNs, err
		}
	case v1pb.ExportFormat_JSON:
		if content, err = exportJSON(result[0]); err != nil {
			return nil, durationNs, err
		}
	case v1pb.ExportFormat_SQL:
		resourceList, err := s.extractResourceList(ctx, instance.Engine, request.ConnectionDatabase, request.Statement, instance)
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
	case v1pb.ExportFormat_XLSX:
		if content, err = exportXLSX(result[0]); err != nil {
			return nil, durationNs, err
		}
	default:
		return nil, durationNs, status.Errorf(codes.InvalidArgument, "unsupported export format: %s", request.Format.String())
	}
	return content, durationNs, nil
}

func (*SQLService) StringifyMetadata(_ context.Context, request *v1pb.StringifyMetadataRequest) (*v1pb.StringifyMetadataResponse, error) {
	switch request.Engine {
	case v1pb.Engine_MYSQL, v1pb.Engine_POSTGRES, v1pb.Engine_TIDB:
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported engine: %v", request.Engine)
	}

	if request.Metadata == nil {
		return nil, status.Errorf(codes.InvalidArgument, "metadata is required")
	}
	if err := checkDatabaseMetadata(request.Engine, request.Metadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid metadata: %v", err))
	}

	sanitizeCommentForSchemaMetadata(request.Metadata)
	schema, err := transformDatabaseMetadataToSchemaString(request.Engine, request.Metadata)
	if err != nil {
		return nil, err
	}

	return &v1pb.StringifyMetadataResponse{
		Schema: schema,
	}, nil
}

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

func exportJSON(result *v1pb.QueryResult) ([]byte, error) {
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

func (s *SQLService) createExportActivity(ctx context.Context, user *store.UserMessage, level api.ActivityLevel, containerID int, payload api.ActivitySQLExportPayload) (*store.ActivityMessage, error) {
	// TODO: use v1 activity API instead of
	activityBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("Failed to marshal activity before exporting sql statement",
			slog.String("database_name", payload.DatabaseName),
			slog.Int("instance_id", payload.InstanceID),
			slog.String("statement", payload.Statement),
			log.BBError(err))
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

func (s *SQLService) Check(ctx context.Context, request *v1pb.CheckRequest) (*v1pb.CheckResponse, error) {
	if len(request.Statement) > common.MaxSheetCheckSize {
		return nil, status.Errorf(codes.FailedPrecondition, "statement size exceeds maximum allowed size %dKB", common.MaxSheetCheckSize/1024)
	}

	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Database)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get instance, error: %v", err)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get database, error: %v", err)
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", request.Database)
	}

	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
		ResourceID: &database.EffectiveEnvironmentID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get environment, error: %v", err)
	}
	if environment == nil {
		return nil, status.Errorf(codes.NotFound, "environment %q not found", database.EffectiveEnvironmentID)
	}

	_, adviceList, err := s.sqlReviewCheck(ctx, request.Statement, environment, instance, database)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to do sql review check, error: %v", err)
	}

	return &v1pb.CheckResponse{
		Advices: adviceList,
	}, nil
}

// Query executes a SQL query.
// We have the following stages:
//  1. pre-query
//  2. do query
//  3. post-query
func (s *SQLService) Query(ctx context.Context, request *v1pb.QueryRequest) (*v1pb.QueryResponse, error) {
	// TODO(zp): Remove this hack after switching all engines to use query span.
	_, _, instance, _, err := s.prepareRelatedMessage(ctx, request.Name, request.ConnectionDatabase)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to prepare related message")
	}
	if instance.Engine == storepb.Engine_POSTGRES {
		return s.QueryV2(ctx, request)
	}

	user, instance, database, adviceStatus, adviceList, sensitiveSchemaInfo, err := s.preCheck(ctx, request.Name, request.ConnectionDatabase, request.Statement, request.Limit, false /* isAdmin */, false /* isExport */)
	if err != nil {
		return nil, err
	}

	// Create query activity.
	level := api.ActivityInfo
	switch adviceStatus {
	case advisor.Error:
		level = api.ActivityError
	case advisor.Warn:
		level = api.ActivityWarn
	}
	databaseID := 0
	if database != nil {
		databaseID = database.UID
	}
	activity, err := s.createQueryActivity(ctx, user, level, instance.UID, api.ActivitySQLEditorQueryPayload{
		Statement:              request.Statement,
		InstanceID:             instance.UID,
		DeprecatedInstanceName: instance.Title,
		DatabaseID:             databaseID,
		DatabaseName:           request.ConnectionDatabase,
	})
	if err != nil {
		return nil, err
	}

	var results []*v1pb.QueryResult
	var queryErr error
	var durationNs int64
	if adviceStatus != advisor.Error {
		results, durationNs, queryErr = s.doQuery(ctx, request, instance, database, sensitiveSchemaInfo)
	}

	err = s.postQuery(ctx, activity, durationNs, queryErr)
	if err != nil {
		return nil, err
	}

	if queryErr != nil {
		return nil, queryErr
	}

	// AllowExport is a validate only check.
	_, _, _, _, _, _, err = s.preCheck(ctx, request.Name, request.ConnectionDatabase, request.Statement, request.Limit, false /* isAdmin */, true /* isExport */)
	allowExport := (err == nil)

	response := &v1pb.QueryResponse{
		Results:     results,
		AllowExport: allowExport,
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
func (s *SQLService) postQuery(ctx context.Context, activity *store.ActivityMessage, durationNs int64, queryErr error) error {
	newLevel := activity.Level

	// Update the activity
	var payload api.ActivitySQLEditorQueryPayload
	if err := json.Unmarshal([]byte(activity.Payload), &payload); err != nil {
		return status.Errorf(codes.Internal, "failed to unmarshal activity payload: %v", err)
	}

	payload.DurationNs = durationNs
	if queryErr != nil {
		payload.Error = queryErr.Error()
		newLevel = api.ActivityError
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("Failed to marshal activity after executing sql statement",
			slog.String("database_name", payload.DatabaseName),
			slog.Int("instance_id", payload.InstanceID),
			slog.String("statement", payload.Statement),
			log.BBError(err))
		return status.Errorf(codes.Internal, "Failed to marshal activity after executing sql statement: %v", err)
	}

	payloadString := string(payloadBytes)
	if _, err := s.store.UpdateActivityV2(ctx, &store.UpdateActivityMessage{
		UID:        activity.UID,
		UpdaterUID: activity.CreatorUID,
		Level:      &newLevel,
		Payload:    &payloadString,
	}); err != nil {
		return status.Errorf(codes.Internal, "Failed to update activity after executing sql statement: %v", err)
	}

	return nil
}

func (s *SQLService) doQuery(ctx context.Context, request *v1pb.QueryRequest, instance *store.InstanceMessage, database *store.DatabaseMessage, sensitiveSchemaInfo *base.SensitiveSchemaInfo) ([]*v1pb.QueryResult, int64, error) {
	driver, err := s.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, database, request.DataSourceId)
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
	results, err := driver.QueryConn(ctx, conn, request.Statement, &db.QueryContext{
		Limit:               int(request.Limit),
		ReadOnly:            true,
		CurrentDatabase:     request.ConnectionDatabase,
		SensitiveSchemaInfo: sensitiveSchemaInfo,
		EnableSensitive:     s.licenseService.IsFeatureEnabledForInstance(api.FeatureSensitiveData, instance) == nil,
		EngineVersion:       instance.EngineVersion,
	})
	select {
	case <-ctx.Done():
		// canceled or timed out
		return nil, time.Now().UnixNano() - start, errors.Errorf("timeout reached: %v", timeout)
	default:
		// So the select will not block
	}

	sanitizeResults(results)

	return results, time.Now().UnixNano() - start, err
}

// sanitizeResults sanitizes the strings in the results by replacing all the invalid UTF-8 characters with its hexadecimal representation.
func sanitizeResults(results []*v1pb.QueryResult) {
	for _, result := range results {
		for _, row := range result.Rows {
			for _, value := range row.Values {
				if value, ok := value.Kind.(*v1pb.RowValue_StringValue); ok {
					value.StringValue = common.SanitizeUTF8String(value.StringValue)
				}
			}
		}
	}
}

// preCheck does the following:
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
func (s *SQLService) preCheck(ctx context.Context, instanceName, connectionDatabase, statement string, limit int32, isAdmin, isExport bool) (*store.UserMessage, *store.InstanceMessage, *store.DatabaseMessage, advisor.Status, []*v1pb.Advice, *base.SensitiveSchemaInfo, error) {
	// Prepare related message.
	user, environment, instance, maybeDatabase, err := s.prepareRelatedMessage(ctx, instanceName, connectionDatabase)
	if err != nil {
		return nil, nil, nil, advisor.Success, nil, nil, err
	}

	// Validate the request.
	if err := validateQueryRequest(instance, connectionDatabase, statement); err != nil {
		return nil, nil, nil, advisor.Success, nil, nil, err
	}

	// dataShare must be false when connecting to instance (not database) in sql editor
	// dataShare must be false when engine is not redshift
	// engine must be MYSQL or TIDB when connecting to instance (not database) in sql editor
	dataShare := false
	if maybeDatabase != nil {
		dataShare = maybeDatabase.DataShare
	}

	if s.licenseService.IsFeatureEnabled(api.FeatureAccessControl) == nil {
		// Check if the caller is admin for exporting with admin mode.
		if isAdmin && (user.Role != api.Owner && user.Role != api.DBA) {
			return nil, nil, nil, advisor.Success, nil, nil, status.Errorf(codes.PermissionDenied, "only workspace owner and DBA can export data using admin mode")
		}

		// Check if the environment is open for query privileges.
		result, err := s.checkWorkspaceIAMPolicy(ctx, environment, isExport)
		if err != nil {
			return nil, nil, nil, advisor.Success, nil, nil, err
		}
		if !result {
			// Check if the user has permission to execute the query.
			if err := s.checkQueryRights(ctx, connectionDatabase, dataShare, statement, limit, user, instance, isExport); err != nil {
				return nil, nil, nil, advisor.Success, nil, nil, err
			}
		}
	}

	// Run SQL review.
	adviceStatus, adviceList, err := s.sqlReviewCheck(ctx, statement, environment, instance, maybeDatabase)
	if err != nil {
		return nil, nil, nil, adviceStatus, adviceList, nil, err
	}

	// Get sensitive schema info.
	maskingType := storepb.MaskingExceptionPolicy_MaskingException_QUERY
	if isExport {
		maskingType = storepb.MaskingExceptionPolicy_MaskingException_EXPORT
	}
	var sensitiveSchemaInfo *base.SensitiveSchemaInfo
	if adviceStatus != advisor.Error {
		databaseMap := make(map[string]bool)
		switch instance.Engine {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			databaseMap[connectionDatabase] = true
			resources, err := base.ExtractResourceList(instance.Engine, connectionDatabase, "", statement)
			if err != nil {
				return nil, nil, nil, advisor.Success, nil, nil, status.Errorf(codes.Internal, "Failed to get resource list: %s with error %v", statement, err)
			}
			for _, resource := range resources {
				databaseMap[resource.Database] = true
			}
		case storepb.Engine_POSTGRES, storepb.Engine_REDSHIFT, storepb.Engine_RISINGWAVE:
			if !allPostgresSystemObjects(statement) {
				databaseMap[connectionDatabase] = true
			}
		case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE, storepb.Engine_DM:
			databaseMap[connectionDatabase] = true
			if instance.Options != nil && instance.Options.SchemaTenantMode {
				resources, err := base.ExtractResourceList(storepb.Engine_ORACLE, connectionDatabase, connectionDatabase, statement)
				if err != nil {
					return nil, nil, nil, advisor.Success, nil, nil, status.Errorf(codes.Internal, "Failed to get resource list: %s", statement)
				}
				for _, resource := range resources {
					databaseMap[resource.Database] = true
				}
			}
		case storepb.Engine_SNOWFLAKE:
			resources, err := base.ExtractResourceList(storepb.Engine_SNOWFLAKE, connectionDatabase, "schema_placeholder", statement)
			if err != nil {
				return nil, nil, nil, advisor.Success, nil, nil, status.Errorf(codes.Internal, "Failed to get resource list: %s with error %v", statement, err)
			}
			for _, resource := range resources {
				databaseMap[resource.Database] = true
			}
		case storepb.Engine_MSSQL:
			resources, err := base.ExtractResourceList(storepb.Engine_MSSQL, connectionDatabase, "dbo", statement)
			if err != nil {
				return nil, nil, nil, advisor.Success, nil, nil, status.Errorf(codes.Internal, "Failed to get resource list: %s with error %v", statement, err)
			}
			for _, resource := range resources {
				databaseMap[resource.Database] = true
			}
		}
		var databases []string
		for k := range databaseMap {
			databases = append(databases, k)
		}
		sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, databases, connectionDatabase, maskingType)
		if err != nil {
			return nil, nil, nil, advisor.Success, nil, nil, status.Errorf(codes.Internal, "Failed to get sensitive schema info for statement: %s, error: %v", statement, err.Error())
		}
	}

	return user, instance, maybeDatabase, adviceStatus, adviceList, sensitiveSchemaInfo, nil
}

func allPostgresSystemObjects(statement string) bool {
	// We need to distinguish between specified public schema and by default.
	resources, err := base.ExtractResourceList(storepb.Engine_POSTGRES, "", "", statement)
	if err != nil {
		slog.Debug("Failed to extract resource list from statement", slog.String("statement", statement), log.BBError(err))
		return false
	}
	for _, resource := range resources {
		if pgparser.IsSystemSchema(resource.Schema) {
			continue
		}
		// If schema is not specified, user can access the pg_catalog schema if the table is pg_catalog's system table.
		// So we need to check this case.
		if resource.Schema == "" && pgparser.IsSystemTable(resource.Table) {
			continue
		}
		return false
	}
	return true
}

func (s *SQLService) createQueryActivity(ctx context.Context, user *store.UserMessage, level api.ActivityLevel, containerID int, payload api.ActivitySQLEditorQueryPayload) (*store.ActivityMessage, error) {
	// TODO: use v1 activity API instead of
	activityBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("Failed to marshal activity before executing sql statement",
			slog.String("database_name", payload.DatabaseName),
			slog.Int("instance_id", payload.InstanceID),
			slog.String("statement", payload.Statement),
			log.BBError(err))
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

func (s *SQLService) getSensitiveSchemaInfo(ctx context.Context, instance *store.InstanceMessage, databaseList []string, currentDatabase string, action storepb.MaskingExceptionPolicy_MaskingException_Action) (*base.SensitiveSchemaInfo, error) {
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	currentPrincipal, err := s.store.GetUser(ctx, &store.FindUserMessage{
		ID: &principalID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find current principal")
	}

	isEmpty := true
	result := &base.SensitiveSchemaInfo{
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		DatabaseList:        []base.DatabaseSchema{},
	}

	classificationSetting, err := s.store.GetDataClassificationSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find classification setting")
	}

	maskingRulePolicy, err := s.store.GetMaskingRulePolicy(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find masking rule policy")
	}

	algorithmSetting, err := s.store.GetMaskingAlgorithmSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find masking algorithm setting")
	}

	semanticTypesSetting, err := s.store.GetSemanticTypesSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find semantic types setting")
	}

	// Multiple databases may belong to the same project, to reduce the protojson unmarshal cost,
	// we store the projectResourceID - maskingExceptionPolicy in a map.
	maskingExceptionPolicyMap := make(map[string]*storepb.MaskingExceptionPolicy)

	m := newEmptyMaskingLevelEvaluator().
		withMaskingRulePolicy(maskingRulePolicy).
		withDataClassificationSetting(classificationSetting).
		withMaskingAlgorithmSetting(algorithmSetting).
		withSemanticTypeSetting(semanticTypesSetting)

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

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instance.ResourceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			return nil, err
		}
		if database == nil {
			return nil, errors.Errorf("database %q not found", databaseName)
		}

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID: &database.ProjectID,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find project %q", database.ProjectID)
		}
		if project == nil {
			return nil, status.Errorf(codes.Internal, "project of database %q should not be nil", database.DatabaseName)
		}

		var maskingExceptionPolicy *storepb.MaskingExceptionPolicy
		// If we cannot find the maskingExceptionPolicy before, we need to find it from the database and record it in cache.
		if _, ok := maskingExceptionPolicyMap[database.ProjectID]; !ok {
			policy, err := s.store.GetMaskingExceptionPolicyByProjectUID(ctx, project.UID)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find masking exception policy for project %q", project.ResourceID)
			}
			// It is safe if policy is nil.
			maskingExceptionPolicyMap[database.ProjectID] = policy
		}
		maskingExceptionPolicy = maskingExceptionPolicyMap[database.ProjectID]

		// Build the filtered maskingExceptionPolicy for current principal.
		var maskingExceptionContainsCurrentPrincipal []*storepb.MaskingExceptionPolicy_MaskingException
		if maskingExceptionPolicy != nil {
			slog.Debug("found masking exception policy for project", slog.String("database", databaseName), slog.String("project", database.ProjectID), slog.Any("masking exception policy", maskingExceptionPolicy))
			for _, maskingException := range maskingExceptionPolicy.MaskingExceptions {
				if maskingException.Action != action {
					continue
				}
				if maskingException.Member == currentPrincipal.Email {
					slog.Debug("hit masking exception for current principal", slog.String("database", databaseName), slog.String("project", database.ProjectID), slog.Any("masking exception", maskingException))
					maskingExceptionContainsCurrentPrincipal = append(maskingExceptionContainsCurrentPrincipal, maskingException)
				}
			}
		}

		// Filtered the current project's data classification config.
		var dataClassificationConfig *storepb.DataClassificationSetting_DataClassificationConfig
		if project.DataClassificationConfigID != "" {
			for _, dataClassificationSetting := range classificationSetting.Configs {
				if dataClassificationSetting.Id == project.DataClassificationConfigID {
					dataClassificationConfig = dataClassificationSetting
					slog.Debug("found data classification config for project", slog.String("project", project.ResourceID), slog.Any("data classification config id", dataClassificationConfig.Id))
					break
				}
			}
		}

		// Convert the maskingPolicy to a map to reduce the time complexity of searching.
		maskingPolicy, err := s.store.GetMaskingPolicyByDatabaseUID(ctx, database.UID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find masking policy for database %q", databaseName)
		}
		maskingPolicyMap := make(map[maskingPolicyKey]*storepb.MaskData)
		if maskingPolicy != nil {
			for _, maskData := range maskingPolicy.MaskData {
				maskingPolicyMap[maskingPolicyKey{
					schema: maskData.Schema,
					table:  maskData.Table,
					column: maskData.Column,
				}] = maskData
			}
		}
		slog.Debug("found masking policy for database", slog.String("database", databaseName), slog.Any("masking policy", maskingPolicy))

		dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to find schema for database %q in instance %q: %v", databaseName, instance.Title, err)
		}

		if instance.Engine == storepb.Engine_ORACLE || instance.Engine == storepb.Engine_DM || instance.Engine == storepb.Engine_OCEANBASE_ORACLE {
			for _, schema := range dbSchema.GetMetadata().Schemas {
				var schemaConfig *storepb.SchemaConfig
				if dbSchema != nil && dbSchema.GetConfig() != nil {
					for _, c := range dbSchema.GetConfig().SchemaConfigs {
						if c != nil && c.Name == schema.Name {
							schemaConfig = c
							break
						}
					}
				}

				databaseSchema := base.DatabaseSchema{
					Name: schema.Name,
				}
				schemaSchema := base.SchemaSchema{
					Name:      schema.Name,
					TableList: []base.TableSchema{},
				}
				for _, table := range schema.Tables {
					var tableConfig *storepb.TableConfig
					if schemaConfig != nil {
						for _, c := range schemaConfig.TableConfigs {
							if c != nil && c.Name == table.Name {
								tableConfig = c
								break
							}
						}
					}

					tableSchema := base.TableSchema{
						Name:       table.Name,
						ColumnList: []base.ColumnInfo{},
					}
					for _, column := range table.Columns {
						var columnConfig *storepb.ColumnConfig
						if tableConfig != nil {
							for _, c := range tableConfig.ColumnConfigs {
								if c != nil && c.Name == column.Name {
									columnConfig = c
									break
								}
							}
						}
						columnSemanticTypeID := ""
						if columnConfig != nil {
							columnSemanticTypeID = columnConfig.SemanticTypeId
						}

						slog.Debug("processing sensitive schema info", slog.String("schema", schema.Name), slog.String("table", table.Name))
						maskingAlgorithm, maskingLevel, err := m.evaluateMaskingAlgorithmOfColumn(database, schema.Name, table.Name, column.Name, columnSemanticTypeID, column.Classification, project.DataClassificationConfigID, maskingPolicyMap, maskingExceptionContainsCurrentPrincipal)
						if err != nil {
							return nil, errors.Wrapf(err, "failed to evaluate masking level of database %q, schema %q, table %q, column %q", databaseName, schema.Name, table.Name, column.Name)
						}
						sensitive := maskingLevel == storepb.MaskingLevel_FULL || maskingLevel == storepb.MaskingLevel_PARTIAL
						if sensitive {
							isEmpty = false
						}
						masker := getMaskerByMaskingAlgorithmAndLevel(maskingAlgorithm, maskingLevel)
						tableSchema.ColumnList = append(tableSchema.ColumnList, base.ColumnInfo{
							Name:              column.Name,
							MaskingAttributes: base.NewMaskingAttributes(masker),
						})
					}
					schemaSchema.TableList = append(schemaSchema.TableList, tableSchema)
				}
				databaseSchema.SchemaList = append(databaseSchema.SchemaList, schemaSchema)
				result.DatabaseList = append(result.DatabaseList, databaseSchema)
			}
			continue
		}

		databaseSchema := base.DatabaseSchema{
			Name:       databaseName,
			SchemaList: []base.SchemaSchema{},
		}
		for _, schema := range dbSchema.GetMetadata().Schemas {
			var schemaConfig *storepb.SchemaConfig
			if dbSchema != nil && dbSchema.GetConfig() != nil {
				for _, c := range dbSchema.GetConfig().SchemaConfigs {
					if c != nil && c.Name == schema.Name {
						schemaConfig = c
						break
					}
				}
			}
			schemaSchema := base.SchemaSchema{
				Name:      schema.Name,
				TableList: []base.TableSchema{},
			}
			for _, table := range schema.Tables {
				var tableConfig *storepb.TableConfig
				if schemaConfig != nil {
					for _, c := range schemaConfig.TableConfigs {
						if c != nil && c.Name == table.Name {
							tableConfig = c
							break
						}
					}
				}
				tableSchema := base.TableSchema{
					Name:       table.Name,
					ColumnList: []base.ColumnInfo{},
				}
				for _, column := range table.Columns {
					var columnConfig *storepb.ColumnConfig
					if tableConfig != nil {
						for _, c := range tableConfig.ColumnConfigs {
							if c != nil && c.Name == column.Name {
								columnConfig = c
							}
						}
					}
					columnSemanticTypeID := ""
					if columnConfig != nil {
						columnSemanticTypeID = columnConfig.SemanticTypeId
					}

					slog.Debug("processing sensitive schema info", slog.String("database", database.DatabaseName), slog.String("schema", schema.Name), slog.String("table", table.Name), slog.String("column", column.Name))
					maskingAlgorithm, maskingLevel, err := m.evaluateMaskingAlgorithmOfColumn(database, schema.Name, table.Name, column.Name, columnSemanticTypeID, column.Classification, project.DataClassificationConfigID, maskingPolicyMap, maskingExceptionContainsCurrentPrincipal)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to evaluate masking level of database %q, schema %q, table %q, column %q", databaseName, schema.Name, table.Name, column.Name)
					}
					sensitive := maskingLevel == storepb.MaskingLevel_FULL || maskingLevel == storepb.MaskingLevel_PARTIAL
					if sensitive {
						isEmpty = false
					}
					masker := getMaskerByMaskingAlgorithmAndLevel(maskingAlgorithm, maskingLevel)
					tableSchema.ColumnList = append(tableSchema.ColumnList, base.ColumnInfo{
						Name:              column.Name,
						MaskingAttributes: base.NewMaskingAttributes(masker),
					})
				}
				schemaSchema.TableList = append(schemaSchema.TableList, tableSchema)
			}
			for _, view := range schema.Views {
				viewSchema := base.ViewSchema{
					Name:       view.Name,
					Definition: view.Definition,
				}
				schemaSchema.ViewList = append(schemaSchema.ViewList, viewSchema)
			}
			databaseSchema.SchemaList = append(databaseSchema.SchemaList, schemaSchema)
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

func getMaskerByMaskingAlgorithmAndLevel(algorithm *storepb.MaskingAlgorithmSetting_Algorithm, level storepb.MaskingLevel) masker.Masker {
	if algorithm == nil {
		switch level {
		case storepb.MaskingLevel_FULL:
			return masker.NewDefaultFullMasker()
		case storepb.MaskingLevel_PARTIAL:
			return masker.NewDefaultRangeMasker()
		default:
			return masker.NewNoneMasker()
		}
	}

	switch m := algorithm.Mask.(type) {
	case *storepb.MaskingAlgorithmSetting_Algorithm_FullMask_:
		return masker.NewFullMasker(m.FullMask.Substitution)
	case *storepb.MaskingAlgorithmSetting_Algorithm_RangeMask_:
		return masker.NewRangeMasker(convertRangeMaskSlices(m.RangeMask.Slices))
	case *storepb.MaskingAlgorithmSetting_Algorithm_Md5Mask:
		return masker.NewMD5Masker(m.Md5Mask.Salt)
	}
	return masker.NewNoneMasker()
}

func convertRangeMaskSlices(slices []*storepb.MaskingAlgorithmSetting_Algorithm_RangeMask_Slice) []*masker.MaskRangeSlice {
	var result []*masker.MaskRangeSlice
	for _, slice := range slices {
		result = append(result, &masker.MaskRangeSlice{
			Start:        slice.Start,
			End:          slice.End,
			Substitution: slice.Substitution,
		})
	}
	return result
}

func isExcludeDatabase(dbType storepb.Engine, database string) bool {
	switch dbType {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB:
		return isMySQLExcludeDatabase(database)
	case storepb.Engine_TIDB:
		if isMySQLExcludeDatabase(database) {
			return true
		}
		return database == "metrics_schema"
	case storepb.Engine_SNOWFLAKE:
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

func (s *SQLService) sqlReviewCheck(ctx context.Context, statement string, environment *store.EnvironmentMessage, instance *store.InstanceMessage, database *store.DatabaseMessage) (advisor.Status, []*v1pb.Advice, error) {
	if !IsSQLReviewSupported(instance.Engine) || database == nil {
		return advisor.Success, nil, nil
	}

	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return advisor.Error, nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
	}
	if dbSchema == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
			return advisor.Error, nil, status.Errorf(codes.Internal, "failed to sync database schema: %v", err)
		}
		dbSchema, err = s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return advisor.Error, nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
		}
		if dbSchema == nil {
			return advisor.Error, nil, status.Errorf(codes.NotFound, "database schema not found: %v", database.UID)
		}
	}

	catalog, err := s.store.NewCatalog(ctx, database.UID, instance.Engine, store.IgnoreDatabaseAndTableCaseSensitive(instance), advisor.SyntaxModeNormal)
	if err != nil {
		return advisor.Error, nil, status.Errorf(codes.Internal, "Failed to create a catalog: %v", err)
	}

	currentSchema := ""
	if instance.Engine == storepb.Engine_ORACLE || instance.Engine == storepb.Engine_DM || instance.Engine == storepb.Engine_OCEANBASE_ORACLE {
		if instance.Options == nil || !instance.Options.SchemaTenantMode {
			currentSchema = getReadOnlyDataSource(instance).Username
		} else {
			currentSchema = database.DatabaseName
		}
	}

	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
	if err != nil {
		return advisor.Error, nil, status.Errorf(codes.Internal, "Failed to get database driver: %v", err)
	}
	defer driver.Close(ctx)
	connection := driver.GetDB()
	adviceLevel, adviceList, err := s.sqlCheck(
		ctx,
		instance.Engine,
		dbSchema.GetMetadata().CharacterSet,
		dbSchema.GetMetadata().Collation,
		environment.UID,
		statement,
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
			Column:  int32(advice.Column),
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
	dbType storepb.Engine,
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

	var database *store.DatabaseMessage
	if databaseName != "" {
		database, err = s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instance.ResourceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			return nil, nil, nil, nil, status.Errorf(codes.Internal, "failed to fetch database: %v", err)
		}
		if database == nil {
			return nil, nil, nil, nil, errors.Errorf("database %q not found", databaseName)
		}
	}

	environmentID := instance.EnvironmentID
	if database != nil {
		environmentID = database.EffectiveEnvironmentID
	}

	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &environmentID})
	if err != nil {
		return nil, nil, nil, nil, status.Errorf(codes.Internal, "failed to fetch environment: %v", err)
	}
	if environment == nil {
		return nil, nil, nil, nil, status.Errorf(codes.NotFound, "environment ID not found: %s", environmentID)
	}

	return user, environment, instance, database, nil
}

// validateQueryRequest validates the query request.
// 1. Check if the instance exists.
// 2. Check connection_database if the instance is postgres.
// 3. Parse statement for Postgres, MySQL, TiDB, Oracle.
// 4. Check if all statements are (EXPLAIN) SELECT statements.
func validateQueryRequest(instance *store.InstanceMessage, databaseName string, statement string) error {
	switch instance.Engine {
	case storepb.Engine_POSTGRES, storepb.Engine_REDSHIFT, storepb.Engine_RISINGWAVE:
		if databaseName == "" {
			return status.Error(codes.InvalidArgument, "connection_database is required for postgres instance")
		}
	case storepb.Engine_ORACLE, storepb.Engine_DM, storepb.Engine_OCEANBASE_ORACLE:
		if instance.Options != nil && instance.Options.SchemaTenantMode && databaseName == "" {
			return status.Error(codes.InvalidArgument, "connection_database is required for oracle schema tenant mode instance")
		}
	case storepb.Engine_MONGODB, storepb.Engine_REDIS:
		// Do nothing.
		return nil
	}

	ok, err := base.ValidateSQLForEditor(instance.Engine, statement)
	if err != nil {
		syntaxErr, ok := err.(*base.SyntaxError)
		if ok {
			querySyntaxError, err := status.New(codes.InvalidArgument, err.Error()).WithDetails(
				&v1pb.PlanCheckRun_Result_SqlReviewReport{
					Line:   int32(syntaxErr.Line),
					Column: int32(syntaxErr.Column),
					Detail: syntaxErr.Message,
				},
			)
			if err != nil {
				return syntaxErr
			}
			return querySyntaxError.Err()
		}
		return err
	}
	if !ok {
		return nonSelectSQLError.Err()
	}
	return nil
}

func (s *SQLService) extractResourceList(ctx context.Context, engine storepb.Engine, databaseName string, statement string, instance *store.InstanceMessage) ([]base.SchemaResource, error) {
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		list, err := base.ExtractResourceList(engine, databaseName, "", statement)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to extract resource list: %s", err.Error())
		} else if databaseName == "" {
			return list, nil
		}

		databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
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

		dbSchema, err := s.store.GetDBSchema(ctx, databaseMessage.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
		}

		var result []base.SchemaResource
		for _, resource := range list {
			if resource.Database != dbSchema.GetMetadata().Name {
				// MySQL allows cross-database query, we should check the corresponding database.
				resourceDB, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
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
				resourceDBSchema, err := s.store.GetDBSchema(ctx, resourceDB.UID)
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

		databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
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

		dbSchema, err := s.store.GetDBSchema(ctx, databaseMessage.UID)
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
		list, err := base.ExtractResourceList(engine, databaseName, currentSchema, statement)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to extract resource list: %s", err.Error())
		}

		databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
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

		dbSchema, err := s.store.GetDBSchema(ctx, databaseMessage.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
		}

		var result []base.SchemaResource
		for _, resource := range list {
			if resource.Database != dbSchema.GetMetadata().Name {
				if instance.Options == nil || !instance.Options.SchemaTenantMode {
					continue
				}
				// Schema tenant mode allows cross-database query, we should check the corresponding database.
				resourceDB, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
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
				resourceDBSchema, err := s.store.GetDBSchema(ctx, resourceDB.UID)
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
		databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
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

		dbSchema, err := s.store.GetDBSchema(ctx, databaseMessage.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch database schema: %v", err)
		}

		var result []base.SchemaResource
		for _, resource := range list {
			if resource.Database != dbSchema.GetMetadata().Name {
				// Snowflake allows cross-database query, we should check the corresponding database.
				resourceDB, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
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
				resourceDBSchema, err := s.store.GetDBSchema(ctx, resourceDB.UID)
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
		databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
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

		dbSchema, err := s.store.GetDBSchema(ctx, databaseMessage.UID)
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
				resourceDB, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
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
				resourceDBSchema, err := s.store.GetDBSchema(ctx, resourceDB.UID)
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

func (s *SQLService) checkWorkspaceIAMPolicy(
	ctx context.Context,
	environment *store.EnvironmentMessage,
	isExport bool,
) (bool, error) {
	role := common.ProjectQuerier
	if isExport {
		role = common.ProjectExporter
	}

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
		"resource.environment_name": fmt.Sprintf("%s%s", common.EnvironmentNamePrefix, environment.ResourceID),
	}
	formattedRole := fmt.Sprintf("roles/%s", role)
	bindings := v1pbPolicy.GetWorkspaceIamPolicy().Bindings
	for _, binding := range bindings {
		if binding.Role != formattedRole {
			continue
		}

		ok, err := evaluateQueryExportPolicyCondition(binding.Condition.Expression, attributes)
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
	isExport bool,
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
	resourceList, err := s.extractResourceList(ctx, instance.Engine, databaseName, extractingStatement, instance)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to extract resource list: %v", err)
	}

	databaseMap := make(map[string]bool)
	for _, resource := range resourceList {
		if resource.LinkedServer != "" && instance.Engine == storepb.Engine_MSSQL {
			continue
		}
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

		ok, err := hasDatabaseAccessRights(user.ID, projectPolicy, attributes, isExport)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check access control for database: %q", resource.Database)
		}
		if !ok {
			return status.Errorf(codes.PermissionDenied, "permission denied to access resource: %q", resource.Pretty())
		}
	}

	return nil
}

func hasDatabaseAccessRights(principalID int, projectPolicy *store.IAMPolicyMessage, attributes map[string]any, isExport bool) (bool, error) {
	// TODO(rebelice): implement table-level query permission check and refactor this function.
	// Project IAM policy evaluation.
	pass := false
	for _, binding := range projectPolicy.Bindings {
		// Project owner has all permissions.
		if binding.Role == api.Role(common.ProjectOwner) {
			for _, member := range binding.Members {
				if member.ID == principalID || member.Email == api.AllUsers {
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
			ok, err := evaluateQueryExportPolicyCondition(binding.Condition.Expression, attributes)
			if err != nil {
				slog.Error("failed to evaluate condition", log.BBError(err), slog.String("condition", binding.Condition.Expression))
				break
			}
			if ok {
				pass = true
				break
			}
		}
		if pass {
			break
		}
	}
	return pass, nil
}

func evaluateMaskingExceptionPolicyCondition(expression string, attributes map[string]any) (bool, error) {
	if expression == "" {
		return true, nil
	}
	maskingExceptionPolicyEnv, err := cel.NewEnv(
		cel.Variable("resource", cel.MapType(cel.StringType, cel.AnyType)),
		cel.Variable("request", cel.MapType(cel.StringType, cel.AnyType)),
	)
	if err != nil {
		return false, errors.Wrapf(err, "failed to create CEL environment for masking exception policy")
	}
	ast, issues := maskingExceptionPolicyEnv.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, errors.Wrapf(issues.Err(), "failed to get the ast of CEL program for masking exception policy")
	}
	prg, err := maskingExceptionPolicyEnv.Program(ast)
	if err != nil {
		return false, errors.Wrapf(err, "failed to create CEL program for masking exception policy")
	}
	out, _, err := prg.Eval(attributes)
	if err != nil {
		return false, errors.Wrapf(err, "failed to eval CEL program for masking exception policy")
	}
	val, err := out.ConvertToNative(reflect.TypeOf(false))
	if err != nil {
		return false, errors.Wrap(err, "expect bool result for masking exception policy")
	}
	boolVar, ok := val.(bool)
	if !ok {
		return false, errors.Wrap(err, "expect bool result for masking exception policy")
	}
	return boolVar, nil
}

func evaluateMaskingRulePolicyCondition(expression string, attributes map[string]any) (bool, error) {
	if expression == "" {
		return true, nil
	}
	maskingRulePolicyEnv, err := cel.NewEnv(common.MaskingRulePolicyCELAttributes...)
	if err != nil {
		return false, errors.Wrapf(err, "failed to create CEL environment for masking rule policy")
	}
	ast, issues := maskingRulePolicyEnv.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, errors.Wrapf(issues.Err(), "failed to get the ast of CEL program for masking rule")
	}
	prg, err := maskingRulePolicyEnv.Program(ast)
	if err != nil {
		return false, errors.Wrapf(err, "failed to create CEL program for masking rule")
	}
	out, _, err := prg.Eval(attributes)
	if err != nil {
		return false, errors.Wrapf(err, "failed to eval CEL program for masking rule")
	}
	val, err := out.ConvertToNative(reflect.TypeOf(false))
	if err != nil {
		return false, errors.Wrap(err, "expect bool result for masking rule")
	}
	boolVar, ok := val.(bool)
	if !ok {
		return false, errors.Wrap(err, "expect bool result for masking rule")
	}
	return boolVar, nil
}

func evaluateQueryExportPolicyCondition(expression string, attributes map[string]any) (bool, error) {
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
	databaseMessage, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instance.ResourceID,
		DatabaseName:        &database,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
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
	principalID, ok := principalPtr.(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
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
	instanceID, err := common.GetInstanceID(name)
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

// IsSQLReviewSupported checks the engine type if SQL review supports it.
func IsSQLReviewSupported(dbType storepb.Engine) bool {
	switch dbType {
	case storepb.Engine_POSTGRES, storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE, storepb.Engine_OCEANBASE, storepb.Engine_SNOWFLAKE, storepb.Engine_DM, storepb.Engine_MSSQL:
		return true
	default:
		return false
	}
}

// encodeToBase64String encodes the statement to base64 string.
func encodeToBase64String(statement string) string {
	base64Encoded := base64.StdEncoding.EncodeToString([]byte(statement))
	return base64Encoded
}

// DifferPreview returns the diff preview of the given SQL statement and metadata.
func (*SQLService) DifferPreview(_ context.Context, request *v1pb.DifferPreviewRequest) (*v1pb.DifferPreviewResponse, error) {
	schema, err := getDesignSchema(request.Engine, request.OldSchema, request.NewMetadata)
	if err != nil {
		return nil, err
	}

	return &v1pb.DifferPreviewResponse{
		Schema: schema,
	}, nil
}
