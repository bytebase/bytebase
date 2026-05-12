package tidb

// Pure-function unit tests for omniBuildColumnTypeString +
// normalizeColumnType. Each case locks one of the empirical
// receipts from a Codex review round on PR #20302 so future
// changes to the type-string rendering path have a CI guard
// against re-introducing the same class of bug.
//
// These tests bypass the advisor / catalog framework entirely —
// they exercise the pure (DataType, alias) → string transforms
// directly. No shared-mock-catalog cross-engine impact concern.

import (
	"strings"
	"testing"

	"github.com/bytebase/omni/tidb/ast"
)

// TestOmniBuildCompactTypeString locks the CompactStr-equivalent
// rendering used by column_type_disallow_list. Codex round-1 catch on
// PR #20308 (Codex P1) flagged that the earlier omniDataTypeNameCompact
// rendering dropped length/literal/canonicalization information that
// pingcap's `column.Tp.CompactStr()` preserved — breaking user
// blocklist entries like "VARCHAR(255)" / "TINYINT(1)" / "ENUM('X','Y')".
// These tests pin the CompactStr-equivalent contract.
func TestOmniBuildCompactTypeString(t *testing.T) {
	cases := []struct {
		name string
		dt   *ast.DataType
		want string
	}{
		// Bare-name types: pingcap CompactStr also renders bare.
		{"JSON", &ast.DataType{Name: "JSON"}, "json"},
		{"BLOB", &ast.DataType{Name: "BLOB"}, "blob"},
		{"DATE", &ast.DataType{Name: "DATE"}, "date"},
		// Length-bearing types: preserve length in CompactStr form.
		{"VARCHAR(255)", &ast.DataType{Name: "VARCHAR", Length: 255}, "varchar(255)"},
		{"CHAR(10)", &ast.DataType{Name: "CHAR", Length: 10}, "char(10)"},
		{"BINARY(16)", &ast.DataType{Name: "BINARY", Length: 16}, "binary(16)"},
		// Bare-form integer: canonical default width applied.
		{"INT bare → int(11)", &ast.DataType{Name: "INT"}, "int(11)"},
		{"TINYINT bare → tinyint(4)", &ast.DataType{Name: "TINYINT"}, "tinyint(4)"},
		{"SMALLINT bare → smallint(6)", &ast.DataType{Name: "SMALLINT"}, "smallint(6)"},
		{"MEDIUMINT bare → mediumint(9)", &ast.DataType{Name: "MEDIUMINT"}, "mediumint(9)"},
		{"BIGINT bare → bigint(20)", &ast.DataType{Name: "BIGINT"}, "bigint(20)"},
		{"BIT bare → bit(1)", &ast.DataType{Name: "BIT"}, "bit(1)"},
		{"BINARY bare → binary(1)", &ast.DataType{Name: "BINARY"}, "binary(1)"},
		{"YEAR bare → year(4)", &ast.DataType{Name: "YEAR"}, "year(4)"},
		// Specified integer length: preserved.
		{"TINYINT(1)", &ast.DataType{Name: "TINYINT", Length: 1}, "tinyint(1)"},
		{"INT(11)", &ast.DataType{Name: "INT", Length: 11}, "int(11)"},
		// Exact-decimal: canonicalize to decimal(M,D).
		{"DECIMAL bare → decimal(10,0)", &ast.DataType{Name: "DECIMAL"}, "decimal(10,0)"},
		{"DECIMAL(10)", &ast.DataType{Name: "DECIMAL", Length: 10}, "decimal(10,0)"},
		{"DECIMAL(10,2)", &ast.DataType{Name: "DECIMAL", Length: 10, Scale: 2}, "decimal(10,2)"},
		{"NUMERIC(10,2) → decimal", &ast.DataType{Name: "NUMERIC", Length: 10, Scale: 2}, "decimal(10,2)"},
		// Approximate float: drop precision hint when Scale=0.
		{"FLOAT bare", &ast.DataType{Name: "FLOAT"}, "float"},
		{"FLOAT(10) → float", &ast.DataType{Name: "FLOAT", Length: 10}, "float"},
		{"FLOAT(10,2)", &ast.DataType{Name: "FLOAT", Length: 10, Scale: 2}, "float(10,2)"},
		// ENUM/SET: value list as body.
		{"ENUM('x','y')", &ast.DataType{Name: "ENUM", EnumValues: []string{"x", "y"}}, "enum('x','y')"},
		{"SET('a','b')", &ast.DataType{Name: "SET", EnumValues: []string{"a", "b"}}, "set('a','b')"},
		// BOOLEAN canonicalizes to tinyint(1).
		{"BOOLEAN → tinyint(1)", &ast.DataType{Name: "BOOLEAN"}, "tinyint(1)"},
		// Crucially: CompactStr does NOT include UNSIGNED/ZEROFILL
		// modifiers, even when set on the DataType. The wrapper
		// omniBuildColumnTypeString appends those; the compact form
		// stays modifier-free (matches pingcap's CompactStr behavior:
		// `BIGINT UNSIGNED` → CompactStr "bigint(20)" — no UNSIGNED).
		{"BIGINT UNSIGNED → compact has no UNSIGNED", &ast.DataType{Name: "BIGINT", Unsigned: true}, "bigint(20)"},
		{"INT(11) UNSIGNED → compact has no UNSIGNED", &ast.DataType{Name: "INT", Length: 11, Unsigned: true}, "int(11)"},
		{"INT ZEROFILL → compact has no ZEROFILL", &ast.DataType{Name: "INT", Zerofill: true}, "int(11)"},
		// nil safety.
		{"nil", nil, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lower := ""
			if tc.dt != nil {
				lower = strings.ToLower(tc.dt.Name)
			}
			got := omniBuildCompactTypeString(tc.dt, lower)
			if got != tc.want {
				t.Errorf("omniBuildCompactTypeString = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOmniBuildColumnTypeString(t *testing.T) {
	cases := []struct {
		name string
		dt   *ast.DataType
		want string
	}{
		// Round 1 / cumulative #21: BLOB-family renders without
		// pingcap's accidental " BINARY" charset suffix.
		{"BLOB", &ast.DataType{Name: "BLOB"}, "blob"},
		{"TINYBLOB", &ast.DataType{Name: "TINYBLOB"}, "tinyblob"},
		{"MEDIUMBLOB", &ast.DataType{Name: "MEDIUMBLOB"}, "mediumblob"},
		{"LONGBLOB", &ast.DataType{Name: "LONGBLOB"}, "longblob"},
		{"TEXT", &ast.DataType{Name: "TEXT"}, "text"},
		// Round 2: exact-decimal types always render (M,D), Scale=0
		// when not explicitly given.
		{"DECIMAL(10)", &ast.DataType{Name: "DECIMAL", Length: 10}, "decimal(10,0)"},
		{"DECIMAL(10,0)", &ast.DataType{Name: "DECIMAL", Length: 10, Scale: 0}, "decimal(10,0)"},
		{"DECIMAL(10,2)", &ast.DataType{Name: "DECIMAL", Length: 10, Scale: 2}, "decimal(10,2)"},
		// Round 9: NUMERIC/FIXED aliases canonicalize to "decimal" in
		// the rendered form. Omni pre-normalizes FIXED→DECIMAL but NOT
		// NUMERIC→DECIMAL, so the builder canonicalizes for both.
		{"NUMERIC(8)", &ast.DataType{Name: "NUMERIC", Length: 8}, "decimal(8,0)"},
		{"NUMERIC(8,2)", &ast.DataType{Name: "NUMERIC", Length: 8, Scale: 2}, "decimal(8,2)"},
		{"FIXED(8,2)-defensive", &ast.DataType{Name: "FIXED", Length: 8, Scale: 2}, "decimal(8,2)"},
		{"NUMERIC(10,2)-UNSIGNED", &ast.DataType{Name: "NUMERIC", Length: 10, Scale: 2, Unsigned: true}, "decimal(10,2) unsigned"},
		// Round 2: ZEROFILL implies UNSIGNED + appends zerofill.
		// Note: round-8 fix updated bare INT ZEROFILL from "int unsigned zerofill"
		// (the original round-2 output, which mismatched catalog) to the
		// canonical "int(11) unsigned zerofill" form that catalog stores.
		{"INT(11)-ZEROFILL", &ast.DataType{Name: "INT", Length: 11, Zerofill: true}, "int(11) unsigned zerofill"},
		// Round 3: FLOAT/DOUBLE/REAL drop precision hint when no
		// explicit scale; render (M,D) only when Scale > 0.
		{"FLOAT", &ast.DataType{Name: "FLOAT"}, "float"},
		{"FLOAT(10)", &ast.DataType{Name: "FLOAT", Length: 10}, "float"},
		{"FLOAT(10,0)", &ast.DataType{Name: "FLOAT", Length: 10, Scale: 0}, "float"},
		{"FLOAT(10,2)", &ast.DataType{Name: "FLOAT", Length: 10, Scale: 2}, "float(10,2)"},
		{"DOUBLE(10,4)", &ast.DataType{Name: "DOUBLE", Length: 10, Scale: 4}, "double(10,4)"},
		{"REAL", &ast.DataType{Name: "REAL"}, "real"},
		// Round 4: ENUM/SET render their value list with SQL
		// single-quote escaping; length/scale/unsigned attributes
		// don't apply.
		{"ENUM-2", &ast.DataType{Name: "ENUM", EnumValues: []string{"a", "b"}}, "enum('a','b')"},
		{"ENUM-3", &ast.DataType{Name: "ENUM", EnumValues: []string{"x", "y", "z"}}, "enum('x','y','z')"},
		{"SET-3", &ast.DataType{Name: "SET", EnumValues: []string{"a", "b", "c"}}, "set('a','b','c')"},
		{"ENUM-escape", &ast.DataType{Name: "ENUM", EnumValues: []string{"with'quote"}}, "enum('with''quote')"},
		// Rounds 5, 6, 7, 8 — the builder now applies MySQL canonical
		// default forms for type families whose info_schema rendering
		// carries explicit length/precision. Previously these were
		// emitted bare and relied on normalizeColumnType; now the
		// builder produces the canonical form directly so that the
		// bare-form × modifier (UNSIGNED/ZEROFILL) Cartesian product
		// composes without combinatorial normalize-map entries.
		{"BOOLEAN", &ast.DataType{Name: "BOOLEAN"}, "tinyint(1)"},
		{"DECIMAL-bare", &ast.DataType{Name: "DECIMAL"}, "decimal(10,0)"},
		{"NUMERIC-bare", &ast.DataType{Name: "NUMERIC"}, "decimal(10,0)"},
		{"BIT-bare", &ast.DataType{Name: "BIT"}, "bit(1)"},
		{"BINARY-bare", &ast.DataType{Name: "BINARY"}, "binary(1)"},
		{"YEAR-bare", &ast.DataType{Name: "YEAR"}, "year(4)"},
		{"INT-bare", &ast.DataType{Name: "INT"}, "int(11)"},
		{"TINYINT-bare", &ast.DataType{Name: "TINYINT"}, "tinyint(4)"},
		{"BIGINT-bare", &ast.DataType{Name: "BIGINT"}, "bigint(20)"},
		// Round 8 / peer-review-prompted: bare-form × modifier
		// combinations. Pre-fix these emitted "decimal unsigned" /
		// "int unsigned zerofill" (bare base + suffix); now emit
		// the canonical-base + suffix forms that match what pingcap
		// Tp.String() and info_schema render.
		{"DECIMAL-bare-UNSIGNED", &ast.DataType{Name: "DECIMAL", Unsigned: true}, "decimal(10,0) unsigned"},
		{"DECIMAL-bare-ZEROFILL", &ast.DataType{Name: "DECIMAL", Zerofill: true}, "decimal(10,0) unsigned zerofill"},
		{"DECIMAL-bare-UNSIGNED-ZEROFILL", &ast.DataType{Name: "DECIMAL", Unsigned: true, Zerofill: true}, "decimal(10,0) unsigned zerofill"},
		{"NUMERIC-bare-UNSIGNED", &ast.DataType{Name: "NUMERIC", Unsigned: true}, "decimal(10,0) unsigned"},
		{"INT-bare-ZEROFILL", &ast.DataType{Name: "INT", Zerofill: true}, "int(11) unsigned zerofill"},
		{"INT-bare-UNSIGNED-ZEROFILL", &ast.DataType{Name: "INT", Unsigned: true, Zerofill: true}, "int(11) unsigned zerofill"},
		{"TINYINT-bare-ZEROFILL", &ast.DataType{Name: "TINYINT", Zerofill: true}, "tinyint(4) unsigned zerofill"},
		{"SMALLINT-bare-ZEROFILL", &ast.DataType{Name: "SMALLINT", Zerofill: true}, "smallint(6) unsigned zerofill"},
		{"MEDIUMINT-bare-ZEROFILL", &ast.DataType{Name: "MEDIUMINT", Zerofill: true}, "mediumint(9) unsigned zerofill"},
		{"BIGINT-bare-ZEROFILL", &ast.DataType{Name: "BIGINT", Zerofill: true}, "bigint(20) unsigned zerofill"},
		{"INT-bare-UNSIGNED", &ast.DataType{Name: "INT", Unsigned: true}, "int(11) unsigned"},
		{"BIGINT-bare-UNSIGNED", &ast.DataType{Name: "BIGINT", Unsigned: true}, "bigint(20) unsigned"},
		// FLOAT/DOUBLE/REAL with bare-form + modifier: float-family
		// does NOT apply default precision, so bare → "float unsigned"
		// (matches pingcap's precision-hint-drop behavior even with
		// modifiers).
		{"FLOAT-bare-UNSIGNED", &ast.DataType{Name: "FLOAT", Unsigned: true}, "float unsigned"},
		{"DOUBLE-bare-UNSIGNED", &ast.DataType{Name: "DOUBLE", Unsigned: true}, "double unsigned"},
		// Regular cases that should "just work".
		{"VARCHAR(255)", &ast.DataType{Name: "VARCHAR", Length: 255}, "varchar(255)"},
		{"CHAR(10)", &ast.DataType{Name: "CHAR", Length: 10}, "char(10)"},
		{"TIMESTAMP(3)", &ast.DataType{Name: "TIMESTAMP", Length: 3}, "timestamp(3)"},
		{"DATETIME", &ast.DataType{Name: "DATETIME"}, "datetime"},
		{"JSON", &ast.DataType{Name: "JSON"}, "json"},
		{"INT(11)-UNSIGNED", &ast.DataType{Name: "INT", Length: 11, Unsigned: true}, "int(11) unsigned"},
		{"nil", nil, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := omniBuildColumnTypeString(tc.dt)
			if got != tc.want {
				t.Errorf("omniBuildColumnTypeString = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeColumnType(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		// Round 5: BOOLEAN ↔ tinyint(1) canonicalization (MySQL
		// stores boolean columns as tinyint(1)).
		{"BOOLEAN", "boolean", "tinyint(1)"},
		{"BOOL", "bool", "tinyint(1)"},
		{"tinyint(1) identity", "tinyint(1)", "tinyint(1)"},
		// Round 6: bare DECIMAL/NUMERIC/FIXED → decimal(10,0)
		// canonicalization (MySQL default precision/scale).
		{"bare decimal", "decimal", "decimal(10,0)"},
		{"bare numeric", "numeric", "decimal(10,0)"},
		{"bare fixed", "fixed", "decimal(10,0)"},
		// Round 7: bare BIT/BINARY/YEAR → default-form
		// canonicalization (MySQL defaults match info_schema).
		{"bare bit", "bit", "bit(1)"},
		{"bare binary", "binary", "binary(1)"},
		{"bare year", "year", "year(4)"},
		// Pre-existing: integer default display widths.
		{"bare tinyint", "tinyint", "tinyint(4)"},
		{"bare smallint", "smallint", "smallint(6)"},
		{"bare mediumint", "mediumint", "mediumint(9)"},
		{"bare int", "int", "int(11)"},
		{"bare bigint", "bigint", "bigint(20)"},
		{"int unsigned", "int unsigned", "int(11) unsigned"},
		{"bigint unsigned", "bigint unsigned", "bigint(20) unsigned"},
		// Default branch: identity-lowercase.
		{"varchar(255) identity", "varchar(255)", "varchar(255)"},
		{"already-canonical decimal", "decimal(10,2)", "decimal(10,2)"},
		{"empty string", "", ""},
		{"uppercase input", "VARCHAR(255)", "varchar(255)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeColumnType(tc.in)
			if got != tc.want {
				t.Errorf("normalizeColumnType(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestTypeStringRoundTripAgainstNormalize asserts that for the
// no-op type-change scenarios across all six Codex rounds + the
// round-7 risk check, normalizeColumnType applied to (a) the
// catalog-side rendering and (b) the omni-built rendering produces
// matching strings. This is the load-bearing invariant for the
// columnTypeChanged comparison — if the two normalize calls ever
// produce different strings for the same logical column type, the
// rule will false-positive.
func TestTypeStringRoundTripAgainstNormalize(t *testing.T) {
	cases := []struct {
		name           string
		omniDT         *ast.DataType
		catalogStored  string // what MySQL info_schema returns
		shouldNoChange bool   // true when this is a no-op-modify scenario
	}{
		// All six Codex rounds + round-7 fixes — the load-bearing
		// invariant: normalize(omni-built) == normalize(catalog).
		{"BLOB", &ast.DataType{Name: "BLOB"}, "blob", true},
		{"TINYBLOB", &ast.DataType{Name: "TINYBLOB"}, "tinyblob", true},
		{"VARBINARY(100)", &ast.DataType{Name: "VARBINARY", Length: 100}, "varbinary(100)", true},
		{"DECIMAL(10)", &ast.DataType{Name: "DECIMAL", Length: 10}, "decimal(10,0)", true},
		{"DECIMAL(10,0)", &ast.DataType{Name: "DECIMAL", Length: 10, Scale: 0}, "decimal(10,0)", true},
		{"FLOAT", &ast.DataType{Name: "FLOAT"}, "float", true},
		{"FLOAT(10)", &ast.DataType{Name: "FLOAT", Length: 10}, "float", true},
		{"FLOAT(10,2)", &ast.DataType{Name: "FLOAT", Length: 10, Scale: 2}, "float(10,2)", true},
		{"DOUBLE(10,4)", &ast.DataType{Name: "DOUBLE", Length: 10, Scale: 4}, "double(10,4)", true},
		{"ENUM-2", &ast.DataType{Name: "ENUM", EnumValues: []string{"a", "b"}}, "enum('a','b')", true},
		{"SET-3", &ast.DataType{Name: "SET", EnumValues: []string{"x", "y", "z"}}, "set('x','y','z')", true},
		{"BOOLEAN", &ast.DataType{Name: "BOOLEAN"}, "tinyint(1)", true},
		{"BOOL", &ast.DataType{Name: "BOOLEAN"}, "tinyint(1)", true},
		{"bare DECIMAL", &ast.DataType{Name: "DECIMAL"}, "decimal(10,0)", true},
		{"bare NUMERIC", &ast.DataType{Name: "NUMERIC"}, "decimal(10,0)", true},
		{"bare BIT", &ast.DataType{Name: "BIT"}, "bit(1)", true},
		{"bare BINARY", &ast.DataType{Name: "BINARY"}, "binary(1)", true},
		{"bare YEAR", &ast.DataType{Name: "YEAR"}, "year(4)", true},
		// Regular pass-through cases.
		{"bare INT", &ast.DataType{Name: "INT"}, "int(11)", true},
		{"INT(11)", &ast.DataType{Name: "INT", Length: 11}, "int(11)", true},
		{"bare TINYINT", &ast.DataType{Name: "TINYINT"}, "tinyint(4)", true},
		{"VARCHAR(255)", &ast.DataType{Name: "VARCHAR", Length: 255}, "varchar(255)", true},
		// Round 8: bare-form × modifier no-op-modify scenarios.
		// Catalog stores the canonical form (default precision +
		// UNSIGNED [+ ZEROFILL]); builder now produces matching form.
		{"DECIMAL UNSIGNED no-op", &ast.DataType{Name: "DECIMAL", Unsigned: true}, "decimal(10,0) unsigned", true},
		// Round 9: NUMERIC alias no-op against catalog stored as decimal.
		{"NUMERIC(8,2) no-op (catalog decimal)", &ast.DataType{Name: "NUMERIC", Length: 8, Scale: 2}, "decimal(8,2)", true},
		{"NUMERIC bare no-op (catalog decimal)", &ast.DataType{Name: "NUMERIC"}, "decimal(10,0)", true},
		{"DECIMAL ZEROFILL no-op", &ast.DataType{Name: "DECIMAL", Zerofill: true}, "decimal(10,0) unsigned zerofill", true},
		{"INT ZEROFILL no-op", &ast.DataType{Name: "INT", Zerofill: true}, "int(11) unsigned zerofill", true},
		{"TINYINT ZEROFILL no-op", &ast.DataType{Name: "TINYINT", Zerofill: true}, "tinyint(4) unsigned zerofill", true},
		{"INT UNSIGNED no-op", &ast.DataType{Name: "INT", Unsigned: true}, "int(11) unsigned", true},
		{"BIGINT UNSIGNED no-op", &ast.DataType{Name: "BIGINT", Unsigned: true}, "bigint(20) unsigned", true},
		// Negative case: actual type change. Normalize should NOT
		// produce matching strings.
		{"INT → BIGINT", &ast.DataType{Name: "BIGINT"}, "int(11)", false},
		{"VARCHAR(50) → VARCHAR(255)", &ast.DataType{Name: "VARCHAR", Length: 255}, "varchar(50)", false},
		{"DECIMAL(10,2) → DECIMAL(10,4)", &ast.DataType{Name: "DECIMAL", Length: 10, Scale: 4}, "decimal(10,2)", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			omniBuilt := omniBuildColumnTypeString(tc.omniDT)
			omniNorm := normalizeColumnType(omniBuilt)
			catalogNorm := normalizeColumnType(tc.catalogStored)
			matched := omniNorm == catalogNorm
			if matched != tc.shouldNoChange {
				t.Errorf("normalize match = %v, want %v (omni-built=%q normalized=%q vs catalog=%q normalized=%q)",
					matched, tc.shouldNoChange, omniBuilt, omniNorm, tc.catalogStored, catalogNorm)
			}
		})
	}
}
