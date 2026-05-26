package doris

import (
	"context"
	"strings"

	"github.com/bytebase/omni/doris/analysis"
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

	// Delegate to omni for table-access extraction + classification.
	omniSpan, err := analysis.GetQuerySpan(single)
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
