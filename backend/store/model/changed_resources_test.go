package model

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestCountAffectedTableRows(t *testing.T) {
	type change struct {
		database, schema, table string
		affected                bool
	}
	for _, tc := range []struct {
		name    string
		engine  storepb.Engine
		changes []change
		want    int64
	}{
		{
			name:    "reviewed_database_table",
			engine:  storepb.Engine_POSTGRES,
			changes: []change{{"db", "public", "t", true}},
			want:    1_000_000,
		},
		{
			name:    "same_named_table_in_another_database",
			engine:  storepb.Engine_POSTGRES,
			changes: []change{{"other_db", "public", "t", true}},
			want:    0,
		},
		{
			name:    "table_changed_without_ddl",
			engine:  storepb.Engine_POSTGRES,
			changes: []change{{"db", "public", "t", false}},
			want:    0,
		},
		{
			name: "partitions_count_their_table_once",
			// The metadata has no per-partition row counts.
			engine: storepb.Engine_POSTGRES,
			changes: []change{
				{"db", "public", "events", true},
				{"db", "public", "events_2020", true},
				{"db", "public", "events_2021", true},
			},
			want: 50_000_000,
		},
		{
			name:    "case_insensitive_database_name",
			engine:  storepb.Engine_MYSQL,
			changes: []change{{"DB", "public", "t", true}},
			want:    1_000_000,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changedResources := NewChangedResources(newChangedResourcesTestMetadata(tc.engine))
			for _, c := range tc.changes {
				changedResources.AddTable(c.database, c.schema, &storepb.ChangedResourceTable{Name: c.table}, c.affected)
			}
			require.Equal(t, tc.want, changedResources.CountAffectedTableRows())
		})
	}
}

func TestBuildTableRows(t *testing.T) {
	changedResources := NewChangedResources(newChangedResourcesTestMetadata(storepb.Engine_POSTGRES))
	changedResources.AddTable("db", "public", &storepb.ChangedResourceTable{Name: "t"}, false)
	changedResources.AddTable("other_db", "public", &storepb.ChangedResourceTable{Name: "t"}, false)

	tableRows := map[string]int64{}
	for _, database := range changedResources.Build().GetDatabases() {
		for _, schema := range database.GetSchemas() {
			for _, table := range schema.GetTables() {
				tableRows[database.GetName()+"."+table.GetName()] = table.GetTableRows()
			}
		}
	}
	require.Equal(t, map[string]int64{"db.t": 1_000_000, "other_db.t": 0}, tableRows)
}

// newChangedResourcesTestMetadata is the reviewed database "db". MySQL metadata is built
// case-insensitive, as on a server with lower_case_table_names set.
func newChangedResourcesTestMetadata(engine storepb.Engine) *DatabaseMetadata {
	return NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{
				{Name: "t", RowCount: 1_000_000},
				{
					Name:       "events",
					RowCount:   50_000_000,
					Partitions: []*metadatapb.TablePartitionMetadata{{Name: "events_2020"}, {Name: "events_2021"}},
				},
			},
		}},
	}, nil, nil, engine, engine != storepb.Engine_MYSQL)
}
