package mssql

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// TestDumpRendersWhatTheOldRendererDropped pins the dump on a corpus the driver
// this replaced could not restore. Measured by rendering the same synced schema
// through both: the old renderer wrote no sequence and no trigger at all, put
// the filtered index's INCLUDE column in its key list and dropped the predicate,
// lost a DATETIMEOFFSET column's precision and a default constraint's name, and
// ended a view definition with two semicolons.
func TestDumpRendersWhatTheOldRendererDropped(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	driver, _ := newSyncTestDatabase(ctx, t, testcontainer.SharedMSSQLContainer(t))
	executeBatches(ctx, t, driver, `CREATE TABLE dbo.dump_t (
	id INT IDENTITY(1,1) PRIMARY KEY,
	at DATETIMEOFFSET NULL,
	flag BIT NOT NULL DEFAULT 0
);
GO

CREATE TABLE dbo.dump_u (id INT PRIMARY KEY, s_id INT NOT NULL, v INT NULL);
CREATE INDEX dump_filtered ON dbo.dump_u (v) INCLUDE (s_id) WHERE v IS NOT NULL;
GO

CREATE SEQUENCE dbo.dump_seq AS DECIMAL(18,0) START WITH 10 INCREMENT BY 5 MINVALUE 1 MAXVALUE 1000 CACHE 20;
GO

CREATE TABLE dbo.dump_w (id INT PRIMARY KEY, v INT NULL);
GO

CREATE TRIGGER dump_w_trg ON dbo.dump_w INSTEAD OF INSERT AS
BEGIN
	INSERT INTO dbo.dump_w (id, v) SELECT id, v FROM inserted;
END
GO

CREATE VIEW dbo.dump_v AS SELECT id FROM dbo.dump_t;
GO
`)

	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	var out bytes.Buffer
	require.NoError(t, driver.Dump(ctx, &out, metadata))
	dump := out.String()

	// A sequence is dumped with what defines it, so the dump restores it.
	require.Contains(t, dump, "CREATE SEQUENCE [dbo].[dump_seq]")
	require.Contains(t, dump, "AS decimal(18,0)")
	require.Contains(t, dump, "START WITH 10")
	require.Contains(t, dump, "INCREMENT BY 5")
	require.Contains(t, dump, "MINVALUE 1")
	require.Contains(t, dump, "MAXVALUE 1000")
	require.Contains(t, dump, "CACHE 20;")

	// A temporal column keeps the precision the server reports.
	require.Contains(t, dump, "[at] datetimeoffset(7)")

	// A default constraint keeps the name the server gave it.
	require.Contains(t, dump, "CONSTRAINT [DF__")

	// A filtered index keeps its INCLUDE list and its predicate, and the
	// INCLUDE column is not a key column.
	require.Contains(t, dump, "CREATE NONCLUSTERED INDEX [dump_filtered] ON\n[dbo].[dump_u] (\n    [v] ASC\n) INCLUDE ([s_id])\nWHERE ([v] IS NOT NULL);")
	require.NotContains(t, dump, "[s_id] ASC")

	// A trigger is dumped, with the timing the server reports.
	require.Contains(t, dump, "CREATE TRIGGER dump_w_trg ON dbo.dump_w INSTEAD OF INSERT")

	// A view definition ends with exactly one semicolon.
	require.Contains(t, dump, "CREATE VIEW dbo.dump_v AS SELECT id FROM dbo.dump_t;")
	require.NotContains(t, dump, ";;")
	require.False(t, strings.Contains(dump, "dump_v AS SELECT id FROM dbo.dump_t;\n;"), dump)
}
