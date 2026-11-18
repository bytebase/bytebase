package tidb

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestDumpRestores proves the dump of a synced schema replays: partitions keep
// their ordinal order in the text, and restoring it into an emptied database and
// dumping again is byte-identical. The old renderer sorted a table's partitions
// by name, so a table created p10, p2, p1 dumped as p1, p10, p2 and the server
// refused it ("VALUES LESS THAN value must be strictly increasing").
func TestDumpRestores(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE %[1]s.orders (
			id BIGINT PRIMARY KEY AUTO_RANDOM,
			note VARCHAR(64) NOT NULL DEFAULT 'n',
			amount DECIMAL(10,2) NOT NULL DEFAULT '0.00',
			KEY orders_note (note)
		) COMMENT = 'orders';
		CREATE TABLE %[1]s.events (
			id INT NOT NULL,
			PRIMARY KEY (id)
		) PARTITION BY RANGE (id) (
			PARTITION p10 VALUES LESS THAN (10),
			PARTITION p2 VALUES LESS THAN (20),
			PARTITION p1 VALUES LESS THAN (30)
		);
		CREATE VIEW %[1]s.order_notes AS SELECT id, note FROM %[1]s.orders;
	`, database))
	require.NoError(t, err)

	driver := openTestDriverOn(ctx, t, container, database)
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })
	first := dumpDatabase(ctx, t, driver)

	// The partitions appear in their ordinal order: the names tell a name-sorted
	// renderer p1, p10, p2, whose VALUES LESS THAN would not increase.
	require.Less(t,
		strings.Index(first, "PARTITION p10"),
		strings.Index(first, "PARTITION p2"),
		"partitions keep their ordinal order: %s", first,
	)

	copied := database + "_copy"
	_, err = container.GetDB().ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", copied))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := container.GetDB().ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE %s", copied))
		require.NoError(t, err)
	})

	copyDriver := openTestDriverOn(ctx, t, container, copied)
	t.Cleanup(func() { require.NoError(t, copyDriver.Close(ctx)) })
	_, err = copyDriver.Execute(ctx, first, db.ExecuteOptions{})
	require.NoError(t, err)

	second := dumpDatabase(ctx, t, copyDriver)
	require.Equal(t, first, second, "dump -> restore -> dump is byte-identical")
}

func dumpDatabase(ctx context.Context, t *testing.T, driver *Driver) string {
	t.Helper()
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	var out bytes.Buffer
	require.NoError(t, driver.Dump(ctx, &out, metadata))
	return out.String()
}
