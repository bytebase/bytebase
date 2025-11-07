package doris

import (
	"context"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/doris"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type querySpanExtractor struct {
	ctx             context.Context
	defaultDatabase string
	gCtx            base.GetQuerySpanContext
	// ctes tracks Common Table Expressions in the current scope
	ctes                map[string]bool
	ignoreCaseSensitive bool
}

func newQuerySpanExtractor(database string, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase:     database,
		gCtx:                gCtx,
		ctes:                make(map[string]bool),
		ignoreCaseSensitive: ignoreCaseSensitive,
	}
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx
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

	accessTables := getAccessTables(q.defaultDatabase, parseResult, q.ctes, q.gCtx, q.ignoreCaseSensitive)

	// We do not support simultaneous access to the system table and the user table
	// because we do not synchronize the schema of the system table.
	// This causes an error (NOT_FOUND) when using querySpanExtractor.findTableSchema.
	// As a result, we exclude getting query span results for accessing only the system table.
	allSystems, mixed := isMixedQuery(accessTables, q.ignoreCaseSensitive)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	queryTypeListener := &queryTypeListener{
		allSystems: allSystems,
		result:     base.QueryTypeUnknown,
	}
	antlr.ParseTreeWalkerDefault.Walk(queryTypeListener, parseResult.Tree)

	return &base.QuerySpan{
		Type:          queryTypeListener.result,
		SourceColumns: accessTables,
		Results:       []base.QuerySpanResult{},
	}, nil
}

func getAccessTables(database string, parseResult *ParseResult, ctes map[string]bool, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) base.SourceColumnSet {
	// First, extract CTEs from the query
	cteListener := newCTEListener()
	antlr.ParseTreeWalkerDefault.Walk(cteListener, parseResult.Tree)

	// Merge extracted CTEs with any existing ones
	for cte := range cteListener.ctes {
		ctes[cte] = true
	}

	accessTableListener := newAccessTableListener(database, ctes, gCtx, ignoreCaseSensitive)
	antlr.ParseTreeWalkerDefault.Walk(accessTableListener, parseResult.Tree)

	return accessTableListener.sourceColumnSet
}

type accessTableListener struct {
	*parser.BaseDorisSQLListener

	defaultDatabase     string
	sourceColumnSet     base.SourceColumnSet
	ctes                map[string]bool
	gCtx                base.GetQuerySpanContext
	ignoreCaseSensitive bool
}

func newAccessTableListener(database string, ctes map[string]bool, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *accessTableListener {
	return &accessTableListener{
		defaultDatabase:     database,
		sourceColumnSet:     base.SourceColumnSet{},
		ctes:                ctes,
		gCtx:                gCtx,
		ignoreCaseSensitive: ignoreCaseSensitive,
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

// isMixedQuery checks whether the query accesses the user table and system table at the same time.
// It returns whether all tables are system tables and whether there is a mixture.
func isMixedQuery(m base.SourceColumnSet, ignoreCaseSensitive bool) (bool, bool) {
	hasSystem, hasUser := false, false
	for table := range m {
		if isSystemResource(table, ignoreCaseSensitive) {
			hasSystem = true
		} else {
			hasUser = true
		}
	}

	if hasSystem && hasUser {
		return false, true
	}

	return !hasUser && hasSystem, false
}

// systemDatabases contains Doris system databases
// Reference: https://doris.apache.org/docs/3.x/admin-manual/system-tables/overview
var systemDatabases = map[string]bool{
	"information_schema": true,
	"mysql":              true,
	"__internal_schema":  true,
	"_statistics_":       true,
}

func isSystemResource(resource base.ColumnResource, ignoreCaseSensitive bool) bool {
	database := resource.Database
	if ignoreCaseSensitive {
		database = strings.ToLower(database)
	}
	return systemDatabases[database]
}
