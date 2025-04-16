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
	// Get a dedicated connection from the pool
	conn, err := d.db.Conn(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get connection")
	}
	defer conn.Close()

	// Set connection ID if callback provided
	if opts.SetConnectionID != nil {
		// Use current_session as an identifier for Trino
		var sessionId string
		if err := conn.QueryRowContext(ctx, "SELECT current_session").Scan(&sessionId); err != nil {
			return 0, errors.Wrap(err, "failed to get session id")
		}
		opts.SetConnectionID(sessionId)

		if opts.DeleteConnectionID != nil {
			defer opts.DeleteConnectionID()
		}
	}

	// Log command execution
	opts.LogCommandExecute([]int32{0})

	// Execute the statement
	result, err := conn.ExecContext(ctx, statement)
	if err != nil {
		opts.LogCommandResponse([]int32{0}, 0, []int32{0}, err.Error())
		return 0, errors.Wrap(err, "failed to execute statement")
	}

	// Get rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// Some Trino operations don't return affected rows
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
			// Determine if this is a query and apply explain/limit if needed
			trimmed := strings.TrimSpace(stmt)
			upperStmt := strings.ToUpper(trimmed)
			isQuery := strings.HasPrefix(upperStmt, "SELECT") ||
				strings.HasPrefix(upperStmt, "SHOW") ||
				strings.HasPrefix(upperStmt, "DESCRIBE") ||
				strings.HasPrefix(upperStmt, "EXPLAIN")

			// Apply explain if needed
			if queryContext.Explain {
				stmt = fmt.Sprintf("EXPLAIN %s", stmt)
				isQuery = true
			}

			// Apply limit for SELECT queries if needed
			if isQuery && queryContext.Limit > 0 && !strings.Contains(upperStmt, " LIMIT ") {
				stmt = fmt.Sprintf("%s LIMIT %d", stmt, queryContext.Limit)
			}

			if isQuery {
				// Run as a query that returns rows
				rows, err := conn.QueryContext(ctx, stmt)
				if err != nil {
					return nil, err
				}
				defer rows.Close()

				// Get columns and create result structure
				columnNames, err := rows.Columns()
				if err != nil {
					return nil, err
				}

				columnTypes, err := rows.ColumnTypes()
				if err != nil {
					return nil, err
				}

				// Create result with column information
				result := &v1pb.QueryResult{
					ColumnNames: columnNames,
				}

				// Add column types
				for _, cType := range columnTypes {
					result.ColumnTypeNames = append(result.ColumnTypeNames,
						strings.ToUpper(cType.DatabaseTypeName()))
				}

				// Process rows
				for rows.Next() {
					// Check if we've exceeded the maximum result size
					if queryContext.MaximumSQLResultSize > 0 &&
						len(result.Rows) > 0 &&
						int64(proto.Size(result)) > queryContext.MaximumSQLResultSize {
						result.Error = common.FormatMaximumSQLResultSizeMessage(queryContext.MaximumSQLResultSize)
						break
					}

					// Create a slice to hold the row values
					rowValues := make([]interface{}, len(columnNames))
					scanValues := make([]interface{}, len(columnNames))
					for i := range rowValues {
						scanValues[i] = &rowValues[i]
					}

					// Scan the row
					if err := rows.Scan(scanValues...); err != nil {
						return nil, err
					}

					// Create a query row
					row := &v1pb.QueryRow{}

					// Convert each column value
					for _, val := range rowValues {
						row.Values = append(row.Values, convertToRowValue(val))
					}

					// Add row to result
					result.Rows = append(result.Rows, row)

					// Stop if we've reached the requested limit
					if queryContext.Limit > 0 && len(result.Rows) >= int(queryContext.Limit) {
						break
					}
				}

				if err := rows.Err(); err != nil {
					return nil, err
				}

				return result, nil
			} else {
				// Run as a statement that doesn't return rows
				result, err := conn.ExecContext(ctx, stmt)
				if err != nil {
					return nil, err
				}

				// Get affected rows
				affectedRows, err := result.RowsAffected()
				if err != nil {
					// Some operations don't return affected rows
					affectedRows = 0
				}

				// Create a result with affected rows count
				return &v1pb.QueryResult{
					ColumnNames:     []string{"Affected Rows"},
					ColumnTypeNames: []string{"INT"},
					Rows: []*v1pb.QueryRow{
						{
							Values: []*v1pb.RowValue{
								{
									Kind: &v1pb.RowValue_Int64Value{
										Int64Value: affectedRows,
									},
								},
							},
						},
					},
				}, nil
			}
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
func convertToRowValue(v interface{}) *v1pb.RowValue {
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
		// For any other type, convert to string
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: fmt.Sprintf("%v", val),
			},
		}
	}
}
