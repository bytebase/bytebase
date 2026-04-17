package pg

import (
	"context"
	"os"
	"testing"

	"github.com/bytebase/omni/pg/catalog"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestClone_BasicIsolation verifies that Clone produces an independent catalog:
// mutations on the clone do not affect the original.
func TestClone_BasicIsolation(t *testing.T) {
	original := catalog.New()
	_, err := original.Exec(`
		CREATE SCHEMA test_schema;
		CREATE TABLE test_schema.t1 (id serial PRIMARY KEY, name text NOT NULL);
		CREATE INDEX t1_name_idx ON test_schema.t1 (name);
	`, nil)
	require.NoError(t, err)

	clone := original.Clone()

	// Mutate clone
	_, err = clone.Exec(`
		ALTER TABLE test_schema.t1 ADD COLUMN email text;
		CREATE TABLE test_schema.t2 (id int PRIMARY KEY);
		DROP INDEX test_schema.t1_name_idx;
	`, nil)
	require.NoError(t, err)

	// Verify original is untouched
	origRel := original.GetRelation("test_schema", "t1")
	require.NotNil(t, origRel, "original t1 should still exist")
	require.Equal(t, 2, len(origRel.Columns), "original t1 should have 2 columns (id, name), not 3")

	require.Nil(t, original.GetRelation("test_schema", "t2"), "original should NOT have t2")

	origIndexes := original.IndexesOf(origRel.OID)
	found := false
	for _, idx := range origIndexes {
		if idx.Name == "t1_name_idx" {
			found = true
		}
	}
	require.True(t, found, "original should still have t1_name_idx")

	// Verify clone has the changes
	cloneRel := clone.GetRelation("test_schema", "t1")
	require.NotNil(t, cloneRel)
	require.Equal(t, 3, len(cloneRel.Columns), "clone t1 should have 3 columns")
	require.NotNil(t, clone.GetRelation("test_schema", "t2"), "clone should have t2")

	// Diff should show changes
	diff := catalog.Diff(original, clone)
	require.False(t, diff.IsEmpty(), "diff should not be empty")
	require.Greater(t, len(diff.Relations), 0, "should have relation diffs")
}

// TestClone_WalkThroughIntegration tests the full walk-through flow using Clone:
// load catalog → clone → exec user DDL on clone → diff → apply.
func TestClone_WalkThroughIntegration(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer", Position: 1},
							{Name: "name", Type: "text", Position: 2, Nullable: true},
						},
						Indexes: []*storepb.IndexMetadata{
							{Name: "users_pkey", Expressions: []string{"id"}, Unique: true, Primary: true},
						},
					},
				},
			},
		},
	}

	catBefore := catalog.New()
	err := loadWalkThroughCatalog(context.Background(), catBefore, meta)
	require.NoError(t, err)

	catAfter := catBefore.Clone()

	userSQL := `
		ALTER TABLE public.users ADD COLUMN email text NOT NULL DEFAULT 'unknown';
		CREATE INDEX users_email_idx ON public.users (email);
		CREATE TABLE public.posts (id serial PRIMARY KEY, user_id int REFERENCES public.users(id), title text);
	`
	results, err := catAfter.Exec(userSQL, &catalog.ExecOptions{ContinueOnError: true})
	require.NoError(t, err)
	for _, r := range results {
		require.NoError(t, r.Error, "DDL should succeed: %s", r.SQL)
	}

	// Diff
	diff := catalog.Diff(catBefore, catAfter)
	require.False(t, diff.IsEmpty())

	// Apply diff to original metadata
	newProto := applyDiffToMetadata(meta, catBefore, catAfter, diff)
	newMeta := model.NewDatabaseMetadata(newProto, nil, nil, storepb.Engine_POSTGRES, true)

	// Verify: users table should have 3 columns
	usersSchema := newMeta.GetSchemaMetadata("public")
	require.NotNil(t, usersSchema)
	usersTbl := usersSchema.GetTable("users")
	require.NotNil(t, usersTbl)
	require.Equal(t, 3, len(usersTbl.GetProto().Columns), "users should have id, name, email")

	// Verify: users should have 2 indexes (pkey + email)
	require.GreaterOrEqual(t, len(usersTbl.GetProto().Indexes), 2)

	// Verify: posts table should exist
	postsTbl := usersSchema.GetTable("posts")
	require.NotNil(t, postsTbl, "posts table should exist")

	// Verify: original metadata should NOT have posts or email column
	origMeta := model.NewDatabaseMetadata(meta, nil, nil, storepb.Engine_POSTGRES, true)
	origUsers := origMeta.GetSchemaMetadata("public").GetTable("users")
	require.Equal(t, 2, len(origUsers.GetProto().Columns), "original should still have 2 columns")
	require.Nil(t, origMeta.GetSchemaMetadata("public").GetTable("posts"), "original should not have posts")
}

// TestClone_SearchPath tests Clone preserves search path and session user.
func TestClone_SearchPath(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
			{Name: "alice"},
		},
	}

	catBefore := catalog.New()
	catBefore.SetSessionUser("alice")
	catBefore.SetSearchPath([]string{"$user", "public"})
	err := loadWalkThroughCatalog(context.Background(), catBefore, meta)
	require.NoError(t, err)

	catAfter := catBefore.Clone()

	// CREATE TABLE without schema should use search path ($user → alice)
	_, err = catAfter.Exec(`CREATE TABLE my_table (id int);`, nil)
	require.NoError(t, err)

	// Table should be in alice schema on clone
	require.NotNil(t, catAfter.GetRelation("alice", "my_table"))

	// Original should NOT have it
	require.Nil(t, catBefore.GetRelation("alice", "my_table"))

	// Diff should show the new table
	diff := catalog.Diff(catBefore, catAfter)
	require.False(t, diff.IsEmpty())

	foundAliceTable := false
	for _, rel := range diff.Relations {
		if rel.SchemaName == "alice" && rel.Name == "my_table" && rel.Action == catalog.DiffAdd {
			foundAliceTable = true
		}
	}
	require.True(t, foundAliceTable, "diff should show alice.my_table as added")
}

// TestClone_BbExportSample tests Clone with a real bb_export metadata file,
// verifying the full walk-through flow produces correct diffs.
func TestClone_BbExportSample(t *testing.T) {
	root := bbExportRoot
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Skipf("bb_export not found at %s; skipping", root)
	}
	jsonFiles := collectPGJsonFiles(t, root)
	if len(jsonFiles) == 0 {
		t.Skip("no files")
	}

	// Pick first 10 files
	if len(jsonFiles) > 10 {
		jsonFiles = jsonFiles[:10]
	}

	userDDLs := []string{
		`CREATE TABLE "__clone_test" (id serial PRIMARY KEY, name text)`,
		`CREATE INDEX "__clone_idx" ON "__clone_test" (name)`,
		`ALTER TABLE "__clone_test" ADD COLUMN created_at timestamptz DEFAULT now()`,
		`DROP TABLE "__clone_test" CASCADE`,
	}

	for _, jf := range jsonFiles {
		data, _ := os.ReadFile(jf)
		meta := &storepb.DatabaseSchemaMetadata{}
		if common.ProtojsonUnmarshaler.Unmarshal(data, meta) != nil {
			continue
		}

		catBefore := catalog.New()
		if loadWalkThroughCatalog(context.Background(), catBefore, meta) != nil {
			continue
		}

		catAfter := catBefore.Clone()
		for _, ddl := range userDDLs {
			catAfter.Exec(ddl, &catalog.ExecOptions{ContinueOnError: true})
		}

		diff := catalog.Diff(catBefore, catAfter)

		// After create+drop cycle, diff should be empty or only contain
		// the sequence created by serial (which DROP CASCADE removes).
		// Key check: no spurious diffs from pseudo objects.
		for _, rel := range diff.Relations {
			if rel.Name != "__clone_test" {
				t.Errorf("[%s] unexpected relation diff: %s.%s action=%d",
					shortPath(jf), rel.SchemaName, rel.Name, rel.Action)
			}
		}
	}
	t.Logf("Clone test passed on %d files", len(jsonFiles))
}

// TestClone_WalkThroughOmniFunction tests the actual WalkThroughOmni function
// using Clone (if enabled) produces correct FinalMetadata.
func TestClone_WalkThroughOmniFunction(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "test",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer", Position: 1},
							{Name: "name", Type: "text", Position: 2, Nullable: true},
						},
						Indexes: []*storepb.IndexMetadata{
							{Name: "test_pkey", Expressions: []string{"id"}, Unique: true, Primary: true},
						},
					},
				},
			},
		},
	}

	state := model.NewDatabaseMetadata(meta, nil, nil, storepb.Engine_POSTGRES, true)
	ctx := schema.WalkThroughContext{
		RawSQL: `CREATE TABLE public.new_table (id int PRIMARY KEY, val text);`,
	}

	advice := WalkThroughOmni(ctx, state, nil)
	require.Nil(t, advice, "walk-through should succeed")

	// Check FinalMetadata has the new table
	newTbl := state.GetSchemaMetadata("public").GetTable("new_table")
	require.NotNil(t, newTbl, "new_table should exist in FinalMetadata")

	// Check original table is preserved
	origTbl := state.GetSchemaMetadata("public").GetTable("test")
	require.NotNil(t, origTbl, "test table should still exist")
	require.Equal(t, 2, len(origTbl.GetProto().Columns))
}
