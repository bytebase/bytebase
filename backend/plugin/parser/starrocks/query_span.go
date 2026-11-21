package starrocks

import (
	"context"
	stderrors "errors"

	"github.com/bytebase/omni/starrocks/analysis"

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
			return nil, base.MixUserSystemTablesError
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

	return &base.QuerySpan{
		Type:             mapQueryType(resolved.Type),
		SourceColumns:    sources,
		PredicateColumns: base.SourceColumnSet{},
		Results:          []base.QuerySpanResult{},
	}, nil
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
