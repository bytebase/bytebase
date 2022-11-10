package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/mysql"
	bbparser "github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/resources/mysqlutil"
)

var (
	generateRollbackSQLChan = make(chan *api.Task, 100)
)

// NewRollbackRunner creates a new backup runner.
func NewRollbackRunner(server *Server) *RollbackRunner {
	return &RollbackRunner{
		server: server,
	}
}

// RollbackRunner is the backup runner scheduling automatic backups.
type RollbackRunner struct {
	server *Server
}

// Run is the runner for backup runner.
func (r *RollbackRunner) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	r.retryGetRollbackSQL(ctx)
	for {
		select {
		case task := <-generateRollbackSQLChan:
			if task.Instance.Engine == db.MySQL && task.Type == api.TaskDatabaseDataUpdate {
				r.getRollbackSQL(ctx, task)
			}
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

// retryGetRollbackSQL retries generating rollback SQL for tasks.
// It is currently called when Bytebase server starts and only rerun unfinished generation.
func (r *RollbackRunner) retryGetRollbackSQL(ctx context.Context) {
	find := &api.TaskFind{
		StatusList: &[]api.TaskStatus{api.TaskRunning},
		TypeList:   &[]api.TaskType{api.TaskDatabaseDataUpdate},
		Payload:    fmt.Sprintf("payload->>'rollbackTaskState' = '%s'", api.RollbackTaskRunning),
	}
	taskList, err := r.server.store.FindTask(ctx, find, true)
	if err != nil {
		log.Error("Failed to get running DML tasks", zap.Error(err))
		return
	}
	for _, task := range taskList {
		if task.Instance.Engine != db.MySQL {
			continue
		}
		r.getRollbackSQL(ctx, task)
	}
}

func (r *RollbackRunner) getRollbackSQL(ctx context.Context, task *api.Task) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			log.Error("Rollback runner PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
		}
	}()

	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		log.Error("Invalid database data update payload", zap.Error(err))
		return
	}
	if payload.ThreadID == "" ||
		payload.BinlogFileStart == "" ||
		payload.BinlogPosStart == 0 ||
		payload.BinlogFileEnd == "" ||
		payload.BinlogPosEnd == 0 {
		log.Error("Cannot generate rollback SQL statement for the data update task with invalid payload", zap.Any("payload", *payload))
		return
	}

	rollbackSQL, err := r.getRollbackSQLImpl(ctx, task, payload)
	if err != nil {
		log.Error("Failed to generate rollback SQL statement", zap.Error(err))
		payload.RollbackTaskState = string(api.RollbackTaskFail)
		payload.RollbackError = err.Error()
	}
	payload.RollbackTaskState = string(api.RollbackTaskSuccess)
	payload.RollbackStatement = rollbackSQL

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Error("Failed to marshal task payload", zap.Error(err))
		return
	}
	payloadString := string(payloadBytes)
	patch := &api.TaskPatch{
		ID:        task.ID,
		UpdaterID: api.SystemBotID,
		Payload:   &payloadString,
	}
	if _, err := r.server.store.PatchTask(ctx, patch); err != nil {
		log.Error("Failed to patch task with the MySQL thread ID", zap.Int("taskID", task.ID))
		return
	}
	log.Debug("Rollback SQL generation success", zap.Int("taskID", task.ID))
}

func (r *RollbackRunner) getRollbackSQLImpl(ctx context.Context, task *api.Task, payload *api.TaskDatabaseDataUpdatePayload) (string, error) {
	basename, seqStart, err := mysql.ParseBinlogName(payload.BinlogFileStart)
	if err != nil {
		return "", errors.WithMessagef(err, "invalid start binlog file name %s", payload.BinlogFileStart)
	}
	_, seqEnd, err := mysql.ParseBinlogName(payload.BinlogFileEnd)
	if err != nil {
		return "", errors.WithMessagef(err, "Invalid end binlog file name %s", payload.BinlogFileEnd)
	}
	binlogFileNameList := mysql.GenBinlogFileNames(basename, seqStart, seqEnd)

	driver, err := r.server.getAdminDatabaseDriver(ctx, task.Instance, "")
	if err != nil {
		return "", errors.WithMessage(err, "failed to get admin database driver")
	}

	list, err := driver.FindMigrationHistoryList(ctx, &db.MigrationHistoryFind{ID: &payload.MigrationID})
	if err != nil {
		return "", errors.WithMessagef(err, "failed to find migration history with ID %d", payload.MigrationID)
	}
	if len(list) == 0 {
		return "", errors.WithMessagef(err, "migration history with ID %d not found", payload.MigrationID)
	}
	history := list[0]
	tableMap, err := parseTableColumns(history.Schema)
	if err != nil {
		return "", errors.WithMessage(err, "failed to parse the schema")
	}

	rollbackSQL, err := r.generateRollbackSQL(ctx, task, payload, binlogFileNameList, tableMap)
	if err != nil {
		return "", errors.WithMessage(err, "failed to generate rollback SQL statement")
	}

	return rollbackSQL, nil
}

func parseTableColumns(schema string) (map[string][]string, error) {
	_, supportStmts, err := bbparser.ExtractTiDBUnsupportStmts(schema)
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

func (r *RollbackRunner) generateRollbackSQL(ctx context.Context, task *api.Task, payload *api.TaskDatabaseDataUpdatePayload, binlogFileNameList []string, tableMap map[string][]string) (string, error) {
	adminDataSource := api.DataSourceFromInstanceWithType(task.Instance, api.Admin)
	args := binlogFileNameList
	args = append(args,
		"--read-from-remote-server",
		// Verify checksum binlog events.
		"--verify-binlog-checksum",
		// Tell mysqlbinlog to suppress the BINLOG statements for row events, which reduces the unneeded output.
		"--base64-output=DECODE-ROWS",
		// Reconstruct pseudo-SQL statements out of row events. This is where we parse the data changes.
		"--verbose",
		"--host", task.Instance.Host,
		"--user", adminDataSource.Username,
		// Start decoding the binary log at the log position, this option applies to the first log file named on the command line.
		"--start-position", fmt.Sprintf("%d", payload.BinlogPosStart),
		// Stop decoding the binary log at the log position, this option applies to the last log file named on the command line.
		"--stop-position", fmt.Sprintf("%d", payload.BinlogPosEnd),
	)
	if task.Instance.Port != "" {
		args = append(args, "--port", task.Instance.Port)
	}
	cmd := exec.CommandContext(ctx, mysqlutil.GetPath(mysqlutil.MySQLBinlog, common.GetResourceDir(r.server.profile.DataDir)), args...)
	log.Debug("mysqlbinlog", zap.String("command", cmd.String()))
	if adminDataSource.Password != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("MYSQL_PWD=%s", adminDataSource.Password))
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
	errScanner := bufio.NewScanner(errPipe)

	if err := cmd.Start(); err != nil {
		return "", errors.Wrap(err, "failed to run mysqlbinlog")
	}

	txnList, err := mysql.ParseBinlogStream(pr)
	if err != nil {
		return "", errors.WithMessage(err, "failed to parse binlog stream")
	}

	txnList, err = mysql.FilterBinlogTransactionsByThreadID(txnList, payload.ThreadID)
	if err != nil {
		return "", errors.WithMessage(err, "failed to filter binlog transactions by thread ID")
	}

	var rollbackSQLList []string
	for i := len(txnList) - 1; i >= 0; i-- {
		sql, err := txnList[i].GetRollbackSQL(tableMap)
		if err != nil {
			return "", errors.WithMessage(err, "failed to generate rollback SQL statement for transaction")
		}
		rollbackSQLList = append(rollbackSQLList, sql)
	}

	var errBuilder strings.Builder
	for errScanner.Scan() {
		line := errScanner.Text()
		// Log the error, but return the first 1024 characters in the error to users.
		log.Warn(line)
		if errBuilder.Len() < 1024 {
			if _, err := errBuilder.WriteString(line); err != nil {
				return "", errors.Wrap(err, "failed to write mysqlbinlog error string")
			}
			if _, err := errBuilder.WriteString("\n"); err != nil {
				return "", errors.Wrap(err, "failed to write mysqlbinlog error string")
			}
		}
	}
	if errScanner.Err() != nil {
		return "", errors.Wrap(errScanner.Err(), "error scanner failed")
	}

	if err = cmd.Wait(); err != nil {
		return "", errors.Wrapf(err, "mysqlbinlog error: %s", errBuilder.String())
	}

	return strings.Join(rollbackSQLList, "\n\n"), nil
}
