package clickhouse

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseDefinition(t *testing.T) {
	table := &metadatapb.TableMetadata{
		Name:   "events",
		Engine: "MergeTree",
		Columns: []*metadatapb.ColumnMetadata{
			{Name: "id", Type: "UInt64", Position: 1, Default: "NULL"},
			{Name: "note", Type: "Nullable(String)", Position: 2, Nullable: true, Default: "'hello'", Comment: "event note"},
		},
		Indexes: []*metadatapb.IndexMetadata{
			{Name: "idx_note", Type: "bloom_filter", Expressions: []string{"note"}, Granularity: 4},
			{Primary: true, Expressions: []string{"id"}},
		},
		SortingKeys: []string{"id"},
		Comment:     "audit events",
	}
	view := &metadatapb.ViewMetadata{
		Name:       "recent_events",
		Definition: "SELECT id, note FROM events",
	}

	gotTable, err := schema.GetTableDefinition(storepb.Engine_CLICKHOUSE, "", table, nil)
	require.NoError(t, err)
	require.Contains(t, gotTable, "CREATE TABLE `events`")
	require.Contains(t, gotTable, "INDEX `idx_note` note TYPE bloom_filter GRANULARITY 4")
	require.Contains(t, gotTable, "ENGINE = MergeTree\nORDER BY id\nPRIMARY KEY id\nCOMMENT 'audit events';")

	gotView, err := schema.GetViewDefinition(storepb.Engine_CLICKHOUSE, "", view)
	require.NoError(t, err)
	require.Contains(t, gotView, "CREATE OR REPLACE VIEW `recent_events` AS SELECT id, note FROM events;")

	gotDB, err := schema.GetDatabaseDefinition(storepb.Engine_CLICKHOUSE, schema.GetDefinitionContext{}, &metadatapb.DatabaseSchemaMetadata{
		Name: "analytics",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{table},
			Views:  []*metadatapb.ViewMetadata{view},
		}},
	})
	require.NoError(t, err)
	require.Contains(t, gotDB, "CREATE TABLE `events`")
	require.Contains(t, gotDB, "CREATE OR REPLACE VIEW `recent_events`")
}
