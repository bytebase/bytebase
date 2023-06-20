package v1

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/google/cel-go/cel"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	tidbast "github.com/pingcap/tidb/parser/ast"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
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
)

// SQLService is the service for SQL.
type SQLService struct {
	v1pb.UnimplementedSQLServiceServer
	store           *store.Store
	schemaSyncer    *schemasync.Syncer
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
}

// NewSQLService creates a SQLService.
func NewSQLService(store *store.Store, schemaSyncer *schemasync.Syncer, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager) *SQLService {
	return &SQLService{
		store:           store,
		schemaSyncer:    schemaSyncer,
		dbFactory:       dbFactory,
		activityManager: activityManager,
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
			return err
		}

		if queryErr != nil {
			return status.Errorf(codes.Internal, "failed to execute statement: %v", queryErr)
		}

		if err := server.Send(&v1pb.AdminExecuteResponse{
			Results: result,
		}); err != nil {
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
	result, err := driver.RunStatement(ctx, conn, request.Statement)
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
	for _, row := range result.Rows {
		for i, value := range row.Values {
			if i != 0 {
				if err := buf.WriteByte(','); err != nil {
					return nil, err
				}
			}
			if _, err := buf.Write(convertValueToBytes(value)); err != nil {
				return nil, err
			}
		}
		if err := buf.WriteByte('\n'); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func convertValueToBytes(value *v1pb.RowValue) []byte {
	switch value.Kind.(type) {
	case *v1pb.RowValue_StringValue:
		var result []byte
		result = append(result, '"')
		result = append(result, []byte(value.GetStringValue())...)
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
		result = append(result, value.GetBytesValue()...)
		result = append(result, '"')
		return result
	case *v1pb.RowValue_NullValue:
		return []byte("")
	case *v1pb.RowValue_ValueValue:
		return convertValueValueToBytes(value.GetValueValue())
	default:
		return []byte("")
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
	return protojson.Marshal(result)
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

	if err := s.checkQueryRights(ctx, request.ConnectionDatabase, request.Statement, request.Limit, user, environment, instance, request.Format); err != nil {
		return nil, nil, nil, nil, err
	}

	// Get sensitive schema info.
	var sensitiveSchemaInfo *db.SensitiveSchemaInfo
	switch instance.Engine {
	case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
		databaseList, err := parser.ExtractDatabaseList(parser.MySQL, request.Statement)
		if err != nil {
			return nil, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get database list: %s", request.Statement)
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

	start := time.Now().UnixNano()
	result, err := driver.QueryConn2(ctx, conn, request.Statement, &db.QueryContext{
		Limit:           int(request.Limit),
		ReadOnly:        true,
		CurrentDatabase: request.ConnectionDatabase,
		// TODO(rebelice): we cannot deal with multi-SensitiveDataMaskType now. Fix it.
		SensitiveDataMaskType: db.SensitiveDataMaskTypeDefault,
		SensitiveSchemaInfo:   sensitiveSchemaInfo,
	})
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

	// Check if the user has permission to execute the query.
	if err := s.checkQueryRights(ctx, request.ConnectionDatabase, request.Statement, request.Limit, user, environment, instance, v1pb.ExportRequest_FORMAT_UNSPECIFIED); err != nil {
		return nil, nil, advisor.Success, nil, nil, nil, err
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
			databaseList, err := parser.ExtractDatabaseList(parser.MySQL, request.Statement)
			if err != nil {
				return nil, nil, advisor.Success, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get database list: %s", request.Statement)
			}

			sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, databaseList, request.ConnectionDatabase)
			if err != nil {
				return nil, nil, advisor.Success, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get sensitive schema info: %s", request.Statement)
			}
		case db.Postgres:
			sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, []string{request.ConnectionDatabase}, request.ConnectionDatabase)
			if err != nil {
				return nil, nil, advisor.Success, nil, nil, nil, status.Errorf(codes.Internal, "Failed to get sensitive schema info: %s", request.Statement)
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

		columnMap := make(sensitiveDataMap)
		policy, err := s.store.GetSensitiveDataPolicy(ctx, database.UID)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find sensitive data policy for database %q in instance %q", databaseName, instance.Title))
		}
		if len(policy.SensitiveDataList) == 0 {
			// If there is no sensitive data policy, return nil to skip mask sensitive data.
			return nil, nil
		}

		for _, data := range policy.SensitiveDataList {
			columnMap[api.SensitiveData{
				Schema: data.Schema,
				Table:  data.Table,
				Column: data.Column,
			}] = data.Type
		}

		dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find table list for database %q", databaseName))
		}

		databaseSchema := db.DatabaseSchema{
			Name:      databaseName,
			TableList: []db.TableSchema{},
		}
		for _, schema := range dbSchema.Metadata.Schemas {
			for _, table := range schema.Tables {
				tableSchema := db.TableSchema{
					Name:       table.Name,
					ColumnList: []db.ColumnInfo{},
				}
				if instance.Engine == db.Postgres {
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
				databaseSchema.TableList = append(databaseSchema.TableList, tableSchema)
			}
		}
		if len(databaseSchema.TableList) > 0 {
			isEmpty = false
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
		Charset:   dbCharacterSet,
		Collation: dbCollation,
		DbType:    dbType,
		Catalog:   catalog,
		Driver:    driver,
		Context:   ctx,
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
		tree, _, err := parser.ParseMySQL(statement)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "failed to parse query: %s", err.Error())
		}
		if err := parser.MySQLValidateForEditor(tree); err != nil {
			return status.Errorf(codes.InvalidArgument, err.Error())
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
	case db.Oracle:
		tree, err := parser.ParsePLSQL(statement)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "failed to parse query: %s", err.Error())
		}
		if err := parser.PLSQLValidateForEditor(tree); err != nil {
			return status.Errorf(codes.InvalidArgument, err.Error())
		}
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
				// Should not happen.
				continue
			}

			if !dbSchema.TableExists(resource.Schema, resource.Table) {
				// If table not found, skip.
				continue
			}

			result = append(result, resource)
		}

		return result, nil
	case parser.Postgres:
		list, err := parser.ExtractResourceList(engine, databaseName, "public", statement)
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
				// Should not happen.
				continue
			}

			if !dbSchema.TableExists(resource.Schema, resource.Table) {
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
				// Should not happen.
				continue
			}

			if !dbSchema.TableExists(resource.Schema, resource.Table) {
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

func (s *SQLService) checkQueryRights(
	ctx context.Context,
	databaseName string,
	statement string,
	limit int32,
	user *store.UserMessage,
	environment *store.EnvironmentMessage,
	instance *store.InstanceMessage,
	exportFormat v1pb.ExportRequest_Format,
) error {
	// Owner and DBA have all rights.
	if user.Role == api.Owner || user.Role == api.DBA {
		return nil
	}

	resourceList, err := s.extractResourceList(ctx, convertToParserEngine(instance.Engine), databaseName, statement, instance)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to extract resource list: %v", err)
	}

	databaseMap := make(map[string]bool)
	for _, resource := range resourceList {
		databaseMap[resource.Database] = true
	}

	var project *store.ProjectMessage
	// var databaseMessages []*store.DatabaseMessage
	databaseMessageMap := make(map[string]*store.DatabaseMessage)
	for database := range databaseMap {
		projectMessage, databaseMessage, err := s.getProjectAndDatabaseMessage(ctx, instance, database)
		if err != nil {
			return err
		}
		if project == nil {
			project = projectMessage
		}
		if project.UID != projectMessage.UID {
			return status.Errorf(codes.InvalidArgument, "allow querying databases within the same project only")
		}
		databaseMessageMap[database] = databaseMessage
	}

	if project == nil {
		return status.Error(codes.NotFound, "project not found")
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
			"request.time":         time.Now(),
			"resource.environment": instance.EnvironmentID,
			"resource.database":    databaseResourceURL,
			"resource.schema":      resource.Schema,
			"resource.table":       resource.Table,
			"request.statement":    base64.StdEncoding.EncodeToString([]byte(statement)),
			"request.row_limit":    limit,
		}

		switch exportFormat {
		case v1pb.ExportRequest_FORMAT_UNSPECIFIED:
			attributes["request.export_format"] = "QUERY"
		case v1pb.ExportRequest_CSV:
			attributes["request.export_format"] = "CSV"
		case v1pb.ExportRequest_JSON:
			attributes["request.export_format"] = "JSON"
		default:
			return status.Errorf(codes.InvalidArgument, "invalid export format: %v", exportFormat)
		}

		ok, expression, err := s.hasDatabaseAccessRights(ctx, user.ID, projectPolicy, databaseMessageMap[resource.Database], environment, attributes, isExport)
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

func (s *SQLService) hasDatabaseAccessRights(ctx context.Context, principalID int, projectPolicy *store.IAMPolicyMessage, database *store.DatabaseMessage, environment *store.EnvironmentMessage, attributes map[string]any, isExport bool) (bool, string, error) {
	// TODO(rebelice): implement table-level query permission check and refactor this function.
	// Project IAM policy evaluation.
	pass := false
	var usedExpression string
	for _, binding := range projectPolicy.Bindings {
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
	if !pass {
		return false, "", nil
	}
	// calculate the effective policy.
	databasePolicy, inheritFromEnvironment, err := s.store.GetAccessControlPolicy(ctx, api.PolicyResourceTypeDatabase, database.UID)
	if err != nil {
		return false, "", err
	}

	environmentPolicy, _, err := s.store.GetAccessControlPolicy(ctx, api.PolicyResourceTypeEnvironment, environment.UID)
	if err != nil {
		return false, "", err
	}

	if !inheritFromEnvironment {
		// Use database policy.
		return databasePolicy != nil && len(databasePolicy.DisallowRuleList) == 0, "", nil
	}
	// Use both database policy and environment policy.
	hasAccessRights := true
	if environmentPolicy != nil {
		// Disallow by environment access policy.
		for _, rule := range environmentPolicy.DisallowRuleList {
			if rule.FullDatabase {
				hasAccessRights = false
				break
			}
		}
	}
	if databasePolicy != nil {
		// Allow by database access policy.
		hasAccessRights = true
	}
	return hasAccessRights, usedExpression, nil
}

func evaluateCondition(expression string, attributes map[string]any) (bool, error) {
	if expression == "" {
		return true, nil
	}
	env, err := cel.NewEnv(iamPolicyCELAttributes...)
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
		if httpErr, ok := err.(*echo.HTTPError); ok && httpErr.Code == echo.ErrNotFound.Code {
			// If database not found, skip.
			return nil, nil, nil
		}
		return nil, nil, err
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
	}
	return parser.Standard
}

// IsSQLReviewSupported checks the engine type if SQL review supports it.
func IsSQLReviewSupported(dbType db.Type) bool {
	switch dbType {
	case db.Postgres, db.MySQL, db.TiDB, db.MariaDB, db.Oracle, db.OceanBase, db.Snowflake:
		advisorDB, err := advisorDB.ConvertToAdvisorDBType(string(dbType))
		if err != nil {
			return false
		}

		return advisor.IsSQLReviewSupported(advisorDB)
	default:
		return false
	}
}
