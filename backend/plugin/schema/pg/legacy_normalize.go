// This file repairs historical PostgreSQL metadata that violates the canonical
// shape documented by IndexMetadata.expressions in
// proto/store/store/database.proto.
//
// Canonical shape (matches pg_get_indexdef(oid, col, true) per-column output):
//   - column key:        bare identifier,            e.g. "id", `"Name"`
//   - function-call key: bare func_expr_windowless,  e.g. "lower(name)"
//   - expression key:    parenthesized a_expr,       e.g. "(payload ->> 'k'::text)"
//
// That form is exactly PostgreSQL's `index_elem` grammar alternative, so the
// emitter can write entries verbatim into a CREATE INDEX key list.
//
// The BYT-9261 reproducer (demo metadata for the bytebase-meta DB) stores
// expression keys without the required outer parens — an older sync code path
// stripped them before persisting. The emitter writing those verbatim produces
// invalid SQL ("CREATE INDEX ... (payload ->> 'k')") which both PostgreSQL and
// the omni parser reject ("syntax error at or near \"->>\"").
//
// normalizeLegacyMetadata re-canonicalizes such rows on read so the emitter
// can rely on the contract.
//
// Removal criteria — this file can be deleted when:
//   1. A one-time migrator has rewritten db_schema.metadata rows to canonical
//      shape (straightforward: run the canonicalizer over each row's
//      IndexMetadata.Expressions and UPSERT); AND
//   2. at least one release containing the migrator has been out for 30+ days.
// Tracked by BYT-9261 follow-up.

package pg

import (
	"regexp"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// normalizeLegacyMetadata mutates meta in place to match the canonical
// IndexMetadata.expressions contract. Idempotent and cheap on canonical input.
func normalizeLegacyMetadata(meta *storepb.DatabaseSchemaMetadata) {
	if meta == nil {
		return
	}
	for _, s := range meta.GetSchemas() {
		for _, t := range s.GetTables() {
			canonicalizeIndexExpressions(t.GetIndexes())
			for _, p := range t.GetPartitions() {
				canonicalizePartitionIndexes(p)
			}
		}
		for _, mv := range s.GetMaterializedViews() {
			canonicalizeIndexExpressions(mv.GetIndexes())
		}
	}
}

func canonicalizePartitionIndexes(p *storepb.TablePartitionMetadata) {
	if p == nil {
		return
	}
	canonicalizeIndexExpressions(p.GetIndexes())
	for _, sub := range p.GetSubpartitions() {
		canonicalizePartitionIndexes(sub)
	}
}

func canonicalizeIndexExpressions(indexes []*storepb.IndexMetadata) {
	for _, idx := range indexes {
		for i, expr := range idx.Expressions {
			idx.Expressions[i] = canonicalizeIndexKeyExpression(expr)
		}
	}
}

// canonicalizeIndexKeyExpression repairs a single index key expression into
// the canonical pg_get_indexdef shape:
//
//   - bare column identifier → returned unchanged
//   - bare function call     → returned unchanged
//   - anything else          → wrapped in a single pair of '(' ')'
//
// Any redundant outer parens in the input are collapsed first, so
// "((payload ->> 'k'))" and "payload ->> 'k'" both produce "(payload ->> 'k')".
//
// Idempotent: canonical input round-trips unchanged.
func canonicalizeIndexKeyExpression(s string) string {
	s = strings.TrimSpace(s)
	// Collapse any fully-matched outer parens — covers both legacy stripped
	// expressions ("expr", no parens to strip) and over-wrapped ones
	// ("((expr))", strip twice).
	for {
		stripped, ok := stripMatchedOuterParens(s)
		if !ok {
			break
		}
		s = strings.TrimSpace(stripped)
	}
	if s == "" {
		return s
	}
	// Canonical bare forms are emitted as-is.
	if isBareColumnIdent(s) || isBareFunctionCall(s) {
		return s
	}
	// Everything else is an a_expr and must be parenthesized per PG's
	// index_elem grammar.
	return "(" + s + ")"
}

// stripMatchedOuterParens returns s without its outermost '(' ')' pair iff
// those parens enclose the entire expression. Single- and double-quoted
// strings are respected so parens inside literals don't confuse matching.
func stripMatchedOuterParens(s string) (string, bool) {
	if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
		return s, false
	}
	depth := 0
	inSingle := false
	inDouble := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inSingle:
			if c == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					i++
				} else {
					inSingle = false
				}
			}
		case inDouble:
			if c == '"' {
				if i+1 < len(s) && s[i+1] == '"' {
					i++
				} else {
					inDouble = false
				}
			}
		case c == '\'':
			inSingle = true
		case c == '"':
			inDouble = true
		case c == '(':
			depth++
		case c == ')':
			depth--
			if depth == 0 && i != len(s)-1 {
				// Opening '(' at index 0 closed before end of string —
				// outer parens don't enclose the whole expression.
				return s, false
			}
		default:
			// Any other character is ignored.
		}
	}
	if depth != 0 {
		return s, false
	}
	return s[1 : len(s)-1], true
}

// reBareColumnIdent matches an unquoted simple identifier. PostgreSQL allows
// Unicode letters and `$` (after the first character) in unquoted identifiers
// — e.g. `col$1`, `naïve`. We mirror that because otherwise such columns would
// be wrapped as expression keys and break PRIMARY KEY / UNIQUE constraint
// emission, which require bare column identifiers, not expression keys.
var reBareColumnIdent = regexp.MustCompile(`^[\p{L}_][\p{L}\p{N}_$]*$`)

// isBareColumnIdent reports whether s is a simple identifier — either unquoted
// (`name`) or double-quoted (`"Name"`, `"has ""quote"" inside"`).
func isBareColumnIdent(s string) bool {
	s = strings.TrimSpace(s)
	if reBareColumnIdent.MatchString(s) {
		return true
	}
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return false
	}
	inner := s[1 : len(s)-1]
	if inner == "" {
		return false
	}
	for i := 0; i < len(inner); i++ {
		if inner[i] != '"' {
			continue
		}
		if i+1 < len(inner) && inner[i+1] == '"' {
			i++
			continue
		}
		return false
	}
	return true
}

// isBareFunctionCall reports whether s is a bare function call — an identifier
// (optionally schema-qualified) directly followed by a parenthesized argument
// list that extends to and balances at the end of s. This is the shape
// pg_get_indexdef returns for function-call index keys like `lower(name)` or
// `tst.foo(a, b)`.
//
// Critically, the opening '(' after the identifier must match the LAST ')' in
// s. Otherwise a compound expression like `lower(name) + abs(score)` — which
// also starts with `ident(` and ends with `)` — would be misclassified as a
// bare call, and a legacy stripped-parens entry would be emitted unwrapped,
// producing invalid CREATE INDEX SQL.
func isBareFunctionCall(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || !isIdentStart(s[0]) {
		return false
	}
	i := scanIdent(s, 0)
	// Optional schema qualifier: ".ident"
	if i < len(s) && s[i] == '.' {
		if j := scanIdent(s, i+1); j > i+1 {
			i = j
		} else {
			return false
		}
	}
	// Optional whitespace between name and '('
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	if i >= len(s) || s[i] != '(' {
		return false
	}
	if !strings.HasSuffix(s, ")") {
		return false
	}
	// Verify the '(' at position i pairs with the ')' at len(s)-1, accounting
	// for nested parens and string literals.
	depth := 0
	inSingle := false
	inDouble := false
	for j := i; j < len(s); j++ {
		c := s[j]
		switch {
		case inSingle:
			if c == '\'' {
				if j+1 < len(s) && s[j+1] == '\'' {
					j++
				} else {
					inSingle = false
				}
			}
		case inDouble:
			if c == '"' {
				if j+1 < len(s) && s[j+1] == '"' {
					j++
				} else {
					inDouble = false
				}
			}
		case c == '\'':
			inSingle = true
		case c == '"':
			inDouble = true
		case c == '(':
			depth++
		case c == ')':
			depth--
			if depth == 0 {
				return j == len(s)-1
			}
		default:
			// Any other character is ignored.
		}
	}
	return false
}

func isIdentStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func isIdentCont(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9')
}

// scanIdent returns the index after the identifier starting at pos, or pos if
// there is no identifier at pos.
func scanIdent(s string, pos int) int {
	if pos >= len(s) || !isIdentStart(s[pos]) {
		return pos
	}
	i := pos + 1
	for i < len(s) && isIdentCont(s[i]) {
		i++
	}
	return i
}
