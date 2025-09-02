package pg

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	pgparser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	// Register PostgreSQL-specific index comparer
	schema.RegisterIndexComparer(storepb.Engine_POSTGRES, &PostgreSQLIndexComparer{})
}

// PostgreSQLIndexComparer provides PostgreSQL-specific index comparison logic.
type PostgreSQLIndexComparer struct{}

// Ensure PostgreSQLIndexComparer implements the IndexComparer interface
var _ schema.IndexComparer = &PostgreSQLIndexComparer{}

// CompareIndexWhereConditions compares WHERE conditions from PostgreSQL index definitions using AST parsing.
func (c *PostgreSQLIndexComparer) CompareIndexWhereConditions(def1, def2 string) bool {
	whereClause1 := c.ExtractWhereClauseFromIndexDef(def1)
	whereClause2 := c.ExtractWhereClauseFromIndexDef(def2)

	// If both have no WHERE clause, they're equal
	if whereClause1 == "" && whereClause2 == "" {
		return true
	}

	// If only one has WHERE clause, they're different
	if whereClause1 == "" || whereClause2 == "" {
		return false
	}

	// Use expression comparer for semantic comparison
	return schema.CompareExpressionsSemantically(storepb.Engine_POSTGRES, whereClause1, whereClause2)
}

// ExtractWhereClauseFromIndexDef extracts the WHERE clause from a CREATE INDEX statement using ANTLR AST parsing.
func (*PostgreSQLIndexComparer) ExtractWhereClauseFromIndexDef(definition string) string {
	if definition == "" {
		return ""
	}

	// Parse the CREATE INDEX statement using ANTLR parser
	inputStream := antlr.NewInputStream(definition)
	lexer := pgparser.NewPostgreSQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := pgparser.NewPostgreSQLParser(stream)

	// Disable error reporting for cleaner parsing
	parser.RemoveErrorListeners()

	// Parse as root statement
	tree := parser.Root()

	// Use visitor to extract WHERE clause from IndexStmt
	extractor := &indexWhereExtractor{}
	extractor.Visit(tree)

	return extractor.whereClause
}

// indexWhereExtractor is a visitor that extracts WHERE clauses from CREATE INDEX statements.
type indexWhereExtractor struct {
	*pgparser.BasePostgreSQLParserVisitor
	whereClause string
}

// Visit implements the visitor pattern to walk the AST.
func (e *indexWhereExtractor) Visit(tree antlr.ParseTree) {
	switch ctx := tree.(type) {
	case *pgparser.IndexstmtContext:
		e.visitIndexStmt(ctx)
	default:
		// Recursively visit children
		if tree != nil {
			for i := 0; i < tree.GetChildCount(); i++ {
				child := tree.GetChild(i)
				if parseTree, ok := child.(antlr.ParseTree); ok {
					e.Visit(parseTree)
				}
			}
		}
	}
}

// visitIndexStmt extracts the WHERE clause from an IndexStmt AST node.
func (e *indexWhereExtractor) visitIndexStmt(ctx *pgparser.IndexstmtContext) {
	if ctx == nil {
		return
	}

	// Get the WHERE clause from the IndexStmt
	whereClauseCtx := ctx.Where_clause()
	if whereClauseCtx == nil {
		e.whereClause = ""
		return
	}

	// Extract the expression from the WHERE clause (excluding the WHERE keyword)
	aExprCtx := whereClauseCtx.A_expr()
	if aExprCtx == nil {
		e.whereClause = ""
		return
	}

	// Get the original text of the expression
	e.whereClause = e.getTextFromContext(aExprCtx)
}

// getTextFromContext extracts the original source text from a parse tree context.
func (*indexWhereExtractor) getTextFromContext(ctx antlr.ParserRuleContext) string {
	if ctx == nil {
		return ""
	}

	start := ctx.GetStart()
	stop := ctx.GetStop()
	if start == nil || stop == nil {
		return ""
	}

	input := start.GetInputStream()
	if input == nil {
		return ""
	}

	startIndex := start.GetStart()
	stopIndex := stop.GetStop()

	if startIndex < 0 || stopIndex < 0 || stopIndex < startIndex {
		return ""
	}

	// Extract the original text from the input stream
	return input.GetText(startIndex, stopIndex)
}

// IndexDefinition represents the parsed structure of a CREATE INDEX statement.
type IndexDefinition struct {
	IndexName   string
	TableName   string
	Unique      bool
	Method      string // btree, gin, gist, etc.
	Expressions []string
	WhereClause string
}

// ParseIndexDefinition parses a complete CREATE INDEX statement and returns structured information.
func (c *PostgreSQLIndexComparer) ParseIndexDefinition(definition string) (*IndexDefinition, error) {
	if definition == "" {
		return nil, nil
	}

	result := &IndexDefinition{
		WhereClause: c.ExtractWhereClauseFromIndexDef(definition),
	}

	// Parse the CREATE INDEX statement using ANTLR parser
	inputStream := antlr.NewInputStream(definition)
	lexer := pgparser.NewPostgreSQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := pgparser.NewPostgreSQLParser(stream)

	// Disable error reporting for cleaner parsing
	parser.RemoveErrorListeners()

	// Parse as root statement
	tree := parser.Root()

	// Use visitor to extract structured information from IndexStmt
	extractor := &indexDefinitionExtractor{result: result}
	extractor.Visit(tree)

	return result, nil
}

// indexDefinitionExtractor is a visitor that extracts structured information from CREATE INDEX statements.
type indexDefinitionExtractor struct {
	*pgparser.BasePostgreSQLParserVisitor
	result *IndexDefinition
}

// Visit implements the visitor pattern to walk the AST.
func (e *indexDefinitionExtractor) Visit(tree antlr.ParseTree) {
	switch ctx := tree.(type) {
	case *pgparser.IndexstmtContext:
		e.visitIndexStmtForDefinition(ctx)
	default:
		// Recursively visit children
		if tree != nil {
			for i := 0; i < tree.GetChildCount(); i++ {
				child := tree.GetChild(i)
				if parseTree, ok := child.(antlr.ParseTree); ok {
					e.Visit(parseTree)
				}
			}
		}
	}
}

// visitIndexStmtForDefinition extracts structured information from an IndexStmt AST node.
func (e *indexDefinitionExtractor) visitIndexStmtForDefinition(ctx *pgparser.IndexstmtContext) {
	if ctx == nil || e.result == nil {
		return
	}

	// Extract UNIQUE flag
	if ctx.Opt_unique() != nil {
		e.result.Unique = true
	}

	// Extract index name
	if ctx.Name() != nil {
		e.result.IndexName = ctx.Name().GetText()
	}

	// Extract table name from relation_expr
	if ctx.Relation_expr() != nil {
		e.result.TableName = ctx.Relation_expr().GetText()
	}

	// Extract access method
	if ctx.Access_method_clause() != nil {
		e.result.Method = strings.ToLower(ctx.Access_method_clause().GetText())
	}

	// WHERE clause is already extracted in the result initialization
}
