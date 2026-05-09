package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mssql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

const (
	// The default schema is 'dbo' for MSSQL.
	// TODO(zp): We should support default schema in the future.
	defaultSchema = "dbo"
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, &StatementPriorBackupCheckAdvisor{})
}

type StatementPriorBackupCheckAdvisor struct {
}

func (*StatementPriorBackupCheckAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	if !checkCtx.EnablePriorBackup {
		return nil, nil
	}

	var adviceList []*storepb.Advice
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	if checkCtx.StatementsTotalSize > common.MaxSheetCheckSize {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       fmt.Sprintf("The size of statements in the sheet exceeds the limit of %d", common.MaxSheetCheckSize),
			Code:          code.BuiltinPriorBackupCheck.Int32(),
			StartPosition: &storepb.Position{Line: 1},
		})
	}

	databaseName := common.BackupDatabaseNameOfEngine(storepb.Engine_MSSQL)
	if !advisor.DatabaseExists(ctx, checkCtx, databaseName) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       fmt.Sprintf("Need database %q to do prior backup but it does not exist", databaseName),
			Code:          code.DatabaseNotExists.Int32(),
			StartPosition: &storepb.Position{Line: 1},
		})
		return adviceList, nil
	}

	// Use omni rule for DDL detection.
	ddlRule := &statementDisallowMixDMLOmniRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: title},
	}
	RunOmniRules(checkCtx.ParsedStatements, []OmniRule{ddlRule})

	if ddlRule.hasDDL {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       "Prior backup cannot deal with mixed DDL and DML statements",
			Code:          int32(code.BuiltinPriorBackupCheck),
			StartPosition: &storepb.Position{Line: 1},
		})
	}

	statementInfoList := prepareTransformation(checkCtx.DBSchema.Name, checkCtx.ParsedStatements)

	groupByTable := make(map[string][]statementInfo)
	for _, item := range statementInfoList {
		key := fmt.Sprintf("%s.%s.%s", item.table.Database, item.table.Schema, item.table.Table)
		groupByTable[key] = append(groupByTable[key], item)
	}

	// Check if the statement type is the same for all statements on the same table.
	for key, list := range groupByTable {
		statementType := StatementTypeUnknown
		for _, item := range list {
			if statementType == StatementTypeUnknown {
				statementType = item.table.StatementType
			}
			if statementType != item.table.StatementType {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("The statement type is not the same for all statements on the same table %q", key),
					Code:          code.BuiltinPriorBackupCheck.Int32(),
					StartPosition: &storepb.Position{Line: 1},
				})
				break
			}
		}
	}

	return adviceList, nil
}

// statementDisallowMixDMLOmniRule uses omni AST to detect DDL statements.
type statementDisallowMixDMLOmniRule struct {
	OmniBaseRule
	hasDDL bool
}

func (*statementDisallowMixDMLOmniRule) Name() string {
	return "StatementDisallowMixDMLOmniRule"
}

func (r *statementDisallowMixDMLOmniRule) OnStatement(node ast.Node) {
	if r.hasDDL {
		return
	}
	switch node.(type) {
	case *ast.CreateTableStmt,
		*ast.AlterTableStmt,
		*ast.DropStmt,
		*ast.TruncateStmt,
		*ast.CreateIndexStmt,
		*ast.CreateViewStmt,
		*ast.CreateFunctionStmt,
		*ast.CreateProcedureStmt,
		*ast.CreateSchemaStmt,
		*ast.CreateDatabaseStmt,
		*ast.CreateTriggerStmt,
		*ast.CreateTypeStmt,
		*ast.CreateSequenceStmt,
		*ast.AlterIndexStmt,
		*ast.AlterSchemaStmt,
		*ast.AlterDatabaseStmt,
		*ast.AlterSequenceStmt,
		*ast.RenameStmt:
		r.hasDDL = true
	default:
	}
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
	Schema        string
	Table         string
	Alias         string
	StatementType StatementType
}

type statementInfo struct {
	table *TableReference
}

func prepareTransformation(databaseName string, parsedStatements []base.ParsedStatement) []statementInfo {
	var dmls []statementInfo
	for _, stmt := range parsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := tsqlparser.GetOmniNode(stmt.AST)
		if !ok || node == nil {
			continue
		}

		var (
			table         *TableReference
			statementType StatementType
		)
		switch n := node.(type) {
		case *ast.UpdateStmt:
			table = resolveDMLTargetTable(n.Relation, n.FromClause, databaseName)
			statementType = StatementTypeUpdate
		case *ast.DeleteStmt:
			table = resolveDMLTargetTable(n.Relation, n.FromClause, databaseName)
			statementType = StatementTypeDelete
		default:
			continue
		}
		if table == nil || table.Table == "" {
			continue
		}
		table.StatementType = statementType
		dmls = append(dmls, statementInfo{
			table: table,
		})
	}

	return dmls
}

func resolveDMLTargetTable(relation ast.TableExpr, fromClause *ast.List, databaseName string) *TableReference {
	table := tableReferenceFromTableExpr(relation, databaseName, defaultSchema)
	if table == nil {
		return table
	}
	if fromClause != nil && table.Database == databaseName && table.Schema == defaultSchema {
		if physical := findPhysicalTableForAlias(fromClause, table); physical != nil {
			return physical
		}
	}
	return table
}

func tableReferenceFromTableExpr(expr ast.TableExpr, defaultDatabase, defaultSchema string) *TableReference {
	ref, ok := expr.(*ast.TableRef)
	if !ok {
		return nil
	}
	schemaName := defaultSchema
	if ref.Schema != "" {
		schemaName = ref.Schema
	}
	databaseName := defaultDatabase
	if ref.Database != "" {
		databaseName = ref.Database
	}
	return &TableReference{
		Database: databaseName,
		Schema:   schemaName,
		Table:    ref.Object,
		Alias:    ref.Alias,
	}
}

func findPhysicalTableForAlias(list *ast.List, table *TableReference) *TableReference {
	if list == nil || table == nil {
		return nil
	}
	for _, item := range list.Items {
		if result := findPhysicalTableForAliasInNode(item, table); result != nil {
			return result
		}
	}
	return nil
}

func findPhysicalTableForAliasInNode(node ast.Node, table *TableReference) *TableReference {
	switch n := node.(type) {
	case *ast.TableRef:
		if n.Alias != "" && n.Alias == table.Table {
			ref := tableReferenceFromTableExpr(n, table.Database, table.Schema)
			if ref == nil {
				return nil
			}
			ref.Alias = n.Alias
			return ref
		}
	case *ast.AliasedTableRef:
		if ref, ok := n.Table.(*ast.TableRef); ok && n.Alias == table.Table {
			result := tableReferenceFromTableExpr(ref, table.Database, table.Schema)
			if result == nil {
				return nil
			}
			result.Alias = n.Alias
			return result
		}
	case *ast.JoinClause:
		if result := findPhysicalTableForAliasInNode(n.Left, table); result != nil {
			return result
		}
		return findPhysicalTableForAliasInNode(n.Right, table)
	default:
	}
	return nil
}
