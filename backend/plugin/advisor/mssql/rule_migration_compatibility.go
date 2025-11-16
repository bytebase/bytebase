package mssql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*MigrationCompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleSchemaBackwardCompatibility, &MigrationCompatibilityAdvisor{})
}

// MigrationCompatibilityAdvisor is the advisor checking for migration compatibility.
type MigrationCompatibilityAdvisor struct {
}

// Check checks for migration compatibility.
func (*MigrationCompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewMigrationCompatibilityRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// MigrationCompatibilityRule is the rule for migration compatibility.
type MigrationCompatibilityRule struct {
	BaseRule
	// normalizedLastCreateTableNameMap contain the last created table name in normalized format.
	normalizedNewCreateTableNameMap map[string]any
	// normalizedLastCreateSchemaNameMap contain the last created schema name in normalized format.
	normalizedNewCreateSchemaNameMap map[string]any
	// normalizedNewCreateDatabaseNameMap contain the new created database name in normalized format.
	normalizedNewCreateDatabaseNameMap map[string]any

	// currentDatabase is the current database name.
	currentDatabase string
}

// NewMigrationCompatibilityRule creates a new MigrationCompatibilityRule.
func NewMigrationCompatibilityRule(level storepb.Advice_Status, title string, currentDatabase string) *MigrationCompatibilityRule {
	return &MigrationCompatibilityRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		normalizedNewCreateTableNameMap:    make(map[string]any),
		normalizedNewCreateSchemaNameMap:   make(map[string]any),
		normalizedNewCreateDatabaseNameMap: make(map[string]any),
		currentDatabase:                    currentDatabase,
	}
}

// Name returns the rule name.
func (*MigrationCompatibilityRule) Name() string {
	return "MigrationCompatibilityRule"
}

// OnEnter is called when entering a parse tree node.
func (r *MigrationCompatibilityRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.enterCreateTable(ctx.(*parser.Create_tableContext))
	case "Create_schema":
		r.enterCreateSchema(ctx.(*parser.Create_schemaContext))
	case NodeTypeCreateDatabase:
		r.enterCreateDatabase(ctx.(*parser.Create_databaseContext))
	case NodeTypeDropTable:
		r.enterDropTable(ctx.(*parser.Drop_tableContext))
	case "Drop_schema":
		r.enterDropSchema(ctx.(*parser.Drop_schemaContext))
	case NodeTypeDropDatabase:
		r.enterDropDatabase(ctx.(*parser.Drop_databaseContext))
	case NodeTypeAlterTable:
		r.enterAlterTable(ctx.(*parser.Alter_tableContext))
	case "Execute_body":
		r.enterExecuteBody(ctx.(*parser.Execute_bodyContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*MigrationCompatibilityRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *MigrationCompatibilityRule) enterCreateTable(ctx *parser.Create_tableContext) {
	tableName := ctx.Table_name()
	if tableName == nil || tableName.GetTable() == nil {
		return
	}
	normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, r.currentDatabase, "dbo", false)
	r.normalizedNewCreateTableNameMap[normalizedTableName] = any(nil)
}

func (r *MigrationCompatibilityRule) enterCreateSchema(ctx *parser.Create_schemaContext) {
	var schemaName string
	if v := ctx.GetSchema_name(); v != nil {
		_, schemaName = tsqlparser.NormalizeTSQLIdentifier(v)
	} else {
		_, schemaName = tsqlparser.NormalizeTSQLIdentifier(ctx.GetOwner_name())
	}

	normalizedDatabaseSchemaName := fmt.Sprintf("%s.%s", r.currentDatabase, schemaName)
	r.normalizedNewCreateSchemaNameMap[normalizedDatabaseSchemaName] = any(nil)
}

func (r *MigrationCompatibilityRule) enterCreateDatabase(ctx *parser.Create_databaseContext) {
	_, databaseName := tsqlparser.NormalizeTSQLIdentifier(ctx.GetDatabase())
	r.normalizedNewCreateDatabaseNameMap[databaseName] = any(nil)
}

func (r *MigrationCompatibilityRule) enterDropTable(ctx *parser.Drop_tableContext) {
	allTableNames := ctx.AllTable_name()
	for _, tableName := range allTableNames {
		if tableName == nil || tableName.GetTable() == nil {
			continue
		}
		normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, r.currentDatabase, "dbo", false)
		if _, ok := r.normalizedNewCreateTableNameMap[normalizedTableName]; !ok {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          advisorcode.CompatibilityDropSchema.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Drop table %s may cause incompatibility with the existing data and code", normalizedTableName),
				StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
			})
		}
		delete(r.normalizedNewCreateTableNameMap, normalizedTableName)
	}
}

func (r *MigrationCompatibilityRule) enterDropSchema(ctx *parser.Drop_schemaContext) {
	schemaName := ctx.GetSchema_name()
	_, normalizedSchemaName := tsqlparser.NormalizeTSQLIdentifier(schemaName)
	normalizedSchemaName = fmt.Sprintf("%s.%s", r.currentDatabase, normalizedSchemaName)
	if _, ok := r.normalizedNewCreateSchemaNameMap[normalizedSchemaName]; !ok {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          advisorcode.CompatibilityDropSchema.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Drop schema %s may cause incompatibility with the existing data and code", normalizedSchemaName),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
	delete(r.normalizedNewCreateSchemaNameMap, normalizedSchemaName)
}

func (r *MigrationCompatibilityRule) enterDropDatabase(ctx *parser.Drop_databaseContext) {
	databaseName := ctx.GetDatabase_name_or_database_snapshot_name()
	_, normalizedDatabaseName := tsqlparser.NormalizeTSQLIdentifier(databaseName)
	if _, ok := r.normalizedNewCreateDatabaseNameMap[normalizedDatabaseName]; !ok {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          advisorcode.CompatibilityDropSchema.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Drop database %s may cause incompatibility with the existing data and code", normalizedDatabaseName),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
	delete(r.normalizedNewCreateDatabaseNameMap, normalizedDatabaseName)
}

func (r *MigrationCompatibilityRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	handleTableName := ctx.Table_name(0)
	normalizedHandleTableName := tsqlparser.NormalizeTSQLTableName(handleTableName, r.currentDatabase, "dbo", false)
	if _, ok := r.normalizedNewCreateTableNameMap[normalizedHandleTableName]; ok {
		return
	}

	if ctx.DROP() != nil && ctx.COLUMN() != nil {
		allDropColumns := ctx.AllId_()
		var allNormalizedDropColumnNames []string
		for _, dropColumn := range allDropColumns {
			_, normalizedDropColumnName := tsqlparser.NormalizeTSQLIdentifier(dropColumn)
			allNormalizedDropColumnNames = append(allNormalizedDropColumnNames, normalizedDropColumnName)
		}
		placeholder := strings.Join(allNormalizedDropColumnNames, ", ")
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          advisorcode.CompatibilityDropSchema.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Drop column %s may cause incompatibility with the existing data and code", placeholder),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.COLUMN().GetSymbol().GetLine()),
		})
		return
	}
	if len(ctx.AllALTER()) == 2 && ctx.COLUMN() != nil {
		normalizedColumnName := ""
		if ctx.Column_definition() != nil {
			_, normalizedColumnName = tsqlparser.NormalizeTSQLIdentifier(ctx.Column_definition().Id_())
		} else if ctx.Column_modifier() != nil {
			_, normalizedColumnName = tsqlparser.NormalizeTSQLIdentifier(ctx.Column_modifier().Id_())
		}

		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          advisorcode.CompatibilityAlterColumn.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Alter COLUMN %s may cause incompatibility with the existing data and code", normalizedColumnName),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.COLUMN().GetSymbol().GetLine()),
		})
		return
	}
	if v := ctx.Column_def_table_constraints(); v != nil {
		allColumnDefTableConstraints := v.AllColumn_def_table_constraint()
		for _, columnDefTableConstraint := range allColumnDefTableConstraints {
			code := advisorcode.Ok
			operation := ""
			tableConstraint := columnDefTableConstraint.Table_constraint()
			if tableConstraint == nil {
				continue
			}
			if tableConstraint.PRIMARY() != nil {
				code = advisorcode.CompatibilityAddPrimaryKey
				operation = "Add PRIMARY KEY"
			}
			if tableConstraint.UNIQUE() != nil {
				code = advisorcode.CompatibilityAddUniqueKey
				operation = "Add UNIQUE KEY"
			}
			if tableConstraint.Check_constraint() != nil {
				code = advisorcode.CompatibilityAddCheck
				operation = "Add CHECK"
			}
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("%s may cause incompatibility with the existing data and code", operation),
				StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
			})
		}
		return
	}
	if ctx.WITH() != nil && ctx.NOCHECK() != nil {
		if ctx.FOREIGN() != nil {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          advisorcode.CompatibilityAddForeignKey.Int32(),
				Title:         r.title,
				Content:       "Add FOREIGN KEY WITH NO CHECK may cause incompatibility with the existing data and code",
				StartPosition: common.ConvertANTLRLineToPosition(ctx.FOREIGN().GetSymbol().GetLine()),
			})
			return
		}
		if len(ctx.AllCHECK()) == 1 {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          advisorcode.CompatibilityAddForeignKey.Int32(),
				Title:         r.title,
				Content:       "Add CHECK WITH NO CHECK may cause incompatibility with the existing data and code",
				StartPosition: common.ConvertANTLRLineToPosition(ctx.CHECK(0).GetSymbol().GetLine()),
			})
			return
		}
	}
}

// enterExecuteBody is called when production execute_body is entered.
func (r *MigrationCompatibilityRule) enterExecuteBody(ctx *parser.Execute_bodyContext) {
	if ctx.Func_proc_name_server_database_schema() == nil {
		return
	}
	if ctx.Func_proc_name_server_database_schema().Func_proc_name_database_schema() == nil {
		return
	}
	if ctx.Func_proc_name_server_database_schema().Func_proc_name_database_schema().Func_proc_name_schema() == nil {
		return
	}
	if ctx.Func_proc_name_server_database_schema().Func_proc_name_database_schema().Func_proc_name_schema().GetSchema() != nil {
		return
	}

	v := ctx.Func_proc_name_server_database_schema().Func_proc_name_database_schema().Func_proc_name_schema().GetProcedure()
	_, normalizedProcedureName := tsqlparser.NormalizeTSQLIdentifier(v)
	if normalizedProcedureName != "sp_rename" {
		return
	}

	unnamedArguments := tsqlparser.FlattenExecuteStatementArgExecuteStatementArgUnnamed(ctx.Execute_statement_arg())

	firstArgument := unnamedArguments[0]
	if firstArgument == nil {
		return
	}
	if firstArgument.Execute_parameter() == nil {
		return
	}
	if firstArgument.Execute_parameter().Constant() == nil {
		return
	}
	if firstArgument.Execute_parameter().Constant().STRING() == nil {
		return
	}
	r.AddAdvice(&storepb.Advice{
		Status:        r.level,
		Code:          advisorcode.CompatibilityRenameTable.Int32(),
		Title:         r.title,
		Content:       "sp_rename may cause incompatibility with the existing data and code, and break scripts and stored procedures.",
		StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
	})
}
