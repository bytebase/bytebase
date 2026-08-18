package pg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

const partitionIndexSetupSQL = `
CREATE TABLE accounts (
    tenant text NOT NULL,
    id integer NOT NULL,
    email text,
    PRIMARY KEY (tenant, id)
) PARTITION BY LIST (tenant);
CREATE TABLE accounts_t1 PARTITION OF accounts FOR VALUES IN ('t1');
CREATE TABLE accounts_t2 PARTITION OF accounts FOR VALUES IN ('t2') PARTITION BY RANGE (id);
CREATE TABLE accounts_t2_a PARTITION OF accounts_t2 FOR VALUES FROM (0) TO (100);
CREATE INDEX accounts_email_idx ON accounts (email);
CREATE INDEX accounts_t1_local_idx ON accounts_t1 (email, id);
CREATE INDEX accounts_t2_local_idx ON accounts_t2 (id, email);
COMMENT ON INDEX accounts_t1_email_idx IS 'clone comment';
`

// TestSyncPartitionIndexesOmitInheritedClones verifies that synced partition
// metadata keeps locally-created indexes and commented clones, but not plain
// clones propagated from a parent partitioned index.
func TestSyncPartitionIndexesOmitInheritedClones(t *testing.T) {
	ctx := context.Background()

	pgContainer := testcontainer.GetTestPgContainer(ctx, t)
	defer pgContainer.Close(ctx)

	pgDB := pgContainer.GetDB()
	require.NoError(t, pgDB.Ping())

	_, err := pgDB.Exec(partitionIndexSetupSQL)
	require.NoError(t, err)

	driver := &Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "postgres",
			Host:     pgContainer.GetHost(),
			Port:     pgContainer.GetPort(),
			Database: "postgres",
		},
		Password: "root-password",
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "16.0",
			DatabaseName:  "postgres",
		},
	}

	openedDriver, err := driver.Open(ctx, storepb.Engine_POSTGRES, config)
	require.NoError(t, err)
	defer openedDriver.Close(ctx)

	pgDriver, ok := openedDriver.(*Driver)
	require.True(t, ok)

	metadata, err := pgDriver.SyncDBSchema(ctx)
	require.NoError(t, err)

	var publicSchema *storepb.SchemaMetadata
	for _, schema := range metadata.Schemas {
		if schema.Name == "public" {
			publicSchema = schema
			break
		}
	}
	require.NotNil(t, publicSchema, "public schema not found in synced metadata")

	var accounts *storepb.TableMetadata
	for _, table := range publicSchema.Tables {
		if table.Name == "accounts" {
			accounts = table
			break
		}
	}
	require.NotNil(t, accounts, "accounts table not found in synced metadata")

	require.ElementsMatch(t, []string{"accounts_pkey", "accounts_email_idx"}, indexNames(accounts.Indexes))

	partitions := make(map[string]*storepb.TablePartitionMetadata)
	for _, partition := range accounts.Partitions {
		partitions[partition.Name] = partition
	}
	require.Len(t, partitions, 2)
	require.Contains(t, partitions, "accounts_t1")
	require.Contains(t, partitions, "accounts_t2")

	require.ElementsMatch(t, []string{"accounts_t1_local_idx", "accounts_t1_email_idx"}, indexNames(partitions["accounts_t1"].Indexes))
	require.ElementsMatch(t, []string{"accounts_t2_local_idx"}, indexNames(partitions["accounts_t2"].Indexes))

	for _, index := range partitions["accounts_t1"].Indexes {
		if index.Name == "accounts_t1_email_idx" {
			require.Equal(t, "clone comment", index.Comment)
			require.Equal(t, "public", index.ParentIndexSchema)
			require.Equal(t, "accounts_email_idx", index.ParentIndexName)
		}
	}

	require.Len(t, partitions["accounts_t2"].Subpartitions, 1)
	subpartition := partitions["accounts_t2"].Subpartitions[0]
	require.Equal(t, "accounts_t2_a", subpartition.Name)
	require.Empty(t, indexNames(subpartition.Indexes))
}

func indexNames(indexes []*storepb.IndexMetadata) []string {
	names := make([]string, 0, len(indexes))
	for _, index := range indexes {
		names = append(names, index.Name)
	}
	return names
}
