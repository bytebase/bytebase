package pg

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// This file translates storepb metadata into omni AST nodes. Each build*
// function is deterministic, allocates fresh nodes, and does not touch the
// catalog. Builders may return errors for inputs the cheat parser rejects;
// the loader catches those errors and installs a pseudo at the same slot.

// buildCreateEnumStmt translates EnumTypeMetadata + schema into a
// CreateEnumStmt. Enum values are preserved as plain string literals.
func buildCreateEnumStmt(schema string, enum *storepb.EnumTypeMetadata) *ast.CreateEnumStmt {
	vals := make([]ast.Node, 0, len(enum.Values))
	for _, v := range enum.Values {
		vals = append(vals, &ast.String{Str: v})
	}
	return &ast.CreateEnumStmt{
		TypeName: qualifiedNameList(schema, enum.Name),
		Vals:     &ast.List{Items: vals},
	}
}

// buildCreateStmt translates TableMetadata into a CreateStmt with real
// column types. Column types are parsed via typeNameFromString; an error on
// any single column propagates back so the loader can pseudo the whole
// relation rather than install a partially-typed catalog entry.
//
// Columns with empty names are skipped.
func buildCreateStmt(schema string, table *storepb.TableMetadata) (*ast.CreateStmt, error) {
	items := make([]ast.Node, 0, len(table.Columns))
	for _, col := range table.Columns {
		if col.Name == "" {
			continue
		}
		if col.Type == "" {
			return nil, errors.Errorf("column %q: empty type", col.Name)
		}
		tn, err := typeNameFromString(col.Type)
		if err != nil {
			return nil, errors.Wrapf(err, "column %q", col.Name)
		}
		items = append(items, &ast.ColumnDef{
			Colname:   col.Name,
			TypeName:  tn,
			IsNotNull: !col.Nullable,
		})
	}
	return &ast.CreateStmt{
		Relation: &ast.RangeVar{
			Schemaname:     schema,
			Relname:        table.Name,
			Relpersistence: 'p',
		},
		TableElts: &ast.List{Items: items},
	}, nil
}

// buildViewStmt translates ViewMetadata into a ViewStmt. The view body is
// parsed via ParsePg; the result must be a *ast.SelectStmt (not a RawStmt
// wrapper) because omni's DefineView type-asserts directly.
//
// If the body is empty or unparseable, returns an error; the loader will
// pseudo the view using dependency_columns metadata.
func buildViewStmt(schema string, view *storepb.ViewMetadata) (*ast.ViewStmt, error) {
	if view.Definition == "" {
		return nil, errors.New("empty view definition")
	}
	sel, err := parseSelectBody(view.Definition)
	if err != nil {
		return nil, errors.Wrapf(err, "view %q body", view.Name)
	}
	return &ast.ViewStmt{
		View: &ast.RangeVar{
			Schemaname:     schema,
			Relname:        view.Name,
			Relpersistence: 'p',
		},
		Query: sel,
	}, nil
}

// buildCreateTableAsStmt translates MaterializedViewMetadata into a
// CreateTableAsStmt with Objtype OBJECT_MATVIEW. Same SelectStmt contract
// as buildViewStmt: body must parse to *ast.SelectStmt.
func buildCreateTableAsStmt(schema string, mv *storepb.MaterializedViewMetadata) (*ast.CreateTableAsStmt, error) {
	if mv.Definition == "" {
		return nil, errors.New("empty matview definition")
	}
	sel, err := parseSelectBody(mv.Definition)
	if err != nil {
		return nil, errors.Wrapf(err, "matview %q body", mv.Name)
	}
	return &ast.CreateTableAsStmt{
		Query:   sel,
		Objtype: ast.OBJECT_MATVIEW,
		Into: &ast.IntoClause{
			Rel: &ast.RangeVar{
				Schemaname:     schema,
				Relname:        mv.Name,
				Relpersistence: 'p',
			},
		},
	}, nil
}

// buildCreateFunctionStmt translates FunctionMetadata into a
// *ast.CreateFunctionStmt. The primary path parses the function's full
// Definition (an `omni`-compatible CREATE FUNCTION/PROCEDURE statement),
// which preserves parameter types, return-type signatures (including
// RETURNS TABLE / RETURNS SETOF), and the dollar-quoted body verbatim.
//
// When the definition string is missing or cannot be parsed as a
// CreateFunctionStmt, we fall back to a minimal text-typed shape
// reconstructed from the signature string. The fallback loses return-type
// fidelity but keeps analyzer name resolution working for queries that
// reference the function by name.
func buildCreateFunctionStmt(schema string, fn *storepb.FunctionMetadata) (*ast.CreateFunctionStmt, error) {
	if fn.Definition != "" {
		stmts, err := ParsePg(fn.Definition)
		if err == nil && len(stmts) == 1 {
			node := stmts[0].AST
			if raw, ok := node.(*ast.RawStmt); ok {
				node = raw.Stmt
			}
			if parsed, ok := node.(*ast.CreateFunctionStmt); ok {
				return parsed, nil
			}
		}
	}
	return buildCreateFunctionStmtFromSignature(schema, fn)
}

// buildCreateFunctionStmtFromSignature is the minimal-shape fallback used
// when the full definition is missing or unparseable. Parameters and return
// type collapse to text; the body is a trivial `SELECT NULL::text`.
func buildCreateFunctionStmtFromSignature(schema string, fn *storepb.FunctionMetadata) (*ast.CreateFunctionStmt, error) {
	argTypes, err := parseFunctionSignatureArgTypes(fn.Signature)
	if err != nil {
		return nil, errors.Wrapf(err, "function %q signature %q", fn.Name, fn.Signature)
	}
	params := make([]ast.Node, 0, len(argTypes))
	for i, argType := range argTypes {
		tn, err := typeNameFromString(argType)
		if err != nil {
			return nil, errors.Wrapf(err, "function %q arg %d", fn.Name, i)
		}
		params = append(params, &ast.FunctionParameter{
			ArgType: tn,
			Mode:    ast.FUNC_PARAM_IN,
		})
	}
	stmt := &ast.CreateFunctionStmt{
		Funcname:   qualifiedNameList(schema, fn.Name),
		ReturnType: pseudoTextTypeName(),
	}
	if len(params) > 0 {
		stmt.Parameters = &ast.List{Items: params}
	}
	stmt.Options = &ast.List{Items: []ast.Node{
		&ast.DefElem{Defname: "language", Arg: &ast.String{Str: "sql"}},
		&ast.DefElem{
			Defname: "as",
			Arg:     &ast.List{Items: []ast.Node{&ast.String{Str: "SELECT NULL::text"}}},
		},
	}}
	return stmt, nil
}

// parseSelectBody parses a SQL string expected to be a single SELECT
// statement and returns the *ast.SelectStmt. Rejects empty input, multiple
// statements, and non-SELECT top-level nodes. Unwraps a RawStmt wrapper if
// ParsePg returns one.
func parseSelectBody(sql string) (*ast.SelectStmt, error) {
	stmts, err := ParsePg(sql)
	if err != nil {
		return nil, errors.Wrap(err, "parse")
	}
	if len(stmts) != 1 {
		return nil, errors.Errorf("expected 1 statement, got %d", len(stmts))
	}
	node := stmts[0].AST
	if raw, ok := node.(*ast.RawStmt); ok {
		node = raw.Stmt
	}
	sel, ok := node.(*ast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("expected SelectStmt, got %T", node)
	}
	return sel, nil
}

// parseFunctionSignatureArgTypes extracts the comma-separated argument types
// from a signature like "fn(integer, text)" or "fn(numeric(10,2), boolean)".
// An empty parameter list returns an empty slice.
func parseFunctionSignatureArgTypes(signature string) ([]string, error) {
	open := strings.Index(signature, "(")
	if open < 0 {
		return nil, nil
	}
	closeIdx := strings.LastIndex(signature, ")")
	if closeIdx < 0 || closeIdx <= open {
		return nil, errors.Errorf("signature %q: unbalanced parentheses", signature)
	}
	return splitTopLevelCommas(signature[open+1 : closeIdx]), nil
}

// splitTopLevelCommas splits a string on commas that are not nested inside
// parentheses. Used for function signature argument lists so that types like
// `numeric(10,2)` are not split.
func splitTopLevelCommas(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		default:
			// other bytes are part of the current segment
		}
	}
	parts = append(parts, strings.TrimSpace(s[start:]))
	return parts
}
