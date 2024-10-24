// Package mssql is the plugin for MSSQL driver.
package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	// Import MSSQL driver.
	gomssqldb "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/integratedauth/krb5"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	tsqlbatch "github.com/bytebase/bytebase/backend/plugin/parser/tsql/batch"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_MSSQL, newDriver)
}

// Driver is the MSSQL driver.
type Driver struct {
	db           *sql.DB
	databaseName string

	// certificate file path should be deleted if calling closed.
	certFilePath         string
	maximumSQLResultSize int64
	isShowPlanAll        bool
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a MSSQL driver.
func (driver *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	query := url.Values{}
	query.Add("app name", "bytebase")
	if config.Database != "" {
		query.Add("database", config.Database)
	}

	// In order to be compatible with db servers that only support old versions of tls.
	// See: https://github.com/microsoft/go-mssqldb/issues/33
	query.Add("tlsmin", "1.0")

	trustServerCertificate := "true"

	var err error
	if config.TLSConfig.UseSSL && config.TLSConfig.SslCA != "" {
		// We should not TrustServerCertificate in production environment, otherwise, TLS is susceptible
		// to man-in-the middle attacks. TrustServerCertificate makes driver accepts any certificate presented by the server
		// and any host name in that certificate.
		// Due to Golang runtime limitation, x509 package will throw the error of 'certificate relies on legacy Common Name field, use SANs instead if
		// TrustServerCertificate is false.
		trustServerCertificate = "false"
		// Driver reads the certificate from file instead of regarding it as certificate content.
		// https://github.com/microsoft/go-mssqldb/blob/main/msdsn/conn_str.go#L159
		// TODO(zp): Driver supports .der format also.
		const pattern string = "cert-*.pem"
		file, err := os.CreateTemp(os.TempDir(), pattern)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create temporary file with pattern %s", pattern)
		}
		fName := file.Name()
		defer func() {
			if err != nil {
				_ = os.Remove(fName)
			} else {
				driver.certFilePath = fName
			}
		}()
		_, err = file.WriteString(config.TLSConfig.SslCA)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write certificate to file %s", fName)
		}
		if err = file.Close(); err != nil {
			return nil, errors.Wrapf(err, "failed to close file %s", fName)
		}
		query.Add("certificate", fName)
	}
	query.Add("TrustServerCertificate", trustServerCertificate)
	u := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(config.Username, config.Password),
		Host:     fmt.Sprintf("%s:%s", config.Host, config.Port),
		RawQuery: query.Encode(),
	}
	var db *sql.DB
	db, err = sql.Open("sqlserver", u.String())
	if err != nil {
		return nil, err
	}
	driver.db = db
	driver.databaseName = config.Database
	driver.maximumSQLResultSize = config.MaximumSQLResultSize
	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(_ context.Context) error {
	if driver.certFilePath != "" {
		if err := os.Remove(driver.certFilePath); err != nil {
			slog.Warn("failed to delete temporary file", slog.String("path", driver.certFilePath), slog.Any("error", err))
		}
	}
	return nil
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// Execute executes a SQL statement and returns the affected rows.
func (driver *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if opts.CreateDatabase {
		if _, err := driver.db.ExecContext(ctx, statement); err != nil {
			return 0, err
		}
		return 0, nil
	}
	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, err.Error())
		return 0, err
	}
	opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, "")

	committed := false
	defer func() {
		err := tx.Rollback()
		if committed {
			return
		}
		var rerr string
		if err != nil {
			rerr = err.Error()
		}
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_ROLLBACK, rerr)
	}()

	totalAffectRows := int64(0)

	batch := NewBatch(statement)

	for idx := 0; ; {
		command, err := batch.Next()
		if err != nil {
			if err == io.EOF {
				// Try send the last batch to server.
				v := batch.String()
				if v != "" {
					indexes := []int32{int32(idx)}
					opts.LogCommandExecute(indexes)
					rowsAffected, err := execute(ctx, tx, v)
					if err != nil {
						opts.LogCommandResponse(indexes, 0, nil, err.Error())
						return 0, err
					}
					opts.LogCommandResponse(indexes, int32(rowsAffected), []int32{int32(rowsAffected)}, "")
					totalAffectRows += rowsAffected
				}
				break
			}
			return 0, errors.Wrapf(err, "failed to get next batch for statement: %s", batch.String())
		}
		if command == nil {
			continue
		}
		switch v := command.(type) {
		case *tsqlbatch.GoCommand:
			stmt := batch.String()
			// Try send the batch to server.
			indexes := []int32{int32(idx)}
			idx++
			for i := uint(0); i < v.Count; i++ {
				opts.LogCommandExecute(indexes)
				rowsAffected, err := execute(ctx, tx, stmt)
				if err != nil {
					opts.LogCommandResponse(indexes, 0, nil, err.Error())
					return 0, err
				}
				opts.LogCommandResponse(indexes, int32(rowsAffected), []int32{int32(rowsAffected)}, "")
				totalAffectRows += rowsAffected
			}
		default:
			return 0, errors.Errorf("unsupported command type: %T", v)
		}
		batch.Reset(nil)
	}

	if err := tx.Commit(); err != nil {
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, err.Error())
		return 0, err
	}
	opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, "")
	committed = true
	return totalAffectRows, nil
}

func execute(ctx context.Context, tx *sql.Tx, statement string) (int64, error) {
	sqlResult, err := tx.ExecContext(ctx, statement)
	if e, ok := err.(gomssqldb.Error); ok {
		err = unpackGoMSSQLDBError(e)
	}
	if err != nil {
		return 0, errors.Wrap(err, "failed to execute statement")
	}
	rowsAffected, err := sqlResult.RowsAffected()
	if err != nil {
		// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
		slog.Debug("rowsAffected returns error", log.BBError(err))
		return 0, nil
	}
	return rowsAffected, nil
}

func unpackGoMSSQLDBError(err gomssqldb.Error) error {
	var msgs []string
	for _, e := range err.All {
		msgs = append(msgs, e.Message)
	}
	return errors.Errorf("%s", strings.Join(msgs, "\n"))
}

// QueryConn queries a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := tsqlparser.SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	isExplain := queryContext.Explain
	if isExplain {
		if _, err := conn.ExecContext(ctx, "SET SHOWPLAN_ALL ON;"); err != nil {
			return nil, err
		}
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		statement := singleSQL.Text
		if !isExplain && queryContext.Limit > 0 {
			statement = getStatementWithResultLimit(statement, queryContext.Limit)
		}

		var allQuery bool
		if isExplain {
			allQuery = true
		} else {
			_, q, err := base.ValidateSQLForEditor(storepb.Engine_MSSQL, statement)
			if err != nil {
				return nil, err
			}
			allQuery = q
		}
		if driver.isShowPlanAll {
			allQuery = true
		}

		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			if allQuery {
				rows, err := conn.QueryContext(ctx, statement)
				if err != nil {
					return nil, err
				}
				defer rows.Close()
				r, err := util.RowsToQueryResult(rows, makeValueByTypeName, convertValue, driver.maximumSQLResultSize)
				if err != nil {
					return nil, err
				}
				if err := rows.Err(); err != nil {
					return nil, err
				}
				return r, nil
			}

			sqlResult, err := conn.ExecContext(ctx, statement)
			if err != nil {
				return nil, err
			}
			affectedRows, err := sqlResult.RowsAffected()
			if err != nil {
				slog.Info("rowsAffected returns error", log.BBError(err))
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
		queryResult.Statement = statement
		queryResult.Latency = durationpb.New(time.Since(startTime))
		// Check SHOWPLAN_ALL.
		if isOn, ok := getShowPlanAll(statement); ok {
			driver.isShowPlanAll = isOn
		}
		results = append(results, queryResult)
		if stop {
			break
		}
	}

	return results, nil
}

func NewBatch(statement string) *tsqlbatch.Batch {
	// Split to batches to support some client commands like GO.
	s := strings.Split(statement, "\n")
	scanner := func() (string, error) {
		if len(s) > 0 {
			z := s[0]
			s = s[1:]
			return z, nil
		}
		return "", io.EOF
	}
	return tsqlbatch.NewBatch(scanner)
}

// getShowPlanAll returns on/off and ok for the statement.
func getShowPlanAll(s string) (bool, bool) {
	if len(s) > 30 {
		return false, false
	}
	s = strings.ToLower(s)
	if !strings.Contains(s, "showplan_all") {
		return false, false
	}
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, ";")
	s = strings.TrimSpace(s)
	tokens := strings.Fields(s)
	if len(tokens) != 3 {
		return false, false
	}
	if tokens[0] != "set" {
		return false, false
	}
	if tokens[1] != "showplan_all" {
		return false, false
	}
	switch tokens[2] {
	case "on":
		return true, true
	case "off":
		return false, true
	}
	return false, false
}
