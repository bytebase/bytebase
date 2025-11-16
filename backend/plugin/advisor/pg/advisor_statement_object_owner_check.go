package pg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*StatementObjectOwnerCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementObjectOwnerCheck, &StatementObjectOwnerCheckAdvisor{})
}

const (
	pgDatabaseOwner = "pg_database_owner"
)

type StatementObjectOwnerCheckAdvisor struct {
}

func (*StatementObjectOwnerCheckAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	dbMetadata := model.NewDatabaseMetadata(checkCtx.DBSchema, checkCtx.IsObjectCaseSensitive, true /* IsDetailCaseSensitive */)
	currentRole := checkCtx.DBSchema.Owner
	if !checkCtx.UsePostgresDatabaseOwner {
		currentRole, err = getCurrentUser(ctx, checkCtx.Driver)
		if err != nil {
			slog.Debug("Failed to get current user", log.BBError(err))
			currentRole = checkCtx.DBSchema.Owner
		}
	}

	rule := &statementObjectOwnerCheckRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		dbMetadata:  dbMetadata,
		currentRole: currentRole,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type statementObjectOwnerCheckRule struct {
	BaseRule
	dbMetadata  *model.DatabaseMetadata
	currentRole string
}

// Name returns the rule name.
func (*statementObjectOwnerCheckRule) Name() string {
	return "statement.object-owner-check"
}

// OnEnter is called when the parser enters a rule context.
func (r *statementObjectOwnerCheckRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Variablesetstmt":
		r.handleVariablesetstmt(ctx.(*parser.VariablesetstmtContext))
	case "Alterseqstmt":
		r.handleAlterseqstmt(ctx.(*parser.AlterseqstmtContext))
	case "Altertablestmt":
		r.handleAltertablestmt(ctx.(*parser.AltertablestmtContext))
	case "Altertypestmt":
		r.handleAltertypestmt(ctx.(*parser.AltertypestmtContext))
	case "Createextensionstmt":
		r.handleCreateextensionstmt(ctx.(*parser.CreateextensionstmtContext))
	case "Createfunctionstmt":
		r.handleCreatefunctionstmt(ctx.(*parser.CreatefunctionstmtContext))
	case "Indexstmt":
		r.handleIndexstmt(ctx.(*parser.IndexstmtContext))
	case "Createschemastmt":
		r.handleCreateschemastmt(ctx.(*parser.CreateschemastmtContext))
	case "Createseqstmt":
		r.handleCreateseqstmt(ctx.(*parser.CreateseqstmtContext))
	case "Createstmt":
		r.handleCreatestmt(ctx.(*parser.CreatestmtContext))
	case "Createtrigstmt":
		r.handleCreatetrigstmt(ctx.(*parser.CreatetrigstmtContext))
	case "Definestmt":
		r.handleDefinestmt(ctx.(*parser.DefinestmtContext))
	case "Creatematviewstmt":
		r.handleCreatematviewstmt(ctx.(*parser.CreatematviewstmtContext))
	case "Removefuncstmt":
		r.handleRemovefuncstmt(ctx.(*parser.RemovefuncstmtContext))
	case "Dropstmt":
		r.handleDropstmt(ctx.(*parser.DropstmtContext))
	case "Renamestmt":
		r.handleRenamestmt(ctx.(*parser.RenamestmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*statementObjectOwnerCheckRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleVariablesetstmt handles SET ROLE statements
func (*statementObjectOwnerCheckRule) handleVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is SET ROLE
	// Note: This is a simplified version that doesn't extract and update the role
	// The actual role name extraction would require parsing the var_list
	// For now, we skip updating currentRole from SET ROLE to keep it simple
}

func (r *statementObjectOwnerCheckRule) checkSchemaOwnership(schemaName string, line int) {
	if schemaName == "" {
		schemaName = defaultSchema
	}

	schemaMeta := r.dbMetadata.GetSchema(schemaName)
	if schemaMeta == nil {
		return
	}

	owner := schemaMeta.GetOwner()
	if owner == pgDatabaseOwner {
		owner = r.dbMetadata.GetOwner()
	}
	if owner != r.currentRole {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Title:   r.title,
			Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, r.currentRole),
			Code:    code.StatementObjectOwnerCheck.Int32(),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}

func (r *statementObjectOwnerCheckRule) checkTableOwnership(schemaName, tableName string, line int) {
	if schemaName == "" {
		schemaName = defaultSchema
	}

	schemaMeta := r.dbMetadata.GetSchema(schemaName)
	if schemaMeta == nil {
		return
	}

	tableMeta := schemaMeta.GetTable(tableName)
	if tableMeta == nil {
		return
	}

	owner := tableMeta.GetOwner()
	if owner == pgDatabaseOwner {
		owner = r.dbMetadata.GetOwner()
	}
	if owner != r.currentRole {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Title:   r.title,
			Content: fmt.Sprintf("Table \"%s\" is owned by \"%s\", but the current role is \"%s\".", tableName, owner, r.currentRole),
			Code:    code.StatementObjectOwnerCheck.Int32(),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}

// handleAlterseqstmt handles ALTER SEQUENCE statements
func (r *statementObjectOwnerCheckRule) handleAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	qualName := ctx.Qualified_name()
	if qualName == nil {
		return
	}

	schemaName := extractSchemaName(qualName)
	r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// handleAltertablestmt handles ALTER TABLE statements
func (r *statementObjectOwnerCheckRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	qualName := ctx.Relation_expr().Qualified_name()
	schemaName := extractSchemaName(qualName)
	tableName := extractTableName(qualName)

	r.checkTableOwnership(schemaName, tableName, ctx.GetStart().GetLine())
}

// handleAltertypestmt handles ALTER TYPE statements
func (r *statementObjectOwnerCheckRule) handleAltertypestmt(ctx *parser.AltertypestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Any_name() == nil {
		return
	}

	anyName := ctx.Any_name()
	parts := pg.NormalizePostgreSQLAnyName(anyName)

	var schemaName string
	if len(parts) >= 2 {
		schemaName = parts[0]
	}

	r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// handleCreateextensionstmt handles CREATE EXTENSION statements
func (r *statementObjectOwnerCheckRule) handleCreateextensionstmt(ctx *parser.CreateextensionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Extract schema name from create_extension_opt_list
	schemaName := defaultSchema
	if optList := ctx.Create_extension_opt_list(); optList != nil {
		for _, opt := range optList.AllCreate_extension_opt_item() {
			if opt.SCHEMA() != nil && opt.Name() != nil {
				schemaName = opt.Name().GetText()
				break
			}
		}
	}

	r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// handleCreatefunctionstmt handles CREATE FUNCTION statements
func (r *statementObjectOwnerCheckRule) handleCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Func_name() == nil {
		return
	}

	schemaName := r.extractSchemaFromFuncName(ctx.Func_name())
	r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// handleIndexstmt handles CREATE INDEX statements
func (r *statementObjectOwnerCheckRule) handleIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	qualName := ctx.Relation_expr().Qualified_name()
	schemaName := extractSchemaName(qualName)

	r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// handleCreateschemastmt handles CREATE SCHEMA statements
func (r *statementObjectOwnerCheckRule) handleCreateschemastmt(ctx *parser.CreateschemastmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	owner := r.dbMetadata.GetOwner()
	if owner != r.currentRole {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Title:   r.title,
			Content: fmt.Sprintf("Database \"%s\" is owned by \"%s\", but the current role is \"%s\".", r.dbMetadata.GetName(), owner, r.currentRole),
			Code:    code.StatementObjectOwnerCheck.Int32(),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// handleCreateseqstmt handles CREATE SEQUENCE statements
func (r *statementObjectOwnerCheckRule) handleCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Qualified_name() == nil {
		return
	}

	qualName := ctx.Qualified_name()
	schemaName := extractSchemaName(qualName)

	r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// handleCreatestmt handles CREATE TABLE statements
func (r *statementObjectOwnerCheckRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	allQualNames := ctx.AllQualified_name()
	if len(allQualNames) == 0 {
		return
	}

	qualName := allQualNames[0]
	schemaName := extractSchemaName(qualName)

	r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// handleCreatetrigstmt handles CREATE TRIGGER statements
func (r *statementObjectOwnerCheckRule) handleCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Qualified_name() == nil {
		return
	}

	qualName := ctx.Qualified_name()
	schemaName := extractSchemaName(qualName)

	r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// handleDefinestmt handles CREATE TYPE statements (via DEFINE)
func (r *statementObjectOwnerCheckRule) handleDefinestmt(ctx *parser.DefinestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is a CREATE TYPE statement
	if ctx.TYPE_P() == nil {
		return
	}

	// Any_name(0) gets the first any_name
	if ctx.Any_name(0) == nil {
		return
	}

	anyName := ctx.Any_name(0)
	parts := pg.NormalizePostgreSQLAnyName(anyName)

	var schemaName string
	if len(parts) >= 2 {
		schemaName = parts[0]
	}

	r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// handleCreatematviewstmt handles CREATE VIEW / MATERIALIZED VIEW statements
func (r *statementObjectOwnerCheckRule) handleCreatematviewstmt(ctx *parser.CreatematviewstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Create_mv_target() == nil || ctx.Create_mv_target().Qualified_name() == nil {
		return
	}

	qualName := ctx.Create_mv_target().Qualified_name()
	schemaName := extractSchemaName(qualName)

	r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// handleRemovefuncstmt handles DROP FUNCTION statements
func (r *statementObjectOwnerCheckRule) handleRemovefuncstmt(ctx *parser.RemovefuncstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Function_with_argtypes_list() == nil {
		return
	}

	for _, funcWithArgs := range ctx.Function_with_argtypes_list().AllFunction_with_argtypes() {
		if funcWithArgs.Func_name() == nil {
			continue
		}

		schemaName := r.extractSchemaFromFuncName(funcWithArgs.Func_name())
		r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
	}
}

// handleDropstmt handles various DROP statements
// Note: PostgreSQL DROP statements don't easily expose the object type in ANTLR,
// so we do a best-effort check by examining the any_name_list
func (r *statementObjectOwnerCheckRule) handleDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// For DROP INDEX, check schema ownership
	if ctx.INDEX() != nil {
		r.handleDropWithAnyNameList(ctx, false)
		return
	}

	// For other DROP statements, check schema ownership
	if ctx.Any_name_list() != nil {
		r.handleDropWithAnyNameList(ctx, false)
	}
}

func (r *statementObjectOwnerCheckRule) handleDropWithAnyNameList(ctx *parser.DropstmtContext, checkTable bool) {
	if ctx.Any_name_list() == nil {
		return
	}

	for _, anyName := range ctx.Any_name_list().AllAny_name() {
		parts := pg.NormalizePostgreSQLAnyName(anyName)
		if len(parts) == 0 {
			continue
		}

		var schemaName, objectName string
		if len(parts) == 1 {
			schemaName = ""
			objectName = parts[0]
		} else {
			schemaName = parts[0]
			objectName = parts[1]
		}

		if checkTable && objectName != "" {
			r.checkTableOwnership(schemaName, objectName, ctx.GetStart().GetLine())
		} else {
			r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
		}
	}
}

// handleRenamestmt handles RENAME statements (including ALTER INDEX RENAME)
func (r *statementObjectOwnerCheckRule) handleRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is an ALTER SCHEMA RENAME
	if ctx.SCHEMA() != nil && ctx.Name(0) != nil {
		schemaName := ctx.Name(0).GetText()
		r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
	} else if ctx.INDEX() != nil && ctx.Qualified_name() != nil {
		// ALTER INDEX RENAME
		qualName := ctx.Qualified_name()
		schemaName := extractSchemaName(qualName)
		r.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
	}
}

// Helper method to extract schema name from function name
func (*statementObjectOwnerCheckRule) extractSchemaFromFuncName(funcName parser.IFunc_nameContext) string {
	if funcName.Type_function_name() != nil {
		// Simple function name without schema
		return ""
	}
	if funcName.Indirection() != nil {
		// Qualified function name with schema
		parts := []string{}
		if funcName.Colid() != nil {
			parts = append(parts, pg.NormalizePostgreSQLColid(funcName.Colid()))
		}
		for _, attr := range funcName.Indirection().AllIndirection_el() {
			if attr.Attr_name() != nil {
				parts = append(parts, attr.Attr_name().GetText())
			}
		}
		if len(parts) >= 2 {
			return parts[0]
		}
	}
	return ""
}

func getCurrentUser(ctx context.Context, driver *sql.DB) (string, error) {
	var user string
	err := driver.QueryRowContext(ctx, "SELECT CURRENT_USER").Scan(&user)
	if err != nil {
		return "", err
	}
	return user, nil
}
