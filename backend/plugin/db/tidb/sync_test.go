package tidb

import (
	"context"
	"fmt"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestSyncDBSchemaPassesTheDriversMetadataThrough proves the adapter answers the
// contract's schema unchanged. What TiDB reports for a table — its columns,
// defaults, generated columns, indexes, partitions, foreign keys and views — is
// tidb/driver's corpus and golden, not this repository's.
func TestSyncDBSchemaPassesTheDriversMetadataThrough(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s.orders (id INT PRIMARY KEY, note VARCHAR(64))", database))
	require.NoError(t, err)

	driver := openTestDriverOn(ctx, t, container, database)
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })

	got, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	want, err := driver.impl.SyncDatabase(ctx)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(want, got, protocmp.Transform()))
}

// TestSyncDBSchemaReportsAMissingDatabase pins the one thing the adapter adds to
// a failed sync: the contract's ErrNotFound becomes the product's not-found
// error, which the sync runner reports as a failed sync.
func TestSyncDBSchemaReportsAMissingDatabase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	driver := openTestDriverOn(ctx, t, container, database+"_missing")
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })

	_, err := driver.SyncDBSchema(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("database %q not found", database+"_missing"))
}

// TestSyncInstanceMapsTheDriversInstance pins the adapter's mapping of the
// contract's instance onto the product's types: the version and the databases
// pass through, a role becomes a storepb.InstanceRole, and the server's
// lower_case_table_names reaches storepb.Instance.
func TestSyncInstanceMapsTheDriversInstance(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf(`
		CREATE USER 'bb_sync_probe'@'%%' IDENTIFIED BY 'probe-pass';
		GRANT SELECT ON %s.* TO 'bb_sync_probe'@'%%';`, database))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := container.GetDB().ExecContext(context.Background(), "DROP USER IF EXISTS 'bb_sync_probe'@'%'")
		require.NoError(t, err)
	})

	driver := openTestDriver(ctx, t, container)
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })

	instance, err := driver.SyncInstance(ctx)
	require.NoError(t, err)
	contract, err := driver.impl.SyncInstance(ctx)
	require.NoError(t, err)

	require.Equal(t, contract.Version, instance.Version)
	require.NotNil(t, instance.Metadata)
	require.Equal(t, contract.LowerCaseTableNames, instance.Metadata.GetMysqlLowerCaseTableNames())

	// A parallel test of this package may create a database between the two
	// reads, so compare the databases both reads saw.
	require.NotEmpty(t, instance.Databases)
	fromContract := map[string]*metadatapb.DatabaseSchemaMetadata{}
	for _, database := range contract.Databases {
		fromContract[database.GetName()] = database
	}
	mapped := 0
	for _, database := range instance.Databases {
		require.Empty(t, database.GetSchemas(), "an instance sync lists databases without their schemas")
		if want, ok := fromContract[database.GetName()]; ok {
			require.Empty(t, cmp.Diff(want, database, protocmp.Transform()), database.GetName())
			mapped++
		}
	}
	require.Positive(t, mapped)

	// The role this test made is the contract's role, in the product's shape.
	var role *storepb.InstanceRole
	for _, candidate := range instance.Metadata.GetRoles() {
		if candidate.GetName() == "'bb_sync_probe'@'%'" {
			role = candidate
		}
	}
	require.NotNil(t, role)
	var contractAttribute *string
	for _, candidate := range contract.Roles {
		if candidate.Name == "'bb_sync_probe'@'%'" {
			contractAttribute = candidate.Attribute
		}
	}
	require.NotNil(t, contractAttribute)
	require.Equal(t, *contractAttribute, role.GetAttribute())
	require.Contains(t, role.GetAttribute(), "GRANT SELECT")
}
