package doris

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/doris-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type querySpanExtractor struct {
	defaultDatabase string
	// ctes tracks Common Table Expressions in the current scope
	ctes map[string]bool
}

func newQuerySpanExtractor(database string, _ base.GetQuerySpanContext, _ bool) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase: database,
		ctes:            make(map[string]bool),
	}
}

func (q *querySpanExtractor) getQuerySpan(_ context.Context, statement string) (*base.QuerySpan, error) {
	parseResult, err := ParseDorisSQL(statement)
	if err != nil {
		return nil, err
	}

	if parseResult == nil {
		return &base.QuerySpan{
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	accessTables := getAccessTables(q.defaultDatabase, parseResult, q.ctes)

	queryTypeListener := &queryTypeListener{
		result: base.QueryTypeUnknown,
	}
	antlr.ParseTreeWalkerDefault.Walk(queryTypeListener, parseResult.Tree)

	return &base.QuerySpan{
		Type:          queryTypeListener.result,
		SourceColumns: accessTables,
		Results:       []base.QuerySpanResult{},
	}, nil
}

func getAccessTables(database string, parseResult *ParseResult, ctes map[string]bool) base.SourceColumnSet {
	// First, extract CTEs from the query
	cteListener := newCTEListener()
	antlr.ParseTreeWalkerDefault.Walk(cteListener, parseResult.Tree)

	// Merge extracted CTEs with any existing ones
	for cte := range cteListener.ctes {
		ctes[cte] = true
	}

	accessTableListener := newAccessTableListener(database, ctes)
	antlr.ParseTreeWalkerDefault.Walk(accessTableListener, parseResult.Tree)

	return accessTableListener.sourceColumnSet
}

type accessTableListener struct {
	*parser.BaseDorisSQLListener

	defaultDatabase string
	sourceColumnSet base.SourceColumnSet
	ctes            map[string]bool
}

func newAccessTableListener(database string, ctes map[string]bool) *accessTableListener {
	return &accessTableListener{
		defaultDatabase: database,
		sourceColumnSet: base.SourceColumnSet{},
		ctes:            ctes,
	}
}

func (l *accessTableListener) EnterTableAtom(ctx *parser.TableAtomContext) {
	if ctx == nil {
		return
	}

	list := NormalizeQualifiedName(ctx.QualifiedName())
	switch len(list) {
	case 1:
		// Check if this is a CTE reference
		if l.ctes[list[0]] {
			// Skip CTE references - they don't need permission checks
			return
		}
		l.sourceColumnSet[base.ColumnResource{
			Database: l.defaultDatabase,
			Table:    list[0],
		}] = true
	case 2:
		// For qualified names (db.table), CTEs cannot have schema qualifiers
		l.sourceColumnSet[base.ColumnResource{
			Database: list[0],
			Table:    list[1],
		}] = true
	default:
		// Ignore qualified names with more than 2 parts
	}
}

// cteListener extracts CTE names from WITH clauses
type cteListener struct {
	*parser.BaseDorisSQLListener

	ctes map[string]bool
}

func newCTEListener() *cteListener {
	return &cteListener{
		ctes: make(map[string]bool),
	}
}

// EnterWithClause is called when entering a WITH clause
func (l *cteListener) EnterWithClause(ctx *parser.WithClauseContext) {
	if ctx == nil {
		return
	}

	// Extract all CTEs from the WITH clause
	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i)
		if cte, ok := child.(*parser.CommonTableExpressionContext); ok {
			l.extractCTEName(cte)
		}
	}
}

// extractCTEName extracts the CTE name from a CommonTableExpression context
func (l *cteListener) extractCTEName(ctx *parser.CommonTableExpressionContext) {
	if ctx == nil {
		return
	}

	// Get the CTE identifier
	if ctx.Identifier() != nil {
		cteName := NormalizeIdentifier(ctx.Identifier())
		l.ctes[cteName] = true
	}
}
