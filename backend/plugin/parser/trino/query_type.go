package trino

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/trino"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// StatementType is the type of the SQL statement.
type StatementType int

// Statement types
const (
	Unsupported StatementType = iota
	Select
	Explain
	Insert
	Update
	Delete
	Merge
	CreateTable
	CreateView
	AlterTable
	DropTable
	DropView
	CreateSchema
	DropSchema
	RenameTable
	CreateTableAsSelect
	Set
	Show
)

// queryTypeListener is a parser listener for determining the query type
type queryTypeListener struct {
	parser.BaseTrinoParserListener

	result           base.QueryType
	isExplainAnalyze bool
}

// getQueryType returns the type of the statement.
func getQueryType(node any) (base.QueryType, bool) {
	stmtCtx, ok := node.(*parser.SingleStatementContext)
	if !ok {
		return base.QueryTypeUnknown, false
	}

	// First check if this is an EXPLAIN statement by examining the statement directly
	if stmt := stmtCtx.Statement(); stmt != nil {
		switch stmt.(type) {
		case *parser.ExplainContext:
			return base.Explain, false
		case *parser.ExplainAnalyzeContext:
			return base.Select, true // EXPLAIN ANALYZE is treated as SELECT since it actually runs the query
		}
	}

	// For other statement types, use the listener
	listener := &queryTypeListener{
		result: base.QueryTypeUnknown,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, stmtCtx)

	return listener.result, listener.isExplainAnalyze
}

// EnterStatementDefault is called when entering a StatementDefaultContext rule.
func (l *queryTypeListener) EnterStatementDefault(ctx *parser.StatementDefaultContext) {
	// Regular SELECT query
	if containsSystemSchema(ctx.GetText()) {
		l.result = base.SelectInfoSchema
	} else {
		l.result = base.Select
	}
}

// EnterExplain is called when entering an ExplainContext rule.
func (l *queryTypeListener) EnterExplain(_ *parser.ExplainContext) {
	l.result = base.Explain
}

// EnterExplainAnalyze is called when entering an ExplainAnalyzeContext rule.
func (l *queryTypeListener) EnterExplainAnalyze(ctx *parser.ExplainAnalyzeContext) {
	// For EXPLAIN ANALYZE, we treat it as a SELECT query because it actually executes the query
	l.result = base.Select
	l.isExplainAnalyze = true

	// Check if this is an EXPLAIN ANALYZE of a query that accesses system tables
	if containsSystemSchema(ctx.GetText()) {
		l.result = base.SelectInfoSchema
	}
}

// EnterInsertInto is called when entering an InsertIntoContext rule.
func (l *queryTypeListener) EnterInsertInto(_ *parser.InsertIntoContext) {
	l.result = base.DML
}

// EnterUpdate is called when entering an UpdateContext rule.
func (l *queryTypeListener) EnterUpdate(_ *parser.UpdateContext) {
	l.result = base.DML
}

// EnterDelete is called when entering a DeleteContext rule.
func (l *queryTypeListener) EnterDelete(_ *parser.DeleteContext) {
	l.result = base.DML
}

// EnterMerge is called when entering a MergeContext rule.
func (l *queryTypeListener) EnterMerge(_ *parser.MergeContext) {
	l.result = base.DML
}

// EnterCreateTable is called when entering a CreateTableContext rule.
func (l *queryTypeListener) EnterCreateTable(_ *parser.CreateTableContext) {
	l.result = base.DDL
}

// EnterCreateTableAsSelect is called when entering a CreateTableAsSelectContext rule.
func (l *queryTypeListener) EnterCreateTableAsSelect(_ *parser.CreateTableAsSelectContext) {
	l.result = base.DDL
}

// EnterCreateView is called when entering a CreateViewContext rule.
func (l *queryTypeListener) EnterCreateView(_ *parser.CreateViewContext) {
	l.result = base.DDL
}

// EnterAddColumn is called when entering an AddColumnContext rule.
func (l *queryTypeListener) EnterAddColumn(_ *parser.AddColumnContext) {
	l.result = base.DDL
}

// EnterDropColumn is called when entering a DropColumnContext rule.
func (l *queryTypeListener) EnterDropColumn(_ *parser.DropColumnContext) {
	l.result = base.DDL
}

// EnterRenameColumn is called when entering a RenameColumnContext rule.
func (l *queryTypeListener) EnterRenameColumn(_ *parser.RenameColumnContext) {
	l.result = base.DDL
}

// EnterSetColumnType is called when entering a SetColumnTypeContext rule.
func (l *queryTypeListener) EnterSetColumnType(_ *parser.SetColumnTypeContext) {
	l.result = base.DDL
}

// EnterSetTableAuthorization is called when entering a SetTableAuthorizationContext rule.
func (l *queryTypeListener) EnterSetTableAuthorization(_ *parser.SetTableAuthorizationContext) {
	l.result = base.DDL
}

// EnterSetTableProperties is called when entering a SetTablePropertiesContext rule.
func (l *queryTypeListener) EnterSetTableProperties(_ *parser.SetTablePropertiesContext) {
	l.result = base.DDL
}

// EnterDropTable is called when entering a DropTableContext rule.
func (l *queryTypeListener) EnterDropTable(_ *parser.DropTableContext) {
	l.result = base.DDL
}

// EnterDropView is called when entering a DropViewContext rule.
func (l *queryTypeListener) EnterDropView(_ *parser.DropViewContext) {
	l.result = base.DDL
}

// EnterCreateSchema is called when entering a CreateSchemaContext rule.
func (l *queryTypeListener) EnterCreateSchema(_ *parser.CreateSchemaContext) {
	l.result = base.DDL
}

// EnterDropSchema is called when entering a DropSchemaContext rule.
func (l *queryTypeListener) EnterDropSchema(_ *parser.DropSchemaContext) {
	l.result = base.DDL
}

// EnterRenameTable is called when entering a RenameTableContext rule.
func (l *queryTypeListener) EnterRenameTable(_ *parser.RenameTableContext) {
	l.result = base.DDL
}

// EnterRenameView is called when entering a RenameViewContext rule.
func (l *queryTypeListener) EnterRenameView(_ *parser.RenameViewContext) {
	l.result = base.DDL
}

// EnterRenameSchema is called when entering a RenameSchemaContext rule.
func (l *queryTypeListener) EnterRenameSchema(_ *parser.RenameSchemaContext) {
	l.result = base.DDL
}

// EnterShowTables is called when entering a ShowTablesContext rule.
func (l *queryTypeListener) EnterShowTables(_ *parser.ShowTablesContext) {
	l.result = base.SelectInfoSchema
}

// EnterShowSchemas is called when entering a ShowSchemasContext rule.
func (l *queryTypeListener) EnterShowSchemas(_ *parser.ShowSchemasContext) {
	l.result = base.SelectInfoSchema
}

// EnterShowColumns is called when entering a ShowColumnsContext rule.
func (l *queryTypeListener) EnterShowColumns(_ *parser.ShowColumnsContext) {
	l.result = base.SelectInfoSchema
}

// EnterShowCreateTable is called when entering a ShowCreateTableContext rule.
func (l *queryTypeListener) EnterShowCreateTable(_ *parser.ShowCreateTableContext) {
	l.result = base.SelectInfoSchema
}

// EnterShowCreateView is called when entering a ShowCreateViewContext rule.
func (l *queryTypeListener) EnterShowCreateView(_ *parser.ShowCreateViewContext) {
	l.result = base.SelectInfoSchema
}

// EnterShowFunctions is called when entering a ShowFunctionsContext rule.
func (l *queryTypeListener) EnterShowFunctions(_ *parser.ShowFunctionsContext) {
	l.result = base.SelectInfoSchema
}

// EnterShowSession is called when entering a ShowSessionContext rule.
func (l *queryTypeListener) EnterShowSession(_ *parser.ShowSessionContext) {
	l.result = base.SelectInfoSchema
}

// EnterShowStats is called when entering a ShowStatsContext rule.
func (l *queryTypeListener) EnterShowStats(_ *parser.ShowStatsContext) {
	l.result = base.SelectInfoSchema
}

// EnterSetSession is called when entering a SetSessionContext rule.
func (l *queryTypeListener) EnterSetSession(_ *parser.SetSessionContext) {
	l.result = base.Select
}

// EnterResetSession is called when entering a ResetSessionContext rule.
func (l *queryTypeListener) EnterResetSession(_ *parser.ResetSessionContext) {
	l.result = base.Select
}

// GetStatementType returns the detailed statement type for query span.
func GetStatementType(tree any) StatementType {
	stmtCtx, ok := tree.(*parser.SingleStatementContext)
	if !ok {
		return Unsupported
	}

	stmt := stmtCtx.Statement()
	if stmt == nil {
		return Unsupported
	}

	switch stmt.(type) {
	case *parser.StatementDefaultContext:
		// SELECT statement
		return Select

	case *parser.ExplainContext, *parser.ExplainAnalyzeContext:
		return Explain

	case *parser.InsertIntoContext:
		return Insert

	case *parser.UpdateContext:
		return Update

	case *parser.DeleteContext:
		return Delete

	case *parser.MergeContext:
		return Merge

	case *parser.CreateTableContext:
		return CreateTable

	case *parser.CreateTableAsSelectContext:
		return CreateTableAsSelect

	case *parser.CreateViewContext:
		return CreateView

	// Alter table operations
	case *parser.AddColumnContext, *parser.DropColumnContext,
		*parser.RenameColumnContext, *parser.SetColumnTypeContext,
		*parser.SetTableAuthorizationContext, *parser.SetTablePropertiesContext:
		return AlterTable

	case *parser.DropTableContext:
		return DropTable

	case *parser.DropViewContext:
		return DropView

	case *parser.CreateSchemaContext:
		return CreateSchema

	case *parser.DropSchemaContext:
		return DropSchema

	case *parser.RenameTableContext, *parser.RenameViewContext:
		return RenameTable

	case *parser.RenameSchemaContext:
		return CreateSchema // Using CreateSchema for schema operations

	// SHOW statements
	case *parser.ShowTablesContext, *parser.ShowSchemasContext,
		*parser.ShowColumnsContext, *parser.ShowCreateTableContext,
		*parser.ShowCreateViewContext, *parser.ShowFunctionsContext,
		*parser.ShowSessionContext, *parser.ShowStatsContext:
		return Show

	case *parser.SetSessionContext, *parser.ResetSessionContext:
		return Set

	default:
		return Unsupported
	}
}

// IsReadOnlyStatement returns whether the statement is read-only.
func IsReadOnlyStatement(tree any) bool {
	queryType, _ := getQueryType(tree)
	return queryType == base.Select ||
		queryType == base.Explain ||
		queryType == base.SelectInfoSchema
}

// IsDataChangingStatement returns whether the statement changes data.
func IsDataChangingStatement(tree any) bool {
	queryType, _ := getQueryType(tree)
	return queryType == base.DML
}

// IsSchemaChangingStatement returns whether the statement changes schema.
func IsSchemaChangingStatement(tree any) bool {
	queryType, _ := getQueryType(tree)
	return queryType == base.DDL
}

// containsSystemSchema checks if a query is accessing system tables/schemas in Trino.
func containsSystemSchema(sql string) bool {
	lowerSQL := strings.ToLower(sql)

	// Trino system schemas/catalogs
	systemPrefixes := []string{
		"system.",
		"information_schema.",
		"$system.",
		"catalog.",
		"metadata.",
	}

	for _, prefix := range systemPrefixes {
		if strings.Contains(lowerSQL, prefix) {
			return true
		}
	}

	return false
}
