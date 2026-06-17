package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	"github.com/bytebase/bytebase/backend/store/model"
)

func TestExtractChangedResources(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)
	statement :=
		`CREATE TABLE t1 (c1 INT);
						DROP TABLE t1;
						ALTER TABLE t1 ADD COLUMN c1 INT;
						ALTER TABLE t1 RENAME TO t2;
						COMMENT ON TABLE t1 IS 'comment';
						INSERT INTO t1 (c1) VALUES (1), (5);
						UPDATE t1 SET c1 = 5;
			`
	changedResources := model.NewChangedResources(dbMetadata)
	changedResources.AddTable(
		"db",
		"public",
		&storepb.ChangedResourceTable{
			Name: "t1",
		},
		true,
	)
	changedResources.AddTable(
		"db",
		"public",
		&storepb.ChangedResourceTable{
			Name: "t2",
		},
		false,
	)
	changedResources.AddTable(
		"db",
		"public",
		&storepb.ChangedResourceTable{
			Name: "t1",
		},
		false,
	)
	want := &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLCount:         1,
		SampleDMLS:       []string{"UPDATE t1 SET c1 = 5;"},
		InsertCount:      2,
	}

	stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("db", "public", dbMetadata, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestExtractChangedResourcesTruncate(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)
	const statement = `TRUNCATE TABLE public.t1, myschema.t2;`

	want := model.NewChangedResources(dbMetadata)
	want.AddTable("db", "public", &storepb.ChangedResourceTable{Name: "t1"}, true)
	want.AddTable("db", "myschema", &storepb.ChangedResourceTable{Name: "t2"}, true)

	stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("db", "public", dbMetadata, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got.ChangedResources)
}

func TestExtractChangedResourcesMerge(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)
	const statement = `MERGE INTO myschema.tgt t USING src s ON t.id = s.id WHEN MATCHED THEN UPDATE SET a = s.a;`

	want := model.NewChangedResources(dbMetadata)
	want.AddTable("db", "myschema", &storepb.ChangedResourceTable{Name: "tgt"}, false)

	stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, statement)
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	got, err := extractChangedResources("db", "public", dbMetadata, asts, statement)
	require.NoError(t, err)
	require.Equal(t, want, got.ChangedResources)
}

func TestExtractChangedResourcesCTAS(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_POSTGRES, true /* caseSensitive */)
	for _, tc := range []struct {
		statement, schema, table string
	}{
		{`CREATE TABLE myschema.t1 AS SELECT * FROM src;`, "myschema", "t1"},
		{`SELECT * INTO myschema.t2 FROM src;`, "myschema", "t2"},
		// INTO sits on the first arm of a set operation, not the root.
		{`SELECT a INTO myschema.t3 FROM src UNION SELECT a FROM src2;`, "myschema", "t3"},
	} {
		want := model.NewChangedResources(dbMetadata)
		want.AddTable("db", tc.schema, &storepb.ChangedResourceTable{Name: tc.table}, false)

		stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, tc.statement)
		require.NoError(t, err, tc.statement)
		asts := base.ExtractASTs(stmts)
		got, err := extractChangedResources("db", "public", dbMetadata, asts, tc.statement)
		require.NoError(t, err, tc.statement)
		require.Equal(t, want, got.ChangedResources, tc.statement)
	}
}
