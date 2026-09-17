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
	"slices"
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

func formatMSSQLError(e gomssqldb.Error) string {
	b := new(strings.Builder)
	if len(e.ProcName) > 0 {
		fmt.Fprintf(b, "Msg %d, Level %d, State %d, Server %s, Procedure %s, Line %d\n", e.Number, e.Class, e.State, e.ServerName, e.ProcName, e.LineNo)
	} else {
		fmt.Fprintf(b, "Msg %d, Level %d, State %d, Server %s, Line %d\n", e.Number, e.Class, e.State, e.ServerName, e.LineNo)
	}
	b.WriteString(e.Message)
	return b.String()
}

func unpackGoMSSQLDBError(err gomssqldb.Error) error {
	if len(err.All) == 0 || len(err.All) == 1 {
		return errors.Errorf("%s", formatMSSQLError(err))
	}
	var msgs []string
	for _, e := range err.All {
		if e.Message == "" {
			continue
		}
		msgs = append(msgs, formatMSSQLError(e))
	}
	return errors.Errorf("%s", strings.Join(msgs, "\n"))
}

func (d *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	// Special handling for EXPLAIN queries in MSSQL is now integrated into queryBatch

	// Regular query processing (unchanged)
	batch := tsqlbatch.NewBatcher(statement)
	// Callers pair results[i] with the request's i-th statement, so the further
	// result sets of procedure calls come after every statement's result.
	var results, procedureResults []*v1pb.QueryResult
	for {
		command, err := batch.Next()
		if err != nil {
			if err == io.EOF {
				v := batch.Batch()
				if v != nil && len(v.Text) > 0 {
					// Query the last batch.
					qr, more, err := d.queryBatch(ctx, conn, v.Text, queryContext)
					results = append(results, qr...)
					procedureResults = append(procedureResults, more...)
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
			qr, more, err := d.queryBatch(ctx, conn, b.Text, queryContext)
			results = append(results, qr...)
			procedureResults = append(procedureResults, more...)
			if err != nil {
				return results, err
			}
		default:
			return results, errors.Errorf("unsupported command type: %T", v)
		}
		batch.Reset(nil)
	}
	return append(results, procedureResults...), nil
}

// showplanStatistic picks the SET statement that turns on the requested explain
// output. SQL Server has no JSON showplan; which formats an engine may be asked
// for is decided at the API boundary (validateExplainFormat), so anything else
// arriving here takes the default plan.
func showplanStatistic(option *v1pb.QueryOption) string {
	if option.GetExplainFormat() == v1pb.QueryOption_XML {
		return "SHOWPLAN_XML"
	}
	return "SHOWPLAN_ALL"
}

// queryBatch queries a batch of SQL statements and returns one result for each: the result set of a Result Set-Generating
// statement, the affected rows of a Row Count-Generating statement, or an empty result. The second slice holds a procedure
// call's result sets after its first.
// https://learn.microsoft.com/en-us/sql/odbc/reference/develop-app/result-generating-and-result-free-statements?view=sql-server-ver16
func (*Driver) queryBatch(ctx context.Context, conn *sql.Conn, batch string, queryContext db.QueryContext) ([]*v1pb.QueryResult, []*v1pb.QueryResult, error) {
	singleSQLs, err := tsqlparser.SplitSQL(batch)
	if err != nil {
		return nil, nil, err
	}
	singleSQLs = base.FilterEmptyStatements(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil, nil
	}

	// Special handling for EXPLAIN queries in MSSQL using explain
	if queryContext.Explain {
		explain := showplanStatistic(queryContext.Option)
		// Enable explain mode once for all statements
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET %s ON", explain)); err != nil { // NOSONAR(go:S2077) explain is a hardcoded constant ("SHOWPLAN_ALL" or "SHOWPLAN_XML"), not user input
			return nil, nil, errors.Wrap(err, "failed to enable explain mode")
		}
		// Ensure explain is turned off after processing
		defer func() {
			if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET %s OFF", explain)); err != nil { // NOSONAR(go:S2077) explain is a hardcoded constant ("SHOWPLAN_ALL" or "SHOWPLAN_XML"), not user input
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

		return results, nil, nil
	}

	// Regular query processing for non-EXPLAIN queries
	startTime := time.Now()
	var stmtTypes []stmtType
	var refinedSQLs []string
	batchBuf := new(strings.Builder)
	for _, singleSQL := range singleSQLs {
		stmtType, err := getStmtType(singleSQL.Text)
		if err != nil {
			return nil, nil, err
		}
		stmtTypes = append(stmtTypes, stmtType)
		// Before sending the batch to server, we add the limit clause to the statement.
		s := singleSQL.Text
		if queryContext.Limit > 0 {
			s = getStatementWithResultLimit(s, queryContext.Limit)
		}
		refinedSQLs = append(refinedSQLs, s)
		if _, err := batchBuf.WriteString(s); err != nil {
			return nil, nil, err
		}
		if _, err := batchBuf.WriteString("\n"); err != nil {
			return nil, nil, err
		}
	}

	refinedBatch := batchBuf.String()
	retmsg := &sqlexp.ReturnMessage{}
	rows, qe := conn.QueryContext(ctx, refinedBatch, retmsg) // NOSONAR(go:S2077) intentional execution of user-authored SQL in SQL Editor
	if qe != nil {
		return nil, nil, qe
	}
	defer rows.Close()
	var resultSets []*v1pb.QueryResult
	var rowCounts []int64
	inResultSet := false
	for more := true; more; {
		switch m := retmsg.Message(ctx).(type) {
		case sqlexp.MsgNotice:
			if err := isExitError(err); err != nil {
				return nil, nil, err
			}
		case sqlexp.MsgError:
			err := m.Error
			var e gomssqldb.Error
			if errors.As(err, &e) {
				err = unpackGoMSSQLDBError(e)
			}
			return nil, nil, err
		case sqlexp.MsgRowsAffected:
			// The count that closes a result set is the result set's own row count.
			if !inResultSet {
				rowCounts = append(rowCounts, m.Count)
			}
		case sqlexp.MsgNextResultSet:
			inResultSet = false
			more = rows.NextResultSet()
			if err = rows.Err(); err != nil {
				return nil, nil, err
			}
		case sqlexp.MsgNext:
			result, err := readResultSet(rows, queryContext)
			if err != nil {
				return nil, nil, err
			}
			resultSets = append(resultSets, result)
			inResultSet = true
		default:
		}
	}

	results, procedureResults, err := matchResults(refinedSQLs, stmtTypes, resultSets, rowCounts)
	if err != nil {
		return nil, nil, err
	}
	if queryContext.MaskingEnabled {
		for i, t := range stmtTypes {
			if t&(stmtTypeProcedure|stmtTypeOutput) != 0 {
				withholdRows(results[i])
			}
		}
		for _, result := range procedureResults {
			withholdRows(result)
		}
	}
	latency := time.Since(startTime)
	for _, res := range slices.Concat(results, procedureResults) {
		res.Latency = durationpb.New(latency)
	}
	return results, procedureResults, nil
}

// readResultSet reads the current result set. The limit rewrite covers only SELECT,
// so the row limit is applied here as well, for the rows of a procedure call or an
// OUTPUT clause.
func readResultSet(rows *sql.Rows, queryContext db.QueryContext) (*v1pb.QueryResult, error) {
	result, err := util.RowsToLimitedQueryResult(rows, makeValueByTypeName, convertValue, queryContext.MaximumSQLResultSize, queryContext.Limit)
	if err != nil {
		return nil, err
	}
	if result.Error != "" {
		// The read stopped at the size limit, and the next message only arrives
		// once the rest of the result set has been read. Past the end of a result
		// set, Next would read the following one.
		for rows.Next() {
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// withholdRows empties a result set of a procedure call or an OUTPUT clause,
// whose rows masking cannot trace to columns.
func withholdRows(result *v1pb.QueryResult) {
	if len(result.ColumnNames) == 0 {
		return
	}
	result.Rows = nil
	result.RowsCount = 0
	result.Error = "Rows returned by a procedure call or an OUTPUT clause are hidden because data masking can't be applied to them."
}

// matchResults gives each statement of a batch its result. SQL Server returns
// results in statement order but does not say which statement returned each, so
// a batch whose procedure call returns result sets must have no other statement
// that returns any. The procedure call's result sets after its first are
// returned separately.
func matchResults(statements []string, types []stmtType, resultSets []*v1pb.QueryResult, rowCounts []int64) ([]*v1pb.QueryResult, []*v1pb.QueryResult, error) {
	var withResultSet, withRowCount, procedures int
	for _, t := range types {
		switch {
		case t&stmtTypeResultSetGenerating != 0:
			withResultSet++
		case t&stmtTypeRowCountGenerating != 0:
			withRowCount++
		case t&stmtTypeProcedure != 0:
			procedures++
		default:
		}
	}
	procedureReturnedSets := len(resultSets) != withResultSet
	if procedureReturnedSets && (withResultSet > 0 || procedures != 1) {
		return nil, nil, errors.New("cannot match the batch's result sets to its statements; a procedure call that returns rows must be the only statement in its batch that does, so separate the others with GO")
	}
	// SET NOCOUNT and procedure calls change how many row counts arrive, so the
	// counts are shown only when each belongs to one statement.
	if len(rowCounts) != withRowCount {
		rowCounts = nil
	}

	results := make([]*v1pb.QueryResult, len(types))
	var procedureResults []*v1pb.QueryResult
	for i, t := range types {
		result := &v1pb.QueryResult{}
		switch {
		case t&stmtTypeResultSetGenerating != 0:
			result, resultSets = resultSets[0], resultSets[1:]
		case t&stmtTypeRowCountGenerating != 0 && len(rowCounts) > 0:
			result, rowCounts = util.BuildAffectedRowsResult(rowCounts[0], nil), rowCounts[1:]
		case t&stmtTypeProcedure != 0 && procedureReturnedSets:
			result, procedureResults = resultSets[0], resultSets[1:]
			for _, r := range procedureResults {
				r.Statement = statements[i]
			}
		default:
		}
		result.Statement = statements[i]
		results[i] = result
	}
	return results, procedureResults, nil
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
