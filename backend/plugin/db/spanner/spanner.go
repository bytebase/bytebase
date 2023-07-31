// Package spanner is the plugin for Spanner driver.
package spanner

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	spanner "cloud.google.com/go/spanner"
	spannerdb "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"

	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	excludedDatabaseList = map[string]bool{
		"bytebase": true,
	}

	dsnRegExp = regexp.MustCompile("projects/(?P<PROJECTGROUP>([a-z]|[-.:]|[0-9])+)/instances/(?P<INSTANCEGROUP>([a-z]|[-]|[0-9])+)/databases/(?P<DATABASEGROUP>([a-z]|[-]|[_]|[0-9])+)")

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Spanner, newDriver)
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

func newDriver(_ db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Spanner driver. It must connect to a specific database.
// If database isn't provided, part of the driver cannot function.
func (d *Driver) Open(ctx context.Context, _ db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	if config.Host == "" {
		return nil, errors.New("host cannot be empty")
	}
	d.config = config
	d.connCtx = connCtx
	if config.Database != "" {
		d.databaseName = d.config.Database
		dsn := getDSN(d.config.Host, d.config.Database)
		client, err := spanner.NewClient(ctx, dsn, option.WithCredentialsJSON([]byte(config.Password)))
		if err != nil {
			return nil, err
		}
		d.client = client
	}

	dbClient, err := spannerdb.NewDatabaseAdminClient(ctx, option.WithCredentialsJSON([]byte(config.Password)))
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
		Parent: d.config.Host,
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

// GetType returns the database type.
func (*Driver) GetType() db.Type {
	return db.Spanner
}

// GetDB gets the database.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute executes a SQL statement.
func (d *Driver) Execute(ctx context.Context, statement string, createDatabase bool, _ db.ExecuteOptions) (int64, error) {
	if createDatabase {
		stmts, err := sanitizeSQL(statement)
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

	var rowCount int64
	stmts, err := sanitizeSQL(statement)
	if err != nil {
		return 0, err
	}

	ddl := func() bool {
		for _, stmt := range stmts {
			if isDDL(stmt) {
				return true
			}
		}
		return false
	}()

	if ddl {
		op, err := d.dbClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database:   getDSN(d.config.Host, d.databaseName),
			Statements: stmts,
		})
		if err != nil {
			return 0, err
		}
		return 0, op.Wait(ctx)
	}

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
		return nil
	}); err != nil {
		return 0, err
	}
	return rowCount, nil
}

func (d *Driver) creataDatabase(ctx context.Context, createStatement string, extraStatement []string) error {
	op, err := d.dbClient.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          d.config.Host,
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
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	stmts, err := sanitizeSQL(statement)
	if err != nil {
		return nil, err
	}

	var results []*v1pb.QueryResult
	for _, statement := range stmts {
		statement = getStatementWithResultLimit(statement, queryContext.Limit)
		result, err := d.querySingleSQL(ctx, statement)
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

func (d *Driver) querySingleSQL(ctx context.Context, statement string) (*v1pb.QueryResult, error) {
	startTime := time.Now()
	iter := d.client.Single().Query(ctx, spanner.NewStatement(statement))
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	data := []*v1pb.QueryRow{}
	columnNames := getColumnNames(iter)
	columnTypeNames, err := getColumnTypeNames(iter)
	if err != nil {
		return nil, err
	}

	for {
		rowData, err := readRow2(row)
		if err != nil {
			return nil, err
		}
		data = append(data, rowData)

		row, err = iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	// spanner doesn't mask the sensitive fields.
	// Return the all false boolean slice here as the placeholder.
	sensitiveInfo := make([]bool, len(columnNames))

	return &v1pb.QueryResult{
		ColumnNames:     columnNames,
		ColumnTypeNames: columnTypeNames,
		Rows:            data,
		Masked:          sensitiveInfo,
		Latency:         durationpb.New(time.Since(startTime)),
		Statement:       statement,
	}, nil
}

func readRow2(row *spanner.Row) (*v1pb.QueryRow, error) {
	result := &v1pb.QueryRow{}
	for i := 0; i < row.Size(); i++ {
		var col spanner.GenericColumnValue
		if err := row.Column(i, &col); err != nil {
			return nil, err
		}
		result.Values = append(result.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_ValueValue{ValueValue: col.Value}})
	}

	return result, nil
}

// RunStatement executes a SQL statement.
func (d *Driver) RunStatement(ctx context.Context, _ *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	stmts, err := sanitizeSQL(statement)
	if err != nil {
		return nil, err
	}

	var results []*v1pb.QueryResult
	for _, statement := range stmts {
		startTime := time.Now()
		if isSelect(statement) {
			result, err := d.querySingleSQL(ctx, statement)
			if err != nil {
				results = append(results, &v1pb.QueryResult{
					Error: err.Error(),
				})
				continue
			}
			results = append(results, result)
			continue
		}

		if isDDL(statement) {
			op, err := d.dbClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   getDSN(d.config.Host, d.databaseName),
				Statements: []string{statement},
			})
			if err != nil {
				results = append(results, &v1pb.QueryResult{
					Error: err.Error(),
				})
				continue
			}
			if err := op.Wait(ctx); err != nil {
				results = append(results, &v1pb.QueryResult{
					Error: err.Error(),
				})
				continue
			}
			results = append(results, &v1pb.QueryResult{})
			continue
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
			results = append(results, &v1pb.QueryResult{
				Error: err.Error(),
			})
			continue
		}

		field := []string{"Affected Rows"}
		types := []string{"INT64"}
		rows := []*v1pb.QueryRow{
			{
				Values: []*v1pb.RowValue{
					{
						Kind: &v1pb.RowValue_Int64Value{Int64Value: rowCount},
					},
				},
			},
		}
		results = append(results, &v1pb.QueryResult{
			ColumnNames:     field,
			ColumnTypeNames: types,
			Rows:            rows,
			Latency:         durationpb.New(time.Since(startTime)),
			Statement:       statement,
		})
	}

	return results, nil
}
