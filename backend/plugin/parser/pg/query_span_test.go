package pg

import (
	"context"
	"errors"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func buildSeamMetadataGetter(metas ...*metadatapb.DatabaseSchemaMetadata) base.GetDatabaseMetadataFunc {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
		for _, m := range metas {
			if m.Name == databaseName {
				return databaseName, model.NewDatabaseMetadata(m, nil, nil, storepb.Engine_POSTGRES, true), nil
			}
		}
		return "", nil, nil
	}
}

func TestGetQuerySpanSeam(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "app",
				Tables: []*metadatapb.TableMetadata{{
					Name:    "t",
					Columns: []*metadatapb.ColumnMetadata{{Name: "email", Type: "text"}},
				}},
			},
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name:    "t",
						Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "int4"}, {Name: "ssn", Type: "text"}},
					},
					{Name: "degraded"},
				},
			},
		},
	}
	gCtx := base.GetQuerySpanContext{
		InstanceID:              "inst",
		GetDatabaseMetadataFunc: buildSeamMetadataGetter(meta),
	}

	// Registry wiring for both PostgreSQL and CockroachDB, plus selected-schema search path.
	for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB} {
		spans, err := base.GetQuerySpan(context.Background(), gCtx, engine, []base.Statement{{Text: "SELECT email FROM t WHERE email = 'a'"}}, "db", "app", false)
		require.NoError(t, err)
		require.Len(t, spans, 1)
		require.Equal(t, base.Select, spans[0].Type)
		require.Equal(t, base.SourceColumnSet{{Database: "db", Schema: "app", Table: "t", Column: "email"}: true}, spans[0].Results[0].SourceColumns)
		require.Equal(t, base.SourceColumnSet{{Database: "db", Schema: "app", Table: "t", Column: "email"}: true}, spans[0].PredicateColumns)
	}

	// ExplainAnalyze and query types convert across the seam.
	span, err := GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "EXPLAIN ANALYZE DELETE FROM public.t"}, "db", "", false)
	require.NoError(t, err)
	require.Equal(t, base.DML, span.Type)
	require.True(t, span.ExplainAnalyze)

	// UnresolvedColumnsError converts to base.UnresolvedColumnsError.
	span, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT * FROM public.degraded"}, "db", "", false)
	require.NoError(t, err)
	require.NotNil(t, span.UnresolvedColumnsError)
	require.Equal(t, []string{"db"}, span.UnresolvedColumnsError.Databases())

	// MixUserSystemTablesError and SyntaxError (with 1-based rune position) convert cleanly.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT * FROM public.t, pg_catalog.pg_class"}, "db", "", false)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT '你好' FROM WHERE"}, "db", "", false)
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.Equal(t, int32(1), syntaxErr.Position.Line)
	require.Greater(t, syntaxErr.Position.Column, int32(1))

	// Nil getter and unsynced (nil) metadata fail cleanly.
	_, err = GetQuerySpan(context.Background(), base.GetQuerySpanContext{}, base.Statement{Text: "SELECT 1"}, "db", "", false)
	require.ErrorContains(t, err, "GetDatabaseMetadataFunc is not set")

	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT 1"}, "unsynced", "", false)
	require.ErrorContains(t, err, `database metadata for database "unsynced" not found`)

	errCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "", nil, errors.New("boom")
		},
	}
	_, err = GetQuerySpan(context.Background(), errCtx, base.Statement{Text: "SELECT 1"}, "db", "", false)
	require.ErrorContains(t, err, "boom")
}
