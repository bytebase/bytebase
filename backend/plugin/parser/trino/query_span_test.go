package trino

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func buildMockDatabaseMetadataGetter(databaseMetadata []*metadatapb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			for _, metadata := range databaseMetadata {
				if metadata.Name == databaseName {
					return "", model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true /* isObjectCaseSensitive */), nil
				}
			}
			return "", nil, nil
		}, func(_ context.Context, _ string) ([]string, error) {
			names := make([]string, 0, len(databaseMetadata))
			for _, metadata := range databaseMetadata {
				names = append(names, metadata.Name)
			}
			return names, nil
		}
}

func TestGetQuerySpanSeam(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "catalog1",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{{
				Name: "users",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "id", Type: "integer"},
					{Name: "email", Type: "varchar"},
				},
			}},
		}},
	}
	getter, lister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{metadata})
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		Engine:                  storepb.Engine_TRINO,
	}

	// Registry dispatch, result lineage, and predicate columns conversion.
	spans, err := base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_TRINO, []base.Statement{
		{Text: "SELECT u.email FROM users u WHERE u.id = 1"},
	}, "catalog1", "public", false)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	require.Equal(t, base.Select, spans[0].Type)
	require.Nil(t, spans[0].NotFoundError)
	require.Len(t, spans[0].Results, 1)
	require.Equal(t, "email", spans[0].Results[0].Name)
	require.True(t, spans[0].Results[0].SourceColumns[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "users", Column: "email"}])
	require.True(t, spans[0].PredicateColumns[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "users", Column: "id"}])

	// QueryType conversion across statement classes.
	for _, tc := range []struct {
		sql  string
		want base.QueryType
	}{
		{"SELECT * FROM system.runtime.nodes;", base.SelectInfoSchema},
		{"EXPLAIN SELECT * FROM users;", base.Explain},
		{"INSERT INTO users (id) VALUES (1);", base.DML},
		{"CREATE TABLE t (id INT);", base.DDL},
	} {
		span, err := GetQuerySpan(context.Background(), gCtx, base.Statement{Text: tc.sql}, "catalog1", "public", false)
		require.NoErrorf(t, err, "%q", tc.sql)
		require.Equalf(t, tc.want, span.Type, "%q", tc.sql)
	}

	// ResourceNotFoundError conversion when callback returns *base.ResourceNotFoundError.
	missingDB := "ghost"
	nfCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "", nil, &base.ResourceNotFoundError{Database: &missingDB}
		},
		ListDatabaseNamesFunc: lister,
		Engine:                storepb.Engine_TRINO,
	}
	span, err := GetQuerySpan(context.Background(), nfCtx, base.Statement{Text: "SELECT * FROM ghost.public.users"}, "catalog1", "public", false)
	require.NoError(t, err)
	var notFound *base.ResourceNotFoundError
	require.ErrorAs(t, span.NotFoundError, &notFound)
	require.NotNil(t, notFound.Database)
	require.Equal(t, "ghost", *notFound.Database)
}
