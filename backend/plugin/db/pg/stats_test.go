package pg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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

	driver := pgTestDriver(ctx, t, pgContainer, dbName, nil)

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

	_, err = rawDB.ExecContext(ctx, `
		CREATE SCHEMA app;
		CREATE TABLE app.only_in_app (id int);
		INSERT INTO app.only_in_app SELECT generate_series(1, 100);
		ANALYZE app.only_in_app;
	`)
	require.NoError(t, err)
	// The search path is set for the estimate's own transaction and rolled back,
	// so the next estimate does not see the schema.
	rows, err := driver.CountAffectedRows(ctx, base.WithSearchPath("DELETE FROM only_in_app;", []string{"app"}))
	require.NoError(t, err)
	require.Equal(t, int64(100), rows)
	_, err = driver.CountAffectedRows(ctx, "DELETE FROM only_in_app;")
	require.ErrorContains(t, err, "does not exist")
}
