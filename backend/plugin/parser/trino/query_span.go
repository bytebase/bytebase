package trino

import (
	"context"
	stderrors "errors"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/trino/analysis"
	"github.com/bytebase/omni/trino/ast"
	"github.com/bytebase/omni/trino/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_TRINO, GetQuerySpan)
}

// GetQuerySpan gets the query span for a Trino statement. It is the entry point
// registered with the base package.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, ignoreCaseSensitive bool) (*base.QuerySpan, error) {
	opts := analysis.ResolveOptions{
		DefaultDatabase:     database,
		DefaultSchema:       schema,
		IgnoreCaseSensitive: ignoreCaseSensitive,
	}
	if gCtx.ListDatabaseNamesFunc != nil {
		opts.ListDatabases = func() ([]string, error) {
			return gCtx.ListDatabaseNamesFunc(ctx, gCtx.InstanceID)
		}
	}
	if gCtx.GetDatabaseMetadataFunc != nil {
		opts.GetDatabaseMetadata = func(db string) (*metadatapb.DatabaseSchemaMetadata, error) {
			_, meta, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, db)
			if err != nil {
				var notFound *base.ResourceNotFoundError
				if stderrors.As(err, &notFound) {
					return nil, &analysis.ResourceNotFoundError{
						Err:      notFound.Err,
						Database: notFound.Database,
						Schema:   notFound.Schema,
						Table:    notFound.Table,
						Column:   notFound.Column,
					}
				}
				return nil, err
			}
			if meta == nil {
				return nil, nil
			}
			return meta.GetProto(), nil
		}
	}

	resolved, err := analysis.ResolveQuerySpan(stmt.Text, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}

	// Work around https://github.com/bytebase/omni/issues/1497: omni's spanQueryType lacks
	// a WithFunctionStmt case (returning DDL with empty Results), so resolve the wrapped query.
	if resolved.Type == analysis.DDL && containsASCIIFold(stmt.Text, "function") {
		if innerSQL, ok := withFunctionInnerQuery(stmt.Text); ok {
			innerResolved, err := analysis.ResolveQuerySpan(innerSQL, opts)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
			}
			resolved = innerResolved
		}
	}

	results := make([]base.QuerySpanResult, 0, len(resolved.Results))
	for _, r := range resolved.Results {
		results = append(results, base.QuerySpanResult{
			Name:          r.Name,
			SourceColumns: toBaseColumnSet(r.SourceColumns),
			IsPlainField:  r.IsPlainField,
		})
	}

	span := &base.QuerySpan{
		Type:             mapQueryType(resolved.Type),
		SourceColumns:    toBaseColumnSet(resolved.SourceColumns),
		PredicateColumns: toBaseColumnSet(resolved.PredicateColumns),
		Results:          results,
	}
	if resolved.NotFoundError != nil {
		var nf *analysis.ResourceNotFoundError
		if stderrors.As(resolved.NotFoundError, &nf) {
			span.NotFoundError = &base.ResourceNotFoundError{
				Err:      nf.Err,
				Database: nf.Database,
				Schema:   nf.Schema,
				Table:    nf.Table,
				Column:   nf.Column,
			}
		} else {
			span.NotFoundError = &base.ResourceNotFoundError{
				Err: resolved.NotFoundError,
			}
		}
	}
	return span, nil
}

func withFunctionInnerQuery(statement string) (string, bool) {
	file, err := parser.Parse(statement)
	if err != nil || file == nil || len(file.Stmts) != 1 {
		return "", false
	}
	wf := unwrapWithFunction(file.Stmts[0])
	if wf == nil || wf.Query == nil {
		return "", false
	}
	start := -1
	if wf.Query.With != nil {
		start = wf.Query.With.Loc.Start
	} else if wf.Query.Body != nil {
		start = wf.Query.Body.Span().Start
	}
	end := wf.Query.Loc.End
	if start < 0 || end > len(statement) || start >= end {
		return "", false
	}
	inner := strings.TrimSpace(statement[start:end])
	if inner == "" {
		return "", false
	}
	return inner, true
}

func unwrapWithFunction(node ast.Node) *parser.WithFunctionStmt {
	switch n := node.(type) {
	case *parser.WithFunctionStmt:
		return n
	case *parser.ExplainStmt:
		if n.Analyze && n.Statement != nil {
			return unwrapWithFunction(n.Statement)
		}
		return nil
	default:
		return nil
	}
}

func mapQueryType(qt analysis.QueryType) base.QueryType {
	switch qt {
	case analysis.Select:
		return base.Select
	case analysis.Explain:
		return base.Explain
	case analysis.SelectInfoSchema:
		return base.SelectInfoSchema
	case analysis.DDL:
		return base.DDL
	case analysis.DML:
		return base.DML
	default:
		return base.QueryTypeUnknown
	}
}

func toBaseColumnSet(cols analysis.ColumnSet) base.SourceColumnSet {
	out := make(base.SourceColumnSet, len(cols))
	for col := range cols {
		out[base.ColumnResource{
			Database: col.Database,
			Schema:   col.Schema,
			Table:    col.Table,
			Column:   col.Column,
		}] = true
	}
	return out
}
