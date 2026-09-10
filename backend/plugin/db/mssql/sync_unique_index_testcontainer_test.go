package mssql

import (
	"context"
	"testing"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestSyncUniqueIndex covers which synced indexes carry Unique. SQL Server
// splits the answer across two catalog columns: is_unique_constraint for a
// UNIQUE constraint and is_unique for a standalone CREATE UNIQUE INDEX. Sync
// reads them in separate queries, and reading only the first reported every
// CREATE UNIQUE INDEX as non-unique, which made the omni parser and sync
// disagree and left every declared unique index looking like a pending change.
func TestSyncUniqueIndex(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container := testcontainer.SharedMSSQLContainer(t)

	driver := newSyncTestDatabase(ctx, t, container)
	executeBatches(ctx, t, driver, `
CREATE TABLE dbo.index_kinds (
    id INT IDENTITY(1,1) PRIMARY KEY,
    code VARCHAR(50) UNIQUE,
    name NVARCHAR(100) NOT NULL,
    status INT,
    tag NVARCHAR(50)
);
GO

CREATE UNIQUE INDEX idx_unique_name ON dbo.index_kinds(name);
CREATE UNIQUE INDEX idx_unique_filtered ON dbo.index_kinds(status) WHERE status IS NOT NULL;
CREATE INDEX idx_plain_tag ON dbo.index_kinds(tag);
CREATE UNIQUE INDEX idx_unique_desc ON dbo.index_kinds(name DESC, status ASC);
GO
`)

	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)

	table := requireTable(t, metadata, "dbo", "index_kinds")
	byName := make(map[string]*storepb.IndexMetadata)
	for _, index := range table.Indexes {
		byName[index.Name] = index
	}

	// A standalone CREATE UNIQUE INDEX is unique but is not a constraint.
	for _, name := range []string{"idx_unique_name", "idx_unique_filtered", "idx_unique_desc"} {
		index := byName[name]
		require.NotNilf(t, index, "index %s not synced", name)
		require.Truef(t, index.Unique, "%s was created UNIQUE", name)
		require.Falsef(t, index.Primary, "%s is not a primary key", name)
		require.Falsef(t, index.IsConstraint, "%s is an index, not a constraint", name)
	}

	plain := byName["idx_plain_tag"]
	require.NotNil(t, plain)
	require.False(t, plain.Unique, "a plain CREATE INDEX must not be reported unique")

	// The UNIQUE column constraint keeps arriving through the constraint query,
	// which is the only path that sets IsConstraint.
	var columnConstraint *storepb.IndexMetadata
	for name, index := range byName {
		if index.IsConstraint && !index.Primary {
			require.Nilf(t, columnConstraint, "expected one unique constraint, also got %s", name)
			columnConstraint = index
		}
	}
	require.NotNil(t, columnConstraint, "the UNIQUE constraint on code should be synced")
	require.True(t, columnConstraint.Unique)
	require.Equal(t, []string{"code"}, columnConstraint.Expressions)

	var primary *storepb.IndexMetadata
	for _, index := range byName {
		if index.Primary {
			primary = index
		}
	}
	require.NotNil(t, primary)
	require.True(t, primary.Unique, "a primary key is unique")
	require.Equal(t, []string{"id"}, primary.Expressions)

	// Column order and direction must survive alongside the unique flag.
	require.Equal(t, []string{"name", "status"}, byName["idx_unique_desc"].Expressions)
	require.Equal(t, []bool{true, false}, byName["idx_unique_desc"].Descending)
}
