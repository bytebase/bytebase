package pg

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/catalog"
	"github.com/stretchr/testify/require"

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
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer", Position: 1},
							{Name: "name", Type: "text", Position: 2, Nullable: true},
						},
						Indexes: []*metadatapb.IndexMetadata{
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
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*metadatapb.SchemaMetadata{
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

// TestClone_WalkThroughFunction tests the actual WalkThroughWithContext function
// using Clone (if enabled) produces correct FinalMetadata.
func TestClone_WalkThroughFunction(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "test",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer", Position: 1},
							{Name: "name", Type: "text", Position: 2, Nullable: true},
						},
						Indexes: []*metadatapb.IndexMetadata{
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

	advice := WalkThroughWithContext(ctx, state, nil)
	require.Nil(t, advice, "walk-through should succeed")

	// Check FinalMetadata has the new table
	newTbl := state.GetSchemaMetadata("public").GetTable("new_table")
	require.NotNil(t, newTbl, "new_table should exist in FinalMetadata")

	// Check original table is preserved
	origTbl := state.GetSchemaMetadata("public").GetTable("test")
	require.NotNil(t, origTbl, "test table should still exist")
	require.Equal(t, 2, len(origTbl.GetProto().Columns))
}

// compositeWalkThroughMetadata has an adversarial name pair (aa_nested sorts
// before its dependency zz_base) plus an enum-referencing composite and a
// table using one, so a successful load proves dependency-ordered install.
func compositeWalkThroughMetadata() *metadatapb.DatabaseSchemaMetadata {
	return &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*metadatapb.EnumTypeMetadata{
					{Name: "status", Values: []string{"a", "b"}},
				},
				CompositeTypes: []*metadatapb.CompositeTypeMetadata{
					{
						// References zz_base only through an array suffix.
						Name: "aa_array_only",
						Attributes: []*metadatapb.CompositeTypeAttribute{
							{Name: "items", Type: "public.zz_base[]"},
						},
					},
					{
						Name: "aa_nested",
						Attributes: []*metadatapb.CompositeTypeAttribute{
							{Name: "home", Type: "public.zz_base"},
							{Name: "s", Type: "public.status"},
						},
					},
					{
						Name:    "zz_base",
						Comment: "base address type",
						Attributes: []*metadatapb.CompositeTypeAttribute{
							{Name: "street", Type: "text", Collation: `"C"`, Comment: "street line"},
							{Name: "city", Type: "character varying(50)"},
						},
					},
					{
						Name:     "ext_owned",
						SkipDump: true,
						Attributes: []*metadatapb.CompositeTypeAttribute{
							{Name: "x", Type: "integer"},
						},
					},
				},
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer", Position: 1},
							{Name: "home", Type: "public.zz_base", Position: 2, Nullable: true},
						},
					},
				},
			},
		},
	}
}

func TestWalkThroughLoadsCompositeTypes(t *testing.T) {
	meta := compositeWalkThroughMetadata()

	cat := catalog.New()
	err := loadWalkThroughCatalog(context.Background(), cat, meta)
	require.NoError(t, err)

	require.NotNil(t, cat.GetRelation("public", "zz_base"), "composite type must be installed")
	require.NotNil(t, cat.GetRelation("public", "aa_nested"), "nested composite type must be installed")
	arrayOnly := cat.GetRelation("public", "aa_array_only")
	require.NotNil(t, arrayOnly, "array-only referencing composite must be installed")
	require.Len(t, arrayOnly.Columns, 1, "array-only composite must install with its real attribute")
	require.NotNil(t, cat.GetRelation("public", "users"), "table using a composite column must install")
}

func TestWalkThroughCompositeFallbackPreservesAttributeNames(t *testing.T) {
	// A domain-typed attribute cannot install (domains are not loader
	// objects), forcing the pseudo fallback — which must keep attribute
	// names so later DDL targeting them still resolves.
	meta := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				CompositeTypes: []*metadatapb.CompositeTypeMetadata{
					{
						Name: "with_domain",
						Attributes: []*metadatapb.CompositeTypeAttribute{
							{Name: "p", Type: "public.pos_int"},
							{Name: "note", Type: "text"},
						},
					},
				},
			},
		},
	}

	cat := catalog.New()
	err := loadWalkThroughCatalog(context.Background(), cat, meta)
	require.NoError(t, err)

	rel := cat.GetRelation("public", "with_domain")
	require.NotNil(t, rel, "composite must fall back, not vanish")
	require.Len(t, rel.Columns, 2, "fallback must preserve attribute names")
	require.Equal(t, "p", rel.Columns[0].Name)
	require.Equal(t, "note", rel.Columns[1].Name)

	// DDL targeting a preserved attribute must succeed against the fallback.
	catAfter := cat.Clone()
	results, err := catAfter.Exec(`ALTER TYPE public.with_domain DROP ATTRIBUTE note;`, &catalog.ExecOptions{ContinueOnError: true})
	require.NoError(t, err)
	for _, r := range results {
		require.NoError(t, r.Error, "DDL against fallback composite should succeed: %s", r.SQL)
	}

	// A rename keeps the attribute number, so the renamed attribute of a
	// degraded composite must also keep its real previous type.
	renameResults, err := catAfter.Exec(`ALTER TYPE public.with_domain RENAME ATTRIBUTE p TO q;`, &catalog.ExecOptions{ContinueOnError: true})
	require.NoError(t, err)
	for _, r := range renameResults {
		require.NoError(t, r.Error, "rename against fallback composite should succeed: %s", r.SQL)
	}

	// Applying the diff must not rewrite the untouched attribute to the
	// fallback's text type — unchanged attributes keep previous metadata.
	diff := catalog.Diff(cat, catAfter)
	require.False(t, diff.IsEmpty())
	newProto := applyDiffToMetadata(meta, cat, catAfter, diff)
	var applied *metadatapb.CompositeTypeMetadata
	for _, composite := range newProto.Schemas[0].CompositeTypes {
		if composite.Name == "with_domain" {
			applied = composite
		}
	}
	require.NotNil(t, applied)
	require.Len(t, applied.Attributes, 1, "dropped attribute must be removed")
	require.Equal(t, "q", applied.Attributes[0].Name, "rename must be reflected")
	require.Equal(t, "public.pos_int", applied.Attributes[0].Type,
		"renamed attribute must keep its real previous type, not the fallback text")
}

func TestWalkThroughDropReaddAttributeReadsCatalogType(t *testing.T) {
	// Dropping and re-adding an attribute with the same name assigns a new
	// attnum; the rebuilt metadata must take the catalog's new type, not
	// carry the stale previous metadata.
	meta := &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				CompositeTypes: []*metadatapb.CompositeTypeMetadata{
					{
						Name: "with_domain",
						Attributes: []*metadatapb.CompositeTypeAttribute{
							{Name: "p", Type: "public.pos_int", Comment: "old comment"},
							{Name: "note", Type: "text"},
						},
					},
				},
			},
		},
	}

	cat := catalog.New()
	err := loadWalkThroughCatalog(context.Background(), cat, meta)
	require.NoError(t, err)

	catAfter := cat.Clone()
	results, err := catAfter.Exec(`
		ALTER TYPE public.with_domain DROP ATTRIBUTE p;
		ALTER TYPE public.with_domain ADD ATTRIBUTE p text;
	`, &catalog.ExecOptions{ContinueOnError: true})
	require.NoError(t, err)
	for _, r := range results {
		require.NoError(t, r.Error, "DDL should succeed: %s", r.SQL)
	}

	diff := catalog.Diff(cat, catAfter)
	require.False(t, diff.IsEmpty())
	newProto := applyDiffToMetadata(meta, cat, catAfter, diff)
	var applied *metadatapb.CompositeTypeMetadata
	for _, composite := range newProto.Schemas[0].CompositeTypes {
		if composite.Name == "with_domain" {
			applied = composite
		}
	}
	require.NotNil(t, applied)
	require.Len(t, applied.Attributes, 2)
	var p *metadatapb.CompositeTypeAttribute
	for _, attribute := range applied.Attributes {
		if attribute.Name == "p" {
			p = attribute
		}
	}
	require.NotNil(t, p)
	require.Equal(t, "text", p.Type, "re-added attribute must take the catalog's new type: %v", applied)
	require.Empty(t, p.Comment, "re-added attribute must not inherit the dropped attribute's comment")
}

func TestWalkThroughAppliesCompositeTypeChanges(t *testing.T) {
	meta := compositeWalkThroughMetadata()

	catBefore := catalog.New()
	err := loadWalkThroughCatalog(context.Background(), catBefore, meta)
	require.NoError(t, err)

	catAfter := catBefore.Clone()
	userSQL := `
		CREATE TYPE public.geo AS (lat numeric(9,6), lng numeric(9,6));
		CREATE TYPE public.geo_wrap AS (g geo, gs geo[]);
		CREATE SCHEMA "select";
		CREATE TYPE "select".kw AS (x int);
		CREATE TYPE public.kw_wrap AS (k "select".kw);
		CREATE TABLE public.places (id int, location public.geo);
		DROP TYPE public.aa_nested;
		ALTER TYPE public.zz_base ADD ATTRIBUTE zip text;
		ALTER TYPE public.ext_owned ADD ATTRIBUTE y integer;
	`
	results, err := catAfter.Exec(userSQL, &catalog.ExecOptions{ContinueOnError: true})
	require.NoError(t, err)
	for _, r := range results {
		require.NoError(t, r.Error, "DDL should succeed: %s", r.SQL)
	}

	diff := catalog.Diff(catBefore, catAfter)
	require.False(t, diff.IsEmpty())

	newProto := applyDiffToMetadata(meta, catBefore, catAfter, diff)
	newMeta := model.NewDatabaseMetadata(newProto, nil, nil, storepb.Engine_POSTGRES, true)

	publicSchema := newMeta.GetSchemaMetadata("public")
	require.NotNil(t, publicSchema)

	composites := make(map[string]*metadatapb.CompositeTypeMetadata)
	for _, composite := range publicSchema.GetProto().CompositeTypes {
		composites[composite.Name] = composite
	}
	require.Contains(t, composites, "geo", "created composite must be applied to metadata")
	require.Len(t, composites["geo"].Attributes, 2)
	require.Equal(t, "lat", composites["geo"].Attributes[0].Name)
	require.Equal(t, "numeric(9,6)", composites["geo"].Attributes[0].Type,
		"built-in attribute types must stay unqualified")

	// User-defined attribute types must come back schema-qualified even when
	// the DDL referenced them unqualified via the search path.
	require.Contains(t, composites, "geo_wrap")
	require.Equal(t, "public.geo", composites["geo_wrap"].Attributes[0].Type)
	require.Equal(t, "public.geo[]", composites["geo_wrap"].Attributes[1].Type)

	// A schema named like a reserved keyword must come back quoted.
	require.Contains(t, composites, "kw_wrap")
	require.Equal(t, `"select".kw`, composites["kw_wrap"].Attributes[0].Type)
	require.NotContains(t, composites, "aa_nested", "dropped composite must be removed from metadata")
	require.Contains(t, composites, "zz_base", "altered composite must remain")

	// The modified composite gains the new attribute while preserving the
	// type comment and surviving attributes' comments/collations.
	base := composites["zz_base"]
	require.Equal(t, "base address type", base.Comment, "type comment must survive modification")
	require.Len(t, base.Attributes, 3)
	require.Equal(t, "street", base.Attributes[0].Name)
	require.Equal(t, `"C"`, base.Attributes[0].Collation, "attribute collation must survive modification")
	require.Equal(t, "street line", base.Attributes[0].Comment, "attribute comment must survive modification")
	require.Equal(t, "zip", base.Attributes[2].Name, "added attribute must appear")

	// skip_dump survives modification (extension-owned types must stay
	// excluded from dumps).
	require.Contains(t, composites, "ext_owned")
	require.True(t, composites["ext_owned"].SkipDump, "skip_dump must survive modification")

	// Original metadata is not mutated.
	require.Len(t, meta.Schemas[0].CompositeTypes, 4)
}
