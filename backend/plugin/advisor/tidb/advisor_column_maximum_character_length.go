package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnMaximumCharacterLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH, &ColumnMaximumCharacterLengthAdvisor{})
}

// ColumnMaximumCharacterLengthAdvisor is the advisor checking for max character length.
type ColumnMaximumCharacterLengthAdvisor struct {
}

// Check checks for maximum character length.
func (*ColumnMaximumCharacterLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for column maximum character length rule")
	}
	maximum := int(numberPayload.Number)
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		advice := checkStmtForCharLength(ostmt, maximum, level, title)
		if advice != nil {
			adviceList = append(adviceList, advice)
		}
	}

	return adviceList, nil
}

// checkStmtForCharLength returns at most ONE advice per top-level
// statement, mirroring pingcap-typed predecessor's `break`-after-first-
// match cardinality contract. Mysql analog emits per-column (no break)
// — cardinality divergence preserved on the tidb side per invariant #7.
//
// Rule fires on CHAR or BINARY columns whose length exceeds the
// configured maximum. Cumulative #22 territory: pingcap's
// `mysql.TypeString` covered BOTH CHAR and BINARY (charset-pair
// unification); my omni port matches both via omniIsCharOrBinaryType.
// MySQL's "max character length" rule conceptually applies to CHAR;
// extending to BINARY preserves the pingcap behavior even though it's
// semantically odd (BINARY length is bytes, not characters).
func checkStmtForCharLength(ostmt OmniStmt, maximum int, level storepb.Advice_Status, title string) *storepb.Advice {
	if maximum <= 0 {
		return nil
	}
	switch n := ostmt.Node.(type) {
	case *ast.CreateTableStmt:
		if n.Table == nil {
			return nil
		}
		tableName := n.Table.Name
		for _, column := range n.Columns {
			if column == nil {
				continue
			}
			if charLength := omniCharLength(column.TypeName); charLength > maximum {
				return buildCharLengthAdvice(level, title, tableName, column.Name, charLength, maximum, ostmt.AbsoluteLine(column.Loc.Start))
			}
		}
	case *ast.AlterTableStmt:
		if n.Table == nil {
			return nil
		}
		tableName := n.Table.Name
		stmtLine := ostmt.AbsoluteLine(n.Loc.Start)
		for _, cmd := range n.Commands {
			if cmd == nil {
				continue
			}
			switch cmd.Type {
			case ast.ATAddColumn:
				// Cumulative #23: pingcap-tidb's pre-omni
				// AlterTableAddColumns inner column loop had no break;
				// LAST violating column in the grouped form overwrites
				// and is reported (asymmetric vs CreateTableStmt which
				// DOES break — first-wins). Track lastViolation across
				// the inner loop, emit once after — mirrors cumulative
				// #15's "preserve pingcap single-advice-per-stmt
				// cardinality with last-wins semantics" prescription
				// on the AlterTable ADD COLUMN call-site.
				var lastCol *ast.ColumnDef
				var lastLen int
				for _, column := range addColumnTargets(cmd) {
					if column == nil {
						continue
					}
					if charLength := omniCharLength(column.TypeName); charLength > maximum {
						lastCol = column
						lastLen = charLength
					}
				}
				if lastCol != nil {
					return buildCharLengthAdvice(level, title, tableName, lastCol.Name, lastLen, maximum, stmtLine)
				}
			case ast.ATChangeColumn, ast.ATModifyColumn:
				if cmd.Column == nil {
					continue
				}
				if charLength := omniCharLength(cmd.Column.TypeName); charLength > maximum {
					return buildCharLengthAdvice(level, title, tableName, cmd.Column.Name, charLength, maximum, stmtLine)
				}
			default:
			}
		}
	default:
	}
	return nil
}

func buildCharLengthAdvice(level storepb.Advice_Status, title, _, columnName string, charLength, maximum, line int) *storepb.Advice {
	return &storepb.Advice{
		Status:        level,
		Code:          code.CharLengthExceedsLimit.Int32(),
		Title:         title,
		Content:       fmt.Sprintf("The length of the CHAR column `%s` is %d, bigger than %d, please use VARCHAR instead", columnName, charLength, maximum),
		StartPosition: common.ConvertANTLRLineToPosition(line),
	}
}
