package pg

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
)

// UserTypeRef names a user-defined type referenced by a column, function
// parameter, or function return type. Extracted from type strings produced by
// the PG sync layer (sync.go:820-872).
type UserTypeRef struct {
	Schema string
	Name   string
}

// typeNameFromString parses a PG type string into an *ast.TypeName by running
// it through omni's SELECT parser via the "SELECT NULL::<type>" trick. This
// reuses the stable SELECT parse path and avoids the DDL parse path that
// breaks on BYT-9215 / BYT-9261 class inputs.
//
// Callers must only pass values originating from bytebase sync metadata. The
// guardrails (single-statement, single-target) reject most malformed inputs
// but are not a sanitization layer for arbitrary user input.
func typeNameFromString(typeStr string) (*ast.TypeName, error) {
	stmts, err := ParsePg("SELECT NULL::" + typeStr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse type %q", typeStr)
	}
	if len(stmts) != 1 {
		return nil, errors.Errorf("type %q: expected 1 statement, got %d", typeStr, len(stmts))
	}
	sel, ok := stmts[0].AST.(*ast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("type %q: expected SelectStmt, got %T", typeStr, stmts[0].AST)
	}
	if sel.TargetList == nil || len(sel.TargetList.Items) != 1 {
		return nil, errors.Errorf("type %q: expected 1 target", typeStr)
	}
	rt, ok := sel.TargetList.Items[0].(*ast.ResTarget)
	if !ok {
		return nil, errors.Errorf("type %q: expected ResTarget, got %T", typeStr, sel.TargetList.Items[0])
	}
	cast, ok := rt.Val.(*ast.TypeCast)
	if !ok {
		return nil, errors.Errorf("type %q: expected TypeCast, got %T", typeStr, rt.Val)
	}
	return cast.TypeName, nil
}

// extractUserTypeRefs returns the user-defined type references embedded in a
// PG type string. Built-in types, PG internal array forms (_name), and
// system-schema-qualified types return nil.
//
// Soundness rule (hard contract C5): false negatives are acceptable (the
// loader's pseudo fallback catches them); false positives are not, because
// they invent edges in the dependency graph.
func extractUserTypeRefs(typeStr string) []UserTypeRef {
	if typeStr == "" {
		return nil
	}
	base := stripTypeModifiers(typeStr)
	if base == "" || isBuiltinType(base) {
		return nil
	}
	if strings.HasPrefix(base, "_") {
		// PG internal array form. The element may be a user type but the
		// schema is not recoverable from this representation — see the
		// sync.go:834 note in the plan.
		return nil
	}
	schema, name, ok := splitQualifiedName(base)
	if !ok {
		return nil
	}
	if IsSystemSchema(schema) {
		return nil
	}
	return []UserTypeRef{{Schema: schema, Name: name}}
}

// stripTypeModifiers normalizes a PG type string to its bare base-type form
// for allow-list lookup. It strips:
//   - parenthesized type modifiers: "numeric(10,2)" -> "numeric"
//   - trailing time-zone suffixes: "timestamp(3) with time zone" -> "timestamp"
//   - surrounding whitespace
//
// The result may still be a qualified name like "public.task_status"; that is
// the caller's job to split.
func stripTypeModifiers(typeStr string) string {
	s := strings.TrimSpace(typeStr)
	s = stripTimeZoneSuffix(s)
	s = stripParens(s)
	return strings.TrimSpace(s)
}

// stripTimeZoneSuffix removes a trailing "with time zone" / "without time
// zone" clause. Idempotent on inputs that do not carry the suffix.
func stripTimeZoneSuffix(s string) string {
	lower := strings.ToLower(s)
	for _, suffix := range []string{" with time zone", " without time zone"} {
		if strings.HasSuffix(lower, suffix) {
			return s[:len(s)-len(suffix)]
		}
	}
	return s
}

// stripParens removes the first parenthesized chunk from a type string.
// "numeric(10,2)" -> "numeric"
// "timestamp(3)" -> "timestamp"
// "character varying(255)" -> "character varying"
func stripParens(s string) string {
	before, _, found := strings.Cut(s, "(")
	if !found {
		return s
	}
	return strings.TrimSpace(before)
}

// splitQualifiedName splits a dotted type name into (schema, name). Quoted
// identifiers are supported.
// "public.task_status"     -> ("public", "task_status", true)
// "myschema.my_domain"     -> ("myschema", "my_domain", true)
// `"MySchema"."MyType"`    -> ("MySchema", "MyType", true)
// "task_status"            -> ("", "", false)
func splitQualifiedName(s string) (schema, name string, ok bool) {
	parts, err := splitDottedIdentifier(s)
	if err != nil || len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// splitDottedIdentifier splits "a.b.c" into ["a", "b", "c"], handling
// double-quoted identifiers like `"weird name"."another"`.
func splitDottedIdentifier(s string) ([]string, error) {
	var parts []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inQuote = !inQuote
		case c == '.' && !inQuote:
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if inQuote {
		return nil, errors.Errorf("unbalanced quote in %q", s)
	}
	parts = append(parts, cur.String())
	return parts, nil
}

// isBuiltinType reports whether a bare (modifier-stripped) type name is a
// PostgreSQL built-in type. Membership is exact, case-insensitive.
//
// The list is hand-maintained. New entries must come with a golden test in
// query_span_e3_test.go.
func isBuiltinType(s string) bool {
	_, ok := pgBuiltinTypes[strings.ToLower(s)]
	return ok
}

// pgBuiltinTypes is the allow-list of PostgreSQL built-in type base names
// (without type modifiers, time-zone suffixes, or array brackets).
//
// Sources: PostgreSQL data types reference + the transforms in
// backend/plugin/db/pg/sync.go:820-872 for the specific surface bytebase
// currently emits.
var pgBuiltinTypes = map[string]struct{}{
	// Numeric.
	"smallint":         {},
	"integer":          {},
	"int":              {},
	"int2":             {},
	"int4":             {},
	"int8":             {},
	"bigint":           {},
	"decimal":          {},
	"numeric":          {},
	"real":             {},
	"double precision": {},
	"float4":           {},
	"float8":           {},
	"smallserial":      {},
	"serial":           {},
	"bigserial":        {},
	"money":            {},

	// Character / binary.
	"character":         {},
	"character varying": {},
	"char":              {},
	"varchar":           {},
	"text":              {},
	"bytea":             {},
	"bpchar":            {},
	"name":              {},

	// Bit.
	"bit":         {},
	"bit varying": {},
	"varbit":      {},

	// Date / time.
	"date":        {},
	"time":        {},
	"timetz":      {},
	"timestamp":   {},
	"timestamptz": {},
	"interval":    {},

	// Boolean and friends.
	"boolean": {},
	"bool":    {},

	// Structured.
	"json":  {},
	"jsonb": {},
	"xml":   {},
	"uuid":  {},

	// Geometric.
	"point":   {},
	"line":    {},
	"lseg":    {},
	"box":     {},
	"path":    {},
	"polygon": {},
	"circle":  {},

	// Network.
	"cidr":     {},
	"inet":     {},
	"macaddr":  {},
	"macaddr8": {},

	// Text search.
	"tsvector": {},
	"tsquery":  {},

	// PG pseudo / system.
	"void":             {},
	"record":           {},
	"anyelement":       {},
	"anyarray":         {},
	"anynonarray":      {},
	"anyenum":          {},
	"anyrange":         {},
	"any":              {},
	"trigger":          {},
	"event_trigger":    {},
	"cstring":          {},
	"internal":         {},
	"language_handler": {},
	"fdw_handler":      {},
	"index_am_handler": {},
	"tsm_handler":      {},
	"pg_lsn":           {},
	"oid":              {},
	"regclass":         {},
	"regproc":          {},
	"regprocedure":     {},
	"regoper":          {},
	"regoperator":      {},
	"regtype":          {},
	"regconfig":        {},
	"regdictionary":    {},
	"regnamespace":     {},
	"regrole":          {},
}
