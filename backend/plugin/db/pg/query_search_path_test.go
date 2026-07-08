package pg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestQueryConnSearchPathIncludesPublicAfterSelectedSchema(t *testing.T) {
	ctx := context.Background()

	pgContainer := testcontainer.GetTestPgContainer(ctx, t)
	defer pgContainer.Close(ctx)

	rawDB := pgContainer.GetDB()
	require.NoError(t, rawDB.Ping())
	_, err := rawDB.ExecContext(ctx, `
		CREATE SCHEMA app;
		CREATE TABLE public.lookup_precedence (marker text);
		CREATE TABLE app.lookup_precedence (marker text);
		INSERT INTO public.lookup_precedence VALUES ('public');
		INSERT INTO app.lookup_precedence VALUES ('app');
		CREATE FUNCTION public.polar_osfs_extract_table_name(rel text) RETURNS text
			LANGUAGE sql
			AS $$ SELECT rel $$;
		CREATE FUNCTION public.polar_alter_relation_to_oss_with_indexes(rel text) RETURNS text
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN polar_osfs_extract_table_name(rel);
			END;
			$$;
	`)
	require.NoError(t, err)

	driver, err := (&Driver{}).Open(ctx, storepb.Engine_POSTGRES, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     pgContainer.GetHost(),
			Port:     pgContainer.GetPort(),
			Username: "postgres",
		},
		Password:          "root-password",
		ConnectionContext: db.ConnectionContext{DatabaseName: "postgres"},
	})
	require.NoError(t, err)
	defer driver.Close(ctx)

	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	queryContext := db.QueryContext{
		Schema:               "app",
		Limit:                5000,
		MaximumSQLResultSize: 1 << 30,
	}
	results, err := driver.QueryConn(ctx, conn, `SELECT public.polar_alter_relation_to_oss_with_indexes('content_messages_2025q1'::text);`, queryContext)
	require.NoError(t, err)
	require.Equal(t, "content_messages_2025q1", firstStringValue(t, results))

	results, err = driver.QueryConn(ctx, conn, `SELECT marker FROM lookup_precedence;`, queryContext)
	require.NoError(t, err)
	require.Equal(t, "app", firstStringValue(t, results))
}

func TestQueryConnSearchPathEscapesSelectedSchemaName(t *testing.T) {
	ctx := context.Background()

	pgContainer := testcontainer.GetTestPgContainer(ctx, t)
	defer pgContainer.Close(ctx)

	rawDB := pgContainer.GetDB()
	require.NoError(t, rawDB.Ping())
	_, err := rawDB.ExecContext(ctx, `
		CREATE SCHEMA "app""schema";
		CREATE TABLE "app""schema".lookup_precedence (marker text);
		INSERT INTO "app""schema".lookup_precedence VALUES ('quoted');
	`)
	require.NoError(t, err)

	driver, err := (&Driver{}).Open(ctx, storepb.Engine_POSTGRES, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     pgContainer.GetHost(),
			Port:     pgContainer.GetPort(),
			Username: "postgres",
		},
		Password:          "root-password",
		ConnectionContext: db.ConnectionContext{DatabaseName: "postgres"},
	})
	require.NoError(t, err)
	defer driver.Close(ctx)

	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	results, err := driver.QueryConn(ctx, conn, `SELECT marker FROM lookup_precedence;`, db.QueryContext{
		Schema:               `app"schema`,
		Limit:                5000,
		MaximumSQLResultSize: 1 << 30,
	})
	require.NoError(t, err)
	require.Equal(t, "quoted", firstStringValue(t, results))
}

func firstStringValue(t *testing.T, results []*v1pb.QueryResult) string {
	t.Helper()
	require.NotEmpty(t, results)
	require.Empty(t, results[0].GetError())
	require.NotEmpty(t, results[0].GetRows())
	require.NotEmpty(t, results[0].GetRows()[0].GetValues())
	return results[0].GetRows()[0].GetValues()[0].GetStringValue()
}
