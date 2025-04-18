package trino

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/trino-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_TRINO, GetQuerySpan)
}

// GetQuerySpan gets the query span for Trino.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, statement, database, schema string, ignoreCaseSensitive bool) (*base.QuerySpan, error) {
	result, err := ParseTrino(statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse Trino statement: %s", statement)
	}

	// Map our StatementType to whatever the span uses
	stmtType := StatementType(result.Tree)
	var spanType base.StatementType
	
	// Maps our StatementType to base.StatementType
	switch stmtType {
	case Select:
		spanType = 0 // Select
	case Explain:
		spanType = 1 // Explain
	case Insert:
		spanType = 2 // Insert
	case Update:
		spanType = 3 // Update
	case Delete:
		spanType = 4 // Delete
	case CreateTable:
		spanType = 5 // CreateTable
	case CreateView:
		spanType = 6 // CreateView
	case AlterTable:
		spanType = 7 // AlterTable
	case DropTable:
		spanType = 8 // DropTable
	case DropView:
		spanType = 9 // DropView
	case Show:
		spanType = 10 // Show
	default:
		spanType = 11 // Unsupported
	}

	span := &base.QuerySpan{
		Type:   spanType,
		Source: statement,
	}

	// Extract tables from the statement
	listener := &querySpanListener{
		currentDatabase:     database,
		currentSchema:       schema,
		ignoreCaseSensitive: ignoreCaseSensitive,
		tables:              make(map[string]*base.PhysicalTable),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, result.Tree)

	// Convert map to slice
	for _, table := range listener.tables {
		span.Tables = append(span.Tables, table)
	}

	return span, nil
}

// querySpanListener implements the TrinoParserListener interface to extract table information.
type querySpanListener struct {
	parser.BaseTrinoParserListener

	currentDatabase     string
	currentSchema       string
	ignoreCaseSensitive bool
	tables              map[string]*base.PhysicalTable
}

// EnterTableName is called when a tableName node is entered
func (l *querySpanListener) EnterTableName(ctx *parser.TableNameContext) {
	// Extract the qualified name parts
	var catalog, schema, table string

	// Extract table name from qualified name
	if ctx.QualifiedName() != nil {
		parts := l.extractQualifiedNameParts(ctx.QualifiedName())
		
		switch len(parts) {
		case 1:
			// Just table name
			catalog = l.currentDatabase
			schema = l.currentSchema
			table = parts[0]
		case 2:
			// schema.table
			catalog = l.currentDatabase
			schema = parts[0]
			table = parts[1]
		case 3:
			// catalog.schema.table
			catalog = parts[0]
			schema = parts[1]
			table = parts[2]
		default:
			return
		}

		// Create a unique key for the table
		key := catalog + "." + schema + "." + table
		
		// Only add if not already added
		if _, exists := l.tables[key]; !exists {
			l.tables[key] = &base.PhysicalTable{
				Database: catalog,
				Schema:   schema,
				Name:     table,
			}
		}
	}
}

// extractQualifiedNameParts extracts the parts of a qualified name.
func (l *querySpanListener) extractQualifiedNameParts(ctx parser.IQualifiedNameContext) []string {
	if ctx == nil {
		return nil
	}

	var parts []string
	for _, ident := range ctx.AllIdentifier() {
		if ident != nil {
			// Normalize the identifier according to Trino rules
			parts = append(parts, NormalizeTrinoIdentifier(ident.GetText()))
		}
	}

	return parts
}