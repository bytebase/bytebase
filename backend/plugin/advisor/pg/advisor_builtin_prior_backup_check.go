package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

const (
	defaultSchema = "public"
)

var (
	_ advisor.Advisor = (*BuiltinPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, &BuiltinPriorBackupCheckAdvisor{})
}

// BuiltinPriorBackupCheckAdvisor is the advisor checking for disallow mix DDL and DML.
type BuiltinPriorBackupCheckAdvisor struct {
}

// Check checks for disallow mix DDL and DML.
func (*BuiltinPriorBackupCheckAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	if !checkCtx.EnablePriorBackup {
		return nil, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice

	// Check for DDL statements in DML context
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		baseLine := stmt.BaseLine()
		if omniIsDDLStatement(node) {
			stmtText := omniTrimmedStmtText(stmt.Text)
			startLine := omniContentStartLine(stmt.Text)
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: fmt.Sprintf("Data change can only run DML, \"%s\" is not DML", stmtText),
				Code:    code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: &storepb.Position{
					Line:   int32(baseLine) + startLine,
					Column: 0,
				},
			})
		}
	}

	// Check if backup schema exists
	schemaName := common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)
	if checkCtx.OriginalMetadata.GetSchemaMetadata(schemaName) == nil {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       fmt.Sprintf("Need schema %q to do prior backup but it does not exist", schemaName),
			Code:          code.SchemaNotExists.Int32(),
			StartPosition: nil,
		})
	}

	// Check statement type consistency for each table
	statementInfoList := prepareTransformation(checkCtx.ParsedStatements)

	groupByTable := make(map[string][]statementInfo)
	for _, item := range statementInfoList {
		key := fmt.Sprintf("%s.%s", item.table.Schema, item.table.Table)
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
					StartPosition: nil,
				})
				break
			}
		}
	}

	return adviceList, nil
}

// omniIsDDLStatement checks if a node is a DDL statement.
func omniIsDDLStatement(node ast.Node) bool {
	switch node.(type) {
	// CREATE statements
	case *ast.CreateStmt,
		*ast.CreateTableAsStmt,
		*ast.CreateDomainStmt,
		*ast.CreateExtensionStmt,
		*ast.CreateFdwStmt,
		*ast.CreateForeignServerStmt,
		*ast.CreateForeignTableStmt,
		*ast.CreateFunctionStmt,
		*ast.CreateRoleStmt,
		*ast.CreatePolicyStmt,
		*ast.CreatePLangStmt,
		*ast.CreateSchemaStmt,
		*ast.CreateSeqStmt,
		*ast.CreateSubscriptionStmt,
		*ast.CreateStatsStmt,
		*ast.CreateTableSpaceStmt,
		*ast.CreateTransformStmt,
		*ast.CreateTrigStmt,
		*ast.CreateEventTrigStmt,
		*ast.CreateUserMappingStmt,
		*ast.CreatedbStmt,
		*ast.CreateAmStmt,
		*ast.CreatePublicationStmt,
		*ast.CreateOpClassStmt,
		*ast.CreateOpFamilyStmt,
		*ast.CreateCastStmt,
		*ast.CreateConversionStmt,
		*ast.CreateRangeStmt,
		*ast.CompositeTypeStmt,
		*ast.CreateEnumStmt,
		*ast.IndexStmt,
		*ast.ViewStmt,
		// ALTER statements
		*ast.AlterEventTrigStmt,
		*ast.AlterCollationStmt,
		*ast.AlterDatabaseStmt,
		*ast.AlterDatabaseSetStmt,
		*ast.AlterDefaultPrivilegesStmt,
		*ast.AlterDomainStmt,
		*ast.AlterEnumStmt,
		*ast.AlterExtensionStmt,
		*ast.AlterExtensionContentsStmt,
		*ast.AlterFdwStmt,
		*ast.AlterForeignServerStmt,
		*ast.AlterFunctionStmt,
		*ast.AlterObjectDependsStmt,
		*ast.AlterObjectSchemaStmt,
		*ast.AlterOwnerStmt,
		*ast.AlterOperatorStmt,
		*ast.AlterTypeStmt,
		*ast.AlterPolicyStmt,
		*ast.AlterSeqStmt,
		*ast.AlterSystemStmt,
		*ast.AlterTableStmt,
		*ast.AlterTableSpaceOptionsStmt,
		*ast.AlterPublicationStmt,
		*ast.AlterRoleSetStmt,
		*ast.AlterRoleStmt,
		*ast.AlterSubscriptionStmt,
		*ast.AlterStatsStmt,
		*ast.AlterTSConfigurationStmt,
		*ast.AlterTSDictionaryStmt,
		*ast.AlterUserMappingStmt,
		*ast.AlterOpFamilyStmt,
		// DROP statements
		*ast.DropStmt,
		*ast.DropOwnedStmt,
		*ast.DropSubscriptionStmt,
		*ast.DropTableSpaceStmt,
		*ast.DropRoleStmt,
		*ast.DropUserMappingStmt,
		*ast.DropdbStmt,
		// Other DDL statements
		*ast.TruncateStmt,
		*ast.RenameStmt,
		*ast.CommentStmt,
		*ast.DefineStmt,
		*ast.ReindexStmt,
		*ast.ClusterStmt,
		*ast.RefreshMatViewStmt,
		*ast.RuleStmt,
		*ast.SecLabelStmt,
		*ast.ReassignOwnedStmt:
		return true
	default:
		return false
	}
}

// StatementType represents the type of DML statement
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
	offset    int
	statement string
	table     *TableReference
}

func prepareTransformation(parsedStatements []base.ParsedStatement) []statementInfo {
	var dmls []statementInfo
	offset := 0
	for _, stmt := range parsedStatements {
		if stmt.AST == nil {
			offset++
			continue
		}
		node, ok := pgparser.GetOmniNode(stmt.AST)
		if !ok {
			offset++
			continue
		}

		switch n := node.(type) {
		case *ast.UpdateStmt:
			table := omniExtractTableReferenceFromRangeVar(n.Relation)
			if table != nil {
				table.StatementType = StatementTypeUpdate
				dmls = append(dmls, statementInfo{
					offset:    offset,
					statement: stmt.Text,
					table:     table,
				})
			}
		case *ast.DeleteStmt:
			// DeleteStmt uses UsingClause for FROM but Relation for the target table
			table := omniExtractTableReferenceFromRangeVar(n.Relation)
			if table != nil {
				table.StatementType = StatementTypeDelete
				dmls = append(dmls, statementInfo{
					offset:    offset,
					statement: stmt.Text,
					table:     table,
				})
			}
		default:
		}

		offset++
	}
	return dmls
}

func omniExtractTableReferenceFromRangeVar(rv *ast.RangeVar) *TableReference {
	if rv == nil {
		return nil
	}

	table := TableReference{}
	if rv.Catalogname != "" {
		table.Database = rv.Catalogname
	}
	if rv.Schemaname != "" {
		table.Schema = rv.Schemaname
	} else {
		table.Schema = defaultSchema
	}
	table.Table = rv.Relname
	if rv.Alias != nil {
		table.Alias = rv.Alias.Aliasname
	}

	if table.Table == "" {
		return nil
	}
	return &table
}

// omniTrimmedStmtText returns the statement text with leading/trailing whitespace
// and trailing semicolons removed.
func omniTrimmedStmtText(text string) string {
	result := text
	// Trim leading/trailing whitespace
	for len(result) > 0 && (result[0] == ' ' || result[0] == '\t' || result[0] == '\n' || result[0] == '\r') {
		result = result[1:]
	}
	for len(result) > 0 {
		last := result[len(result)-1]
		if last == ' ' || last == '\t' || last == '\n' || last == '\r' || last == ';' {
			result = result[:len(result)-1]
		} else {
			break
		}
	}
	return result
}

// omniContentStartLine returns the 1-based line number of the first non-whitespace
// character in the text.
func omniContentStartLine(text string) int32 {
	line := int32(1)
	for _, c := range text {
		if c == '\n' {
			line++
		} else if c != ' ' && c != '\t' && c != '\r' {
			break
		}
	}
	return line
}
