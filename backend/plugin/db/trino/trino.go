package trino

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	// Import Trino driver for side effects
	_ "github.com/trinodb/trino-go-client/trino"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
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

func newDriver(db.DriverConfig) db.Driver {
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

	if opts.SetConnectionID != nil {
		var sessionID string
		if err := conn.QueryRowContext(ctx, "SELECT current_session").Scan(&sessionID); err != nil {
			return 0, errors.Wrap(err, "failed to get session id")
		}
		opts.SetConnectionID(sessionID)

		if opts.DeleteConnectionID != nil {
			defer opts.DeleteConnectionID()
		}
	}

	opts.LogCommandExecute([]int32{0})

	result, err := conn.ExecContext(ctx, statement)
	if err != nil {
		opts.LogCommandResponse([]int32{0}, 0, []int32{0}, err.Error())
		return 0, errors.Wrap(err, "failed to execute statement")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		opts.LogCommandResponse([]int32{0}, 0, []int32{0}, "")
		return 0, nil
	}

	opts.LogCommandResponse([]int32{0}, int32(rowsAffected), []int32{int32(rowsAffected)}, "")
	return rowsAffected, nil
}

// QueryConn executes a query using the provided connection.
func (d *Driver) QueryConn(ctx context.Context, conn *sql.Conn, rawStatement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	stmts, err := util.SanitizeSQL(rawStatement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split sql")
	}

	var results []*v1pb.QueryResult
	for _, stmt := range stmts {
		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			trimmed := strings.TrimSpace(stmt)
			upperStmt := strings.ToUpper(trimmed)
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
				// Add catalog.schema qualification for unqualified table references if needed
				if strings.Contains(upperStmt, " FROM ") &&
					!strings.Contains(upperStmt, ".") &&
					d.databaseName != "" && queryContext.Schema != "" {
					fromIndex := strings.Index(upperStmt, " FROM ")
					if fromIndex != -1 {
						beforeFrom := stmt[:fromIndex+6]
						afterFrom := stmt[fromIndex+6:]

						tableEnd := len(afterFrom)
						if spaceIndex := strings.Index(afterFrom, " "); spaceIndex != -1 {
							tableEnd = spaceIndex
						}
						tableName := afterFrom[:tableEnd]
						restOfQuery := afterFrom[tableEnd:]

						qualifiedTable := fmt.Sprintf("%s.%s.%s", d.databaseName, queryContext.Schema, tableName)
						stmt = beforeFrom + qualifiedTable + restOfQuery
					}
				}

				rows, err := conn.QueryContext(ctx, stmt)
				if err != nil {
					return nil, err
				}
				defer rows.Close()

				columnNames, err := rows.Columns()
				if err != nil {
					return nil, err
				}

				columnTypes, err := rows.ColumnTypes()
				if err != nil {
					return nil, err
				}

				result := &v1pb.QueryResult{
					ColumnNames: columnNames,
				}

				for _, cType := range columnTypes {
					result.ColumnTypeNames = append(result.ColumnTypeNames,
						strings.ToUpper(cType.DatabaseTypeName()))
				}

				for rows.Next() {
					if queryContext.MaximumSQLResultSize > 0 &&
						len(result.Rows) > 0 &&
						int64(proto.Size(result)) > queryContext.MaximumSQLResultSize {
						result.Error = common.FormatMaximumSQLResultSizeMessage(queryContext.MaximumSQLResultSize)
						break
					}

					rowValues := make([]any, len(columnNames))
					scanValues := make([]any, len(columnNames))
					for i := range rowValues {
						scanValues[i] = &rowValues[i]
					}

					if err := rows.Scan(scanValues...); err != nil {
						return nil, err
					}

					row := &v1pb.QueryRow{}
					for _, val := range rowValues {
						row.Values = append(row.Values, convertToRowValue(val))
					}

					result.Rows = append(result.Rows, row)

					if queryContext.Limit > 0 && len(result.Rows) >= int(queryContext.Limit) {
						break
					}
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

			return util.BuildAffectedRowsResult(affectedRows), nil
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

// convertToRowValue converts a value to a RowValue for the query result.
func convertToRowValue(v any) *v1pb.RowValue {
	if v == nil {
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_NullValue{
				NullValue: structpb.NullValue_NULL_VALUE,
			},
		}
	}

	switch val := v.(type) {
	case string:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: val,
			},
		}
	case []byte:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_BytesValue{
				BytesValue: val,
			},
		}
	case int64:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_Int64Value{
				Int64Value: val,
			},
		}
	case int:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_Int64Value{
				Int64Value: int64(val),
			},
		}
	case int32:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_Int32Value{
				Int32Value: val,
			},
		}
	case float64:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_DoubleValue{
				DoubleValue: val,
			},
		}
	case float32:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_FloatValue{
				FloatValue: val,
			},
		}
	case bool:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_BoolValue{
				BoolValue: val,
			},
		}
	case time.Time:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: val.Format(time.RFC3339),
			},
		}
	default:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: fmt.Sprintf("%v", val),
			},
		}
	}
}
