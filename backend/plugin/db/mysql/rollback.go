package mysql

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"regexp"
	"strings"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/resources/mysqlutil"
)

var (
	// The (?s) is a modifier that makes "." to match the new line character, which is a valid character in the MySQL database and name.
	reDatabaseTable = regexp.MustCompile("(?s)`(.+)`\\.`(.+)`")
)

// GetRollbackSQL generates the rollback SQL for the list of binlog events in the reversed order.
func (txn BinlogTransaction) GetRollbackSQL(tables map[string][]string) (string, error) {
	if len(txn) == 0 {
		return "", nil
	}
	var sqlList []string
	// Generate rollback SQL for each statement of the transaction in reversed order.
	// Each statement may have multiple affected rows in a single binlog event. The order between them is irrelevant.
	for i := len(txn) - 1; i >= 0; i-- {
		e := txn[i]
		if e.Type != WriteRowsEventType && e.Type != DeleteRowsEventType && e.Type != UpdateRowsEventType {
			continue
		}
		sql, err := e.getRollbackSQL(tables)
		if err != nil {
			return "", err
		}
		if sql != "" {
			sqlList = append(sqlList, sql)
		}
	}
	return strings.Join(sqlList, "\n"), nil
}

// GenerateRollbackSQL generates the rollback SQL statements from the binlog.
// binlogFileNameList is a list of binlog file names, such as ["binlog.000001", "binlog.000002"].
// binlogPosStart is the start position in the first binlog file.
// binlogPosEnd is the end position in the last binlog file.
// threadID is used to filter the binlog events of the target transaction. It is recorded after executing the transaction using the same connection.
// The binlog file names and positions are used to specify the binlog events range for rollback SQL generation.
// tableCatalog is a map from table names to column names. It is used to map positional placeholders in the binlog events to the actual columns to generate valid SQL statements.
// TODO(dragonly): parse/filter/generate rollback SQL in stream. Limit the generated SQL size to 8MB for now.
func (driver *Driver) GenerateRollbackSQL(ctx context.Context, binlogSizeLimit int, binlogFileNameList []string, binlogPosStart, binlogPosEnd int64, threadID string, tableCatalog map[string][]string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	args := binlogFileNameList
	args = append(args,
		"--read-from-remote-server",
		// Verify checksum binlog events.
		"--verify-binlog-checksum",
		// Tell mysqlbinlog to suppress the BINLOG statements for row events, which reduces the unneeded output.
		"--base64-output=DECODE-ROWS",
		// Reconstruct pseudo-SQL statements out of row events. This is where we parse the data changes.
		"--verbose",
		"--host", driver.connCfg.Host,
		"--user", driver.connCfg.Username,
		// Start decoding the binary log at the log position, this option applies to the first log file named on the command line.
		"--start-position", fmt.Sprintf("%d", binlogPosStart),
		// Stop decoding the binary log at the log position, this option applies to the last log file named on the command line.
		"--stop-position", fmt.Sprintf("%d", binlogPosEnd),
	)
	if driver.connCfg.Port != "" {
		args = append(args, "--port", driver.connCfg.Port)
	}
	cmd := exec.CommandContext(ctx, mysqlutil.GetPath(mysqlutil.MySQLBinlog, driver.dbBinDir), args...)
	slog.Debug("mysqlbinlog", slog.String("command", cmd.String()))
	if driver.connCfg.Password != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("MYSQL_PWD=%s", driver.connCfg.Password))
	}
	pr, err := cmd.StdoutPipe()
	if err != nil {
		return "", errors.Wrap(err, "failed to create stdout pipe")
	}

	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", errors.Wrap(err, "failed to create stderr pipe")
	}
	defer errPipe.Close()

	if err := cmd.Start(); err != nil {
		return "", errors.Wrap(err, "failed to run mysqlbinlog")
	}

	txnList, err := ParseBinlogStream(ctx, pr, threadID, binlogSizeLimit)
	if err != nil {
		return "", errors.WithMessage(err, "failed to parse binlog stream")
	}

	var rollbackSQLList []string
	for i := len(txnList) - 1; i >= 0; i-- {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		sql, err := txnList[i].GetRollbackSQL(tableCatalog)
		if err != nil {
			return "", errors.WithMessage(err, "failed to generate rollback SQL statement for transaction")
		}
		if sql != "" {
			rollbackSQLList = append(rollbackSQLList, sql)
		}
	}

	errReader := bufio.NewReader(errPipe)
	errBuilder := strings.Builder{}
	for {
		line, err := errReader.ReadString('\n')
		if err != io.EOF && err != nil {
			return "", errors.Wrap(err, "failed to read from stderr")
		}
		if strings.HasPrefix(line, "ERROR: ") {
			_, _ = errBuilder.WriteString(line)
			_, _ = errBuilder.WriteString("\n")
		}
		if err == io.EOF {
			break
		}
	}

	if errBuilder.Len() > 0 {
		return "", errors.Errorf("mysqlbinlog error: %s", errBuilder.String())
	}

	if err = cmd.Wait(); err != nil {
		return "", errors.Wrapf(err, "failed to run mysqlbinlog")
	}

	return strings.Join(rollbackSQLList, "\n\n"), nil
}

// GetTableColumns parses the schema to get the table columns map.
// This is used to generate rollback SQL from the binlog events.
func GetTableColumns(schema string) (map[string][]string, error) {
	_, supportStmts, err := tidbparser.ExtractTiDBUnsupportedStmts(schema)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract TiDB unsupported statements from old statements %q", schema)
	}
	nodes, _, err := parser.New().Parse(supportStmts, "", "")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse old statement %q", schema)
	}
	tableMap := make(map[string][]string)
	for _, node := range nodes {
		if stmt, ok := node.(*ast.CreateTableStmt); ok {
			var columnNames []string
			for _, col := range stmt.Cols {
				columnNames = append(columnNames, col.Name.Name.O)
			}
			tableMap[stmt.Table.Name.O] = columnNames
		}
	}
	return tableMap, nil
}

func (e *BinlogEvent) getRollbackSQL(tables map[string][]string) (string, error) {
	// 0. early return if the body is empty.
	switch e.Type {
	case WriteRowsEventType, DeleteRowsEventType, UpdateRowsEventType:
		if e.Body == "" {
			return "", nil
		}
	default:
		return "", errors.Errorf("invalid binlog event type %s", e.Type.String())
	}

	// 1. Remove the "### " prefix of each line.
	// mysqlbinlog output is separated by "\n", ref https://sourcegraph.com/github.com/mysql/mysql-server@a246bad76b9271cb4333634e954040a970222e0a/-/blob/sql/log_event.cc?L2398
	body := strings.Split(e.Body, "\n")
	body = replaceAllPrefix(body, "### ", "")

	matches := reDatabaseTable.FindStringSubmatch(e.Body)
	if len(matches) != 3 {
		return "", errors.Errorf("failed to match database and table names in binlog event %q", e.Body)
	}
	tableName := matches[2]
	columnNames, ok := tables[tableName]
	if !ok {
		return "", errors.Errorf("table %s does not exist in the provided table map", tableName)
	}

	// 2. Switch "DELETE FROM" and "INSERT INTO".
	// 3. Replace "WHERE" and "SET" with each other.
	// 4. Replace "@i" with the column names.
	// 5. Add a ";" at the end of each row.
	var err error
	switch e.Type {
	case WriteRowsEventType:
		body = replaceAllPrefix(body, "INSERT INTO", "DELETE FROM")
		body = replaceAllPrefix(body, "SET", "WHERE")
		body, err = replaceColumns(columnNames, body, "WHERE", " AND", ";")
	case DeleteRowsEventType:
		body = replaceAllPrefix(body, "DELETE FROM", "INSERT INTO")
		body = replaceAllPrefix(body, "WHERE", "SET")
		body, err = replaceColumns(columnNames, body, "SET", ",", ";")
	case UpdateRowsEventType:
		body = replaceAllPrefix(body, "WHERE", "OLDWHERE")
		body = replaceAllPrefix(body, "SET", "WHERE")
		body = replaceAllPrefix(body, "OLDWHERE", "SET")
		body, err = replaceColumns(columnNames, body, "SET", ",", "")
		if err != nil {
			return "", err
		}
		body, err = replaceColumns(columnNames, body, "WHERE", " AND", ";")
	default:
		return "", errors.Errorf("invalid binlog event type %s", e.Type.String())
	}

	return strings.Join(body, "\n"), err
}

func replaceAllPrefix(body []string, old, new string) []string {
	var ret []string
	for _, line := range body {
		ret = append(ret, replacePrefix(line, old, new))
	}
	return ret
}

func replacePrefix(line, old, new string) string {
	if strings.HasPrefix(line, old) {
		return new + strings.TrimPrefix(line, old)
	}
	return line
}

func replaceColumns(columnNames []string, body []string, sepLine, lineSuffix, sectionSuffix string) ([]string, error) {
	var equal string
	switch sepLine {
	case "SET":
		equal = "="
	case "WHERE":
		// Use NULL-safe equal operator.
		equal = "<=>"
	default:
		return nil, errors.Errorf("invalid sepLine %q", sepLine)
	}

	var ret []string
	for i := 0; i < len(body); {
		line := body[i]
		if line != sepLine {
			ret = append(ret, line)
			i++
			continue
		}
		// Found the "WHERE" or "SET" line
		ret = append(ret, line)
		i++
		if i+len(columnNames) > len(body) {
			return nil, errors.Errorf("binlog event body has a section with less columns than %d: %q", len(columnNames), strings.Join(body, "\n"))
		}
		for j := range columnNames {
			prefix := fmt.Sprintf("  @%d=", j+1)
			line := body[i+j]
			if !strings.HasPrefix(line, prefix) {
				return nil, errors.Errorf("invalid value line %q, must starts with %q", line, prefix)
			}
			if j == len(columnNames)-1 {
				ret = append(ret, fmt.Sprintf("  `%s`%s%s%s", columnNames[j], equal, strings.TrimPrefix(line, prefix), sectionSuffix))
			} else {
				ret = append(ret, fmt.Sprintf("  `%s`%s%s%s", columnNames[j], equal, strings.TrimPrefix(line, prefix), lineSuffix))
			}
		}
		i += len(columnNames)
	}
	return ret, nil
}

func (t BinlogEventType) String() string {
	switch t {
	case DeleteRowsEventType:
		return "DELETE"
	case UpdateRowsEventType:
		return "UPDATE"
	case WriteRowsEventType:
		return "INSERT"
	case QueryEventType:
		return "QUERY"
	case XidEventType:
		return "XID"
	default:
		return "UNKNOWN"
	}
}
