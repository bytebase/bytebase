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
// "name[(length[,scale])] [unsigned]" form that normalizeColumnType
// expects. Zerofill / charset / collation are intentionally NOT
// rendered — neither the catalog nor the rule's normalize map covers
// them.
func omniBuildColumnTypeString(dt *ast.DataType) string {
	if dt == nil {
		return ""
	}
	tp := strings.ToLower(dt.Name)
	switch {
	case dt.Length > 0 && dt.Scale > 0:
		tp = fmt.Sprintf("%s(%d,%d)", tp, dt.Length, dt.Scale)
	case dt.Length > 0:
		tp = fmt.Sprintf("%s(%d)", tp, dt.Length)
	default:
		// No length specified — keep tp as the bare type name.
	}
	if dt.Unsigned {
		tp += " unsigned"
	}
	return tp
}

// normalizeColumnType canonicalizes bare integer type names to their
// default-length forms so that pingcap's `int(11)`-style rendering
// matches the catalog's `int` (or vice-versa, depending on which side
// elides the default length). Kept structurally identical to the
// pingcap-typed predecessor + the mysql analog for fixture parity.
func normalizeColumnType(tp string) string {
	switch strings.ToLower(tp) {
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
