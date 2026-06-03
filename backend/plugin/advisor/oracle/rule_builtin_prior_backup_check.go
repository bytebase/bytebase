package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, &StatementPriorBackupCheckAdvisor{})
}

type StatementPriorBackupCheckAdvisor struct {
}

func (*StatementPriorBackupCheckAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	if !checkCtx.EnablePriorBackup {
		return nil, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewStatementPriorBackupCheckRule(ctx, level, checkCtx.Rule.Type.String(), checkCtx)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

type StatementType int

const (
	StatementTypeUnknown StatementType = iota
	StatementTypeUpdate
	StatementTypeInsert
	StatementTypeDelete
)

type TableReference struct {
	Database      string
	HasSchema     bool
	Schema        string
	Table         string
	Alias         string
	StatementType StatementType
}

type statementInfo struct {
	statement string
	table     *TableReference
}

// StatementPriorBackupCheckRule is the rule implementation for prior backup checks.
type StatementPriorBackupCheckRule struct {
	BaseRule

	ctx      context.Context
	checkCtx advisor.Context

	statementInfoList []statementInfo
	hasDDL            bool
}

// NewStatementPriorBackupCheckRule creates a new StatementPriorBackupCheckRule.
func NewStatementPriorBackupCheckRule(ctx context.Context, level storepb.Advice_Status, title string, checkCtx advisor.Context) *StatementPriorBackupCheckRule {
	return &StatementPriorBackupCheckRule{
		BaseRule: NewBaseRule(level, title, 0),
		ctx:      ctx,
		checkCtx: checkCtx,
	}
}

// Name returns the rule name.
func (*StatementPriorBackupCheckRule) Name() string {
	return "builtin.prior-backup-check"
}

// OnStatement collects top-level DML/DDL facts from omni statements.
func (r *StatementPriorBackupCheckRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.UpdateStmt:
		r.statementInfoList = append(r.statementInfoList, statementInfo{
			statement: r.stmtText,
			table:     oracleTableReferenceFromObjectName(r.checkCtx.DBSchema.Name, n.Table, StatementTypeUpdate),
		})
	case *ast.DeleteStmt:
		r.statementInfoList = append(r.statementInfoList, statementInfo{
			statement: r.stmtText,
			table:     oracleTableReferenceFromObjectName(r.checkCtx.DBSchema.Name, n.Table, StatementTypeDelete),
		})
	default:
		if omniIsOracleDDL(node) {
			r.hasDDL = true
		}
	}
}

// GetAdviceList returns final prior-backup advice after all omni statements are processed.
func (r *StatementPriorBackupCheckRule) GetAdviceList() ([]*storepb.Advice, error) {
	r.handleSQLScriptExit()
	return r.BaseRule.GetAdviceList()
}

func (r *StatementPriorBackupCheckRule) handleSQLScriptExit() {
	var adviceList []*storepb.Advice

	if r.checkCtx.StatementsTotalSize > common.MaxSheetCheckSize {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        r.level,
			Title:         r.title,
			Content:       fmt.Sprintf("The size of the SQL statements exceeds the maximum limit of %d bytes for backup", common.MaxSheetCheckSize),
			Code:          code.BuiltinPriorBackupCheck.Int32(),
			StartPosition: nil,
		})
	}

	databaseName := common.BackupDatabaseNameOfEngine(storepb.Engine_ORACLE)
	if !advisor.DatabaseExists(r.ctx, r.checkCtx, databaseName) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        r.level,
			Title:         r.title,
			Content:       fmt.Sprintf("Need database %q to do prior backup but it does not exist", databaseName),
			Code:          code.DatabaseNotExists.Int32(),
			StartPosition: nil,
		})
		r.adviceList = append(r.adviceList, adviceList...)
		return
	}

	if r.hasDDL {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        r.level,
			Title:         r.title,
			Content:       "Prior backup cannot deal with mixed DDL and DML statements",
			Code:          int32(code.BuiltinPriorBackupCheck),
			StartPosition: nil,
		})
	}

	groupByTable := make(map[string][]statementInfo)
	for _, item := range r.statementInfoList {
		if item.table == nil {
			continue
		}
		key := fmt.Sprintf("%s.%s", item.table.Schema, item.table.Table)
		groupByTable[key] = append(groupByTable[key], item)
	}

	// Check if the statement type is the same for all statements in the group.
	for key, list := range groupByTable {
		statementType := StatementTypeUnknown
		for _, item := range list {
			if statementType == StatementTypeUnknown {
				statementType = item.table.StatementType
			}
			if statementType != item.table.StatementType {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        r.level,
					Title:         r.title,
					Content:       fmt.Sprintf("Prior backup cannot handle mixed DML statements on the same table %q", key),
					Code:          code.BuiltinPriorBackupCheck.Int32(),
					StartPosition: nil,
				})
				break
			}
		}
	}

	r.adviceList = append(r.adviceList, adviceList...)
}

func oracleTableReferenceFromObjectName(databaseName string, name *ast.ObjectName, typ StatementType) *TableReference {
	if name == nil {
		return nil
	}
	schemaName := name.Schema
	hasSchema := schemaName != ""
	if schemaName == "" {
		schemaName = databaseName
	}
	return &TableReference{
		Database:      schemaName,
		HasSchema:     hasSchema,
		Schema:        schemaName,
		Table:         name.Name,
		StatementType: typ,
	}
}

func omniIsOracleDDL(node ast.Node) bool {
	switch node.(type) {
	case *ast.CreateTableStmt, *ast.AlterTableStmt, *ast.DropStmt, *ast.CreateIndexStmt,
		*ast.CreateViewStmt, *ast.TruncateStmt, *ast.CommentStmt:
		return true
	default:
		return false
	}
}
