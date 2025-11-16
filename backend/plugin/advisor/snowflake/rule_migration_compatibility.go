// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*MigrationCompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SchemaRuleSchemaBackwardCompatibility, &MigrationCompatibilityAdvisor{})
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

	rule := NewMigrationCompatibilityRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase)
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// MigrationCompatibilityRule checks for migration compatibility issues.
type MigrationCompatibilityRule struct {
	BaseRule
	// normalizedNewCreateTableNameMap contain the new created table name in normalized format, e.g. "SNOWFLAKE.PUBLIC.TABLE", If there are IF NOT EXISTS, the value will be false.
	normalizedNewCreateTableNameMap map[string]bool
	// normalizedNewCreateSchemaNameMap contain the new created schema name in normalized format, e.g. "SNOWFLAKE.PUBLIC", If there are IF NOT EXISTS, the value will be false.
	normalizedNewCreateSchemaNameMap map[string]bool
	// normalizedNewCreateDatabaseNameMap contain the new created database name in normalized format, e.g. "SNOWFLAKE", If there are IF NOT EXISTS, the value will be false.
	normalizedNewCreateDatabaseNameMap map[string]bool

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
		currentDatabase:                    currentDatabase,
		normalizedNewCreateTableNameMap:    make(map[string]bool),
		normalizedNewCreateSchemaNameMap:   make(map[string]bool),
		normalizedNewCreateDatabaseNameMap: make(map[string]bool),
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
	case "Create_table_as_select":
		r.enterCreateTableAsSelect(ctx.(*parser.Create_table_as_selectContext))
	case NodeTypeCreateSchema:
		r.enterCreateSchema(ctx.(*parser.Create_schemaContext))
	case NodeTypeCreateDatabase:
		r.enterCreateDatabase(ctx.(*parser.Create_databaseContext))
	case NodeTypeDropTable:
		r.enterDropTable(ctx.(*parser.Drop_tableContext))
	case NodeTypeDropSchema:
		r.enterDropSchema(ctx.(*parser.Drop_schemaContext))
	case NodeTypeDropDatabase:
		r.enterDropDatabase(ctx.(*parser.Drop_databaseContext))
	case NodeTypeAlterTable:
		r.enterAlterTable(ctx.(*parser.Alter_tableContext))
	default:
		// Other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*MigrationCompatibilityRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *MigrationCompatibilityRule) enterCreateTable(ctx *parser.Create_tableContext) {
	normalizedFullTableName := snowsqlparser.NormalizeSnowSQLObjectName(ctx.Object_name(), r.currentDatabase, "PUBLIC")
	r.normalizedNewCreateTableNameMap[normalizedFullTableName] = ctx.If_not_exists() == nil
}

func (r *MigrationCompatibilityRule) enterCreateTableAsSelect(ctx *parser.Create_table_as_selectContext) {
	normalizedFullTableName := snowsqlparser.NormalizeSnowSQLObjectName(ctx.Object_name(), r.currentDatabase, "PUBLIC")
	r.normalizedNewCreateTableNameMap[normalizedFullTableName] = ctx.If_not_exists() == nil
}

func (r *MigrationCompatibilityRule) enterCreateSchema(ctx *parser.Create_schemaContext) {
	normalizedFullSchemaName := snowsqlparser.NormalizeSnowSQLSchemaName(ctx.Schema_name(), r.currentDatabase)
	r.normalizedNewCreateSchemaNameMap[normalizedFullSchemaName] = ctx.If_not_exists() == nil
}

func (r *MigrationCompatibilityRule) enterCreateDatabase(ctx *parser.Create_databaseContext) {
	normalizedFullDatabaseName := snowsqlparser.NormalizeSnowSQLObjectNamePart(ctx.Id_())
	r.normalizedNewCreateDatabaseNameMap[normalizedFullDatabaseName] = ctx.If_not_exists() == nil
}

func (r *MigrationCompatibilityRule) enterDropTable(ctx *parser.Drop_tableContext) {
	normalizedFullDropTableName := snowsqlparser.NormalizeSnowSQLObjectName(ctx.Object_name(), r.currentDatabase, "PUBLIC")
	mustNewCreate, ok := r.normalizedNewCreateTableNameMap[normalizedFullDropTableName]
	if ok && mustNewCreate {
		return
	}
	level := r.level
	if ok && !mustNewCreate {
		level = storepb.Advice_WARNING
	}
	r.AddAdvice(&storepb.Advice{
		Status:        level,
		Code:          code.CompatibilityDropTable.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("Drop table %q may cause incompatibility with the existing data and code", normalizedFullDropTableName),
		StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
	})
}

func (r *MigrationCompatibilityRule) enterDropSchema(ctx *parser.Drop_schemaContext) {
	normalizedFullDropSchemaName := snowsqlparser.NormalizeSnowSQLSchemaName(ctx.Schema_name(), r.currentDatabase)
	mustNewCreate, ok := r.normalizedNewCreateSchemaNameMap[normalizedFullDropSchemaName]
	if ok && mustNewCreate {
		return
	}
	level := r.level
	if ok && !mustNewCreate {
		level = storepb.Advice_WARNING
	}
	r.AddAdvice(&storepb.Advice{
		Status:        level,
		Code:          code.CompatibilityDropSchema.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("Drop schema %q may cause incompatibility with the existing data and code", normalizedFullDropSchemaName),
		StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
	})
}

func (r *MigrationCompatibilityRule) enterDropDatabase(ctx *parser.Drop_databaseContext) {
	normalizedFullDropDatabaseName := snowsqlparser.NormalizeSnowSQLObjectNamePart(ctx.Id_())
	mustNewCreate, ok := r.normalizedNewCreateDatabaseNameMap[normalizedFullDropDatabaseName]
	if ok && mustNewCreate {
		return
	}
	level := r.level
	if ok && !mustNewCreate {
		level = storepb.Advice_WARNING
	}
	r.AddAdvice(&storepb.Advice{
		Status:        level,
		Code:          code.CompatibilityDropDatabase.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("Drop database %q may cause incompatibility with the existing data and code", normalizedFullDropDatabaseName),
		StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
	})
}

func (r *MigrationCompatibilityRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	tableColumnAction := ctx.Table_column_action()
	if tableColumnAction == nil {
		return
	}
	dropColumn := tableColumnAction.DROP(0)
	if dropColumn == nil {
		return
	}
	allColumnName := tableColumnAction.Column_list().AllColumn_name()
	normalizedAllColumnNames := make([]string, 0, len(allColumnName))
	for _, columnName := range allColumnName {
		normalizedAllColumnNames = append(normalizedAllColumnNames, fmt.Sprintf("%q", snowsqlparser.NormalizeSnowSQLObjectNamePart(columnName.Id_())))
	}
	r.AddAdvice(&storepb.Advice{
		Status:        r.level,
		Code:          code.CompatibilityDropColumn.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("Drop column %s may cause incompatibility with the existing data and code", strings.Join(normalizedAllColumnNames, ",")),
		StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
	})
}
