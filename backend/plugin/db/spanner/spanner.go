// Package spanner is the plugin for Spanner driver.
package spanner

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	spannerdb "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/pkg/errors"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

var (
	dsnRegExp = regexp.MustCompile("projects/(?P<PROJECTGROUP>([a-z]|[-.:]|[0-9])+)/instances/(?P<INSTANCEGROUP>([a-z]|[-]|[0-9])+)/databases/(?P<DATABASEGROUP>([a-z]|[-]|[_]|[0-9])+)")

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_SPANNER, newDriver)
}

// Driver is the Spanner driver.
type Driver struct {
	config   db.ConnectionConfig
	connCtx  db.ConnectionContext
	client   *spanner.Client
	dbClient *spannerdb.DatabaseAdminClient

	// databaseName is the currently connected database name.
	databaseName string
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a Spanner driver. It must connect to a specific database.
// If database isn't provided, part of the driver cannot function.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if config.DataSource.Host == "" {
		return nil, errors.New("host cannot be empty")
	}
	d.config = config
	d.connCtx = config.ConnectionContext

	var o []option.ClientOption
	if gcpCredential := config.DataSource.GetGcpCredential(); gcpCredential != nil {
		o = append(o, option.WithCredentialsJSON([]byte(gcpCredential.Content)))
	}
	if config.ConnectionContext.DatabaseName != "" {
		d.databaseName = config.ConnectionContext.DatabaseName
		dsn := getDSN(d.config.DataSource.Host, config.ConnectionContext.DatabaseName)
		client, err := spanner.NewClient(
			ctx,
			dsn,
			o...,
		)
		if err != nil {
			return nil, err
		}
		d.client = client
	}

	dbClient, err := spannerdb.NewDatabaseAdminClient(ctx, o...)
	if err != nil {
		return nil, err
	}

	d.dbClient = dbClient
	return d, nil
}

// Close closes the driver.
func (d *Driver) Close(_ context.Context) error {
	if d.client != nil {
		d.client.Close()
	}
	return d.dbClient.Close()
}

// Ping pings the instance.
func (d *Driver) Ping(ctx context.Context) error {
	iter := d.dbClient.ListDatabases(ctx, &databasepb.ListDatabasesRequest{
		Parent: d.config.DataSource.Host,
	})
	_, err := iter.Next()
	if err == iterator.Done {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "spanner: bad connection")
	}
	return nil
}

// GetDB gets the database.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute executes a SQL statement.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if opts.CreateDatabase {
		stmts, err := util.SanitizeSQL(statement)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to sanitize %v", statement)
		}
		if len(stmts) == 0 {
			return 0, errors.Errorf("expect sanitized SQLs to have at least one entry, original statement: %v", statement)
		}
		if !strings.HasPrefix(stmts[0], "CREATE DATABASE") {
			return 0, errors.Errorf("expect the first entry of the sanitized SQLs to start with 'CREATE DATABASE', sql %v", stmts[0])
		}
		if err := d.creataDatabase(ctx, stmts[0], stmts[1:]); err != nil {
			return 0, errors.Wrap(err, "failed to create database")
		}
		return 0, nil
	}

	// Parse transaction mode from the script
	config, cleanedStatement := base.ParseTransactionConfig(statement)
	statement = cleanedStatement
	transactionMode := config.Mode

	// Apply default when transaction mode is not specified
	if transactionMode == common.TransactionModeUnspecified {
		transactionMode = common.GetDefaultTransactionMode()
	}

	stmts, err := util.SanitizeSQL(statement)
	if err != nil {
		return 0, err
	}

	// Check if any statement is DDL
	ddl := func() bool {
		for _, stmt := range stmts {
			if util.IsDDL(stmt) {
				return true
			}
		}
		return false
	}()

	// Spanner DDL is always non-transactional
	if ddl {
		return d.executeDDL(ctx, stmts, opts)
	}

	// Execute based on transaction mode
	if transactionMode == common.TransactionModeOff {
		return d.executeInAutoCommitMode(ctx, stmts, opts)
	}
	return d.executeInTransactionMode(ctx, stmts, opts)
}

func (d *Driver) executeDDL(ctx context.Context, stmts []string, _ db.ExecuteOptions) (int64, error) {
	op, err := d.dbClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   getDSN(d.config.DataSource.Host, d.databaseName),
		Statements: stmts,
	})
	if err != nil {
		return 0, err
	}
	return 0, op.Wait(ctx)
}

func (d *Driver) executeInTransactionMode(ctx context.Context, stmts []string, opts db.ExecuteOptions) (int64, error) {
	var rowCount int64

	// Log transaction start
	opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, "")

	committed := false
	if _, err := d.client.ReadWriteTransaction(ctx, func(ctx context.Context, rwt *spanner.ReadWriteTransaction) error {
		spannerStmts := []spanner.Statement{}
		for _, stmt := range stmts {
			spannerStmts = append(spannerStmts, spanner.NewStatement(stmt))
		}
		counts, err := rwt.BatchUpdate(ctx, spannerStmts)
		if err != nil {
			return err
		}
		for _, count := range counts {
			rowCount += count
		}
		committed = true
		return nil
	}); err != nil {
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_ROLLBACK, "")
		return 0, err
	}

	if committed {
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, "")
	}
	return rowCount, nil
}

func (d *Driver) executeInAutoCommitMode(ctx context.Context, stmts []string, _ db.ExecuteOptions) (int64, error) {
	var rowCount int64
	// Execute statements individually in auto-commit mode
	for _, stmt := range stmts {
		spannerStmt := spanner.NewStatement(stmt)
		count, err := d.client.PartitionedUpdate(ctx, spannerStmt)
		if err != nil {
			return rowCount, err
		}
		rowCount += count
	}
	return rowCount, nil
}

func (d *Driver) creataDatabase(ctx context.Context, createStatement string, extraStatement []string) error {
	op, err := d.dbClient.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          d.config.DataSource.Host,
		CreateStatement: createStatement,
		ExtraStatements: extraStatement,
	})
	if err != nil {
		return err
	}
	if _, err := op.Wait(ctx); err != nil {
		return err
	}
	return nil
}

func getColumnNames(iter *spanner.RowIterator) []string {
	var names []string
	for _, field := range iter.Metadata.RowType.Fields {
		names = append(names, field.Name)
	}
	return names
}

func getColumnTypeNames(iter *spanner.RowIterator) ([]string, error) {
	var names []string
	for _, field := range iter.Metadata.RowType.Fields {
		typeName, err := getColumnTypeName(field.Type)
		if err != nil {
			return nil, err
		}
		names = append(names, typeName)
	}
	return names, nil
}

func getColumnTypeName(columnType *sppb.Type) (string, error) {
	if columnType.Code == sppb.TypeCode_STRUCT {
		return "", errors.New("spanner STRUCT type is not supported")
	}
	if columnType.Code == sppb.TypeCode_ARRAY {
		if columnType.ArrayElementType.Code == sppb.TypeCode_STRUCT {
			return "", errors.New("spanner STRUCT type is not supported")
		}
		return fmt.Sprintf("[]%s", columnType.ArrayElementType.Code.String()), nil
	}
	return columnType.Code.String(), nil
}

// getStatementWithResultLimit wraps a SQL statement in a CTE to enforce a result limit.
// This is a simple approach that works for SELECT queries but has a critical limitation:
// Spanner does NOT support DML statements (INSERT/UPDATE/DELETE) inside CTEs.
//
// This function should ONLY be called for SELECT statements (verify with util.IsSelect first).
// For a more robust parser-based approach that can handle complex queries, see the
// PostgreSQL/MySQL implementations which parse and inject LIMIT clauses directly.
func getStatementWithResultLimit(stmt string, limit int) string {
	stmt = strings.TrimRightFunc(stmt, utils.IsSpaceOrSemicolon)
	limitPart := fmt.Sprintf(" LIMIT %d", limit)
	return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result%s;", stmt, limitPart)
}

func getDSN(host, database string) string {
	return fmt.Sprintf("%s/databases/%s", host, database)
}

// get `<database>` from `projects/<project>/instances/<instance>/databases/<database>`.
func getDatabaseFromDSN(dsn string) (string, error) {
	match := dsnRegExp.FindStringSubmatch(dsn)
	if match == nil {
		return "", errors.New("invalid DSN")
	}
	matches := make(map[string]string)
	for i, name := range dsnRegExp.SubexpNames() {
		if i != 0 && name != "" {
			matches[name] = match[i]
		}
	}
	return matches["DATABASEGROUP"], nil
}

// QueryConn queries a SQL statement in a given connection.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext.Explain {
		return d.explainStatement(ctx, statement)
	}

	stmts, err := util.SanitizeSQL(statement)
	if err != nil {
		return nil, err
	}

	var results []*v1pb.QueryResult
	for _, statement := range stmts {
		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			if util.IsSelect(statement) {
				// Only apply limit wrapper for SELECT statements
				limitedStatement := statement
				if queryContext.Limit > 0 {
					limitedStatement = getStatementWithResultLimit(statement, queryContext.Limit)
				}
				return d.queryStatement(ctx, limitedStatement, queryContext)
			}
			if util.IsDDL(statement) {
				op, err := d.dbClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
					Database:   getDSN(d.config.DataSource.Host, d.databaseName),
					Statements: []string{statement},
				})
				if err != nil {
					return nil, err
				}
				if err := op.Wait(ctx); err != nil {
					return nil, err
				}
				return &v1pb.QueryResult{}, nil
			}
			var rowCount int64
			if _, err := d.client.ReadWriteTransaction(ctx, func(ctx context.Context, rwt *spanner.ReadWriteTransaction) error {
				count, err := rwt.Update(ctx, spanner.NewStatement(statement))
				if err != nil {
					return err
				}
				rowCount = count
				return nil
			}); err != nil {
				return nil, err
			}
			return util.BuildAffectedRowsResult(rowCount, nil), nil
		}()
		stop := false
		if err != nil {
			queryResult = &v1pb.QueryResult{
				Error: err.Error(),
			}
			stop = true
		}
		queryResult.Statement = statement
		queryResult.Latency = durationpb.New(time.Since(startTime))
		queryResult.RowsCount = int64(len(queryResult.Rows))
		results = append(results, queryResult)
		if stop {
			break
		}
	}

	return results, nil
}

func (d *Driver) queryStatement(ctx context.Context, statement string, queryContext db.QueryContext) (*v1pb.QueryResult, error) {
	iter := d.client.Single().Query(ctx, spanner.NewStatement(statement))
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		// Empty result set - return empty QueryResult with column info
		columnTypeNames, typeErr := getColumnTypeNames(iter)
		if typeErr != nil {
			return nil, typeErr
		}
		return &v1pb.QueryResult{
			ColumnNames:     getColumnNames(iter),
			ColumnTypeNames: columnTypeNames,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	columnTypeNames, err := getColumnTypeNames(iter)
	if err != nil {
		return nil, err
	}
	result := &v1pb.QueryResult{
		ColumnNames:     getColumnNames(iter),
		ColumnTypeNames: columnTypeNames,
	}

	for {
		rowData, err := readRow(row)
		if err != nil {
			return nil, err
		}
		result.Rows = append(result.Rows, rowData)
		n := len(result.Rows)
		if (n&(n-1) == 0) && int64(proto.Size(result)) > queryContext.MaximumSQLResultSize {
			result.Error = common.FormatMaximumSQLResultSizeMessage(queryContext.MaximumSQLResultSize)
			break
		}

		row, err = iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// convertSpannerValue converts google.protobuf.Value to RowValue.
// Uses type metadata to preserve INT64 precision and properly handle all Spanner types.
func convertSpannerValue(colType *sppb.Type, v *structpb.Value) *v1pb.RowValue {
	if v == nil || v.Kind == nil {
		return util.NullRowValue
	}

	switch v.Kind.(type) {
	case *structpb.Value_NullValue:
		return util.NullRowValue
	case *structpb.Value_StringValue:
		stringValue := v.GetStringValue()
		if colType == nil {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: stringValue,
			}}
		}

		switch colType.Code {
		case sppb.TypeCode_INT64:
			// Spanner encodes INT64 as strings to preserve precision
			val, err := strconv.ParseInt(stringValue, 10, 64)
			if err != nil {
				slog.Error("failed to parse INT64 string value", log.BBError(err))
				return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
					StringValue: stringValue,
				}}
			}
			return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{
				Int64Value: val,
			}}
		case sppb.TypeCode_TIMESTAMP:
			// Spanner encodes TIMESTAMP as RFC3339 string with nanosecond precision
			t, err := time.Parse(time.RFC3339Nano, stringValue)
			if err != nil {
				slog.Error("failed to parse TIMESTAMP string value", log.BBError(err))
				return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
					StringValue: stringValue,
				}}
			}
			// Determine accuracy from the string
			accuracy := int32(6) // Default to microsecond precision
			if dotIndex := strings.Index(stringValue, "."); dotIndex >= 0 {
				// Find the end of fractional seconds (before 'Z' or timezone)
				endIndex := strings.IndexAny(stringValue[dotIndex:], "Z+-")
				if endIndex > 0 {
					accuracy = int32(endIndex - 1)
					if accuracy > 9 {
						accuracy = 9 // Cap at nanosecond precision
					}
				}
			}
			return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{
				TimestampValue: &v1pb.RowValue_Timestamp{
					GoogleTimestamp: timestamppb.New(t),
					Accuracy:        accuracy,
				},
			}}
		case sppb.TypeCode_BYTES:
			// Spanner encodes BYTES as base64 string
			bytes, err := base64.StdEncoding.DecodeString(stringValue)
			if err != nil {
				slog.Error("failed to decode BYTES base64 string value", log.BBError(err))
				return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
					StringValue: stringValue,
				}}
			}
			return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{
				BytesValue: bytes,
			}}
		default:
			// DATE, JSON, STRING all stay as StringValue
			return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: stringValue,
			}}
		}
	case *structpb.Value_NumberValue:
		// Check if this is INT64 to preserve precision
		if colType != nil && colType.Code == sppb.TypeCode_INT64 {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{
				Int64Value: int64(v.GetNumberValue()),
			}}
		}
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{
			DoubleValue: v.GetNumberValue(),
		}}
	case *structpb.Value_BoolValue:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{
			BoolValue: v.GetBoolValue(),
		}}
	case *structpb.Value_ListValue, *structpb.Value_StructValue:
		// Flatten complex types to JSON strings (no array_value/struct_value in RowValue proto).
		goValue := v.AsInterface()
		jsonBytes, err := json.Marshal(goValue)
		if err != nil {
			slog.Error("failed to marshal Spanner complex value", log.BBError(err))
			return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
				StringValue: fmt.Sprintf("%v", goValue),
			}}
		}
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{
			StringValue: string(jsonBytes),
		}}
	default:
		return util.NullRowValue
	}
}

func readRow(row *spanner.Row) (*v1pb.QueryRow, error) {
	result := &v1pb.QueryRow{}
	for i := 0; i < row.Size(); i++ {
		var col spanner.GenericColumnValue
		if err := row.Column(i, &col); err != nil {
			return nil, err
		}
		result.Values = append(result.Values, convertSpannerValue(col.Type, col.Value))
	}

	return result, nil
}

// explainStatement returns the query plan for the given statement as JSON.
func (d *Driver) explainStatement(ctx context.Context, statement string) ([]*v1pb.QueryResult, error) {
	if d.client == nil {
		return nil, errors.New("spanner client is not initialized, database name may be missing")
	}

	stmts, err := util.SanitizeSQL(statement)
	if err != nil {
		return nil, err
	}

	var results []*v1pb.QueryResult
	for _, stmt := range stmts {
		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			plan, err := d.client.Single().AnalyzeQuery(ctx, spanner.NewStatement(stmt))
			if err != nil {
				return nil, err
			}

			// Convert the query plan to JSON
			planJSON, err := convertQueryPlanToJSON(plan)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert query plan to JSON")
			}

			return &v1pb.QueryResult{
				ColumnNames:     []string{"QUERY PLAN"},
				ColumnTypeNames: []string{"JSON"},
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{Kind: &v1pb.RowValue_StringValue{StringValue: planJSON}},
						},
					},
				},
			}, nil
		}()
		if err != nil {
			queryResult = &v1pb.QueryResult{
				Error: err.Error(),
			}
		}
		queryResult.Statement = stmt
		queryResult.Latency = durationpb.New(time.Since(startTime))
		queryResult.RowsCount = int64(len(queryResult.Rows))
		results = append(results, queryResult)
	}

	return results, nil
}

// PlanNode represents a node in the Spanner query plan for JSON serialization.
type PlanNode struct {
	Index               int32                `json:"index"`
	Kind                string               `json:"kind"`
	DisplayName         string               `json:"displayName"`
	ChildLinks          []ChildLink          `json:"childLinks,omitempty"`
	ShortRepresentation *ShortRepresentation `json:"shortRepresentation,omitempty"`
	Metadata            map[string]any       `json:"metadata,omitempty"`
	ExecutionStats      map[string]any       `json:"executionStats,omitempty"`
}

// ChildLink represents a child link in the query plan.
type ChildLink struct {
	ChildIndex int32  `json:"childIndex"`
	Type       string `json:"type,omitempty"`
	Variable   string `json:"variable,omitempty"`
}

// ShortRepresentation represents the short representation of a scalar node.
type ShortRepresentation struct {
	Description string           `json:"description"`
	Subqueries  map[string]int32 `json:"subqueries,omitempty"`
}

// QueryPlan represents the full query plan for JSON serialization.
type QueryPlan struct {
	PlanNodes []PlanNode `json:"planNodes"`
}

// convertQueryPlanToJSON converts a Spanner QueryPlan to JSON string.
func convertQueryPlanToJSON(plan *sppb.QueryPlan) (string, error) {
	if plan == nil {
		return "{}", nil
	}

	queryPlan := QueryPlan{
		PlanNodes: make([]PlanNode, 0, len(plan.PlanNodes)),
	}

	for _, node := range plan.PlanNodes {
		planNode := PlanNode{
			Index:       node.Index,
			Kind:        node.Kind.String(),
			DisplayName: node.DisplayName,
		}

		// Convert child links
		if len(node.ChildLinks) > 0 {
			planNode.ChildLinks = make([]ChildLink, 0, len(node.ChildLinks))
			for _, link := range node.ChildLinks {
				planNode.ChildLinks = append(planNode.ChildLinks, ChildLink{
					ChildIndex: link.ChildIndex,
					Type:       link.Type,
					Variable:   link.Variable,
				})
			}
		}

		// Convert short representation
		if node.ShortRepresentation != nil {
			planNode.ShortRepresentation = &ShortRepresentation{
				Description: node.ShortRepresentation.Description,
				Subqueries:  node.ShortRepresentation.Subqueries,
			}
		}

		// Convert metadata
		if node.Metadata != nil {
			planNode.Metadata = node.Metadata.AsMap()
		}

		// Convert execution stats
		if node.ExecutionStats != nil {
			planNode.ExecutionStats = node.ExecutionStats.AsMap()
		}

		queryPlan.PlanNodes = append(queryPlan.PlanNodes, planNode)
	}

	jsonBytes, err := json.Marshal(queryPlan)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}
