package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestSyncInstanceBasicMetaSkipsDatabaseDiscovery(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	databaseName := "basic_meta_should_not_list_me"

	sqlDB, err := sql.Open("sqlite3", filepath.Join(dir, databaseName+".db"))
	require.NoError(t, err)
	defer sqlDB.Close()
	_, err = sqlDB.Exec("CREATE TABLE t(id INTEGER PRIMARY KEY)")
	require.NoError(t, err)

	driver, err := newDriver().Open(ctx, storepb.Engine_SQLITE, db.ConnectionConfig{
		DataSource: &storepb.DataSource{Host: dir},
	})
	require.NoError(t, err)
	defer driver.Close(ctx)

	meta, err := driver.SyncInstanceBasicMeta(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, meta.Version)
	require.Empty(t, meta.Databases)
}
