package trino

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	// Import Trino driver for side effects
	_ "github.com/trinodb/trino-go-client/trino"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func init() {
	db.Register(storepb.Engine_TRINO, newDriver)
}

type Driver struct {
	config       db.ConnectionConfig
	db           *sql.DB
	databaseName string
}

func newDriver() db.Driver {
	return &Driver{}
}

func (*Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	// Construct Trino DSN
	scheme := "http"
	if config.DataSource.UseSsl {
		scheme = "https"
	}

	// Get user and password
	user := config.DataSource.Username
	if user == "" {
		user = "trino" // default user if not specified
	}

	password := config.Password

	// Set host and port
	host := config.DataSource.Host
	port := config.DataSource.Port
	if port == "" {
		port = "8080" // default Trino port
	}

	// Build URL with query parameters
	u := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%s", host, port),
	}

	// Set user info based on whether password exists
	if password != "" {
		u.User = url.UserPassword(user, password)
	} else {
		u.User = url.User(user)
	}

	// Add query parameters
	query := u.Query()
	query.Add("source", "bytebase")

	database := config.DataSource.Database
	if config.ConnectionContext.DatabaseName != "" {
		database = config.ConnectionContext.DatabaseName
	}
	if database == "" {
		database = "system"
	}
	query.Add("catalog", database)
	u.RawQuery = query.Encode()

	// Get DSN from URL
	dsn := u.String()

	// Connect using the Trino driver
	db, err := sql.Open("trino", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Trino")
	}

	// Set connection pool parameters
	db.SetConnMaxLifetime(30 * time.Second)
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(20)

	d := &Driver{
		config:       config,
		db:           db,
		databaseName: config.ConnectionContext.DatabaseName,
	}
	return d, nil
}

func (d *Driver) Close(context.Context) error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *Driver) Ping(ctx context.Context) error {
	if d.db != nil {
		return d.db.PingContext(ctx)
	}
	return errors.New("database connection not established")
}

func (d *Driver) GetDB() *sql.DB {
	return d.db
}

// Execute executes the SQL statement with the given options.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	conn, err := d.db.Conn(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get connection")
	}
	defer conn.Close()

	rawStmts, err := util.SanitizeSQL(statement)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to split sql")
	}

	singleSQLs := make([]base.SingleSQL, len(rawStmts))
	for i, stmt := range rawStmts {
		singleSQLs[i] = base.SingleSQL{Text: stmt}
	}

	commands, originalIndex := base.FilterEmptySQLWithIndexes(singleSQLs)
	if len(commands) == 0 {
		return 0, nil
	}

	var totalRowsAffected int64
	for i, command := range commands {
		indexes := []int32{originalIndex[i]}
		opts.LogCommandExecute(indexes)

		result, err := conn.ExecContext(ctx, command.Text)
		if err != nil {
			opts.LogCommandResponse(indexes, 0, []int32{0}, err.Error())
			return totalRowsAffected, errors.Wrapf(err, "failed to execute statement")
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			rowsAffected = 0
		}

		totalRowsAffected += rowsAffected
		opts.LogCommandResponse(indexes, int32(rowsAffected), []int32{int32(rowsAffected)}, "")
	}

	return totalRowsAffected, nil
}

// QueryConn executes a query using the provided connection.
func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	stmts, err := util.SanitizeSQL(statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split sql")
	}

	if queryContext.Schema != "" {
		escapedSchema := strings.ReplaceAll(queryContext.Schema, `"`, `""`)
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("USE \"%s\"", escapedSchema)); err != nil {
			return nil, errors.Wrapf(err, "failed to set schema")
		}
	}

	var results []*v1pb.QueryResult
	for _, stmt := range stmts {
		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			upperStmt := strings.ToUpper(strings.TrimSpace(stmt))
			isQuery := strings.HasPrefix(upperStmt, "SELECT") ||
				strings.HasPrefix(upperStmt, "SHOW") ||
				strings.HasPrefix(upperStmt, "DESCRIBE") ||
				strings.HasPrefix(upperStmt, "EXPLAIN")

			if queryContext.Explain {
				stmt = fmt.Sprintf("EXPLAIN %s", stmt)
				isQuery = true
			}

			if isQuery && queryContext.Limit > 0 && !strings.Contains(upperStmt, " LIMIT ") {
				stmt = fmt.Sprintf("%s LIMIT %d", stmt, queryContext.Limit)
			}

			if isQuery {
				rows, err := conn.QueryContext(ctx, stmt)
				if err != nil {
					return nil, err
				}
				defer rows.Close()

				result, err := util.RowsToQueryResult(rows, makeValueByTypeName, convertValue, queryContext.MaximumSQLResultSize)
				if err != nil {
					return nil, err
				}

				if err := rows.Err(); err != nil {
					return nil, err
				}

				return result, nil
			}

			result, err := conn.ExecContext(ctx, stmt)
			if err != nil {
				return nil, err
			}

			affectedRows, err := result.RowsAffected()
			if err != nil {
				affectedRows = 0
			}

			return util.BuildAffectedRowsResult(affectedRows, nil), nil
		}()

		stop := false
		if err != nil {
			queryResult = &v1pb.QueryResult{
				Error: err.Error(),
			}
			stop = true
		}

		queryResult.Statement = stmt
		queryResult.Latency = durationpb.New(time.Since(startTime))
		queryResult.RowsCount = int64(len(queryResult.Rows))
		results = append(results, queryResult)

		if stop {
			break
		}
	}

	return results, nil
}
