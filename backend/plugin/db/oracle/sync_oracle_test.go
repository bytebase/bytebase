package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	"github.com/bytebase/bytebase/backend/plugin/schema"

	// Registers the Oracle dump.
	_ "github.com/bytebase/bytebase/backend/plugin/schema/oracle"
)

// newSyncTarget returns a driver opened on a fresh schema the way Bytebase
// opens one for a database: as the admin, with CURRENT_SCHEMA set to it.
func newSyncTarget(ctx context.Context, t *testing.T) (*Driver, *sql.DB, string) {
	t.Helper()
	container, schemaName := testcontainer.NewOracleSchema(t)
	opened, err := newDriver().Open(ctx, storepb.Engine_ORACLE, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:        container.GetHost(),
			Port:        container.GetPort(),
			Username:    container.GetUsername(),
			ServiceName: testcontainer.OracleServiceName,
		},
		Password:          container.GetPassword(),
		ConnectionContext: db.ConnectionContext{DatabaseName: schemaName},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = opened.Close(context.Background()) })
	driver, ok := opened.(*Driver)
	require.True(t, ok)
	instance, err := driver.SyncInstance(ctx)
	require.NoError(t, err)
	driver.connectionCtx.EngineVersion = instance.Version
	return driver, container.GetDB(), schemaName
}

func syncTestContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	t.Cleanup(cancel)
	return ctx
}

func execAll(ctx context.Context, t *testing.T, conn *sql.DB, statements ...string) {
	t.Helper()
	for _, statement := range statements {
		_, err := conn.ExecContext(ctx, statement)
		require.NoError(t, err, statement)
	}
}

func syncedSchema(ctx context.Context, t *testing.T, driver *Driver) *metadatapb.SchemaMetadata {
	t.Helper()
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Len(t, metadata.Schemas, 1)
	return metadata.Schemas[0]
}

func tableByName(t *testing.T, schemaMetadata *metadatapb.SchemaMetadata, name string) *metadatapb.TableMetadata {
	t.Helper()
	index := slices.IndexFunc(schemaMetadata.Tables, func(table *metadatapb.TableMetadata) bool { return table.Name == name })
	require.NotEqual(t, -1, index, "table %s", name)
	return schemaMetadata.Tables[index]
}

func names[T interface{ GetName() string }](items []T) []string {
	var result []string
	for _, item := range items {
		result = append(result, item.GetName())
	}
	return result
}

// syncedColumn is the part of a column these tests assert on.
type syncedColumn struct {
	Name       string
	Position   int32
	Invisible  bool
	Nullable   bool
	Default    string
	Expression string
}

func syncedColumns(columns []*metadatapb.ColumnMetadata) []syncedColumn {
	var result []syncedColumn
	for _, column := range columns {
		result = append(result, syncedColumn{
			Name:       column.Name,
			Position:   column.Position,
			Invisible:  column.IsInvisible,
			Nullable:   column.Nullable,
			Default:    column.Default,
			Expression: column.GetGeneration().GetExpression(),
		})
	}
	return result
}

func TestSyncListsObjectAndXMLTypeTables(t *testing.T) {
	t.Parallel()
	ctx := syncTestContext(t)
	driver, admin, s := newSyncTarget(ctx, t)
	execAll(ctx, t, admin,
		fmt.Sprintf(`CREATE TYPE %s.TY AS OBJECT (X NUMBER, Y VARCHAR2(10))`, s),
		fmt.Sprintf(`CREATE TABLE %[1]s.OT OF %[1]s.TY (CONSTRAINT OT_PK PRIMARY KEY (X))`, s),
		fmt.Sprintf(`CREATE INDEX %[1]s.OT_Y ON %[1]s.OT (Y)`, s),
		fmt.Sprintf(`CREATE TABLE %s.XT OF XMLTYPE`, s),
		fmt.Sprintf(`CREATE TABLE %s.REL (A NUMBER)`, s),
	)

	synced := syncedSchema(ctx, t, driver)
	require.ElementsMatch(t, []string{"OT", "REL", "XT"}, names(synced.Tables))
	require.False(t, tableByName(t, synced, "REL").SkipDump)

	objectTable := tableByName(t, synced, "OT")
	require.True(t, objectTable.SkipDump)
	require.Equal(t, []string{"X", "Y"}, names(objectTable.Columns))
	// The rest is the SYS_C unique constraint on the hidden object identifier.
	require.Subset(t, names(objectTable.Indexes), []string{"OT_PK", "OT_Y"})

	xmlTable := tableByName(t, synced, "XT")
	require.True(t, xmlTable.SkipDump)
	require.Equal(t, []string{"SYS_NC_ROWINFO$"}, names(xmlTable.Columns))
}

func TestSyncListsInvisibleColumns(t *testing.T) {
	t.Parallel()
	ctx := syncTestContext(t)
	driver, admin, s := newSyncTarget(ctx, t)
	execAll(ctx, t, admin,
		fmt.Sprintf(`CREATE TABLE %s.T (
			A NUMBER,
			H NUMBER INVISIBLE,
			B VARCHAR2(10),
			HV NUMBER INVISIBLE DEFAULT 7 NOT NULL,
			VC AS (A+1) VIRTUAL,
			IHV INVISIBLE AS (A*2) VIRTUAL,
			U NUMBER
		)`, s),
		// Each of these adds a hidden system column that must stay out.
		fmt.Sprintf(`CREATE INDEX %[1]s.T_FBI ON %[1]s.T (UPPER(B))`, s),
		fmt.Sprintf(`ALTER TABLE %s.T SET UNUSED COLUMN U`, s),
		fmt.Sprintf(`BEGIN DBMS_OUTPUT.PUT_LINE(DBMS_STATS.CREATE_EXTENDED_STATS('%s', 'T', '(A, B)')); END;`, s),
		fmt.Sprintf(`CREATE VIEW %[1]s.V (A, B INVISIBLE) AS SELECT A, B FROM %[1]s.T`, s),
	)
	want := []syncedColumn{
		{Name: "A", Position: 1, Nullable: true},
		{Name: "B", Position: 2, Nullable: true},
		{Name: "VC", Position: 3, Nullable: true, Expression: `"A"+1`},
		{Name: "H", Position: 4, Invisible: true, Nullable: true},
		{Name: "HV", Position: 5, Invisible: true, Default: "7"},
		{Name: "IHV", Position: 6, Invisible: true, Nullable: true, Expression: `"A"*2`},
	}

	synced := syncedSchema(ctx, t, driver)
	require.Equal(t, want, syncedColumns(tableByName(t, synced, "T").Columns))
	require.Len(t, synced.Views, 1)
	require.Equal(t, []syncedColumn{
		{Name: "A", Position: 1, Nullable: true},
		{Name: "B", Position: 2, Invisible: true, Nullable: true},
	}, syncedColumns(synced.Views[0].Columns))

	// Before 12.2 the columns come from ALL_TAB_COLUMNS and the virtual ones
	// from a query of their own.
	txn, err := driver.db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer txn.Rollback()
	columnMap, err := getTableColumns(txn, s, &plsql.Version{First: 12, Second: 1})
	require.NoError(t, err)
	require.Equal(t, want, syncedColumns(columnMap[db.TableKey{Schema: s, Table: "T"}]))
}

func TestSyncAttachesMaterializedViewIndexesAndTriggers(t *testing.T) {
	t.Parallel()
	ctx := syncTestContext(t)
	driver, admin, s := newSyncTarget(ctx, t)
	execAll(ctx, t, admin,
		fmt.Sprintf(`CREATE TABLE %s.BASE (ID NUMBER CONSTRAINT BASE_PK PRIMARY KEY, V NUMBER)`, s),
		fmt.Sprintf(`CREATE MATERIALIZED VIEW LOG ON %s.BASE WITH PRIMARY KEY, ROWID`, s),
		// A fast-refreshed ROWID materialized view gets the index I_SNAP$_MV.
		fmt.Sprintf(`CREATE MATERIALIZED VIEW %[1]s.MV REFRESH FAST WITH ROWID AS SELECT ID, V FROM %[1]s.BASE`, s),
		fmt.Sprintf(`CREATE INDEX %[1]s.MV_V ON %[1]s.MV (V)`, s),
		fmt.Sprintf(`CREATE TRIGGER %[1]s.MV_TRG BEFORE INSERT ON %[1]s.MV FOR EACH ROW BEGIN NULL; END;`, s),
	)

	synced := syncedSchema(ctx, t, driver)
	require.NotContains(t, names(synced.Tables), "MV")
	require.Len(t, synced.MaterializedViews, 1)
	materializedView := synced.MaterializedViews[0]
	require.ElementsMatch(t, []string{"I_SNAP$_MV", "MV_V"}, names(materializedView.Indexes))
	require.Equal(t, []string{"MV_TRG"}, names(materializedView.Triggers))
}

func TestSyncLeavesOutTriggersOnOtherSchemasTables(t *testing.T) {
	t.Parallel()
	ctx := syncTestContext(t)
	driver, admin, s := newSyncTarget(ctx, t)
	_, other := testcontainer.NewOracleSchema(t)
	execAll(ctx, t, admin,
		fmt.Sprintf(`CREATE TABLE %s.T (A NUMBER)`, s),
		fmt.Sprintf(`CREATE TABLE %s.T (A NUMBER)`, other),
		fmt.Sprintf(`CREATE TRIGGER %[1]s.ON_OWN BEFORE INSERT ON %[1]s.T FOR EACH ROW BEGIN NULL; END;`, s),
		fmt.Sprintf(`CREATE TRIGGER %s.ON_OTHER BEFORE INSERT ON %s.T FOR EACH ROW BEGIN NULL; END;`, s, other),
	)

	synced := syncedSchema(ctx, t, driver)
	require.Equal(t, []string{"ON_OWN"}, names(tableByName(t, synced, "T").Triggers))
}

// TestSyncDumpRoundTrip applies the dump of a synced schema to an empty one and
// expects the second sync to match the first.
func TestSyncDumpRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := syncTestContext(t)
	source, admin, s := newSyncTarget(ctx, t)
	execAll(ctx, t, admin,
		fmt.Sprintf(`CREATE TABLE %s.PARENT (ID NUMBER CONSTRAINT PARENT_PK PRIMARY KEY, NAME VARCHAR2(20) DEFAULT 'x' NOT NULL)`, s),
		fmt.Sprintf(`CREATE TABLE %[1]s.CHILD (
			ID NUMBER,
			PARENT_ID NUMBER CONSTRAINT CHILD_FK REFERENCES %[1]s.PARENT (ID),
			H NUMBER INVISIBLE,
			HV VARCHAR2(5) INVISIBLE DEFAULT 'y' NOT NULL,
			VC AS (ID*10) VIRTUAL,
			IHV INVISIBLE AS (ID+PARENT_ID) VIRTUAL
		)`, s),
		fmt.Sprintf(`CREATE INDEX %[1]s.CHILD_VC ON %[1]s.CHILD (VC)`, s),
		fmt.Sprintf(`CREATE TYPE %s.TY AS OBJECT (X NUMBER)`, s),
		fmt.Sprintf(`CREATE TABLE %[1]s.OT OF %[1]s.TY`, s),
	)
	want := syncedSchema(ctx, t, source)

	dump, err := schema.GetDatabaseDefinition(storepb.Engine_ORACLE, schema.GetDefinitionContext{}, &metadatapb.DatabaseSchemaMetadata{Name: s, Schemas: []*metadatapb.SchemaMetadata{want}})
	require.NoError(t, err)
	require.NotContains(t, dump, `"OT"`)

	target, _, targetSchema := newSyncTarget(ctx, t)
	_, err = target.Execute(ctx, fmt.Sprintf("ALTER SESSION SET CURRENT_SCHEMA = %s;\n%s", targetSchema, dump), db.ExecuteOptions{})
	require.NoError(t, err, dump)
	got := syncedSchema(ctx, t, target)

	require.Equal(t, []string{"CHILD", "PARENT"}, names(got.Tables))
	for _, name := range []string{"CHILD", "PARENT"} {
		wantTable, gotTable := tableByName(t, want, name), tableByName(t, got, name)
		require.Equal(t, syncedColumns(wantTable.Columns), syncedColumns(gotTable.Columns), name)
		require.ElementsMatch(t, names(wantTable.Indexes), names(gotTable.Indexes), name)
		require.ElementsMatch(t, names(wantTable.ForeignKeys), names(gotTable.ForeignKeys), name)
	}
}
