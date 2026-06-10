package starrocks

import (
	"context"
	"strings"

	"github.com/bytebase/omni/starrocks/analysis"
	"github.com/bytebase/omni/starrocks/ast"
	"github.com/bytebase/omni/starrocks/parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// querySpanExtractor analyses a single Doris statement and produces a
// base.QuerySpan: the set of physical tables it reads and its classification
// as a Select / SelectInfoSchema / DML / DDL statement.
//
// The extractor delegates the heavy lifting to omni's analysis.GetQuerySpan,
// then applies bytebase-specific post-processing:
//   - default-database fill-in for bare table references,
//   - system-vs-user mixed-query rejection (MixUserSystemTablesError),
//   - QueryType promotion to SelectInfoSchema when every accessed table is a
//     system table (matching the legacy ANTLR listener behaviour).
type querySpanExtractor struct {
	ctx             context.Context
	defaultDatabase string
	gCtx            base.GetQuerySpanContext
	// ctes tracks Common Table Expressions in the current scope. omni's
	// GetQuerySpan already filters CTE references from AccessTables, but we
	// keep the field to preserve the construction signature used by callers
	// (and any future logic that needs it).
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

	// Split into top-level statements so we can reject inputs that contain
	// multiple statements (matching the legacy behaviour) before doing any
	// per-statement analysis.
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	nonEmpty := 0
	var single string
	for _, s := range stmts {
		if s.Empty {
			continue
		}
		nonEmpty++
		single = s.Text
	}
	if nonEmpty == 0 {
		return &base.QuerySpan{
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}
	if nonEmpty > 1 {
		return nil, errors.Errorf("expecting only one statement to get query span, but got %d", nonEmpty)
	}

	// omni's span walker only descends into SELECT/SetOp at the top level,
	// so EXPLAIN <query> would return zero AccessTables. Unwrap an EXPLAIN
	// to the inner statement's text before delegating; that way table-level
	// ACL checks still see what the underlying query reads.
	spanInput := unwrapExplainForSpan(single)

	// Delegate to omni for table-access extraction + classification.
	omniSpan, err := analysis.GetQuerySpan(spanInput)
	if err != nil {
		return nil, err
	}

	sources := q.toSourceColumnSet(omniSpan)

	// Track CTE names omni discovered for callers that want to inspect them.
	for _, name := range omniSpan.CTEs {
		q.ctes[strings.ToLower(name)] = true
	}

	allSystems, mixed := isMixedQuery(sources, q.ignoreCaseSensitive)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	queryType := getQueryType(single, allSystems)

	return &base.QuerySpan{
		Type:          queryType,
		SourceColumns: sources,
		Results:       []base.QuerySpanResult{},
	}, nil
}

// unwrapExplainForSpan returns the inner statement's text when `statement`
// parses to a top-level EXPLAIN; otherwise it returns the original string.
//
// omni's span walker only descends into SELECT/SetOp at the top level, so
// without this unwrap an `EXPLAIN SELECT ... FROM t` would yield zero
// AccessTables — table-level ACL checks need to see `t`.
func unwrapExplainForSpan(statement string) string {
	file, errs := parser.Parse(statement)
	if len(errs) > 0 || file == nil || len(file.Stmts) == 0 {
		return statement
	}
	explain, ok := file.Stmts[0].(*ast.ExplainStmt)
	if !ok || explain.Query == nil {
		return statement
	}
	inner := ast.NodeLoc(explain.Query)
	if inner.Start < 0 || inner.End > len(statement) || inner.Start >= inner.End {
		return statement
	}
	return statement[inner.Start:inner.End]
}

// toSourceColumnSet converts an omni QuerySpan's AccessTables into the
// base.SourceColumnSet shape bytebase expects. Tables with no database
// qualifier fall back to the extractor's default database.
func (q *querySpanExtractor) toSourceColumnSet(span *analysis.QuerySpan) base.SourceColumnSet {
	out := base.SourceColumnSet{}
	if span == nil {
		return out
	}
	for _, t := range span.AccessTables {
		db := t.Database
		if db == "" {
			db = q.defaultDatabase
		}
		out[base.ColumnResource{
			Database: db,
			Table:    t.Table,
		}] = true
	}
	return out
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

// systemDatabases contains StarRocks system databases. StarRocks exposes a
// read-only `sys` metadatabase and does not have Doris's `mysql` /
// `__internal_schema`.
// Reference: https://docs.starrocks.io/docs/administration/management/system_catalogs/sys/
var systemDatabases = map[string]bool{
	"information_schema": true,
	"_statistics_":       true,
	"sys":                true,
}

func isSystemResource(resource base.ColumnResource, ignoreCaseSensitive bool) bool {
	database := resource.Database
	if ignoreCaseSensitive {
		database = strings.ToLower(database)
	}
	return systemDatabases[database]
}
