package mysql

import (
	"context"
	"strings"
	"testing"

	"github.com/bytebase/omni/mysql/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// These tests exercise loadWalkThroughCatalog against synthetic metadata.
// Nothing here touches bb_export or any external fixture — every input is a
// storepb.*Metadata constructed in memory.

const testDBName = "testdb"

// newLoaderTestCatalog returns a catalog with the test database created and
// selected, matching the setup WalkThroughOmni performs before the loader runs.
func newLoaderTestCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	c := catalog.New()
	if _, err := c.Exec(
		"SET foreign_key_checks = 0;\nCREATE DATABASE IF NOT EXISTS `"+testDBName+"`;\nUSE `"+testDBName+"`;",
		&catalog.ExecOptions{ContinueOnError: true},
	); err != nil {
		t.Fatalf("init catalog: %v", err)
	}
	return c
}

func runLoader(t *testing.T, c *catalog.Catalog, meta *storepb.DatabaseSchemaMetadata) {
	t.Helper()
	if err := loadWalkThroughCatalog(context.Background(), c, testDBName, meta); err != nil {
		t.Fatalf("loadWalkThroughCatalog: %v", err)
	}
}

func mustGetTable(t *testing.T, c *catalog.Catalog, name string) *catalog.Table {
	t.Helper()
	db := c.GetDatabase(testDBName)
	if db == nil {
		t.Fatalf("database %q not in catalog", testDBName)
	}
	tbl := db.Tables[strings.ToLower(name)]
	if tbl == nil {
		t.Fatalf("table %q not in catalog", name)
	}
	return tbl
}

func mustGetView(t *testing.T, c *catalog.Catalog, name string) *catalog.View {
	t.Helper()
	db := c.GetDatabase(testDBName)
	if db == nil {
		t.Fatalf("database %q not in catalog", testDBName)
	}
	v := db.Views[strings.ToLower(name)]
	if v == nil {
		t.Fatalf("view %q not in catalog", name)
	}
	return v
}

// ----------------------------------------------------------------------
// wtParseTypeName — covers a representative cross-section of MySQL types.
// ----------------------------------------------------------------------

func TestWtParseTypeName(t *testing.T) {
	cases := []struct {
		in       string
		wantName string
	}{
		{"int", "INT"},
		{"int unsigned", "INT"},
		{"bigint(20) unsigned zerofill", "BIGINT"},
		{"varchar(255)", "VARCHAR"},
		{"char(1)", "CHAR"},
		{"text", "TEXT"},
		{"decimal(10,2)", "DECIMAL"},
		{"datetime(6)", "DATETIME"},
		{"timestamp", "TIMESTAMP"},
		{"tinyint(1)", "TINYINT"},
		{"blob", "BLOB"},
		{"json", "JSON"},
		{"enum('a','b','c')", "ENUM"},
		{"set('x','y')", "SET"},
		{"geometry", "GEOMETRY"},
		{"bit(1)", "BIT"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			dt, err := wtParseTypeName(tc.in)
			if err != nil {
				t.Fatalf("wtParseTypeName(%q): %v", tc.in, err)
			}
			if !strings.EqualFold(dt.Name, tc.wantName) {
				t.Errorf("wtParseTypeName(%q).Name = %q, want %q", tc.in, dt.Name, tc.wantName)
			}
		})
	}
}

func TestWtParseTypeName_Failures(t *testing.T) {
	for _, in := range []string{"", "not a real type (((", "int UNSIGNED ZEROFILL NOT_A_THING"} {
		if _, err := wtParseTypeName(in); err == nil {
			t.Errorf("wtParseTypeName(%q): expected error, got nil", in)
		}
	}
}

// ----------------------------------------------------------------------
// wtParseExpr — default/ON UPDATE/generated/check expressions.
// ----------------------------------------------------------------------

func TestWtParseExpr(t *testing.T) {
	for _, in := range []string{
		`'hello'`,
		`CURRENT_TIMESTAMP`,
		`NULL`,
		`0`,
		`a + b`,
		`CONCAT('x', 'y')`,
		`DATE_FORMAT(NOW(), '%Y-%m-%d')`,
	} {
		if _, err := wtParseExpr(in); err != nil {
			t.Errorf("wtParseExpr(%q) unexpectedly failed: %v", in, err)
		}
	}
}

func TestWtParseExpr_Failures(t *testing.T) {
	// "SELECT 1" is intentionally NOT in this list: wrapped in our SELECT
	// probe it parses as a scalar subquery expression, which is valid.
	for _, in := range []string{"", "(((", "this is not sql at all ###"} {
		if _, err := wtParseExpr(in); err == nil {
			t.Errorf("wtParseExpr(%q): expected error, got nil", in)
		}
	}
}

// ----------------------------------------------------------------------
// Real table install — exercises the full column + constraint + option path.
// ----------------------------------------------------------------------

func TestLoader_TableBasic(t *testing.T) {
	meta := schemaWithTables(&storepb.TableMetadata{
		Name:      "users",
		Engine:    "InnoDB",
		Charset:   "utf8mb4",
		Collation: "utf8mb4_0900_ai_ci",
		Comment:   "users table",
		Columns: []*storepb.ColumnMetadata{
			{
				Name:     "id",
				Type:     "bigint unsigned",
				Nullable: false,
				Default:  autoIncrementSentinel,
			},
			{
				Name:         "email",
				Type:         "varchar(255)",
				Nullable:     false,
				CharacterSet: "utf8mb4",
				Collation:    "utf8mb4_0900_ai_ci",
			},
			{
				Name:     "created_at",
				Type:     "timestamp",
				Nullable: false,
				Default:  "CURRENT_TIMESTAMP",
				OnUpdate: "CURRENT_TIMESTAMP",
			},
			{
				Name:     "bio",
				Type:     "text",
				Nullable: true,
				Comment:  "free form text",
			},
		},
		Indexes: []*storepb.IndexMetadata{
			{Name: "PRIMARY", Type: "BTREE", Primary: true, Unique: true, Expressions: []string{"id"}},
			{Name: "uniq_email", Type: "BTREE", Unique: true, Expressions: []string{"email"}},
		},
	})

	c := newLoaderTestCatalog(t)
	runLoader(t, c, meta)

	tbl := mustGetTable(t, c, "users")
	if got := len(tbl.Columns); got != 4 {
		t.Fatalf("Columns: want 4, got %d", got)
	}
	if got := tbl.Engine; got != "InnoDB" {
		t.Errorf("Engine: %q", got)
	}
	if got := tbl.Charset; got != "utf8mb4" {
		t.Errorf("Charset: %q", got)
	}
	if got := tbl.Comment; got != "users table" {
		t.Errorf("Comment: %q", got)
	}

	// Column properties.
	idCol := tbl.GetColumn("id")
	if idCol == nil {
		t.Fatal("id column missing")
	}
	if !idCol.AutoIncrement {
		t.Error("id should be AUTO_INCREMENT (sentinel detection)")
	}
	if idCol.Nullable {
		t.Error("id should be NOT NULL")
	}
	if strings.ToLower(idCol.DataType) != "bigint" {
		t.Errorf("id.DataType = %q", idCol.DataType)
	}

	emailCol := tbl.GetColumn("email")
	if emailCol == nil || emailCol.Charset != "utf8mb4" {
		t.Errorf("email.Charset: %q", emailCol.Charset)
	}

	createdCol := tbl.GetColumn("created_at")
	// Omni's catalog canonicalizes CURRENT_TIMESTAMP to now() when deparsing
	// the parsed expression; both forms are equivalent so assert presence of
	// a default and ON UPDATE rather than a specific string.
	if createdCol == nil || createdCol.Default == nil || *createdCol.Default == "" {
		t.Errorf("created_at default missing: %+v", createdCol)
	}
	if createdCol == nil || createdCol.OnUpdate == "" {
		t.Error("created_at OnUpdate missing")
	}

	bioCol := tbl.GetColumn("bio")
	if bioCol == nil || bioCol.Comment != "free form text" {
		t.Errorf("bio.Comment: %q", bioCol.Comment)
	}

	// Primary key must be present as a constraint.
	var hasPK, hasUnique bool
	for _, con := range tbl.Constraints {
		switch con.Type {
		case catalog.ConPrimaryKey:
			hasPK = true
		case catalog.ConUniqueKey:
			if con.Name == "uniq_email" {
				hasUnique = true
			}
		default:
		}
	}
	if !hasPK {
		t.Error("missing PRIMARY KEY constraint")
	}
	if !hasUnique {
		t.Error("missing uniq_email unique constraint")
	}
}

// ----------------------------------------------------------------------
// Fulltext / Spatial index fix — the regression we're covering in this PR.
// ----------------------------------------------------------------------

func TestLoader_FulltextAndSpatialIndex(t *testing.T) {
	meta := schemaWithTables(&storepb.TableMetadata{
		Name: "t",
		Columns: []*storepb.ColumnMetadata{
			{Name: "id", Type: "bigint unsigned", Nullable: false, Default: autoIncrementSentinel},
			{Name: "body", Type: "text", Nullable: true},
			{Name: "location", Type: "geometry", Nullable: true},
		},
		Indexes: []*storepb.IndexMetadata{
			{Name: "PRIMARY", Type: "BTREE", Primary: true, Unique: true, Expressions: []string{"id"}},
			{Name: "ft_body", Type: "FULLTEXT", Expressions: []string{"body"}},
			{Name: "sp_location", Type: "SPATIAL", Expressions: []string{"location"}},
		},
	})

	c := newLoaderTestCatalog(t)
	runLoader(t, c, meta)

	tbl := mustGetTable(t, c, "t")
	var gotFulltext, gotSpatial bool
	for _, idx := range tbl.Indexes {
		switch idx.Name {
		case "ft_body":
			gotFulltext = idx.Fulltext
			if !idx.Fulltext {
				t.Errorf("ft_body installed but not marked as Fulltext; IndexType=%q", idx.IndexType)
			}
		case "sp_location":
			gotSpatial = idx.Spatial
			if !idx.Spatial {
				t.Errorf("sp_location installed but not marked as Spatial; IndexType=%q", idx.IndexType)
			}
		default:
			// Other indexes (e.g. PRIMARY) are irrelevant here.
		}
	}
	if !gotFulltext {
		t.Error("fulltext index not found on table")
	}
	if !gotSpatial {
		t.Error("spatial index not found on table")
	}
}

// ----------------------------------------------------------------------
// FK install — foreign_key_checks off, the constraint is recorded unchecked.
// ----------------------------------------------------------------------

func TestLoader_ForeignKeyAcrossTables(t *testing.T) {
	meta := schemaWithTables(
		&storepb.TableMetadata{
			Name: "parent",
			Columns: []*storepb.ColumnMetadata{
				{Name: "id", Type: "bigint unsigned", Nullable: false, Default: autoIncrementSentinel},
			},
			Indexes: []*storepb.IndexMetadata{
				{Name: "PRIMARY", Type: "BTREE", Primary: true, Unique: true, Expressions: []string{"id"}},
			},
		},
		&storepb.TableMetadata{
			Name: "child",
			Columns: []*storepb.ColumnMetadata{
				{Name: "id", Type: "bigint unsigned", Nullable: false, Default: autoIncrementSentinel},
				{Name: "parent_id", Type: "bigint unsigned", Nullable: false},
			},
			Indexes: []*storepb.IndexMetadata{
				{Name: "PRIMARY", Type: "BTREE", Primary: true, Unique: true, Expressions: []string{"id"}},
				{Name: "idx_parent_id", Type: "BTREE", Expressions: []string{"parent_id"}},
			},
			ForeignKeys: []*storepb.ForeignKeyMetadata{
				{
					Name:              "fk_child_parent",
					Columns:           []string{"parent_id"},
					ReferencedTable:   "parent",
					ReferencedColumns: []string{"id"},
					OnDelete:          "CASCADE",
					OnUpdate:          "RESTRICT",
				},
			},
		},
	)

	c := newLoaderTestCatalog(t)
	runLoader(t, c, meta)

	child := mustGetTable(t, c, "child")
	var fk *catalog.Constraint
	for _, con := range child.Constraints {
		if con.Type == catalog.ConForeignKey && con.Name == "fk_child_parent" {
			fk = con
			break
		}
	}
	if fk == nil {
		t.Fatal("fk_child_parent constraint missing")
	}
	if strings.ToUpper(fk.OnDelete) != "CASCADE" {
		t.Errorf("OnDelete = %q, want CASCADE", fk.OnDelete)
	}
	if strings.ToUpper(fk.OnUpdate) != "RESTRICT" {
		t.Errorf("OnUpdate = %q, want RESTRICT", fk.OnUpdate)
	}
}

// ----------------------------------------------------------------------
// Generated columns.
// ----------------------------------------------------------------------

func TestLoader_GeneratedColumn(t *testing.T) {
	meta := schemaWithTables(&storepb.TableMetadata{
		Name: "t",
		Columns: []*storepb.ColumnMetadata{
			{Name: "a", Type: "int", Nullable: true},
			{Name: "b", Type: "int", Nullable: true},
			{
				Name:     "c",
				Type:     "int",
				Nullable: true,
				Generation: &storepb.GenerationMetadata{
					Type:       storepb.GenerationMetadata_TYPE_STORED,
					Expression: "a + b",
				},
			},
		},
	})

	c := newLoaderTestCatalog(t)
	runLoader(t, c, meta)

	tbl := mustGetTable(t, c, "t")
	col := tbl.GetColumn("c")
	if col == nil || col.Generated == nil {
		t.Fatalf("generated column not recorded: %+v", col)
	}
	if !col.Generated.Stored {
		t.Error("expected STORED, got VIRTUAL")
	}
	if !strings.Contains(col.Generated.Expr, "a") || !strings.Contains(col.Generated.Expr, "b") {
		t.Errorf("Generated.Expr = %q", col.Generated.Expr)
	}
}

// ----------------------------------------------------------------------
// View real install.
// ----------------------------------------------------------------------

func TestLoader_ViewReal(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Name: testDBName,
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "bigint unsigned", Nullable: false, Default: autoIncrementSentinel},
							{Name: "name", Type: "varchar(255)", Nullable: true},
						},
						Indexes: []*storepb.IndexMetadata{
							{Name: "PRIMARY", Type: "BTREE", Primary: true, Unique: true, Expressions: []string{"id"}},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name:       "v_users",
						Definition: "SELECT id, name FROM users",
					},
				},
			},
		},
	}

	c := newLoaderTestCatalog(t)
	runLoader(t, c, meta)

	v := mustGetView(t, c, "v_users")
	if !strings.Contains(strings.ToUpper(v.Definition), "SELECT") {
		t.Errorf("view Definition not preserved: %q", v.Definition)
	}
}

// ----------------------------------------------------------------------
// Pseudo fallbacks. We exercise them directly by calling the helpers because
// the AST-direct real path is permissive enough that natural failures are
// rare; going through the helper means we really test the Define* + ast
// construction that the fallback relies on.
// ----------------------------------------------------------------------

func TestLoader_PseudoTable(t *testing.T) {
	c := newLoaderTestCatalog(t)
	// Minimal metadata — just column names.
	tblMeta := &storepb.TableMetadata{
		Name: "pt",
		Columns: []*storepb.ColumnMetadata{
			{Name: "a"},
			{Name: "b"},
			{Name: "c"},
		},
	}
	if err := wtInstallPseudoTable(c, testDBName, tblMeta); err != nil {
		t.Fatalf("wtInstallPseudoTable: %v", err)
	}
	tbl := mustGetTable(t, c, "pt")
	if len(tbl.Columns) != 3 {
		t.Fatalf("pseudo columns: got %d want 3", len(tbl.Columns))
	}
	for _, col := range tbl.Columns {
		if !strings.EqualFold(col.DataType, "text") {
			t.Errorf("pseudo column %q: DataType=%q, want text", col.Name, col.DataType)
		}
	}
}

func TestLoader_PseudoTable_EmptyColumns(t *testing.T) {
	c := newLoaderTestCatalog(t)
	if err := wtInstallPseudoTable(c, testDBName, &storepb.TableMetadata{Name: "empty"}); err != nil {
		t.Fatalf("wtInstallPseudoTable(no columns): %v", err)
	}
	tbl := mustGetTable(t, c, "empty")
	if len(tbl.Columns) != 1 {
		t.Fatalf("placeholder columns: got %d want 1", len(tbl.Columns))
	}
	if tbl.Columns[0].Name != "__bb_placeholder" {
		t.Errorf("placeholder column name: %q", tbl.Columns[0].Name)
	}
}

func TestLoader_PseudoView(t *testing.T) {
	c := newLoaderTestCatalog(t)
	if err := wtInstallPseudoView(c, testDBName, &storepb.ViewMetadata{
		Name: "pv",
		Columns: []*storepb.ColumnMetadata{
			{Name: "x"},
			{Name: "y"},
		},
	}); err != nil {
		t.Fatalf("wtInstallPseudoView: %v", err)
	}
	v := mustGetView(t, c, "pv")
	if len(v.Columns) != 2 {
		t.Fatalf("pseudo view columns: got %d want 2", len(v.Columns))
	}
}

// ----------------------------------------------------------------------
// Fallback chain — real install fails, pseudo kicks in.
// Simulate by giving a column an unparseable type string.
// ----------------------------------------------------------------------

func TestLoader_RealFailsFallsBackToPseudo(t *testing.T) {
	meta := schemaWithTables(&storepb.TableMetadata{
		Name: "broken",
		Columns: []*storepb.ColumnMetadata{
			{Name: "good", Type: "int"},
			// Type with unbalanced parens — wtParseTypeName returns an error
			// for this, so wtBuildCreateTableStmt propagates and install_real
			// fails, triggering pseudo install.
			{Name: "bad", Type: "varchar((("},
		},
	})

	c := newLoaderTestCatalog(t)
	runLoader(t, c, meta)

	tbl := mustGetTable(t, c, "broken")
	if len(tbl.Columns) != 2 {
		t.Fatalf("pseudo columns: got %d want 2", len(tbl.Columns))
	}
	// Pseudo collapses everything to TEXT.
	for _, col := range tbl.Columns {
		if !strings.EqualFold(col.DataType, "text") {
			t.Errorf("pseudo fallback should collapse columns to TEXT; %q.DataType=%q", col.Name, col.DataType)
		}
	}
}

// ----------------------------------------------------------------------
// Topological ordering — FK forward references work with FK checks off.
// ----------------------------------------------------------------------

func TestLoader_ForwardFKInstallsViaFKChecksOff(t *testing.T) {
	// Child declared first in the metadata, referencing parent that appears
	// after. Because the loader flips foreign_key_checks off during bulk
	// install, this must succeed.
	meta := schemaWithTables(
		&storepb.TableMetadata{
			Name: "child",
			Columns: []*storepb.ColumnMetadata{
				{Name: "id", Type: "bigint unsigned", Nullable: false, Default: autoIncrementSentinel},
				{Name: "parent_id", Type: "bigint unsigned", Nullable: false},
			},
			Indexes: []*storepb.IndexMetadata{
				{Name: "PRIMARY", Type: "BTREE", Primary: true, Unique: true, Expressions: []string{"id"}},
				{Name: "idx_pid", Type: "BTREE", Expressions: []string{"parent_id"}},
			},
			ForeignKeys: []*storepb.ForeignKeyMetadata{
				{
					Name:              "fk_fwd",
					Columns:           []string{"parent_id"},
					ReferencedTable:   "parent",
					ReferencedColumns: []string{"id"},
				},
			},
		},
		&storepb.TableMetadata{
			Name: "parent",
			Columns: []*storepb.ColumnMetadata{
				{Name: "id", Type: "bigint unsigned", Nullable: false, Default: autoIncrementSentinel},
			},
			Indexes: []*storepb.IndexMetadata{
				{Name: "PRIMARY", Type: "BTREE", Primary: true, Unique: true, Expressions: []string{"id"}},
			},
		},
	)

	c := newLoaderTestCatalog(t)
	runLoader(t, c, meta)

	// Both tables present, child's FK is recorded.
	mustGetTable(t, c, "parent")
	child := mustGetTable(t, c, "child")
	var found bool
	for _, con := range child.Constraints {
		if con.Type == catalog.ConForeignKey && con.Name == "fk_fwd" {
			found = true
		}
	}
	if !found {
		t.Error("fk_fwd not recorded on child; forward-FK install failed")
	}
}

// schemaWithTables wraps tables into a DatabaseSchemaMetadata with the
// expected single empty-named schema that MySQL uses.
func schemaWithTables(tables ...*storepb.TableMetadata) *storepb.DatabaseSchemaMetadata {
	return &storepb.DatabaseSchemaMetadata{
		Name: testDBName,
		Schemas: []*storepb.SchemaMetadata{
			{
				Name:   "",
				Tables: tables,
			},
		},
	}
}
