package cockroachdb

import (
	"context"

	"github.com/bytebase/omni/cockroachdb/analysis"
	"github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// getQuerySpan resolves a statement on CockroachDB's own AST. A column read
// through a view names the view's column and the table column behind it, as
// CockroachDB's result-set metadata does, so a masking policy on either applies.
func getQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, _ bool) (*base.QuerySpan, error) {
	meta, err := base.RequireDatabaseMetadata(ctx, gCtx, database)
	if err != nil {
		return nil, err
	}
	searchPath := meta.GetSearchPath()
	switch schema {
	case "":
	case "public":
		searchPath = []string{schema}
	default:
		// As for PostgreSQL, a selected schema goes before public.
		searchPath = []string{schema, "public"}
	}
	span, err := analysis.ResolveQuerySpan(ctx, stmt.Text, analysis.ResolveOptions{
		DefaultDatabase: database,
		SearchPath:      searchPath,
		Catalog: analysis.Catalog{
			GetDatabase: base.QuerySpanDatabaseGetter(ctx, gCtx, false, func(db *model.DatabaseMetadata) analysis.Database {
				return &catalogDatabase{database: db}
			}),
		},
	})
	var syntaxErr *analysis.SyntaxError
	if errors.As(err, &syntaxErr) {
		err = base.SyntaxErrorAtOffset(stmt.Text, syntaxErr.Position, syntaxErr.Error())
	}
	querySpan, err := base.OmniResolvedSpanResult(span, err)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}
	return querySpan, nil
}

// catalogDatabase answers omni's lookups with the metadata model's own name matching.
type catalogDatabase struct {
	database *model.DatabaseMetadata
}

func (d *catalogDatabase) Metadata() *metadata.DatabaseSchemaMetadata {
	return d.database.GetProto()
}

func (d *catalogDatabase) GetTable(schema, table string) *metadata.TableMetadata {
	tableMetadata := d.database.GetSchemaMetadata(schema).GetTable(table)
	if tableMetadata == nil {
		return nil
	}
	return tableMetadata.GetProto()
}

func (d *catalogDatabase) SearchRelation(searchPath []string, name string) (string, string) {
	return d.database.SearchRelation(searchPath, name)
}
