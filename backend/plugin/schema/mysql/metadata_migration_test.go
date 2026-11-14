package mysql

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
)

func oneTableSnapshot(name string, columns []*metadatapb.ColumnMetadata, views ...*metadatapb.ViewMetadata) *metadatapb.DatabaseSchemaMetadata {
	return &metadatapb.DatabaseSchemaMetadata{
		Name: name,
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{Name: "t", Columns: columns}},
			Views:  views,
		}},
	}
}

func TestOmniDiffMetadataMigrationRefusesAChangedUnloadableObject(t *testing.T) {
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "int"}
	_, err := omniDiffMetadataMigration(
		metadataFromProto(oneTableSnapshot("d", []*metadatapb.ColumnMetadata{id}, &metadatapb.ViewMetadata{Name: "v", Definition: "select ((broken from"})),
		metadataFromProto(oneTableSnapshot("d", []*metadatapb.ColumnMetadata{id}, &metadatapb.ViewMetadata{Name: "v", Definition: "select ((still broken from"})),
	)
	require.ErrorContains(t, err, "view:v")

	// A default that does not parse leaves the table incomplete.
	_, err = omniDiffMetadataMigration(
		metadataFromProto(oneTableSnapshot("d", []*metadatapb.ColumnMetadata{id})),
		metadataFromProto(oneTableSnapshot("d", []*metadatapb.ColumnMetadata{{Name: "id", Type: "int", Default: "((("}})),
	)
	require.ErrorContains(t, err, "table:t")
}

// A database diffed against another database's changelog: sync qualifies each
// view body with its own database.
func TestOmniDiffMetadataMigrationAcrossDatabaseNames(t *testing.T) {
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "int"}
	view := func(db string) *metadatapb.ViewMetadata {
		return &metadatapb.ViewMetadata{Name: "v", Definition: "select `" + db + "`.`t`.`id` AS `id` from `" + db + "`.`t`"}
	}
	sql, err := omniDiffMetadataMigration(
		metadataFromProto(oneTableSnapshot("prod", []*metadatapb.ColumnMetadata{id}, view("prod"))),
		metadataFromProto(oneTableSnapshot("staging", []*metadatapb.ColumnMetadata{id}, view("staging"))),
	)
	require.NoError(t, err)
	require.Empty(t, sql)

	sql, err = omniDiffMetadataMigration(
		metadataFromProto(oneTableSnapshot("prod", []*metadatapb.ColumnMetadata{id})),
		metadataFromProto(oneTableSnapshot("staging", []*metadatapb.ColumnMetadata{id}, view("staging"))),
	)
	require.NoError(t, err)
	require.NotContains(t, sql, "staging")
	require.NotContains(t, sql, sdlDatabaseContext)
	require.Contains(t, sql, "VIEW `v` AS select `t`.`id` AS `id` from `t`")
}
