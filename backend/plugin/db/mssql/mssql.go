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

	"github.com/golang-sql/sqlexp"
	gomssqldb "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/azuread"

	// Kerberos Active Directory authentication outside Windows.
	_ "github.com/microsoft/go-mssqldb/integratedauth/krb5"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	tsqlbatch "github.com/bytebase/bytebase/backend/plugin/parser/tsql/batch"
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
	certFilePath string
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a MSSQL driver.
func (d *Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	query := url.Values{}
	query.Add("app name", "bytebase")
	if config.ConnectionContext.DatabaseName != "" {
		query.Add("database", config.ConnectionContext.DatabaseName)
	} else if config.DataSource.Database != "" {
		query.Add("database", config.DataSource.Database)
	}

	// In order to be compatible with db servers that only support old versions of tls.
	// See: https://github.com/microsoft/go-mssqldb/issues/33
	query.Add("tlsmin", "1.0")

	// Add extra connection parameters if specified in the DataSource
	for key, value := range config.DataSource.GetExtraConnectionParameters() {
		query.Add(key, value)
	}

	var err error
	if config.DataSource.GetUseSsl() && config.DataSource.GetSslCa() != "" {
		// Due to Golang runtime limitation, x509 package will throw the error of 'certificate relies on legacy Common Name field, use SANs instead.
		// Driver reads the certificate from file instead of regarding it as certificate content.
		// https://github.com/microsoft/go-mssqldb/blob/main/msdsn/conn_str.go#L159
		// TODO(zp): Driver supports .der format also.
		const pattern string = "cert-*.pem"
		file, err := os.CreateTemp(os.TempDir(), pattern)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create temporary file with pattern %s", pattern)
		}
		fName := file.Name()
		defer func(err error) {
			if err != nil {
				_ = os.Remove(fName)
			} else {
				d.certFilePath = fName
			}
		}(err)
		_, err = file.WriteString(config.DataSource.GetSslCa())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write certificate to file %s", fName)
		}
		if err = file.Close(); err != nil {
			return nil, errors.Wrapf(err, "failed to close file %s", fName)
		}
		query.Add("certificate", fName)
	}
	query.Add("TrustServerCertificate", "true")

	driverName := "sqlserver"
	password := config.Password
	if config.DataSource.GetAuthenticationType() == storepb.DataSource_AZURE_IAM {
		driverName = azuread.DriverName
		if azureCredential := config.DataSource.GetAzureCredential(); azureCredential != nil {
			query.Add("fedauth", azuread.ActiveDirectoryServicePrincipal)
			query.Add("user id", fmt.Sprintf("%s@%s", azureCredential.ClientId, azureCredential.TenantId))
			query.Add("password", azureCredential.ClientSecret)
			password = ""
		} else {
			query.Add("fedauth", azuread.ActiveDirectoryDefault)
		}
	}
	u := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(config.DataSource.Username, password),
		Host:     fmt.Sprintf("%s:%s", config.DataSource.Host, config.DataSource.Port),
		RawQuery: query.Encode(),
	}
	var db *sql.DB
	db, err = sql.Open(driverName, u.String())
	if err != nil {
		return nil, err
	}
	d.db = db
	d.databaseName = config.ConnectionContext.DatabaseName
	return d, nil
}

// Close closes the driver.
func (d *Driver) Close(_ context.Context) error {
	if d.certFilePath != "" {
		if err := os.Remove(d.certFilePath); err != nil {
			slog.Warn("failed to delete temporary file", slog.String("path", d.certFilePath), log.BBError(err))
		}
	}
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Ping pings the database.
func (d *Driver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// GetDB gets the database.
func (d *Driver) GetDB() *sql.DB {
	return d.db
}

// Execute executes a SQL statement and returns the affected rows.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if opts.CreateDatabase {
		if _, err := d.db.ExecContext(ctx, statement); err != nil {
			return 0, err
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

	// Execute based on transaction mode
	if transactionMode == common.TransactionModeOff {
		return d.executeInAutoCommitMode(ctx, statement, opts)
	}
	return d.executeInTransactionMode(ctx, statement, opts)
}

// executeInTransactionMode executes statements within a single transaction
func (d *Driver) executeInTransactionMode(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	tx, err := d.db.BeginTx(ctx, nil)
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

	batch := tsqlbatch.NewBatcher(statement)

	for idx := 0; ; {
		command, err := batch.Next()
		if err != nil {
			if err == io.EOF {
				// Try send the last batch to server.
				v := batch.Batch()
				if v != nil && len(v.Text) > 0 {
					opts.LogCommandExecute(&storepb.Range{Start: int32(v.Start), End: int32(v.End)}, v.Text)
					rowsAffected, err := execute(ctx, tx, v.Text)
					if err != nil {
						opts.LogCommandResponse(0, nil, err.Error())
						return 0, err
					}
					opts.LogCommandResponse(rowsAffected, []int64{rowsAffected}, "")
					totalAffectRows += rowsAffected
				}
				break
			}
			return 0, errors.Wrapf(err, "failed to get next batch for statement: %s", batch.Batch().Text)
		}
		if command == nil {
			continue
		}
		switch v := command.(type) {
		case *tsqlbatch.GoCommand:
			b := batch.Batch()
			// Try send the batch to server.
			idx++
			for i := uint(0); i < v.Count; i++ {
				opts.LogCommandExecute(&storepb.Range{Start: int32(b.Start), End: int32(b.End)}, b.Text)
				rowsAffected, err := execute(ctx, tx, b.Text)
				if err != nil {
					opts.LogCommandResponse(0, nil, err.Error())
					return 0, err
				}
				opts.LogCommandResponse(rowsAffected, []int64{rowsAffected}, "")
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

// executeInAutoCommitMode executes statements sequentially in auto-commit mode
func (d *Driver) executeInAutoCommitMode(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	totalAffectRows := int64(0)

	batch := tsqlbatch.NewBatcher(statement)

	for idx := 0; ; {
		command, err := batch.Next()
		if err != nil {
			if err == io.EOF {
				// Try send the last batch to server.
				v := batch.Batch()
				if v != nil && len(v.Text) > 0 {
					opts.LogCommandExecute(&storepb.Range{Start: int32(v.Start), End: int32(v.End)}, v.Text)
					rowsAffected, err := d.executeAutoCommit(ctx, v.Text)
					if err != nil {
						opts.LogCommandResponse(0, nil, err.Error())
						return totalAffectRows, err
					}
					opts.LogCommandResponse(rowsAffected, []int64{rowsAffected}, "")
					totalAffectRows += rowsAffected
				}
				break
			}
			return 0, errors.Wrapf(err, "failed to get next batch for statement: %s", batch.Batch().Text)
		}
		if command == nil {
			continue
		}
		switch v := command.(type) {
		case *tsqlbatch.GoCommand:
			b := batch.Batch()
			// Execute the batch in auto-commit mode
			idx++
			for i := uint(0); i < v.Count; i++ {
				opts.LogCommandExecute(&storepb.Range{Start: int32(b.Start), End: int32(b.End)}, b.Text)
				rowsAffected, err := d.executeAutoCommit(ctx, b.Text)
				if err != nil {
					opts.LogCommandResponse(0, nil, err.Error())
					// In auto-commit mode, we stop at the first error
					return totalAffectRows, err
				}
				opts.LogCommandResponse(rowsAffected, []int64{rowsAffected}, "")
				totalAffectRows += rowsAffected
			}
		default:
			return 0, errors.Errorf("unsupported command type: %T", v)
		}
		batch.Reset(nil)
	}

	return totalAffectRows, nil
}

// executeAutoCommit executes a single statement in auto-commit mode
func (d *Driver) executeAutoCommit(ctx context.Context, statement string) (int64, error) {
	sqlResult, err := d.db.ExecContext(ctx, statement)
	if err != nil {
		return 0, errors.Wrap(err, "failed to execute statement in auto-commit mode")
	}
	rowsAffected, err := sqlResult.RowsAffected()
	if err != nil {
		// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
		slog.Debug("rowsAffected returns error in auto-commit mode", log.BBError(err))
		return 0, nil
	}
	return rowsAffected, nil
}

func execute(ctx context.Context, txn *sql.Tx, statement string) (int64, error) {
	sqlResult, err := txn.ExecContext(ctx, statement)
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

func (d *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	// Special handling for EXPLAIN queries in MSSQL is now integrated into queryBatch

	// Regular query processing (unchanged)
	batch := tsqlbatch.NewBatcher(statement)
	var results []*v1pb.QueryResult
	for {
		command, err := batch.Next()
		if err != nil {
			if err == io.EOF {
				v := batch.Batch()
				if v != nil && len(v.Text) > 0 {
					// Query the last batch.
					qr, err := d.queryBatch(ctx, conn, v.Text, queryContext)
					results = append(results, qr...)
					if err != nil {
						return results, err
					}
				}
				batch.Reset(nil)
				break
			}
			return results, errors.Wrapf(err, "failed to get next batch for statement: %s", batch.Batch().Text)
		}
		if command == nil {
			continue
		}
		switch v := command.(type) {
		case *tsqlbatch.GoCommand:
			b := batch.Batch()
			// Query the batch.
			qr, err := d.queryBatch(ctx, conn, b.Text, queryContext)
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
func (*Driver) queryBatch(ctx context.Context, conn *sql.Conn, batch string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := tsqlparser.SplitSQL(batch)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptyStatements(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	// Special handling for EXPLAIN queries in MSSQL using explain
	if queryContext.Explain {
		explain := "SHOWPLAN_ALL"
		if queryContext.Option.MssqlExplainFormat == v1pb.QueryOption_MSSQL_EXPLAIN_FORMAT_XML {
			explain = "SHOWPLAN_XML"
		}
		// Enable explain mode once for all statements
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET %s ON", explain)); err != nil {
			return nil, errors.Wrap(err, "failed to enable explain mode")
		}
		// Ensure explain is turned off after processing
		defer func() {
			if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET %s OFF", explain)); err != nil {
				slog.Warn("failed to disable explain mode", log.BBError(err))
			}
		}()

		var results []*v1pb.QueryResult

		// Process each statement with explain enabled
		for _, singleSQL := range singleSQLs {
			startTime := time.Now()

			queryResult, err := func() (*v1pb.QueryResult, error) {
				// Execute query to get execution plan
				rows, err := conn.QueryContext(ctx, singleSQL.Text)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get execution plan")
				}
				defer rows.Close()

				// Convert to query result
				r, err := util.RowsToQueryResult(rows, makeValueByTypeName, convertValue, queryContext.MaximumSQLResultSize)
				if err != nil {
					return nil, errors.Wrap(err, "failed to convert execution plan results")
				}

				if err = rows.Err(); err != nil {
					return nil, errors.Wrap(err, "error after processing rows")
				}

				return r, nil
			}()

			stop := false
			if err != nil {
				queryResult = &v1pb.QueryResult{
					Error: err.Error(),
				}
				stop = true
			}

			queryResult.Statement = singleSQL.Text
			queryResult.Latency = durationpb.New(time.Since(startTime))
			queryResult.RowsCount = int64(len(queryResult.Rows))

			results = append(results, queryResult)
			if stop {
				break
			}
		}

		return results, nil
	}

	// Regular query processing for non-EXPLAIN queries
	startTime := time.Now()
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
		if queryContext.Limit > 0 {
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
				nextAffectedRowsIdx = getNextAffectedRowsIdx(stmtTypes, nextAffectedRowsIdx+1)
				continue
			}
			queryResult = util.BuildAffectedRowsResult(m.Count, nil)
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
			r, err := util.RowsToQueryResult(rows, makeValueByTypeName, convertValue, queryContext.MaximumSQLResultSize)
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

	latency := time.Since(startTime)
	for i, res := range ret {
		res.Latency = durationpb.New(latency)
		res.Statement = singleSQLs[i].Text
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
		if s[i]&stmtTypeRowCountGenerating != 0 {
			return i
		}
	}
	return len(s)
}

func getNextResultSetIdx(s []stmtType, beginIdx int) int {
	for i := beginIdx; i < len(s); i++ {
		if s[i]&stmtTypeResultSetGenerating != 0 {
			return i
		}
	}
	return len(s)
}
