package trino

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/trino-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_TRINO, TransformDMLToSelect)
}

// TransformDMLToSelect transforms DML statements to SELECT statements.
func TransformDMLToSelect(_ context.Context, _ base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	// Parse the SQL statement
	parseResult, err := ParseTrino(statement)
	if err != nil {
		return nil, err
	}

	// Create a listener to transform DML to SELECT
	listener := &dmlTransformListener{
		sourceDatabase: sourceDatabase,
		targetDatabase: targetDatabase,
		tablePrefix:    tablePrefix,
		statement:      statement,
	}

	// Walk the parse tree with our listener
	antlr.ParseTreeWalkerDefault.Walk(listener, parseResult.Tree)

	// Return the transformed statements
	return listener.backupStatements, nil
}

// dmlTransformListener implements the TrinoParserListener interface to transform DML to SELECT.
type dmlTransformListener struct {
	parser.BaseTrinoParserListener

	sourceDatabase   string
	targetDatabase   string
	tablePrefix      string
	statement        string
	backupStatements []base.BackupStatement

	// Current statement context
	currentTable      string
	currentSchema     string
	currentDatabase   string
	currentColumns    []string
	currentCondition  string
	currentValues     []string
	currentSetClauses []string
}

// EnterInsertInto is called when the parser enters an insertInto rule
func (l *dmlTransformListener) EnterInsertInto(ctx *parser.InsertIntoContext) {
	// Extract table information
	if ctx.QualifiedName() == nil {
		return
	}

	// Extract catalog, schema, and table name
	l.currentDatabase, l.currentSchema, l.currentTable = ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.sourceDatabase,
		"",
	)

	// Extract column names if specified
	l.currentColumns = []string{}
	if ctx.ColumnAliases() != nil {
		for _, identifier := range ctx.ColumnAliases().AllIdentifier() {
			l.currentColumns = append(l.currentColumns, NormalizeTrinoIdentifier(identifier.GetText()))
		}
	}
}

// ExitInsertInto is called when the parser exits an insertInto rule
func (l *dmlTransformListener) ExitInsertInto(_ *parser.InsertIntoContext) {
	// Skip if we don't have valid table information
	if l.currentTable == "" {
		return
	}

	// For a simple INSERT with VALUES, we create a SELECT that would retrieve the same data
	// This is simplified and works for basic cases
	selectSQL := fmt.Sprintf("SELECT * FROM %s.%s.%s",
		l.getQualifiedName(l.currentDatabase),
		l.getQualifiedName(l.currentSchema),
		l.getQualifiedName(l.currentTable))

	// Add the backup statement
	l.backupStatements = append(l.backupStatements, base.BackupStatement{
		Statement:       selectSQL,
		SourceSchema:    l.currentSchema,
		SourceTableName: l.currentTable,
		TargetTableName: l.tablePrefix + l.currentTable,
	})

	// Reset state
	l.currentTable = ""
	l.currentSchema = ""
	l.currentDatabase = ""
	l.currentColumns = nil
	l.currentValues = nil
}

// EnterUpdate is called when the parser enters an update rule
func (l *dmlTransformListener) EnterUpdate(ctx *parser.UpdateContext) {
	// Extract table information
	if ctx.QualifiedName() == nil {
		return
	}

	// Extract catalog, schema, and table name
	l.currentDatabase, l.currentSchema, l.currentTable = ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.sourceDatabase,
		"",
	)

	// Extract SET clauses
	l.currentSetClauses = []string{}
	if ctx.AllUpdateAssignment() != nil {
		for _, assignment := range ctx.AllUpdateAssignment() {
			if assignment.Identifier() != nil && assignment.Expression() != nil {
				column := NormalizeTrinoIdentifier(assignment.Identifier().GetText())
				l.currentSetClauses = append(l.currentSetClauses, column)
			}
		}
	}

	// Extract WHERE condition if present
	l.currentCondition = ""
	if ctx.BooleanExpression() != nil {
		// This is simplified - extracting the full condition would require more sophisticated parsing
		l.currentCondition = ctx.BooleanExpression().GetText()
	}
}

// ExitUpdate is called when the parser exits an update rule
func (l *dmlTransformListener) ExitUpdate(_ *parser.UpdateContext) {
	// Skip if we don't have valid table information
	if l.currentTable == "" {
		return
	}

	// Build the SELECT statement for the update
	// We need to select the rows that would be affected by the update
	selectSQL := fmt.Sprintf("SELECT * FROM %s.%s.%s",
		l.getQualifiedName(l.currentDatabase),
		l.getQualifiedName(l.currentSchema),
		l.getQualifiedName(l.currentTable))

	// Add WHERE clause if available
	if l.currentCondition != "" {
		selectSQL += " WHERE " + l.currentCondition
	}

	// Add the backup statement
	l.backupStatements = append(l.backupStatements, base.BackupStatement{
		Statement:       selectSQL,
		SourceSchema:    l.currentSchema,
		SourceTableName: l.currentTable,
		TargetTableName: l.tablePrefix + l.currentTable,
	})

	// Reset state
	l.currentTable = ""
	l.currentSchema = ""
	l.currentDatabase = ""
	l.currentSetClauses = nil
	l.currentCondition = ""
}

// EnterDelete is called when the parser enters a delete rule
func (l *dmlTransformListener) EnterDelete(ctx *parser.DeleteContext) {
	// Extract table information
	if ctx.QualifiedName() == nil {
		return
	}

	// Extract catalog, schema, and table name
	l.currentDatabase, l.currentSchema, l.currentTable = ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.sourceDatabase,
		"",
	)

	// Extract WHERE condition if present
	l.currentCondition = ""
	if ctx.BooleanExpression() != nil {
		// This is simplified - extracting the full condition would require more sophisticated parsing
		l.currentCondition = ctx.BooleanExpression().GetText()
	}
}

// ExitDelete is called when the parser exits a delete rule
func (l *dmlTransformListener) ExitDelete(_ *parser.DeleteContext) {
	// Skip if we don't have valid table information
	if l.currentTable == "" {
		return
	}

	// Build the SELECT statement for the delete
	// We need to select the rows that would be deleted
	selectSQL := fmt.Sprintf("SELECT * FROM %s.%s.%s",
		l.getQualifiedName(l.currentDatabase),
		l.getQualifiedName(l.currentSchema),
		l.getQualifiedName(l.currentTable))

	// Add WHERE clause if available
	if l.currentCondition != "" {
		selectSQL += " WHERE " + l.currentCondition
	}

	// Add the backup statement
	l.backupStatements = append(l.backupStatements, base.BackupStatement{
		Statement:       selectSQL,
		SourceSchema:    l.currentSchema,
		SourceTableName: l.currentTable,
		TargetTableName: l.tablePrefix + l.currentTable,
	})

	// Reset state
	l.currentTable = ""
	l.currentSchema = ""
	l.currentDatabase = ""
	l.currentCondition = ""
}

// getQualifiedName returns a properly quoted identifier
func (dmlTransformListener) getQualifiedName(name string) string {
	// For Trino, use double quotes for identifiers
	return "\"" + name + "\""
}
