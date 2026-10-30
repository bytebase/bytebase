package store_test

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
)

func publicSchema(synonyms ...string) *metadatapb.SchemaMetadata {
	schema := &metadatapb.SchemaMetadata{Name: "PUBLIC"}
	for _, name := range synonyms {
		schema.Synonyms = append(schema.Synonyms, &metadatapb.SynonymMetadata{Name: name, TargetSchema: "SYS", TargetName: name})
	}
	return schema
}

func TestInstanceSchemaUpsert(t *testing.T) {
	t.Parallel()
	f := newStorePostgresFixture(t, `
		INSERT INTO instance (resource_id, workspace) VALUES ('oracle', 'default');
	`)

	got, err := f.store.GetInstanceSchema(f.ctx, "oracle")
	require.NoError(t, err)
	require.Nil(t, got)

	require.NoError(t, f.store.UpsertInstanceSchema(f.ctx, "oracle", publicSchema("DUAL")))
	got, err = f.store.GetInstanceSchema(f.ctx, "oracle")
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(publicSchema("DUAL"), got, protocmp.Transform()))

	rowVersion := func() string {
		var xmin string
		require.NoError(t, f.db.QueryRowContext(f.ctx, `SELECT xmin::text FROM instance_schema WHERE instance = 'oracle'`).Scan(&xmin))
		return xmin
	}
	before := rowVersion()
	require.NoError(t, f.store.UpsertInstanceSchema(f.ctx, "oracle", publicSchema("DUAL")))
	require.Equal(t, before, rowVersion())

	require.NoError(t, f.store.UpsertInstanceSchema(f.ctx, "oracle", publicSchema("DUAL", "PLAN_TABLE")))
	require.NotEqual(t, before, rowVersion())
	got, err = f.store.GetInstanceSchema(f.ctx, "oracle")
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(publicSchema("DUAL", "PLAN_TABLE"), got, protocmp.Transform()))
}

func TestDeleteInstancePurgesInstanceSchema(t *testing.T) {
	t.Parallel()
	f := newStorePostgresFixture(t, `
		INSERT INTO instance (resource_id, workspace, deleted) VALUES
			('purged', 'default', TRUE),
			('kept', 'default', TRUE);
	`)
	require.NoError(t, f.store.UpsertInstanceSchema(f.ctx, "purged", publicSchema("DUAL")))
	require.NoError(t, f.store.UpsertInstanceSchema(f.ctx, "kept", publicSchema("DUAL")))

	require.NoError(t, f.store.DeleteInstance(f.ctx, "default", "purged"))

	got, err := f.store.GetInstanceSchema(f.ctx, "purged")
	require.NoError(t, err)
	require.Nil(t, got)
	got, err = f.store.GetInstanceSchema(f.ctx, "kept")
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestDeleteProjectsPurgesProjectInstanceSchema(t *testing.T) {
	t.Parallel()
	f := newStorePostgresFixture(t, `
		INSERT INTO instance (resource_id, workspace, project) VALUES
			('project-instance', 'default', 'project-a'),
			('workspace-instance', 'default', NULL);
	`)
	require.NoError(t, f.store.UpsertInstanceSchema(f.ctx, "project-instance", publicSchema("DUAL")))
	require.NoError(t, f.store.UpsertInstanceSchema(f.ctx, "workspace-instance", publicSchema("DUAL")))

	require.NoError(t, f.store.DeleteProjects(f.ctx, "default", "project-a"))

	got, err := f.store.GetInstanceSchema(f.ctx, "project-instance")
	require.NoError(t, err)
	require.Nil(t, got)
	got, err = f.store.GetInstanceSchema(f.ctx, "workspace-instance")
	require.NoError(t, err)
	require.NotNil(t, got)
}
