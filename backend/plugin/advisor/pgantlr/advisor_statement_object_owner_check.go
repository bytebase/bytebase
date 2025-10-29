package pgantlr

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

	checker := &statementObjectOwnerCheckChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		dbMetadata:                   dbMetadata,
		currentRole:                  currentRole,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type statementObjectOwnerCheckChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList  []*storepb.Advice
	level       storepb.Advice_Status
	title       string
	dbMetadata  *model.DatabaseMetadata
	currentRole string
}

// EnterVariablesetstmt handles SET ROLE statements
func (*statementObjectOwnerCheckChecker) EnterVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is SET ROLE
	// Note: This is a simplified version that doesn't extract and update the role
	// The actual role name extraction would require parsing the var_list
	// For now, we skip updating currentRole from SET ROLE to keep it simple
}

func (c *statementObjectOwnerCheckChecker) checkSchemaOwnership(schemaName string, line int) {
	if schemaName == "" {
		schemaName = defaultSchema
	}

	schemaMeta := c.dbMetadata.GetSchema(schemaName)
	if schemaMeta == nil {
		return
	}

	owner := schemaMeta.GetOwner()
	if owner == pgDatabaseOwner {
		owner = c.dbMetadata.GetOwner()
	}
	if owner != c.currentRole {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Title:   c.title,
			Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, c.currentRole),
			Code:    advisor.StatementObjectOwnerCheck.Int32(),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}

func (c *statementObjectOwnerCheckChecker) checkTableOwnership(schemaName, tableName string, line int) {
	if schemaName == "" {
		schemaName = defaultSchema
	}

	schemaMeta := c.dbMetadata.GetSchema(schemaName)
	if schemaMeta == nil {
		return
	}

	tableMeta := schemaMeta.GetTable(tableName)
	if tableMeta == nil {
		return
	}

	owner := tableMeta.GetOwner()
	if owner == pgDatabaseOwner {
		owner = c.dbMetadata.GetOwner()
	}
	if owner != c.currentRole {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Title:   c.title,
			Content: fmt.Sprintf("Table \"%s\" is owned by \"%s\", but the current role is \"%s\".", tableName, owner, c.currentRole),
			Code:    advisor.StatementObjectOwnerCheck.Int32(),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}

// EnterAlterseqstmt handles ALTER SEQUENCE statements
func (c *statementObjectOwnerCheckChecker) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	qualName := ctx.Qualified_name()
	if qualName == nil {
		return
	}

	schemaName := extractSchemaName(qualName)
	c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// EnterAltertablestmt handles ALTER TABLE statements
func (c *statementObjectOwnerCheckChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	qualName := ctx.Relation_expr().Qualified_name()
	schemaName := extractSchemaName(qualName)
	tableName := extractTableName(qualName)

	c.checkTableOwnership(schemaName, tableName, ctx.GetStart().GetLine())
}

// EnterAltertypestmt handles ALTER TYPE statements
func (c *statementObjectOwnerCheckChecker) EnterAltertypestmt(ctx *parser.AltertypestmtContext) {
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

	c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// EnterCreateextensionstmt handles CREATE EXTENSION statements
func (c *statementObjectOwnerCheckChecker) EnterCreateextensionstmt(ctx *parser.CreateextensionstmtContext) {
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

	c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// EnterCreatefunctionstmt handles CREATE FUNCTION statements
func (c *statementObjectOwnerCheckChecker) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Func_name() == nil {
		return
	}

	schemaName := c.extractSchemaFromFuncName(ctx.Func_name())
	c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// EnterIndexstmt handles CREATE INDEX statements
func (c *statementObjectOwnerCheckChecker) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	qualName := ctx.Relation_expr().Qualified_name()
	schemaName := extractSchemaName(qualName)

	c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// EnterCreateschemastmt handles CREATE SCHEMA statements
func (c *statementObjectOwnerCheckChecker) EnterCreateschemastmt(ctx *parser.CreateschemastmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	owner := c.dbMetadata.GetOwner()
	if owner != c.currentRole {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Title:   c.title,
			Content: fmt.Sprintf("Database \"%s\" is owned by \"%s\", but the current role is \"%s\".", c.dbMetadata.GetName(), owner, c.currentRole),
			Code:    advisor.StatementObjectOwnerCheck.Int32(),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// EnterCreateseqstmt handles CREATE SEQUENCE statements
func (c *statementObjectOwnerCheckChecker) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Qualified_name() == nil {
		return
	}

	qualName := ctx.Qualified_name()
	schemaName := extractSchemaName(qualName)

	c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// EnterCreatestmt handles CREATE TABLE statements
func (c *statementObjectOwnerCheckChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	allQualNames := ctx.AllQualified_name()
	if len(allQualNames) == 0 {
		return
	}

	qualName := allQualNames[0]
	schemaName := extractSchemaName(qualName)

	c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// EnterCreatetrigstmt handles CREATE TRIGGER statements
func (c *statementObjectOwnerCheckChecker) EnterCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Qualified_name() == nil {
		return
	}

	qualName := ctx.Qualified_name()
	schemaName := extractSchemaName(qualName)

	c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// EnterDefinestmt handles CREATE TYPE statements (via DEFINE)
func (c *statementObjectOwnerCheckChecker) EnterDefinestmt(ctx *parser.DefinestmtContext) {
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

	c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// EnterCreatematviewstmt handles CREATE VIEW / MATERIALIZED VIEW statements
func (c *statementObjectOwnerCheckChecker) EnterCreatematviewstmt(ctx *parser.CreatematviewstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Create_mv_target() == nil || ctx.Create_mv_target().Qualified_name() == nil {
		return
	}

	qualName := ctx.Create_mv_target().Qualified_name()
	schemaName := extractSchemaName(qualName)

	c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
}

// EnterRemovefuncstmt handles DROP FUNCTION statements
func (c *statementObjectOwnerCheckChecker) EnterRemovefuncstmt(ctx *parser.RemovefuncstmtContext) {
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

		schemaName := c.extractSchemaFromFuncName(funcWithArgs.Func_name())
		c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
	}
}

// EnterDropstmt handles various DROP statements
// Note: PostgreSQL DROP statements don't easily expose the object type in ANTLR,
// so we do a best-effort check by examining the any_name_list
func (c *statementObjectOwnerCheckChecker) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// For DROP INDEX, check schema ownership
	if ctx.INDEX() != nil {
		c.handleDropWithAnyNameList(ctx, false)
		return
	}

	// For other DROP statements, check schema ownership
	if ctx.Any_name_list() != nil {
		c.handleDropWithAnyNameList(ctx, false)
	}
}

func (c *statementObjectOwnerCheckChecker) handleDropWithAnyNameList(ctx *parser.DropstmtContext, checkTable bool) {
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
			c.checkTableOwnership(schemaName, objectName, ctx.GetStart().GetLine())
		} else {
			c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
		}
	}
}

// EnterRenamestmt handles RENAME statements (including ALTER INDEX RENAME)
func (c *statementObjectOwnerCheckChecker) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is an ALTER SCHEMA RENAME
	if ctx.SCHEMA() != nil && ctx.Name(0) != nil {
		schemaName := ctx.Name(0).GetText()
		c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
	} else if ctx.INDEX() != nil && ctx.Qualified_name() != nil {
		// ALTER INDEX RENAME
		qualName := ctx.Qualified_name()
		schemaName := extractSchemaName(qualName)
		c.checkSchemaOwnership(schemaName, ctx.GetStart().GetLine())
	}
}

// Helper method to extract schema name from function name
func (*statementObjectOwnerCheckChecker) extractSchemaFromFuncName(funcName parser.IFunc_nameContext) string {
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
