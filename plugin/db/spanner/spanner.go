// Package spanner is the plugin for Spanner driver.
package spanner

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	spanner "cloud.google.com/go/spanner"
	spannerdb "cloud.google.com/go/spanner/admin/database/apiv1"
	adminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"go.uber.org/zap"

	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"

	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	//go:embed spanner_migration_schema.sql
	migrationSchema string

	excludedDatabaseList = map[string]bool{
		"bytebase": true,
	}

	createBytebaseDatabaseStatement = `CREATE DATABASE bytebase`

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

	// dbName is the currently connected database name.
	dbName string
}

func newDriver(_ db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Spanner driver. It must connect to a specific database.
// If database isn't provided, the driver tries to connect to "bytebase" database.
// If connecting to "bytebase" also fails, part of the driver cannot function.
func (d *Driver) Open(ctx context.Context, _ db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	if config.Host == "" {
		return nil, errors.New("host cannot be empty")
	}
	d.config = config
	d.connCtx = connCtx
	if config.Database == "" {
		// try to connect to bytebase
		d.dbName = db.BytebaseDatabase
		dsn := getDSN(d.config.Host, db.BytebaseDatabase)
		client, err := spanner.NewClient(ctx, dsn, option.WithCredentialsJSON([]byte(config.Password)))
		if status.Code(err) == codes.NotFound {
			log.Debug(`spanner driver: no database provided, try connecting to "bytebase" database which is not found`, zap.Error(err))
		} else if err != nil {
			return nil, err
		} else {
			d.client = client
		}
	} else {
		d.dbName = d.config.Database
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

func (d *Driver) switchDatabase(ctx context.Context, dbName string) error {
	if d.dbName == dbName {
		return nil
	}
	if d.client != nil {
		d.client.Close()
	}
	dsn := getDSN(d.config.Host, dbName)
	client, err := spanner.NewClient(ctx, dsn, option.WithCredentialsJSON([]byte(d.config.Password)))
	if err != nil {
		return err
	}
	d.client = client
	d.dbName = dbName
	return nil
}

// Close closes the driver.
func (d *Driver) Close(_ context.Context) error {
	if d.client != nil {
		d.client.Close()
	}
	return d.dbClient.Close()
}

// Ping pings the database.
func (d *Driver) Ping(ctx context.Context) error {
	iter := d.client.Single().Query(ctx, spanner.NewStatement("SELECT 1"))
	defer iter.Stop()

	var i int64
	row, err := iter.Next()
	if err != nil {
		return err
	}
	if err := row.Column(0, &i); err != nil {
		return err
	}
	if i != 1 {
		return errors.New("expect to get 1")
	}
	if _, err = iter.Next(); err != iterator.Done {
		return errors.New("expect no more rows")
	}
	return nil
}

// GetType returns the database type.
func (*Driver) GetType() db.Type {
	return db.Spanner
}

// GetDBConnection gets a database connection.
func (*Driver) GetDBConnection(_ context.Context, _ string) (*sql.DB, error) {
	panic("not implemented")
}

// Execute executes a SQL statement.
func (d *Driver) Execute(ctx context.Context, statement string, createDatabase bool) (int64, error) {
	if createDatabase {
		return 0, errors.Errorf("cannot set createDatabase to true")
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
		op, err := d.dbClient.UpdateDatabaseDdl(ctx, &adminpb.UpdateDatabaseDdlRequest{
			Database:   getDSN(d.config.Host, d.dbName),
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

// Query queries a SQL statement.
func (d *Driver) Query(ctx context.Context, statement string, queryContext *db.QueryContext) ([]interface{}, error) {
	stmts, err := sanitizeSQL(statement)
	if err != nil {
		return nil, err
	}
	if len(stmts) != 1 {
		return nil, errors.Errorf("expect to get 1 statement, get %d", len(stmts))
	}

	statement = stmts[0]
	if !queryContext.ReadOnly && !isSelect(statement) {
		return d.queryAdmin(ctx, statement)
	}

	statement = getStatementWithResultLimit(statement, queryContext.Limit)
	iter := d.client.Single().Query(ctx, spanner.NewStatement(statement))
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	data := []interface{}{}
	columnNames := getColumnNames(iter)
	columnTypeNames, err := getColumnTypeNames(iter)
	if err != nil {
		return nil, err
	}

	for {
		rowData, err := readRow(row)
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
	return []interface{}{columnNames, columnTypeNames, data}, nil
}

func (d *Driver) queryAdmin(ctx context.Context, statement string) ([]interface{}, error) {
	if isDDL(statement) {
		op, err := d.dbClient.UpdateDatabaseDdl(ctx, &adminpb.UpdateDatabaseDdlRequest{
			Database:   getDSN(d.config.Host, d.dbName),
			Statements: []string{statement},
		})
		if err != nil {
			return nil, err
		}
		return nil, op.Wait(ctx)
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

	field := []string{"Affected Rows"}
	types := []string{"INT64"}
	rows := [][]interface{}{{rowCount}}
	return []interface{}{field, types, rows}, nil
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

func readRow(row *spanner.Row) ([]interface{}, error) {
	dest := make([]interface{}, row.Size())
	for i := 0; i < row.Size(); i++ {
		var col spanner.GenericColumnValue
		if err := row.Column(i, &col); err != nil {
			return nil, err
		}
		switch col.Type.Code {
		case sppb.TypeCode_INT64:
			var v spanner.NullInt64
			if err := col.Decode(&v); err != nil {
				return nil, err
			}
			if v.Valid {
				dest[i] = v.Int64
			} else {
				dest[i] = nil
			}
		case sppb.TypeCode_FLOAT64:
			var v spanner.NullFloat64
			if err := col.Decode(&v); err != nil {
				return nil, err
			}
			if v.Valid {
				dest[i] = v.Float64
			} else {
				dest[i] = nil
			}
		case sppb.TypeCode_NUMERIC:
			var v spanner.NullNumeric
			if err := col.Decode(&v); err != nil {
				return nil, err
			}
			if v.Valid {
				dest[i] = v.Numeric
			} else {
				dest[i] = nil
			}
		case sppb.TypeCode_STRING:
			var v spanner.NullString
			if err := col.Decode(&v); err != nil {
				return nil, err
			}
			if v.Valid {
				dest[i] = v.StringVal
			} else {
				dest[i] = nil
			}
		case sppb.TypeCode_JSON:
			var v spanner.NullJSON
			if err := col.Decode(&v); err != nil {
				return nil, err
			}
			// We always assign `v` to dest[i] here because there is no native type
			// for JSON in the Go sql package. That means that instead of returning
			// nil we should return a NullJSON with valid=false.
			dest[i] = v
		case sppb.TypeCode_BYTES:
			// The column value is a base64 encoded string.
			var v []byte
			if err := col.Decode(&v); err != nil {
				return nil, err
			}
			dest[i] = v
		case sppb.TypeCode_BOOL:
			var v spanner.NullBool
			if err := col.Decode(&v); err != nil {
				return nil, err
			}
			if v.Valid {
				dest[i] = v.Bool
			} else {
				dest[i] = nil
			}
		case sppb.TypeCode_DATE:
			var v spanner.NullDate
			if err := col.Decode(&v); err != nil {
				return nil, err
			}
			if v.Valid {
				dest[i] = v.Date
			} else {
				dest[i] = nil
			}
		case sppb.TypeCode_TIMESTAMP:
			var v spanner.NullTime
			if err := col.Decode(&v); err != nil {
				return nil, err
			}
			if v.Valid {
				dest[i] = v.Time
			} else {
				dest[i] = nil
			}
		case sppb.TypeCode_ARRAY:
			switch col.Type.ArrayElementType.Code {
			case sppb.TypeCode_INT64:
				var v []spanner.NullInt64
				if err := col.Decode(&v); err != nil {
					return nil, err
				}
				dest[i] = v
			case sppb.TypeCode_FLOAT64:
				var v []spanner.NullFloat64
				if err := col.Decode(&v); err != nil {
					return nil, err
				}
				dest[i] = v
			case sppb.TypeCode_NUMERIC:
				var v []spanner.NullNumeric
				if err := col.Decode(&v); err != nil {
					return nil, err
				}
				dest[i] = v
			case sppb.TypeCode_STRING:
				var v []spanner.NullString
				if err := col.Decode(&v); err != nil {
					return nil, err
				}
				dest[i] = v
			case sppb.TypeCode_JSON:
				var v []spanner.NullJSON
				if err := col.Decode(&v); err != nil {
					return nil, err
				}
				dest[i] = v
			case sppb.TypeCode_BYTES:
				var v [][]byte
				if err := col.Decode(&v); err != nil {
					return nil, err
				}
				dest[i] = v
			case sppb.TypeCode_BOOL:
				var v []spanner.NullBool
				if err := col.Decode(&v); err != nil {
					return nil, err
				}
				dest[i] = v
			case sppb.TypeCode_DATE:
				var v []spanner.NullDate
				if err := col.Decode(&v); err != nil {
					return nil, err
				}
				dest[i] = v
			case sppb.TypeCode_TIMESTAMP:
				var v []spanner.NullTime
				if err := col.Decode(&v); err != nil {
					return nil, err
				}
				dest[i] = v
			}
		}
	}
	return dest, nil
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
