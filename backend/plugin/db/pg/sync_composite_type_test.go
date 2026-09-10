package pg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestSyncCompositeTypes pins the exact metadata sync produces for composite
// types. The definition round-trip in backend/plugin/schema/pg compares two
// synced databases against each other, so a sync that is wrong the same way on
// both sides passes it vacuously; these assertions do not.
func TestSyncCompositeTypes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgContainer := testcontainer.SharedPgContainer(t)
	dbName, pgDB := testcontainer.NewPgDatabase(t)

	_, err := pgDB.ExecContext(ctx, `
CREATE TYPE zz_base AS (street text COLLATE "C", city varchar(50));
COMMENT ON TYPE zz_base IS 'base address type';
COMMENT ON COLUMN zz_base.street IS 'street line';
CREATE TYPE aa_nested AS (home zz_base, homes zz_base[], tags text[]);
CREATE TYPE empty_type AS ();
CREATE SCHEMA geo;
CREATE TYPE geo.point2 AS (lat numeric(9,6), lng numeric(9,6));
CREATE TYPE wrapper AS (p geo.point2);
CREATE TABLE plain_table (id int);
CREATE SCHEMA locale;
CREATE COLLATION locale.mycoll (locale = 'C');
CREATE TYPE coll_qualified AS (v text COLLATE locale.mycoll);
`)
	require.NoError(t, err)

	driver := &Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "postgres",
			Host:     pgContainer.GetHost(),
			Port:     pgContainer.GetPort(),
			Database: dbName,
		},
		Password: "root-password",
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "16.0",
			DatabaseName:  dbName,
		},
	}

	openedDriver, err := driver.Open(ctx, storepb.Engine_POSTGRES, config)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, openedDriver.Close(ctx)) })

	pgDriver, ok := openedDriver.(*Driver)
	require.True(t, ok)

	metadata, err := pgDriver.SyncDBSchema(ctx)
	require.NoError(t, err)

	compositesBySchema := make(map[string]map[string]*storepb.CompositeTypeMetadata)
	for _, schemaMeta := range metadata.Schemas {
		m := make(map[string]*storepb.CompositeTypeMetadata)
		for _, composite := range schemaMeta.CompositeTypes {
			m[composite.Name] = composite
		}
		compositesBySchema[schemaMeta.Name] = m
	}

	public := compositesBySchema["public"]
	require.Len(t, public, 5, "public schema should have exactly the created composite types")
	require.NotContains(t, public, "plain_table", "table row types must not be synced as composite types")

	collQualified := public["coll_qualified"]
	require.NotNil(t, collQualified)
	require.Len(t, collQualified.Attributes, 1)
	require.Equal(t, "locale.mycoll", collQualified.Attributes[0].Collation,
		"non-pg_catalog collations must be schema-qualified")

	base := public["zz_base"]
	require.NotNil(t, base)
	require.Equal(t, "base address type", base.Comment)
	require.Len(t, base.Attributes, 2)
	require.Equal(t, "street", base.Attributes[0].Name)
	require.Equal(t, "text", base.Attributes[0].Type)
	require.Equal(t, `"C"`, base.Attributes[0].Collation)
	require.Equal(t, "street line", base.Attributes[0].Comment)
	require.Equal(t, "city", base.Attributes[1].Name)
	require.Equal(t, "character varying(50)", base.Attributes[1].Type)
	require.Empty(t, base.Attributes[1].Collation)

	nested := public["aa_nested"]
	require.NotNil(t, nested)
	require.Len(t, nested.Attributes, 3)
	require.Equal(t, "public.zz_base", nested.Attributes[0].Type, "user-defined attribute types must be schema-qualified")
	require.Equal(t, "public.zz_base[]", nested.Attributes[1].Type)
	require.Equal(t, "text[]", nested.Attributes[2].Type)

	empty := public["empty_type"]
	require.NotNil(t, empty, "zero-attribute composite types must be synced")
	require.Empty(t, empty.Attributes)

	wrapper := public["wrapper"]
	require.NotNil(t, wrapper)
	require.Len(t, wrapper.Attributes, 1)
	require.Equal(t, "geo.point2", wrapper.Attributes[0].Type, "cross-schema attribute types must be schema-qualified")

	geo := compositesBySchema["geo"]
	require.Len(t, geo, 1)
	require.NotNil(t, geo["point2"])
}
