package starrocks

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/bytebase/omni/starrocks/analysis"
	"github.com/bytebase/omni/starrocks/ast"
	"github.com/bytebase/omni/starrocks/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_STARROCKS, GetQuerySpan)
}

func GetQuerySpan(
	_ context.Context,
	_ base.GetQuerySpanContext,
	stmt base.Statement,
	database, _ string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	resolved, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase:     database,
		IgnoreCaseSensitive: ignoreCaseSensitive,
	})
	if err != nil {
		if stderrors.Is(err, analysis.ErrMixUserSystemTables) {
			return resolveWithStatisticsSystemDB(stmt, database, ignoreCaseSensitive)
		}
		return nil, convertParseError(stmt.Text, err, stmt.Start)
	}

	sources := make(base.SourceColumnSet, len(resolved.SourceColumns))
	for res := range resolved.SourceColumns {
		sources[base.ColumnResource{
			Database: res.Database,
			Table:    res.Table,
		}] = true
	}

	allSystems, mixed := classifySystemMix(sources, ignoreCaseSensitive)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	spanType := mapQueryType(resolved.Type)
	if spanType == base.Select && allSystems {
		spanType = base.SelectInfoSchema
	}

	return &base.QuerySpan{
		Type:             spanType,
		SourceColumns:    sources,
		PredicateColumns: base.SourceColumnSet{},
		Results:          []base.QuerySpanResult{},
	}, nil
}

// resolveWithStatisticsSystemDB handles omni ErrMixUserSystemTables when _statistics_
// (excluded from omni's StarRocks systemDatabases; https://github.com/bytebase/omni/issues/1497)
// is queried alongside information_schema or sys.
func resolveWithStatisticsSystemDB(stmt base.Statement, database string, ignoreCaseSensitive bool) (*base.QuerySpan, error) {
	stmts := analysis.SplitStatements(stmt.Text)
	var single string
	for _, s := range stmts {
		if !s.Empty {
			single = s.Text
			break
		}
	}
	if single == "" {
		return nil, base.MixUserSystemTablesError
	}
	file, err := parser.Parse(single)
	if err != nil || file == nil || len(file.Stmts) == 0 {
		return nil, base.MixUserSystemTablesError
	}
	defaultDB := database
	if submit, ok := file.Stmts[0].(*ast.SubmitTaskStmt); ok && submit.Name != nil && len(submit.Name.Parts) >= 2 {
		defaultDB = submit.Name.Parts[len(submit.Name.Parts)-2]
	}

	spanInput := single
	if exp, ok := file.Stmts[0].(*ast.ExplainStmt); ok && exp.Query != nil {
		if inner := ast.NodeLoc(exp.Query); inner.Start >= 0 && inner.Start < len(single) {
			spanInput = single[inner.Start:]
		}
	}

	rawSpan, err := analysis.GetQuerySpan(spanInput)
	if err != nil {
		return nil, convertParseError(stmt.Text, err, stmt.Start)
	}

	sources := make(base.SourceColumnSet, len(rawSpan.AccessTables))
	for _, table := range rawSpan.AccessTables {
		db := table.Database
		if db == "" {
			db = defaultDB
		}
		sources[base.ColumnResource{
			Database: db,
			Table:    table.Table,
		}] = true
	}

	allSystems, mixed := classifySystemMix(sources, ignoreCaseSensitive)
	if mixed || !allSystems {
		return nil, base.MixUserSystemTablesError
	}

	spanType := mapQueryType(rawSpan.Type)
	if spanType == base.Select {
		spanType = base.SelectInfoSchema
	}
	return &base.QuerySpan{
		Type:             spanType,
		SourceColumns:    sources,
		PredicateColumns: base.SourceColumnSet{},
		Results:          []base.QuerySpanResult{},
	}, nil
}

func classifySystemMix(resources base.SourceColumnSet, ignoreCaseSensitive bool) (allSystems bool, mixed bool) {
	var hasSystem, hasUser bool
	for res := range resources {
		if isSystemDatabase(res.Database, ignoreCaseSensitive) {
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

// isSystemDatabase treats _statistics_ as a StarRocks system database in addition to
// omni/starrocks/analysis.IsSystemDatabase (https://github.com/bytebase/omni/issues/1497).
func isSystemDatabase(name string, ignoreCaseSensitive bool) bool {
	if analysis.IsSystemDatabase(name, ignoreCaseSensitive) {
		return true
	}
	if ignoreCaseSensitive {
		return strings.EqualFold(name, "_statistics_")
	}
	return name == "_statistics_"
}

func mapQueryType(qt analysis.QueryType) base.QueryType {
	switch qt {
	case analysis.QueryTypeSelect:
		return base.Select
	case analysis.QueryTypeSelectInfoSchema:
		return base.SelectInfoSchema
	case analysis.QueryTypeDML:
		return base.DML
	case analysis.QueryTypeDDL:
		return base.DDL
	default:
		return base.QueryTypeUnknown
	}
}
