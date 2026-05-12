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
	"testing"

	"github.com/bytebase/omni/tidb/ast"
)

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
		{"NUMERIC(8)", &ast.DataType{Name: "NUMERIC", Length: 8}, "numeric(8,0)"},
		// Round 2: ZEROFILL implies UNSIGNED + appends zerofill.
		{"INT-ZEROFILL", &ast.DataType{Name: "INT", Zerofill: true}, "int unsigned zerofill"},
		{"INT-UNSIGNED-ZEROFILL", &ast.DataType{Name: "INT", Unsigned: true, Zerofill: true}, "int unsigned zerofill"},
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
		// Round 5 / 6 are normalizeColumnType concerns — tested in
		// TestNormalizeColumnType below. The builder for BOOLEAN
		// and bare DECIMAL emits the user-typed form; normalize
		// canonicalizes.
		{"BOOLEAN-builder", &ast.DataType{Name: "BOOLEAN"}, "boolean"},
		{"DECIMAL-bare-builder", &ast.DataType{Name: "DECIMAL"}, "decimal"},
		// Round-7 risk check (BIT / BINARY / YEAR bare-form
		// defaults). Builder emits bare; normalize canonicalizes.
		{"BIT-bare-builder", &ast.DataType{Name: "BIT"}, "bit"},
		{"BINARY-bare-builder", &ast.DataType{Name: "BINARY"}, "binary"},
		{"YEAR-bare-builder", &ast.DataType{Name: "YEAR"}, "year"},
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
