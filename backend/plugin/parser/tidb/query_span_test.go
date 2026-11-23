package tidb

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func buildMockDatabaseMetadataGetter(databaseMetadata []*metadatapb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			for _, metadata := range databaseMetadata {
				if metadata.Name == databaseName {
					return databaseName, model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_TIDB, false /* isObjectCaseSensitive */), nil
				}
			}
			return "", nil, errors.Errorf("database %q not found", databaseName)
		}, func(_ context.Context, _ string) ([]string, error) {
			names := make([]string, 0, len(databaseMetadata))
			for _, metadata := range databaseMetadata {
				names = append(names, metadata.Name)
			}
			return names, nil
		}
}

func TestGetQuerySpanSeam(t *testing.T) {
	metadata := []*metadatapb.DatabaseSchemaMetadata{
		{
			Name: "db",
			Schemas: []*metadatapb.SchemaMetadata{{
				Name: "",
				Tables: []*metadatapb.TableMetadata{{
					Name:    "t1",
					Columns: []*metadatapb.ColumnMetadata{{Name: "a"}, {Name: "b"}},
				}},
				Views: []*metadatapb.ViewMetadata{
					{Name: "mixed_v", Definition: "SELECT * FROM t1, mysql.user"},
					{Name: "v1", Definition: "SELECT * FROM v2"},
					{Name: "v2", Definition: "SELECT * FROM v1"},
				},
			}},
		},
		{
			Name: "mysql",
			Schemas: []*metadatapb.SchemaMetadata{{
				Name:   "",
				Tables: []*metadatapb.TableMetadata{{Name: "user", Columns: []*metadatapb.ColumnMetadata{{Name: "user"}}}},
			}},
		},
	}
	getter, lister := buildMockDatabaseMetadataGetter(metadata)
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		Engine:                  storepb.Engine_TIDB,
	}

	// Registry dispatch and result lineage.
	spans, err := base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_TIDB, []base.Statement{
		{Text: "SELECT a FROM t1 WHERE b = 1"},
	}, "db", "", false)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	require.Equal(t, base.Select, spans[0].Type)
	require.Nil(t, spans[0].NotFoundError)
	require.Equal(t, []base.QuerySpanResult{{
		Name:          "a",
		SourceColumns: base.SourceColumnSet{{Database: "db", Table: "t1", Column: "a"}: true},
	}}, spans[0].Results)

	// ExplainAnalyze flag conversion.
	span, err := GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "EXPLAIN ANALYZE SELECT a FROM t1"}, "db", "", false)
	require.NoError(t, err)
	require.Equal(t, base.Select, span.Type)
	require.True(t, span.ExplainAnalyze)

	// ResourceNotFoundError conversion on missing column.
	span, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT nope FROM t1"}, "db", "", false)
	require.NoError(t, err)
	var notFound *base.ResourceNotFoundError
	require.ErrorAs(t, span.NotFoundError, &notFound)
	require.NotNil(t, notFound.Column)
	require.Equal(t, "nope", *notFound.Column)

	// MixUserSystemTablesError conversion (direct and wrapped via view).
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT * FROM t1, mysql.user"}, "db", "", false)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT * FROM mixed_v"}, "db", "", false)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	// Parse error propagation.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{
		Text:  "SELECT * FORM invalid_table;",
		Start: &storepb.Position{Line: 3, Column: 1},
	}, "db", "", false)
	require.ErrorContains(t, err, "syntax error")

	// Cyclic view reference error propagation.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT * FROM v1"}, "db", "", false)
	require.ErrorContains(t, err, "cyclic view reference")
}
