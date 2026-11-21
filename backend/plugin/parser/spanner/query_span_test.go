package spanner

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/googlesql/googlesqltest"
)

func TestGetQuerySpanSeam(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "",
				Tables: []*metadatapb.TableMetadata{
					{Name: "users", Columns: []*metadatapb.ColumnMetadata{{Name: "id"}, {Name: "email"}}},
				},
			},
			{
				Name: "analytics",
				Tables: []*metadatapb.TableMetadata{
					{Name: "events", Columns: []*metadatapb.ColumnMetadata{{Name: "id"}, {Name: "payload"}}},
				},
			},
		},
	}
	getter, lister := googlesqltest.BuildMockDatabaseMetadataGetter(storepb.Engine_SPANNER, []*metadatapb.DatabaseSchemaMetadata{metadata})
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		Engine:                  storepb.Engine_SPANNER,
	}

	// Plain SELECT 1 must report Type=Select for new-ACL access checks (BYT-9627).
	spans, err := base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_SPANNER, []base.Statement{
		{Text: "SELECT 1"},
	}, "db", "", false)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	require.Equal(t, base.Select, spans[0].Type)

	// Named schema lookup with Spanner's IgnoreWrittenDatabase + SchemaScopedMetadata config.
	span, err := GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT id, payload FROM other.analytics.events"}, "db", "", false)
	require.NoError(t, err)
	require.Equal(t, base.Select, span.Type)
	require.Equal(t, base.SourceColumnSet{{Database: "db", Schema: "analytics", Table: "events"}: true}, span.SourceColumns)
	require.Equal(t, []string{"analytics.events.id"}, googlesqltest.SourcesOf(span.Results[0]))
	require.Equal(t, []string{"analytics.events.payload"}, googlesqltest.SourcesOf(span.Results[1]))

	// System-only queries early-return an empty SelectInfoSchema span (including SPANNER_SYS).
	span, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT * FROM SPANNER_SYS.QUERY_STATS_TOP_MINUTE"}, "db", "", false)
	require.NoError(t, err)
	require.Equal(t, base.SelectInfoSchema, span.Type)
	require.Empty(t, span.Results)

	// MixUserSystemTablesError conversion.
	_, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT * FROM users JOIN INFORMATION_SCHEMA.TABLES ON TRUE"}, "db", "", false)
	require.ErrorIs(t, err, base.MixUserSystemTablesError)

	// ResourceNotFoundError conversion on missing table.
	span, err = GetQuerySpan(context.Background(), gCtx, base.Statement{Text: "SELECT id FROM missing"}, "db", "", false)
	require.NoError(t, err)
	var notFound *base.ResourceNotFoundError
	require.ErrorAs(t, span.NotFoundError, &notFound)
}
