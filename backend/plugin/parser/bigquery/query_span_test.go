package bigquery

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/googlesql/googlesqltest"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestGetQuerySpanSeam(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "ds1",
		Schemas: googlesqltest.DefaultSchemaTables(
			&metadatapb.TableMetadata{
				Name:    "users",
				Columns: []*metadatapb.ColumnMetadata{{Name: "id"}, {Name: "email"}},
			},
		),
	}
	getter, lister := googlesqltest.BuildMockDatabaseMetadataGetter(storepb.Engine_BIGQUERY, []*metadatapb.DatabaseSchemaMetadata{metadata})
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		Engine:                  storepb.Engine_BIGQUERY,
	}

	// Registry dispatch, dataset-as-database metadata lookup, and lineage conversion.
	spans, err := base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_BIGQUERY, []base.Statement{
		{Text: "SELECT id, email FROM ds1.users"},
	}, "ds1", "", false)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	require.Equal(t, base.Select, spans[0].Type)
	require.Nil(t, spans[0].NotFoundError)
	require.Equal(t, base.SourceColumnSet{{Database: "ds1", Table: "users"}: true}, spans[0].SourceColumns)
	require.Len(t, spans[0].Results, 2)
	require.Equal(t, []string{"users.id"}, googlesqltest.SourcesOf(spans[0].Results[0]))
	require.Equal(t, []string{"users.email"}, googlesqltest.SourcesOf(spans[0].Results[1]))

	// BigQuery dialect flag: SET classifies as Select.
	span, err := GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SET x = 1"}, "ds1", "", false)
	require.NoError(t, err)
	require.Equal(t, base.Select, span.Type)

	// Non-SELECT query types map through base.OmniQueryType.
	span, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "INSERT INTO users (id) VALUES (1)"}, "ds1", "", false)
	require.NoError(t, err)
	require.Equal(t, base.DML, span.Type)

	// MixUserSystemTablesError conversion.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT * FROM users JOIN INFORMATION_SCHEMA.TABLES ON TRUE"}, "ds1", "", false)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	// ResourceNotFoundError conversion on missing table and callback returning ResourceNotFoundError.
	span, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT id FROM missing"}, "ds1", "", false)
	require.NoError(t, err)
	var notFound *base.ResourceNotFoundError
	require.ErrorAs(t, span.NotFoundError, &notFound)

	missingDB := "ghost"
	nfCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "", nil, &base.ResourceNotFoundError{Database: &missingDB}
		},
		ListDatabaseNamesFunc: lister,
		Engine:                storepb.Engine_BIGQUERY,
	}
	span, err = GetQuerySpan(context.Background(), nfCtx, base.Statement{Text: "SELECT id FROM ghost.users"}, "ds1", "", false)
	require.NoError(t, err)
	require.ErrorAs(t, span.NotFoundError, &notFound)
	require.NotNil(t, notFound.Database)
	require.Equal(t, "ghost", *notFound.Database)
}
