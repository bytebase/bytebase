package pg

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
)

// This file builds "pseudo" AST nodes used by the Catalog loader when a real
// install fails. Every pseudo form is backed by built-in types only (text,
// primarily) so pseudo install itself cannot cascade into further failures.
//
// Pseudo forms preserve object names and — where metadata allows — column
// names. Type fidelity is intentionally dropped. For query span this is
// sufficient in the vast majority of cases: lineage is a name-resolution
// problem, not a type-resolution problem.
//
// Every form in this file corresponds to a PoC test in
// query_span_v5_poc_test.go (TestLoaderPoC_Pseudo*).

// pseudoTextTypeName is the single pseudo column/parameter/return type omni
// accepts uniformly. Every pseudo form ends up routing to this shape.
func pseudoTextTypeName() *ast.TypeName {
	return &ast.TypeName{
		Names:   &ast.List{Items: []ast.Node{&ast.String{Str: "text"}}},
		Typemod: -1,
	}
}

// pseudoCreateEnumStmt builds a minimal enum with zero values. omni's
// DefineEnum accepts an empty Vals list.
func pseudoCreateEnumStmt(schema, name string) *ast.CreateEnumStmt {
	return &ast.CreateEnumStmt{
		TypeName: qualifiedNameList(schema, name),
		Vals:     &ast.List{Items: []ast.Node{}},
	}
}

// pseudoCreateDomainStmt builds a domain over text with no constraints.
// Query span does not inspect domain constraints, so dropping them is safe.
func pseudoCreateDomainStmt(schema, name string) *ast.CreateDomainStmt {
	return &ast.CreateDomainStmt{
		Domainname: qualifiedNameList(schema, name),
		Typname:    pseudoTextTypeName(),
	}
}

// pseudoCompositeTypeStmt builds a composite type with a single text field
// named "_broken". Without CompositeTypeMetadata in storepb we cannot
// preserve real field names, so (col).field access against a pseudo composite
// will fall through to extractFallbackColumns.
func pseudoCompositeTypeStmt(schema, name string) *ast.CompositeTypeStmt {
	return &ast.CompositeTypeStmt{
		Typevar: &ast.RangeVar{
			Schemaname: schema,
			Relname:    name,
		},
		Coldeflist: &ast.List{Items: []ast.Node{
			&ast.ColumnDef{
				Colname:  "_broken",
				TypeName: pseudoTextTypeName(),
			},
		}},
	}
}

// pseudoCreateRangeStmt builds a range over text. text has btree ordering so
// omni accepts it as a valid subtype.
func pseudoCreateRangeStmt(schema, name string) *ast.CreateRangeStmt {
	return &ast.CreateRangeStmt{
		TypeName: qualifiedNameList(schema, name),
		Params: &ast.List{Items: []ast.Node{
			&ast.DefElem{
				Defname: "subtype",
				Arg:     pseudoTextTypeName(),
			},
		}},
	}
}

// pseudoCreateTableStmt builds a table whose columns are all text-typed but
// retain their metadata names. This is the fallback used when real install
// fails because of unresolved column types (broken enum, unknown composite,
// etc.).
func pseudoCreateTableStmt(schema, name string, columnNames []string) *ast.CreateStmt {
	items := make([]ast.Node, 0, len(columnNames))
	for _, col := range columnNames {
		if col == "" {
			continue
		}
		items = append(items, &ast.ColumnDef{
			Colname:  col,
			TypeName: pseudoTextTypeName(),
		})
	}
	return &ast.CreateStmt{
		Relation: &ast.RangeVar{
			Schemaname:     schema,
			Relname:        name,
			Relpersistence: 'p',
		},
		TableElts: &ast.List{Items: items},
	}
}

// pseudoViewStmt builds a view whose body is a constant SELECT target list
// exposing each metadata column name as a nullable text expression. The body
// has no FROM clause.
//
// Returns an error if the body cannot be parsed; callers should treat this
// as a "truly broken" object.
func pseudoViewStmt(schema, name string, columnNames []string) (*ast.ViewStmt, error) {
	sel, err := pseudoConstantSelect(columnNames)
	if err != nil {
		return nil, err
	}
	return &ast.ViewStmt{
		View: &ast.RangeVar{
			Schemaname:     schema,
			Relname:        name,
			Relpersistence: 'p',
		},
		Query: sel,
	}, nil
}

// pseudoCreateTableAsStmt builds a pseudo materialized view using the same
// constant-target-list body as pseudoViewStmt.
func pseudoCreateTableAsStmt(schema, name string, columnNames []string) (*ast.CreateTableAsStmt, error) {
	sel, err := pseudoConstantSelect(columnNames)
	if err != nil {
		return nil, err
	}
	return &ast.CreateTableAsStmt{
		Query:   sel,
		Objtype: ast.OBJECT_MATVIEW,
		Into: &ast.IntoClause{
			Rel: &ast.RangeVar{
				Schemaname:     schema,
				Relname:        name,
				Relpersistence: 'p',
			},
		},
	}, nil
}

// pseudoCreateFunctionStmt builds a function whose signature collapses every
// parameter to text and whose return type is text. argCount is taken from the
// real function's signature so analyzer overload resolution by arity still
// matches against the pseudo.
//
// The function body is `SELECT $1` when argCount > 0, `SELECT NULL::text`
// otherwise — always a valid SQL-language body.
func pseudoCreateFunctionStmt(schema, name string, argCount int) *ast.CreateFunctionStmt {
	stmt := &ast.CreateFunctionStmt{
		Funcname:   qualifiedNameList(schema, name),
		ReturnType: pseudoTextTypeName(),
	}
	if argCount > 0 {
		params := make([]ast.Node, argCount)
		for i := range argCount {
			params[i] = &ast.FunctionParameter{
				ArgType: pseudoTextTypeName(),
				Mode:    ast.FUNC_PARAM_IN,
			}
		}
		stmt.Parameters = &ast.List{Items: params}
	}
	body := "SELECT NULL::text"
	if argCount > 0 {
		body = "SELECT $1::text"
	}
	stmt.Options = &ast.List{Items: []ast.Node{
		&ast.DefElem{Defname: "language", Arg: &ast.String{Str: "sql"}},
		&ast.DefElem{
			Defname: "as",
			Arg:     &ast.List{Items: []ast.Node{&ast.String{Str: body}}},
		},
	}}
	return stmt
}

// pseudoConstantSelect parses "SELECT NULL::text AS col1, NULL::text AS col2 ..."
// and returns the resulting *ast.SelectStmt. Columns with empty names are
// skipped; if the resulting target list is empty, a single anonymous column
// is emitted so the SELECT is well-formed.
func pseudoConstantSelect(columnNames []string) (*ast.SelectStmt, error) {
	var targets []string
	for _, col := range columnNames {
		if col == "" {
			continue
		}
		targets = append(targets, "NULL::text AS "+quoteIdent(col))
	}
	if len(targets) == 0 {
		targets = []string{"NULL::text"}
	}
	sql := "SELECT " + strings.Join(targets, ", ")
	stmts, err := ParsePg(sql)
	if err != nil {
		return nil, errors.Wrap(err, "parse pseudo constant select")
	}
	if len(stmts) != 1 {
		return nil, errors.Errorf("pseudo constant select: expected 1 statement, got %d", len(stmts))
	}
	sel, ok := stmts[0].AST.(*ast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("pseudo constant select: expected SelectStmt, got %T", stmts[0].AST)
	}
	return sel, nil
}

// functionArgCountFromSignature parses a signature string like
// "my_func(integer, text)" and returns the number of arguments. A signature
// with no parentheses or an empty argument list returns 0.
func functionArgCountFromSignature(signature string) int {
	openIdx := strings.Index(signature, "(")
	if openIdx < 0 {
		return 0
	}
	closeIdx := strings.LastIndex(signature, ")")
	if closeIdx < 0 || closeIdx <= openIdx {
		return 0
	}
	inner := strings.TrimSpace(signature[openIdx+1 : closeIdx])
	if inner == "" {
		return 0
	}
	// Split on top-level commas. Type names inside parens (rare, e.g.
	// numeric(10,2)) must not count as separators.
	depth := 0
	count := 1
	for _, r := range inner {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				count++
			}
		default:
			// other runes contribute nothing to arg-count accounting
		}
	}
	return count
}

// ---------------- small AST helpers shared with builders.go ----------------

// qualifiedNameList builds an omni List of String nodes representing a
// schema-qualified object name. An empty schema emits only the name.
func qualifiedNameList(schema, name string) *ast.List {
	items := make([]ast.Node, 0, 2)
	if schema != "" {
		items = append(items, &ast.String{Str: schema})
	}
	items = append(items, &ast.String{Str: name})
	return &ast.List{Items: items}
}

// quoteIdent wraps an identifier in double quotes, doubling any internal
// double quotes. Used when a metadata column name may contain special chars
// or collide with SQL reserved words.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
