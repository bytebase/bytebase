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

	// Import MSSQL driver.
	"github.com/golang-sql/sqlexp"
	gomssqldb "github.com/microsoft/go-mssqldb"

	// Kerberos Active Directory authentication outside Windows.
	_ "github.com/microsoft/go-mssqldb/integratedauth/krb5"
	"github.com/pkg/errors"

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
	var e gomssqldb.Error
	if errors.As(err, &e) {
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
	if len(err.All) == 0 || len(err.All) == 1 {
		return errors.Errorf("%s", err.Message)
	}
	var msgs []string
	for _, e := range err.All {
		cerr := unpackGoMSSQLDBError(e)
		if cerr == nil {
			continue
		}
		msgs = append(msgs, cerr.Error())
	}
	return errors.Errorf("%s", strings.Join(msgs, "\n"))
}

func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	batch := NewBatch(statement)
	var results []*v1pb.QueryResult
	for {
		command, err := batch.Next()
		if err != nil {
			if err == io.EOF {
				v := batch.String()
				if v != "" {
					// Query the last batch.
					qr, err := driver.queryBatch(ctx, conn, v, queryContext)
					results = append(results, qr...)
					if err != nil {
						return results, err
					}
				}
				batch.Reset(nil)
				break
			}
			return results, errors.Wrapf(err, "failed to get next batch for statement: %s", batch.String())
		}
		if command == nil {
			continue
		}
		switch v := command.(type) {
		case *tsqlbatch.GoCommand:
			stmt := batch.String()
			// Query the batch.
			qr, err := driver.queryBatch(ctx, conn, stmt, queryContext)
			results = append(results, qr...)
			if err != nil {
				return results, err
			}
		default:
			return results, errors.Errorf("unsupported command type: %T", v)
		}
		batch.Reset(nil)
	}
	return results, nil
}

// queryBatch queries a batch of SQL statements, for Result Set-Generating statements, it returns the results, for Row Count-Generating statements,
// it returns the affected rows, for other statements, it returns the empty query result.
// https://learn.microsoft.com/en-us/sql/odbc/reference/develop-app/result-generating-and-result-free-statements?view=sql-server-ver16
func (driver *Driver) queryBatch(ctx context.Context, conn *sql.Conn, batch string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := tsqlparser.SplitSQL(batch)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}
	var stmtTypes []stmtType
	batchBuf := new(strings.Builder)
	for _, singleSQL := range singleSQLs {
		stmtType, err := getStmtType(singleSQL.Text)
		if err != nil {
			return nil, err
		}
		stmtTypes = append(stmtTypes, stmtType)
		// Before sending the batch to server, we add the limit clause to the statement.
		s := singleSQL.Text
		if !queryContext.Explain && queryContext.Limit > 0 {
			s = getStatementWithResultLimit(s, queryContext.Limit)
		}
		if _, err := batchBuf.WriteString(s); err != nil {
			return nil, err
		}
		if _, err := batchBuf.WriteString("\n"); err != nil {
			return nil, err
		}
	}

	refinedBatch := batchBuf.String()
	retmsg := &sqlexp.ReturnMessage{}
	rows, qe := conn.QueryContext(ctx, refinedBatch, retmsg)
	if qe != nil {
		return nil, qe
	}
	defer rows.Close()
	nextResultSetIdx := getNextResultSetIdx(stmtTypes, 0)
	nextAffectedRowsIdx := getNextAffectedRowsIdx(stmtTypes, 0)
	skipRowsAffected := false
	results := true
	var ret []*v1pb.QueryResult
	for results {
		queryResult := new(v1pb.QueryResult)
		msg := retmsg.Message(ctx)
		// While meeting the RowsAffected and MsgNext, fill up the lap for the other statement types.
		switch m := msg.(type) {
		case sqlexp.MsgNotice:
			if err := isExitError(err); err != nil {
				return ret, err
			}
		case sqlexp.MsgError:
			err := m.Error
			var e gomssqldb.Error
			if errors.As(err, &e) {
				err = unpackGoMSSQLDBError(e)
			}
			queryResult.Error = err.Error()
			ret = append(ret, queryResult)
			return ret, err
		case sqlexp.MsgRowsAffected:
			// Assuming the rows affected appears after the MsgNext for SELECT statement, and ignore the MsgRowsAffected
			// for SELECT statement for now.
			if skipRowsAffected {
				skipRowsAffected = false
				continue
			}
			queryResult = util.BuildAffectedRowsResult(m.Count)
			emptyResultSets := make([]*v1pb.QueryResult, nextAffectedRowsIdx-len(ret))
			for i := 0; i < len(emptyResultSets); i++ {
				emptyResultSets[i] = &v1pb.QueryResult{}
			}
			ret = append(ret, emptyResultSets...)
			ret = append(ret, queryResult)
			nextAffectedRowsIdx = getNextAffectedRowsIdx(stmtTypes, nextAffectedRowsIdx+1)
		case sqlexp.MsgNextResultSet:
			results = rows.NextResultSet()
			if err = rows.Err(); err != nil {
				return ret, err
			}
		case sqlexp.MsgNext:
			r, err := util.RowsToQueryResult(rows, makeValueByTypeName, convertValue, driver.maximumSQLResultSize)
			if err != nil {
				queryResult.Error = err.Error()
				ret = append(ret, queryResult)
				return ret, err
			}
			if err := rows.Err(); err != nil {
				return ret, err
			}
			queryResult = r
			// Fill up the lap for the other statement types.
			emptyResultSets := make([]*v1pb.QueryResult, nextResultSetIdx-len(ret))
			for i := 0; i < len(emptyResultSets); i++ {
				emptyResultSets[i] = &v1pb.QueryResult{}
			}
			ret = append(ret, emptyResultSets...)
			ret = append(ret, queryResult)
			nextResultSetIdx = getNextResultSetIdx(stmtTypes, nextResultSetIdx+1)
			skipRowsAffected = true
		}
	}
	return ret, nil
}

func isExitError(err error) error {
	if err == nil {
		return nil
	}
	var errState uint8
	switch sqlError := err.(type) {
	case gomssqldb.Error:
		errState = sqlError.State
	default:
	}
	// 127 is the magic exit code
	if errState == 127 {
		return errors.Errorf("meet exit error, state: %d", errState)
	}
	return nil
}

func getNextAffectedRowsIdx(s []stmtType, beginIdx int) int {
	for i := beginIdx; i < len(s); i++ {
		if s[i] == stmtTypeResultSetRowCountGenerating {
			return i
		}
	}
	return len(s)
}

func getNextResultSetIdx(s []stmtType, beginIdx int) int {
	for i := beginIdx; i < len(s); i++ {
		if s[i] == stmtTypeResultSetGenerating {
			return i
		}
	}
	return len(s)
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
