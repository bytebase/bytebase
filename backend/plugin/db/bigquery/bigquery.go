// Package bigquery is the plugin for BigQuery driver.
package bigquery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_BIGQUERY, newDriver)
}

// Driver is the BigQuery driver.
type Driver struct {
	config  db.ConnectionConfig
	connCtx db.ConnectionContext
	client  *bigquery.Client

	// databaseName is the currently connected database name.
	databaseName string
}

func newDriver(_ db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a BigQuery driver. It must connect to a specific database.
// If database isn't provided, part of the driver cannot function.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if config.Host == "" {
		return nil, errors.New("host cannot be empty")
	}
	d.config = config
	d.connCtx = config.ConnectionContext
	d.databaseName = config.Database

	var o []option.ClientOption
	if config.AuthenticationType != storepb.DataSourceOptions_GOOGLE_CLOUD_SQL_IAM {
		o = append(o, option.WithCredentialsJSON([]byte(config.Password)))
	}
	client, err := bigquery.NewClient(ctx, config.Host, o...)
	if err != nil {
		return nil, err
	}
	d.client = client
	return d, nil
}

// Close closes the driver.
func (d *Driver) Close(_ context.Context) error {
	return d.client.Close()
}

// Ping pings the instance.
func (d *Driver) Ping(ctx context.Context) error {
	q := d.client.Query("SELECT 1")
	if _, err := q.Read(ctx); err != nil {
		return err
	}
	return nil
}

// GetType returns the database type.
func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_BIGQUERY
}

// GetDB gets the database.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute executes a SQL statement.
func (d *Driver) Execute(ctx context.Context, statement string, _ db.ExecuteOptions) (int64, error) {
	q := d.client.Query(statement)
	q.DefaultDatasetID = d.databaseName
	job, err := q.Run(ctx)
	if err != nil {
		return 0, err
	}
	status, err := job.Wait(ctx)
	if err != nil {
		return 0, err
	}
	if err := status.Err(); err != nil {
		return 0, err
	}
	return 0, nil
}

// QueryConn queries a SQL statement in a given connection.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	stmts, err := util.SanitizeSQL(statement)
	if err != nil {
		return nil, err
	}

	var results []*v1pb.QueryResult
	for _, stmt := range stmts {
		stmt = getStatementWithResultLimit(stmt, queryContext.Limit)
		result, err := d.querySingleSQL(ctx, stmt)
		if err != nil {
			results = append(results, &v1pb.QueryResult{
				Error: err.Error(),
			})
		} else {
			results = append(results, result)
		}
	}

	return results, nil
}

// RunStatement executes a SQL statement.
func (d *Driver) RunStatement(ctx context.Context, _ *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	stmts, err := util.SanitizeSQL(statement)
	if err != nil {
		return nil, err
	}

	var results []*v1pb.QueryResult
	for _, stmt := range stmts {
		startTime := time.Now()
		if util.IsSelect(stmt) {
			result, err := d.querySingleSQL(ctx, stmt)
			if err != nil {
				results = append(results, &v1pb.QueryResult{
					Error: err.Error(),
				})
				continue
			}
			results = append(results, result)
			continue
		}

		q := d.client.Query(stmt)
		q.DefaultDatasetID = d.databaseName
		job, err := q.Run(ctx)
		if err != nil {
			return nil, err
		}
		status, err := job.Wait(ctx)
		if err != nil {
			results = append(results, &v1pb.QueryResult{
				Error: err.Error(),
			})
			continue
		}
		if err := status.Err(); err != nil {
			results = append(results, &v1pb.QueryResult{
				Error: err.Error(),
			})
			continue
		}
		switch r := status.Statistics.Details.(type) {
		case *bigquery.QueryStatistics:
			field := []string{"Affected Rows"}
			types := []string{"INT64"}
			rows := []*v1pb.QueryRow{
				{
					Values: []*v1pb.RowValue{
						{
							Kind: &v1pb.RowValue_Int64Value{Int64Value: r.NumDMLAffectedRows},
						},
					},
				},
			}
			results = append(results, &v1pb.QueryResult{
				ColumnNames:     field,
				ColumnTypeNames: types,
				Rows:            rows,
				Latency:         durationpb.New(time.Since(startTime)),
				Statement:       stmt,
			})
		default:
			results = append(results, &v1pb.QueryResult{})
		}
	}

	return results, nil
}

func (d *Driver) querySingleSQL(ctx context.Context, statement string) (*v1pb.QueryResult, error) {
	startTime := time.Now()
	q := d.client.Query(statement)
	q.DefaultDatasetID = d.databaseName
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}

	result := &v1pb.QueryResult{
		Statement: statement,
	}
	readOnce := false

	var fieldTypes []bigquery.FieldType
	for {
		var values []bigquery.Value
		err := it.Next(&values)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		// Get schema columns.
		if !readOnce {
			readOnce = true
			for _, s := range it.Schema {
				result.ColumnNames = append(result.ColumnNames, s.Name)
				result.ColumnTypeNames = append(result.ColumnTypeNames, string(s.Type))
				fieldTypes = append(fieldTypes, s.Type)
			}
		}

		row := &v1pb.QueryRow{}
		for i, v := range values {
			row.Values = append(row.Values, convertValue(v, fieldTypes[i]))
		}
		result.Rows = append(result.Rows, row)

		if proto.Size(result) > common.MaximumSQLResultSize {
			result.Error = common.MaximumSQLResultSizeExceeded
			result.Latency = durationpb.New(time.Since(startTime))
			return result, nil
		}
	}

	// BigQuery doesn't mask the sensitive fields.
	// Return the all false boolean slice here as the placeholder.
	result.Masked = make([]bool, len(result.ColumnNames))
	result.Latency = durationpb.New(time.Since(startTime))

	return result, nil
}

func getStatementWithResultLimit(stmt string, limit int) string {
	stmt = strings.TrimRight(stmt, " \n\t;")
	if !strings.HasPrefix(stmt, "EXPLAIN") {
		limitPart := ""
		if limit > 0 {
			limitPart = fmt.Sprintf(" LIMIT %d", limit)
		}
		return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result%s;", stmt, limitPart)
	}
	return stmt
}

func convertValue(v bigquery.Value, fieldType bigquery.FieldType) *v1pb.RowValue {
	if v == nil {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	}
	switch fieldType {
	case bigquery.StringFieldType:
		if s, ok := v.(string); ok {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: s}}
		}
	case bigquery.BytesFieldType:
		if bytes, ok := v.([]byte); ok {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: bytes}}
		}
	case bigquery.IntegerFieldType:
		if i, ok := v.(int64); ok {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: i}}
		}
	case bigquery.FloatFieldType:
		if f, ok := v.(float64); ok {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: f}}
		}
	case bigquery.BooleanFieldType:
		if b, ok := v.(bool); ok {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: b}}
		}
	case bigquery.TimestampFieldType:
	case bigquery.RecordFieldType:
	case bigquery.DateFieldType:
	case bigquery.TimeFieldType:
	case bigquery.DateTimeFieldType:
	case bigquery.NumericFieldType:
	case bigquery.GeographyFieldType:
	case bigquery.BigNumericFieldType:
	case bigquery.IntervalFieldType:
	case bigquery.JSONFieldType:
	case bigquery.RangeFieldType:
	}

	// Fall back to string representation.
	s := fmt.Sprintf("%v", v)
	return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: s}}
}
