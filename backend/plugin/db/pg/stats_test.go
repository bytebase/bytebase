package pg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestCountAffectedRows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, rawDB := testcontainer.NewPgDatabase(t)
	_, err := rawDB.ExecContext(ctx, `
		CREATE TABLE s (id int PRIMARY KEY, flag bool NOT NULL);
		INSERT INTO s SELECT g, g <= 10 FROM generate_series(1, 100) g;
		CREATE TABLE big (id int PRIMARY KEY, s_id int NOT NULL, v int);
		INSERT INTO big SELECT g, (g % 100) + 1, 0 FROM generate_series(1, 10000) g;
		CREATE TABLE big_archive (LIKE big);
		ANALYZE;
	`)
	require.NoError(t, err)

	openedDriver, err := (&Driver{}).Open(ctx, storepb.Engine_POSTGRES, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     pgContainer.GetHost(),
			Port:     pgContainer.GetPort(),
			Username: "postgres",
		},
		Password:          "root-password",
		ConnectionContext: db.ConnectionContext{DatabaseName: dbName},
	})
	require.NoError(t, err)
	defer openedDriver.Close(ctx)
	driver, ok := openedDriver.(*Driver)
	require.True(t, ok)

	// Each statement modifies the 1000 big rows joined to the 10 flagged s rows.
	for _, statement := range []string{
		"DELETE FROM big USING s WHERE big.s_id = s.id AND s.flag;",
		"UPDATE big SET v = 1 FROM s WHERE big.s_id = s.id AND s.flag;",
		"DELETE FROM big WHERE s_id IN (SELECT id FROM s WHERE flag);",
		"INSERT INTO big_archive SELECT big.* FROM big JOIN s ON big.s_id = s.id WHERE s.flag;",
		"MERGE INTO big USING s ON big.s_id = s.id AND s.flag WHEN MATCHED THEN UPDATE SET v = 1;",
	} {
		rows, err := driver.CountAffectedRows(ctx, statement)
		require.NoError(t, err, statement)
		require.Greater(t, rows, int64(500), statement)
		require.LessOrEqual(t, rows, int64(2000), statement)
	}

	_, err = driver.CountAffectedRows(ctx, "SELECT * FROM big;")
	require.Error(t, err)
}
