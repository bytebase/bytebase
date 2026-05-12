package tidb

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*ColumnDisallowChangingTypeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE, &ColumnDisallowChangingTypeAdvisor{})
}

// ColumnDisallowChangingTypeAdvisor is the advisor checking for disallow changing column type.
type ColumnDisallowChangingTypeAdvisor struct {
}

// Check checks for disallow changing column type.
func (*ColumnDisallowChangingTypeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	originalMetadata := checkCtx.OriginalMetadata

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		alter, ok := ostmt.Node.(*ast.AlterTableStmt)
		if !ok || alter.Table == nil {
			continue
		}
		tableName := alter.Table.Name
		changed := false
		for _, cmd := range alter.Commands {
			if cmd == nil {
				continue
			}
			var oldColumnName string
			switch cmd.Type {
			case ast.ATChangeColumn:
				// CHANGE COLUMN: cmd.Name is the OLD column name.
				oldColumnName = cmd.Name
			case ast.ATModifyColumn:
				// MODIFY COLUMN: column name comes from the column def.
				if cmd.Column != nil {
					oldColumnName = cmd.Column.Name
				}
			default:
				continue
			}
			if cmd.Column == nil || cmd.Column.TypeName == nil || oldColumnName == "" {
				continue
			}
			if columnTypeChanged(originalMetadata, tableName, oldColumnName, cmd.Column.TypeName) {
				changed = true
				break
			}
		}
		if changed {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.ChangeColumnType.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" changes column type", ostmt.TrimmedText()),
				StartPosition: common.ConvertANTLRLineToPosition(ostmt.AbsoluteLine(alter.Loc.Start)),
			})
		}
	}

	return adviceList, nil
}

// columnTypeChanged compares the new type (rendered from the omni
// DataType) against the catalog-stored type for the given column. Both
// sides are routed through normalizeColumnType to canonicalize the
// integer default-length forms (`int` ↔ `int(11)` etc.). Returns false
// if the column is not in the catalog (treated as "no change to detect").
//
// Note: pingcap-typed predecessor used `Tp.String()` which appended a
// ` BINARY` charset annotation on BLOB/TINYBLOB/VARBINARY etc., causing
// false-positive flags on no-op type changes (catalog stores the type
// without the charset suffix; pingcap rendered it with). Omni's
// DataType.Name is the bare type name with no charset suffix, so the
// omni port correctly compares "blob" against catalog "blob" — fixes
// the latent false-positive. See cumulative #21.
func columnTypeChanged(metadata *model.DatabaseMetadata, tableName, columnName string, dt *ast.DataType) bool {
	column := metadata.GetSchemaMetadata("").GetTable(tableName).GetColumn(columnName)
	if column == nil {
		return false
	}
	return normalizeColumnType(column.GetProto().Type) != normalizeColumnType(omniBuildColumnTypeString(dt))
}

// omniBuildColumnTypeString renders an omni DataType into the lowercase
// "name[(...)] [unsigned] [zerofill]" form that the normalizeColumnType
// helper + catalog comparison expects.
//
// Rendering rules per type family (each verified empirically against
// pingcap's Tp.String() during pre-batch protocol; cumulative #21):
//
//   - **ENUM / SET:** render the literal value list as
//     `enum('v1','v2',…)` / `set('v1','v2',…)`, single quotes inside
//     values doubled per SQL convention. Pingcap preserved the literals
//     in `Tp.String()`; catalog stores them via MySQL's COLUMN_TYPE.
//     A naive builder that dropped EnumValues would false-positive on
//     no-op `MODIFY status ENUM('a','b')` (Codex round-4 catch).
//
//   - **DECIMAL / NUMERIC / FIXED (exact-precision):** always (M,D)
//     when Length > 0; Scale defaults to 0. Pingcap and MySQL
//     info_schema both canonicalize `DECIMAL(10)` → `decimal(10,0)`.
//
//   - **FLOAT / DOUBLE / REAL (approximate):** (M,D) ONLY when Scale
//     was explicitly given (Scale > 0); otherwise render bare type
//     name. MySQL drops the precision hint for `FLOAT(10)` —
//     pingcap's `Tp.String()` returns `"float"` for `FLOAT(10)` and
//     `"float"` for `FLOAT(10,0)`. Treating these like DECIMAL caused
//     false-positives on no-op `MODIFY x FLOAT(10)` (Codex round-3
//     catch).
//
//   - **All other types** (VARCHAR / CHAR / BINARY / VARBINARY /
//     integer family / etc.): (Length) when Length > 0; no scale.
//
// Additional attributes:
//
//   - **ZEROFILL**: pingcap appended `" ZEROFILL"` (and implied
//     UNSIGNED) for zerofill columns. Render `" zerofill"` when
//     `Zerofill` is set; treat Zerofill as implying Unsigned in the
//     output. Caveat: MySQL canonicalizes ZEROFILL display widths to
//     maximums (e.g. `int(11)` → `int(10)`); the normalizeColumnType
//     map doesn't cover zerofill cases, so display-width-
//     canonicalization differences may still surface as false-
//     positives on rare ZEROFILL no-op modifies. Out of scope
//     (ZEROFILL is deprecated in MySQL 8.0.17+).
//
//   - **Charset / collation:** pingcap appended a `" BINARY"` charset
//     annotation on BLOB/TINYBLOB/VARBINARY; omni's DataType has no
//     charset suffix. Latent pingcap false-positive — silently fixed
//     by the migration (cumulative #21).
func omniBuildColumnTypeString(dt *ast.DataType) string {
	if dt == nil {
		return ""
	}
	lower := strings.ToLower(dt.Name)

	// Build the compact (no-modifier) form first; this is the
	// CompactStr-equivalent that maps directly to pingcap's
	// `column.Tp.CompactStr()` and to MySQL info_schema's
	// non-attribute column-type rendering. Modifier handling is
	// centralized below so canonical-bare-form types (DECIMAL →
	// decimal(10,0), INT → int(11), etc.) compose correctly with
	// UNSIGNED/ZEROFILL — this avoids the bare-form × modifier
	// Cartesian product false-positive class (Codex round-8 catch
	// on PR #20302).
	base := omniBuildCompactTypeString(dt, lower)

	// ZEROFILL implies UNSIGNED in MySQL storage; pingcap's Tp.String()
	// rendered both. Match that convention.
	if dt.Unsigned || dt.Zerofill {
		base += " unsigned"
	}
	if dt.Zerofill {
		base += " zerofill"
	}
	return base
}

// omniBuildCompactTypeString renders the type body in the
// CompactStr-equivalent form: type name + canonicalized length/scale
// (ENUM/SET value list) without UNSIGNED/ZEROFILL/charset suffixes.
// Matches pingcap's `column.Tp.CompactStr()` output and MySQL
// info_schema's non-attribute column-type rendering.
//
// Use this when comparing against:
//   - User-provided blocklist/allowlist entries (column_type_disallow_list)
//   - Pingcap CompactStr-equivalent fixture content
//
// For catalog comparison that needs UNSIGNED/ZEROFILL preservation,
// wrap with omniBuildColumnTypeString which appends the modifiers.
//
// The `lower` argument should be `strings.ToLower(dt.Name)` — callers
// often have it available already; passing in avoids re-computing.
//
// Length/scale rendering rules per type family (each verified
// empirically against pingcap's Tp.String()/CompactStr() during
// batch 11 + 12 pre-batch protocol):
//
//   - **ENUM / SET:** value list as body.
//   - **DECIMAL / NUMERIC / FIXED (exact-precision):** always (M,D),
//     defaulting Scale=0; canonicalize the name to `decimal`.
//   - **FLOAT / DOUBLE / REAL (approximate):** (M,D) only when Scale > 0;
//     otherwise bare name (matches pingcap's precision-hint drop).
//   - **Bare-form types with MySQL canonical defaults**: apply via
//     `canonicalBareTypeForm` (INT → int(11), TINYINT → tinyint(4),
//     BIT → bit(1), BINARY → binary(1), YEAR → year(4), BOOLEAN →
//     tinyint(1), DECIMAL → decimal(10,0), …).
//   - **All other types**: (Length) when Length > 0; bare name otherwise.
func omniBuildCompactTypeString(dt *ast.DataType, lower string) string {
	if dt == nil {
		return ""
	}
	switch {
	case isEnumOrSetTypeName(lower):
		return fmt.Sprintf("%s(%s)", lower, formatEnumValueList(dt.EnumValues))
	case dt.Length > 0 && isExactDecimalTypeName(lower):
		// DECIMAL / NUMERIC / FIXED with explicit length: always (M,D),
		// defaulting Scale=0. Canonicalize the type name to "decimal"
		// — pingcap's Tp.String() and MySQL info_schema both render
		// `NUMERIC(8,2)` and `FIXED(8,2)` as `decimal(8,2)`. Empirical:
		// omni pre-normalizes FIXED/DEC/INTEGER but does NOT
		// pre-normalize NUMERIC (Name stays "NUMERIC"); render
		// "decimal(...)" regardless to match catalog (Codex round-9).
		return fmt.Sprintf("decimal(%d,%d)", dt.Length, dt.Scale)
	case dt.Length > 0 && dt.Scale > 0:
		// Any other type with explicit (M,D) — render both.
		return fmt.Sprintf("%s(%d,%d)", lower, dt.Length, dt.Scale)
	case dt.Length > 0 && !isApproximateFloatTypeName(lower):
		// VARCHAR / CHAR / integer-with-display-width / etc.: (Length).
		// FLOAT(N) / DOUBLE(N) / REAL(N) fall through to default below →
		// bare name, matching pingcap's precision-hint drop.
		return fmt.Sprintf("%s(%d)", lower, dt.Length)
	}
	// No length, OR float-family without explicit scale.
	// Apply MySQL canonical default precision for type families whose
	// catalog/info_schema rendering carries the default.
	if canonical, ok := canonicalBareTypeForm(lower); ok {
		return canonical
	}
	return lower
}

// canonicalBareTypeForm returns the MySQL canonical default rendering
// for type families whose info_schema / pingcap Tp.String() always
// includes an explicit length/precision even when the user wrote the
// bare form. Returns "", false for type families that store/render
// bare (FLOAT, DOUBLE, JSON, DATE, …).
//
// This centralizes the "what does MySQL canonicalize?" knowledge in
// one place. Adding a new family is a single switch entry rather
// than: one normalize entry per type-modifier combination.
func canonicalBareTypeForm(lower string) (string, bool) {
	switch lower {
	case "decimal", "numeric", "fixed":
		return "decimal(10,0)", true
	case "tinyint":
		return "tinyint(4)", true
	case "smallint":
		return "smallint(6)", true
	case "mediumint":
		return "mediumint(9)", true
	case "int", "integer":
		return "int(11)", true
	case "bigint":
		return "bigint(20)", true
	case "bit":
		return "bit(1)", true
	case "binary":
		return "binary(1)", true
	case "year":
		return "year(4)", true
	case "boolean", "bool":
		// BOOLEAN is a TINYINT(1) alias; UNSIGNED/ZEROFILL append
		// to "tinyint(1)" if syntactically present (e.g. malformed
		// SQL like `BOOLEAN UNSIGNED`) — matches pingcap rendering.
		return "tinyint(1)", true
	default:
		return "", false
	}
}

// isExactDecimalTypeName returns true for column types whose canonical
// MySQL rendering always carries an explicit scale, even when zero.
// Pingcap's `Tp.String()` canonicalizes `DECIMAL(M)` → `decimal(M,0)`;
// MySQL info_schema does the same.
func isExactDecimalTypeName(lower string) bool {
	switch lower {
	case "decimal", "numeric", "fixed":
		return true
	default:
		return false
	}
}

// isApproximateFloatTypeName returns true for column types whose
// canonical MySQL rendering drops the precision hint when scale is
// not explicitly given. Pingcap's `Tp.String()` returns `"float"` for
// `FLOAT(10)` and `"float"` for `FLOAT(10,0)` — only `FLOAT(M,D)`
// with D > 0 renders as `float(M,D)`. Same applies to DOUBLE and
// REAL (REAL is an alias for DOUBLE in pingcap + MySQL).
func isApproximateFloatTypeName(lower string) bool {
	switch lower {
	case "float", "double", "real":
		return true
	default:
		return false
	}
}

// isEnumOrSetTypeName returns true for the ENUM and SET pseudo-types
// whose canonical rendering carries a value list rather than a length
// or scale.
func isEnumOrSetTypeName(lower string) bool {
	return lower == "enum" || lower == "set"
}

// formatEnumValueList renders an ENUM / SET value list in pingcap's
// `'v1','v2',…` canonical form: each value SQL-escaped (single quotes
// doubled) and wrapped in single quotes, comma-separated with no
// spaces. Matches pingcap's `Tp.String()` output for ENUM/SET as well
// as MySQL info_schema's COLUMN_TYPE rendering.
func formatEnumValueList(values []string) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = "'" + strings.ReplaceAll(v, "'", "''") + "'"
	}
	return strings.Join(parts, ",")
}

// normalizeColumnType canonicalizes type-name forms so that
// equivalent renderings on either side of the comparison
// (omni-built new type vs catalog-stored existing type) match.
// Two classes of normalization:
//
//   - **Integer default-length**: bare `int` ↔ `int(11)`, `tinyint`
//     ↔ `tinyint(4)`, etc. Pingcap's `Tp.String()` always rendered
//     the default display width; catalog stores it too. The omni
//     port may emit either form depending on whether the user
//     specified a length.
//
//   - **Alias normalization**: `boolean` ↔ `tinyint(1)`. Omni
//     preserves `DataType.Name = "BOOLEAN"` from the user's
//     literal source; pingcap and MySQL info_schema both
//     canonicalize boolean columns to `tinyint(1)` (verified via
//     `backend/plugin/schema/{tidb,mysql}/get_database_metadata.go`
//     and pingcap's Tp.String() for BOOLEAN columns). Without
//     this entry, no-op `MODIFY x BOOLEAN` on a tinyint(1)
//     column would false-positive (Codex round-5 catch).
//
// Kept structurally aligned with the mysql analog + pingcap-typed
// predecessor for fixture parity.
func normalizeColumnType(tp string) string {
	switch strings.ToLower(tp) {
	case "boolean", "bool":
		// MySQL canonicalizes BOOL/BOOLEAN to tinyint(1). Both
		// pingcap rendering and INFORMATION_SCHEMA agree.
		return "tinyint(1)"
	case "decimal", "numeric", "fixed":
		// MySQL applies default precision (10) and scale (0) when
		// user writes bare DECIMAL (or its NUMERIC / FIXED aliases).
		// Pingcap's Tp.String() and INFORMATION_SCHEMA both
		// canonicalize the bare form to decimal(10,0). The
		// omniBuildColumnTypeString builder renders the bare form
		// when Length=0 because there's no length to render;
		// catch the canonicalization here.
		return "decimal(10,0)"
	case "bit":
		// Pre-merge round-7 risk check surfaced this: MySQL's
		// default BIT precision is 1, pingcap renders `BIT` as
		// `bit(1)`, INFORMATION_SCHEMA stores `bit(1)`.
		return "bit(1)"
	case "binary":
		// Pre-merge round-7 risk check surfaced this: MySQL's
		// default BINARY length is 1, pingcap renders bare BINARY
		// as `binary(1)`, INFORMATION_SCHEMA stores `binary(1)`.
		return "binary(1)"
	case "year":
		// Pre-merge round-7 risk check surfaced this: MySQL's
		// default YEAR display width is 4 (legacy; YEAR(2) was
		// deprecated then removed). Pingcap renders bare YEAR as
		// `year(-1)` (an internal sentinel for "no width specified"
		// — a pingcap-side artifact); INFORMATION_SCHEMA stores
		// `year(4)`. Canonicalize bare year to year(4) to match the
		// catalog form.
		return "year(4)"
	case "tinyint":
		return "tinyint(4)"
	case "tinyint unsigned":
		return "tinyint(4) unsigned"
	case "smallint":
		return "smallint(6)"
	case "smallint unsigned":
		return "smallint(6) unsigned"
	case "mediumint":
		return "mediumint(9)"
	case "mediumint unsigned":
		return "mediumint(9) unsigned"
	case "int":
		return "int(11)"
	case "int unsigned":
		return "int(11) unsigned"
	case "bigint":
		return "bigint(20)"
	case "bigint unsigned":
		return "bigint(20) unsigned"
	default:
		return strings.ToLower(tp)
	}
}
