package pg

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ---------------- typeNameFromString ----------------

func TestLoaderTypeNameFromString(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"int4", false},
		{"text", false},
		{"numeric(10,2)", false},
		{"timestamp(3) with time zone", false},
		{"character varying(255)", false},
		{"text[]", false},
		{"public.task_status", false},
		{`"public"."task_status"`, false},
		{"", true}, // empty cannot parse as SELECT NULL::
	}
	for _, c := range cases {
		tn, err := typeNameFromString(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("typeNameFromString(%q): expected error, got tn=%+v", c.in, tn)
			}
			continue
		}
		if err != nil {
			t.Errorf("typeNameFromString(%q): unexpected error: %v", c.in, err)
			continue
		}
		if tn == nil {
			t.Errorf("typeNameFromString(%q): nil TypeName with no error", c.in)
		}
	}
}

// ---------------- extractUserTypeRefs ----------------

func TestLoaderExtractUserTypeRefs(t *testing.T) {
	cases := []struct {
		in   string
		want []UserTypeRef
	}{
		// Built-in scalars.
		{"integer", nil},
		{"int4", nil},
		{"text", nil},
		{"bigint", nil},
		{"boolean", nil},
		{"json", nil},
		{"jsonb", nil},
		{"uuid", nil},
		{"date", nil},
		{"interval", nil},

		// Built-in with length / precision / timezone.
		{"numeric(10,2)", nil},
		{"numeric(8)", nil},
		{"decimal(10,2)", nil},
		{"character varying(255)", nil},
		{"character(10)", nil},
		{"varchar(255)", nil},
		{"bit(8)", nil},
		{"bit varying(8)", nil},
		{"timestamp(3) with time zone", nil},
		{"time(6) without time zone", nil},
		{"timestamp without time zone", nil},

		// USER-DEFINED.
		{"public.task_status", []UserTypeRef{{Schema: "public", Name: "task_status"}}},
		{"myschema.my_domain", []UserTypeRef{{Schema: "myschema", Name: "my_domain"}}},

		// Quoted qualified name.
		{`"MySchema"."MyType"`, []UserTypeRef{{Schema: "MySchema", Name: "MyType"}}},

		// PG internal array form.
		{"_text", nil},
		{"_int4", nil},
		{"_task_status", nil}, // sync.go:834 drops the schema; we do not topo-sort it.

		// System schemas.
		{"pg_catalog.int4", nil},
		{"information_schema.cardinal_number", nil},
		{"pg_toast.pg_toast_2619", nil},

		// Empty.
		{"", nil},

		// Whitespace.
		{"  text  ", nil},
	}
	for _, c := range cases {
		got := extractUserTypeRefs(c.in)
		if !userTypeRefsEqual(got, c.want) {
			t.Errorf("extractUserTypeRefs(%q): got %+v, want %+v", c.in, got, c.want)
		}
	}
}

func userTypeRefsEqual(a, b []UserTypeRef) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ---------------- helpers ----------------

func TestLoaderStripTypeModifiers(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"numeric(10,2)", "numeric"},
		{"timestamp(3) with time zone", "timestamp"},
		{"timestamp without time zone", "timestamp"},
		{"character varying(255)", "character varying"},
		{"  text  ", "text"},
		{"public.task_status", "public.task_status"},
		{"", ""},
	}
	for _, c := range cases {
		if got := stripTypeModifiers(c.in); got != c.want {
			t.Errorf("stripTypeModifiers(%q): got %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLoaderSplitQualifiedName(t *testing.T) {
	cases := []struct {
		in         string
		wantSchema string
		wantName   string
		wantOk     bool
	}{
		{"public.foo", "public", "foo", true},
		{`"weird name"."another"`, "weird name", "another", true},
		{"foo", "", "", false},
		{"", "", "", false},
		{"a.b.c", "", "", false}, // too many parts
	}
	for _, c := range cases {
		s, n, ok := splitQualifiedName(c.in)
		if s != c.wantSchema || n != c.wantName || ok != c.wantOk {
			t.Errorf("splitQualifiedName(%q): got (%q,%q,%v), want (%q,%q,%v)",
				c.in, s, n, ok, c.wantSchema, c.wantName, c.wantOk)
		}
	}
}

func TestLoaderIsSystemSchema(t *testing.T) {
	// Sanity check that extractUserTypeRefs routes system schemas to nil via
	// IsSystemSchema. The underlying predicate is tested in system_objects.go.
	cases := map[string]bool{
		"pg_catalog.foo":         true,
		"information_schema.bar": true,
		"pg_toast.baz":           true,
		"pg_temp_1.qux":          true,
		"public.foo":             false,
		"myschema.bar":           false,
	}
	for in, wantFiltered := range cases {
		got := extractUserTypeRefs(in)
		if wantFiltered && got != nil {
			t.Errorf("extractUserTypeRefs(%q): expected filtered, got %+v", in, got)
		}
		if !wantFiltered && got == nil {
			t.Errorf("extractUserTypeRefs(%q): expected non-nil, got nil", in)
		}
	}
}

// ---------------- classifyAnalyzeError ----------------

func TestLoaderClassifyAnalyzeError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want fallbackReason
	}{
		{"nil", nil, reasonNone},
		{
			"undefined function",
			&catalog.Error{Code: catalog.CodeUndefinedFunction, Message: "function fn(text) does not exist"},
			reasonExpectedPseudoSemantic,
		},
		{
			"ambiguous function",
			&catalog.Error{Code: catalog.CodeAmbiguousFunction, Message: "function fn is not unique"},
			reasonExpectedPseudoSemantic,
		},
		{
			"datatype mismatch",
			&catalog.Error{Code: catalog.CodeDatatypeMismatch, Message: "UNION types text and integer cannot be matched"},
			reasonExpectedPseudoSemantic,
		},
		{
			"feature not supported",
			&catalog.Error{Code: catalog.CodeFeatureNotSupported, Message: "not supported"},
			reasonExpectedPseudoSemantic,
		},
		{
			"ambiguous column",
			&catalog.Error{Code: catalog.CodeAmbiguousColumn, Message: "column x is ambiguous"},
			reasonExpectedPseudoSemantic,
		},
		{
			"undefined table",
			&catalog.Error{Code: catalog.CodeUndefinedTable, Message: `relation "foo" does not exist`},
			reasonUndefinedReference,
		},
		{
			"undefined column",
			&catalog.Error{Code: catalog.CodeUndefinedColumn, Message: `column "x" does not exist`},
			reasonUndefinedReference,
		},
		{
			"undefined object",
			&catalog.Error{Code: catalog.CodeUndefinedObject, Message: `type "t" does not exist`},
			reasonUndefinedReference,
		},
		{
			"undefined schema",
			&catalog.Error{Code: catalog.CodeUndefinedSchema, Message: `schema "s" does not exist`},
			reasonUndefinedReference,
		},
		{
			"other catalog error",
			&catalog.Error{Code: catalog.CodeDuplicateTable, Message: "duplicate"},
			reasonAnalyzerUnsupported,
		},
		{
			"plain error",
			errors.New("unsupported node type"),
			reasonAnalyzerUnsupported,
		},
		{
			"wrapped catalog error",
			errors.Wrap(&catalog.Error{Code: catalog.CodeDatatypeMismatch, Message: "mismatch"}, "ctx"),
			reasonExpectedPseudoSemantic,
		},
	}
	for _, c := range cases {
		if got := classifyAnalyzeError(c.err); got != c.want {
			t.Errorf("%s: classifyAnalyzeError got %v, want %v", c.name, got, c.want)
		}
	}
}

// ---------------- pseudo builders ----------------

func TestLoaderPseudoEnum(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	if err := cat.DefineEnum(pseudoCreateEnumStmt("public", "broken")); err != nil {
		t.Fatalf("DefineEnum: %v", err)
	}
	// Reference it from a table column and analyze.
	tn, err := typeNameFromString("public.broken")
	if err != nil {
		t.Fatalf("typeNameFromString: %v", err)
	}
	tbl := &ast.CreateStmt{
		Relation: &ast.RangeVar{Schemaname: "public", Relname: "t", Relpersistence: 'p'},
		TableElts: &ast.List{Items: []ast.Node{
			&ast.ColumnDef{Colname: "s", TypeName: tn},
		}},
	}
	if err := cat.DefineRelation(tbl, 'r'); err != nil {
		t.Fatalf("DefineRelation with pseudo enum column: %v", err)
	}
}

func TestLoaderPseudoDomain(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	if err := cat.DefineDomain(pseudoCreateDomainStmt("public", "d")); err != nil {
		t.Fatalf("DefineDomain: %v", err)
	}
}

func TestLoaderPseudoComposite(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	if err := cat.DefineCompositeType(pseudoCompositeTypeStmt("public", "c")); err != nil {
		t.Fatalf("DefineCompositeType: %v", err)
	}
}

func TestLoaderPseudoRange(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	if err := cat.DefineRange(pseudoCreateRangeStmt("public", "r")); err != nil {
		t.Fatalf("DefineRange: %v", err)
	}
}

func TestLoaderPseudoTable(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	stmt := pseudoCreateTableStmt("public", "t", []string{"id", "name", "email"})
	if err := cat.DefineRelation(stmt, 'r'); err != nil {
		t.Fatalf("DefineRelation: %v", err)
	}
	rel := cat.GetRelation("public", "t")
	if rel == nil {
		t.Fatal("relation not installed")
	}
	if len(rel.Columns) != 3 {
		t.Errorf("column count: got %d, want 3", len(rel.Columns))
	}
}

func TestLoaderPseudoView(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	stmt, err := pseudoViewStmt("public", "v", []string{"id", "label"})
	if err != nil {
		t.Fatalf("build pseudo view: %v", err)
	}
	if err := cat.DefineView(stmt); err != nil {
		t.Fatalf("DefineView: %v", err)
	}
	rel := cat.GetRelation("public", "v")
	if rel == nil {
		t.Fatal("view not installed")
	}
	if len(rel.Columns) != 2 {
		t.Errorf("view column count: got %d, want 2", len(rel.Columns))
	}
}

func TestLoaderPseudoMatView(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	stmt, err := pseudoCreateTableAsStmt("public", "m", []string{"x", "y"})
	if err != nil {
		t.Fatalf("build pseudo matview: %v", err)
	}
	if err := cat.ExecCreateTableAs(stmt); err != nil {
		t.Fatalf("ExecCreateTableAs: %v", err)
	}
}

func TestLoaderPseudoFunction(t *testing.T) {
	cases := []struct {
		name     string
		argCount int
	}{
		{"zero args", 0},
		{"one arg", 1},
		{"three args", 3},
	}
	for _, c := range cases {
		cat := catalog.New()
		cat.SetSearchPath([]string{"public"})
		stmt := pseudoCreateFunctionStmt("public", "fn", c.argCount)
		if err := cat.CreateFunctionStmt(stmt); err != nil {
			t.Errorf("%s: CreateFunctionStmt: %v", c.name, err)
			continue
		}
		procs := cat.LookupProcByName("fn")
		if len(procs) == 0 {
			t.Errorf("%s: no procs installed", c.name)
		}
	}
}

func TestLoaderFunctionArgCountFromSignature(t *testing.T) {
	cases := map[string]int{
		"":                        0,
		"fn()":                    0,
		"fn(integer)":             1,
		"fn(integer, text)":       2,
		"fn(integer, text, uuid)": 3,
		"fn(numeric(10,2))":       1, // inner paren must not split
		"fn(numeric(10,2), int)":  2,
	}
	for in, want := range cases {
		if got := functionArgCountFromSignature(in); got != want {
			t.Errorf("functionArgCountFromSignature(%q): got %d, want %d", in, got, want)
		}
	}
}

func TestLoaderQualifiedNameList(t *testing.T) {
	list := qualifiedNameList("public", "foo")
	if list == nil || len(list.Items) != 2 {
		t.Fatalf("qualifiedNameList(public, foo): got %+v, want 2 items", list)
	}
	bare := qualifiedNameList("", "foo")
	if bare == nil || len(bare.Items) != 1 {
		t.Fatalf("qualifiedNameList(\"\", foo): got %+v, want 1 item", bare)
	}
}

// ---------------- real builders ----------------

func TestLoaderBuildCreateEnumStmt(t *testing.T) {
	stmt := buildCreateEnumStmt("public", &storepb.EnumTypeMetadata{
		Name:   "task_status",
		Values: []string{"pending", "running", "done"},
	})
	if stmt == nil || stmt.Vals == nil || len(stmt.Vals.Items) != 3 {
		t.Fatalf("unexpected: %+v", stmt)
	}
	cat := catalog.New()
	if err := cat.DefineEnum(stmt); err != nil {
		t.Fatalf("DefineEnum: %v", err)
	}
}

func TestLoaderBuildCreateStmt(t *testing.T) {
	stmt, err := buildCreateStmt("public", &storepb.TableMetadata{
		Name: "t",
		Columns: []*storepb.ColumnMetadata{
			{Name: "id", Type: "int4", Nullable: false},
			{Name: "name", Type: "text", Nullable: true},
			{Name: "amount", Type: "numeric(10,2)", Nullable: true},
		},
	})
	if err != nil {
		t.Fatalf("buildCreateStmt: %v", err)
	}
	if stmt == nil || stmt.TableElts == nil || len(stmt.TableElts.Items) != 3 {
		t.Fatalf("unexpected: %+v", stmt)
	}
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	if err := cat.DefineRelation(stmt, 'r'); err != nil {
		t.Fatalf("DefineRelation: %v", err)
	}
	rel := cat.GetRelation("public", "t")
	if rel == nil || len(rel.Columns) != 3 {
		t.Fatalf("rel: %+v", rel)
	}
}

func TestLoaderBuildCreateStmt_BadTypeErrors(t *testing.T) {
	_, err := buildCreateStmt("public", &storepb.TableMetadata{
		Name: "t",
		Columns: []*storepb.ColumnMetadata{
			{Name: "bad", Type: "this is not a type at all;;"},
		},
	})
	if err == nil {
		t.Fatal("expected error for bad column type")
	}
}

func TestLoaderBuildCreateStmt_EmptyTypeErrors(t *testing.T) {
	_, err := buildCreateStmt("public", &storepb.TableMetadata{
		Name:    "t",
		Columns: []*storepb.ColumnMetadata{{Name: "x", Type: ""}},
	})
	if err == nil {
		t.Fatal("expected error for empty column type")
	}
}

func TestLoaderBuildViewStmt(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	base, err := buildCreateStmt("public", &storepb.TableMetadata{
		Name: "orders",
		Columns: []*storepb.ColumnMetadata{
			{Name: "id", Type: "int4"},
			{Name: "total", Type: "int4"},
		},
	})
	if err != nil {
		t.Fatalf("build base: %v", err)
	}
	if err := cat.DefineRelation(base, 'r'); err != nil {
		t.Fatalf("install base: %v", err)
	}
	stmt, err := buildViewStmt("public", &storepb.ViewMetadata{
		Name:       "v",
		Definition: "SELECT id, total FROM orders",
	})
	if err != nil {
		t.Fatalf("buildViewStmt: %v", err)
	}
	if err := cat.DefineView(stmt); err != nil {
		t.Fatalf("DefineView: %v", err)
	}
}

func TestLoaderBuildViewStmt_EmptyDefinition(t *testing.T) {
	_, err := buildViewStmt("public", &storepb.ViewMetadata{Name: "v"})
	if err == nil {
		t.Fatal("expected error for empty definition")
	}
}

func TestLoaderBuildCreateTableAsStmt(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	base, err := buildCreateStmt("public", &storepb.TableMetadata{
		Name: "orders",
		Columns: []*storepb.ColumnMetadata{
			{Name: "id", Type: "int4"},
		},
	})
	if err != nil {
		t.Fatalf("build base: %v", err)
	}
	if err := cat.DefineRelation(base, 'r'); err != nil {
		t.Fatalf("install base: %v", err)
	}
	stmt, err := buildCreateTableAsStmt("public", &storepb.MaterializedViewMetadata{
		Name:       "m",
		Definition: "SELECT id FROM orders",
	})
	if err != nil {
		t.Fatalf("buildCreateTableAsStmt: %v", err)
	}
	if err := cat.ExecCreateTableAs(stmt); err != nil {
		t.Fatalf("ExecCreateTableAs: %v", err)
	}
}

func TestLoaderBuildCreateFunctionStmt(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	stmt, err := buildCreateFunctionStmt("public", &storepb.FunctionMetadata{
		Name:      "fn",
		Signature: "fn(integer, text)",
	})
	if err != nil {
		t.Fatalf("buildCreateFunctionStmt: %v", err)
	}
	if stmt.Parameters == nil || len(stmt.Parameters.Items) != 2 {
		t.Fatalf("expected 2 params, got %+v", stmt.Parameters)
	}
	if err := cat.CreateFunctionStmt(stmt); err != nil {
		t.Fatalf("CreateFunctionStmt: %v", err)
	}
}

func TestLoaderBuildCreateFunctionStmt_ZeroArgs(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	stmt, err := buildCreateFunctionStmt("public", &storepb.FunctionMetadata{
		Name:      "fn",
		Signature: "fn()",
	})
	if err != nil {
		t.Fatalf("buildCreateFunctionStmt: %v", err)
	}
	if err := cat.CreateFunctionStmt(stmt); err != nil {
		t.Fatalf("CreateFunctionStmt: %v", err)
	}
}

func TestLoaderParseFunctionSignatureArgTypes(t *testing.T) {
	cases := map[string][]string{
		"fn()":                    nil,
		"fn(integer)":             {"integer"},
		"fn(integer, text)":       {"integer", "text"},
		"fn(numeric(10,2), text)": {"numeric(10,2)", "text"},
		"fn(  int4 , text )":      {"int4", "text"},
		"fn":                      nil,
	}
	for in, want := range cases {
		got, err := parseFunctionSignatureArgTypes(in)
		if err != nil {
			t.Errorf("%q: unexpected error %v", in, err)
			continue
		}
		if !stringSlicesEqual(got, want) {
			t.Errorf("%q: got %v, want %v", in, got, want)
		}
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestLoaderParseSelectBody(t *testing.T) {
	sel, err := parseSelectBody("SELECT 1 AS x")
	if err != nil {
		t.Fatalf("parseSelectBody: %v", err)
	}
	if sel == nil {
		t.Fatal("nil sel")
	}
	if _, err := parseSelectBody(""); err == nil {
		t.Error("expected error for empty body")
	}
	if _, err := parseSelectBody("SELECT 1; SELECT 2"); err == nil {
		t.Error("expected error for multi-statement body")
	}
}

// ---------------- loader ----------------

func TestLoaderLoader_HappyPath(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	meta := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			EnumTypes: []*storepb.EnumTypeMetadata{
				{Name: "task_status", Values: []string{"pending", "running"}},
			},
			Tables: []*storepb.TableMetadata{{
				Name: "tasks",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					{Name: "status", Type: "public.task_status"},
					{Name: "title", Type: "text"},
				},
			}},
			Views: []*storepb.ViewMetadata{{
				Name:       "open_tasks",
				Definition: "SELECT id, title, status FROM tasks",
				DependencyColumns: []*storepb.DependencyColumn{
					{Schema: "public", Table: "tasks", Column: "id"},
					{Schema: "public", Table: "tasks", Column: "title"},
					{Schema: "public", Table: "tasks", Column: "status"},
				},
			}},
		}},
	}
	loader := newCatalogLoader(cat, meta)
	if err := loader.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loader.degraded) != 0 {
		t.Errorf("expected zero degraded, got %+v", loader.degraded)
	}
	if len(loader.trulyBroken) != 0 {
		t.Errorf("expected zero truly broken, got %+v", loader.trulyBroken)
	}
	if cat.GetRelation("public", "tasks") == nil {
		t.Error("tasks not installed")
	}
	if cat.GetRelation("public", "open_tasks") == nil {
		t.Error("open_tasks not installed")
	}
}

func TestLoaderLoader_BrokenEnumCascadesToPseudo(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	meta := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Tables: []*storepb.TableMetadata{{
				Name: "t",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "int4"},
					// Reference a user type that is not in metadata. The
					// real install will fail; pseudo should take over.
					{Name: "status", Type: "public.nonexistent_enum"},
				},
			}},
		}},
	}
	loader := newCatalogLoader(cat, meta)
	if err := loader.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	key := "rel:public.t"
	if _, ok := loader.degraded[key]; !ok {
		t.Errorf("expected public.t to be degraded, got %+v", loader.degraded)
	}
	if _, ok := loader.trulyBroken[key]; ok {
		t.Errorf("unexpected truly broken: %+v", loader.trulyBroken)
	}
	rel := cat.GetRelation("public", "t")
	if rel == nil {
		t.Fatal("pseudo t not installed")
	}
	if len(rel.Columns) != 2 {
		t.Errorf("pseudo t: got %d cols, want 2", len(rel.Columns))
	}
}

func TestLoaderLoader_TopoOrder_DependencyBeforeUse(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	meta := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			// View is listed BEFORE its base table in metadata. The loader
			// must reorder so the table is installed first; otherwise the
			// view body analyzes against an empty catalog.
			Views: []*storepb.ViewMetadata{{
				Name:       "v",
				Definition: "SELECT id FROM base",
				DependencyColumns: []*storepb.DependencyColumn{
					{Schema: "public", Table: "base", Column: "id"},
				},
			}},
			Tables: []*storepb.TableMetadata{{
				Name: "base",
				Columns: []*storepb.ColumnMetadata{
					{Name: "id", Type: "int4"},
				},
			}},
		}},
	}
	loader := newCatalogLoader(cat, meta)
	if err := loader.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loader.degraded["rel:public.v"] != nil {
		t.Errorf("view should install real, got degraded: %v", loader.degraded["rel:public.v"])
	}
	if loader.degraded["rel:public.base"] != nil {
		t.Errorf("base should install real, got degraded: %v", loader.degraded["rel:public.base"])
	}
}

func TestLoaderLoader_CycleBreaking(t *testing.T) {
	cat := catalog.New()
	cat.SetSearchPath([]string{"public"})
	// Two views referencing each other — a metadata-level cycle.
	meta := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			Views: []*storepb.ViewMetadata{
				{
					Name:       "v_alpha",
					Definition: "SELECT id FROM v_beta",
					DependencyColumns: []*storepb.DependencyColumn{
						{Schema: "public", Table: "v_beta", Column: "id"},
					},
				},
				{
					Name:       "v_beta",
					Definition: "SELECT id FROM v_alpha",
					DependencyColumns: []*storepb.DependencyColumn{
						{Schema: "public", Table: "v_alpha", Column: "id"},
					},
				},
			},
		}},
	}
	loader := newCatalogLoader(cat, meta)
	if err := loader.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Both views should be in the catalog (real or pseudo). At least one
	// should be degraded (the lex-first member of the SCC).
	if cat.GetRelation("public", "v_alpha") == nil {
		t.Error("v_alpha missing")
	}
	if cat.GetRelation("public", "v_beta") == nil {
		t.Error("v_beta missing")
	}
	if len(loader.degraded) == 0 {
		t.Error("expected at least one degraded in SCC")
	}
	// v_alpha is lex-first by sortKey, so it should be the pseudo root.
	if _, ok := loader.degraded["rel:public.v_alpha"]; !ok {
		t.Errorf("lex-first v_alpha should be degraded, got %+v", loader.degraded)
	}
}

func TestLoaderLoader_LoaderObjectsTracksEverything(t *testing.T) {
	cat := catalog.New()
	meta := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{{
			Name: "public",
			EnumTypes: []*storepb.EnumTypeMetadata{
				{Name: "e", Values: []string{"a"}},
			},
			Tables: []*storepb.TableMetadata{{
				Name:    "t",
				Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "int4"}},
			}},
		}},
	}
	loader := newCatalogLoader(cat, meta)
	_ = loader.Load(context.Background())
	if !loader.loaderObjects["schema:public"] {
		t.Error("schema:public not tracked")
	}
	if !loader.loaderObjects["type:public.e"] {
		t.Error("type:public.e not tracked")
	}
	if !loader.loaderObjects["rel:public.t"] {
		t.Error("rel:public.t not tracked")
	}
}

func TestLoaderLoader_NilMetaIsNoop(t *testing.T) {
	cat := catalog.New()
	loader := newCatalogLoader(cat, nil)
	if err := loader.Load(context.Background()); err != nil {
		t.Errorf("Load(nil meta): %v", err)
	}
}

func TestLoaderLoader_NilCatalogErrors(t *testing.T) {
	loader := newCatalogLoader(nil, &storepb.DatabaseSchemaMetadata{})
	if err := loader.Load(context.Background()); err == nil {
		t.Error("expected error for nil catalog")
	}
}

func TestLoaderTopoSort_Determinism(t *testing.T) {
	// Build a few fixtures and ensure topoSortObjects returns the same order
	// for the same input.
	objects := []*objectEntry{
		{kind: kindSchema, schema: "public", name: "public"},
		{kind: kindTable, schema: "public", name: "b", tableMeta: &storepb.TableMetadata{Name: "b"}},
		{kind: kindTable, schema: "public", name: "a", tableMeta: &storepb.TableMetadata{Name: "a"}},
		{kind: kindEnum, schema: "public", name: "e", enumMeta: &storepb.EnumTypeMetadata{Name: "e"}},
	}
	first := topoSortObjects(objects)
	second := topoSortObjects(objects)
	if len(first) != len(second) {
		t.Fatal("length mismatch")
	}
	for i := range first {
		if first[i].key() != second[i].key() {
			t.Errorf("non-deterministic: [%d] %s vs %s", i, first[i].key(), second[i].key())
		}
	}
}

func TestLoaderFallbackReasonString(t *testing.T) {
	cases := map[fallbackReason]string{
		reasonNone:                   "none",
		reasonExpectedPseudoSemantic: "expected_pseudo_semantic",
		reasonUndefinedReference:     "undefined_reference",
		reasonAnalyzerUnsupported:    "analyzer_unsupported",
	}
	for r, want := range cases {
		if got := r.String(); got != want {
			t.Errorf("fallbackReason(%d).String(): got %q, want %q", r, got, want)
		}
	}
}
