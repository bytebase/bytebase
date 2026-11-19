package tidb

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// TestDumpRendersTheSyncedMetadata proves the adapter hands the schema it synced
// to the driver's dump. The fidelity of the rendering — a table's DDL, its
// partitions' order, the script's byte shape — is omni's tidb/driver cover.
func TestDumpRendersTheSyncedMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE %[1]s.orders (
			id BIGINT PRIMARY KEY AUTO_RANDOM,
			note VARCHAR(64) NOT NULL DEFAULT 'n',
			KEY orders_note (note)
		) COMMENT = 'orders';
	`, database))
	require.NoError(t, err)

	driver := openTestDriverOn(ctx, t, container, database)
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })

	dump := dumpDatabase(ctx, t, driver)
	require.Contains(t, dump, "`orders`", "the dump renders the table the adapter synced")
	require.Contains(t, dump, "`orders_note`", "the dump renders the table's index")
}

func dumpDatabase(ctx context.Context, t *testing.T, driver *Driver) string {
	t.Helper()
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	var out bytes.Buffer
	require.NoError(t, driver.Dump(ctx, &out, metadata))
	return out.String()
}
