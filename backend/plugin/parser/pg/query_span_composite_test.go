package pg

import (
	"context"
	"testing"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestLoaderInstallsCompositeTypes proves the query-span catalog loader
// installs composite types from metadata with real attributes, in dependency
// order (aa_nested sorts before its dependency zz_base alphabetically), and
// that tables typed with them install cleanly.
func TestLoaderInstallsCompositeTypes(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{Name: "status", Values: []string{"a", "b"}},
				},
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						// References zz_base only through an array suffix.
						Name: "aa_array_only",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "items", Type: "public.zz_base[]"},
						},
					},
					{
						Name: "aa_nested",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "home", Type: "public.zz_base"},
							{Name: "s", Type: "public.status"},
						},
					},
					{
						Name: "zz_base",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "street", Type: "text"},
							{Name: "city", Type: "character varying(50)"},
						},
					},
				},
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "home", Type: "public.zz_base", Nullable: true},
						},
					},
				},
			},
		},
	}

	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	loader := newCatalogLoader(cat, meta)
	if err := loader.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}

	base := cat.GetRelation("public", "zz_base")
	if base == nil {
		t.Fatal("composite type zz_base not installed")
	}
	if len(base.Columns) != 2 {
		t.Errorf("zz_base attribute count: got %d, want 2", len(base.Columns))
	}
	if base.Columns[0].Name != "street" {
		t.Errorf("zz_base first attribute: got %q, want street", base.Columns[0].Name)
	}
	nested := cat.GetRelation("public", "aa_nested")
	if nested == nil {
		t.Fatal("composite type aa_nested not installed")
	}
	if len(nested.Columns) != 2 {
		t.Errorf("aa_nested attribute count: got %d, want 2", len(nested.Columns))
	}
	arrayOnly := cat.GetRelation("public", "aa_array_only")
	if arrayOnly == nil {
		t.Fatal("composite type aa_array_only not installed")
	}
	if len(arrayOnly.Columns) != 1 || arrayOnly.Columns[0].Name != "items" {
		t.Errorf("aa_array_only must install with its real attribute, got %d columns", len(arrayOnly.Columns))
	}
	if cat.GetRelation("public", "users") == nil {
		t.Fatal("table with composite column not installed")
	}
}

// TestLoaderCompositeFallbackPreservesAttributeNames forces the pseudo
// fallback via a domain-typed attribute (domains are not loader objects) and
// asserts the fallback keeps the metadata attribute names.
func TestLoaderCompositeFallbackPreservesAttributeNames(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "with_domain",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "p", Type: "public.pos_int"},
							{Name: "note", Type: "text"},
						},
					},
				},
			},
		},
	}

	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	loader := newCatalogLoader(cat, meta)
	if err := loader.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}

	rel := cat.GetRelation("public", "with_domain")
	if rel == nil {
		t.Fatal("composite must fall back, not vanish")
	}
	if len(rel.Columns) != 2 {
		t.Fatalf("fallback must preserve attribute names, got %d columns", len(rel.Columns))
	}
	if rel.Columns[0].Name != "p" || rel.Columns[1].Name != "note" {
		t.Errorf("fallback attribute names: got %q, %q", rel.Columns[0].Name, rel.Columns[1].Name)
	}
}

// TestRangeVarFallbackSkipsCompositeTypes proves the query-span fallback does
// not expand a standalone composite type as a FROM source — PostgreSQL
// rejects composite types as table sources.
func TestRangeVarFallbackSkipsCompositeTypes(t *testing.T) {
	meta := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				CompositeTypes: []*storepb.CompositeTypeMetadata{
					{
						Name: "addr",
						Attributes: []*storepb.CompositeTypeAttribute{
							{Name: "street", Type: "text"},
						},
					},
				},
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
					},
				},
			},
		},
	}

	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	loader := newCatalogLoader(cat, meta)
	if err := loader.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}

	e := &omniQuerySpanExtractor{
		cat:             cat,
		searchPath:      []string{"public"},
		defaultDatabase: "db",
	}
	if results := e.extractColumnsFromRangeVar(&ast.RangeVar{Schemaname: "public", Relname: "addr"}); results != nil {
		t.Errorf("composite type must not expand as a FROM source, got %d columns", len(results))
	}
	if results := e.extractColumnsFromRangeVar(&ast.RangeVar{Schemaname: "public", Relname: "t"}); len(results) != 1 {
		t.Errorf("real table must still expand, got %d columns", len(results))
	}
}
