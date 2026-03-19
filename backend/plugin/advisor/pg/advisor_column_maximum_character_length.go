package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnMaximumCharacterLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH, &ColumnMaximumCharacterLengthAdvisor{})
}

// ColumnMaximumCharacterLengthAdvisor is the advisor checking for maximum character length.
type ColumnMaximumCharacterLengthAdvisor struct {
}

// Check checks for maximum character length.
func (*ColumnMaximumCharacterLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	if int(numberPayload.Number) <= 0 {
		return nil, nil
	}

	rule := &columnMaximumCharacterLengthRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		maximum: int(numberPayload.Number),
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnMaximumCharacterLengthRule struct {
	OmniBaseRule

	maximum int
}

func (*columnMaximumCharacterLengthRule) Name() string {
	return "column-maximum-character-length"
}

func (r *columnMaximumCharacterLengthRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	default:
	}
}

func (r *columnMaximumCharacterLengthRule) handleCreateStmt(n *ast.CreateStmt) {
	tableName := omniFormatTableName(n.Relation)
	if tableName == "" {
		return
	}

	cols, _ := omniTableElements(n)
	for _, col := range cols {
		if col.TypeName == nil {
			continue
		}
		charLength := r.getCharLength(col.TypeName)
		if charLength > r.maximum {
			r.addAdvice(tableName, col.Colname, col.Colname)
			return // Only report first violation
		}
	}
}

func (r *columnMaximumCharacterLengthRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniFormatTableName(n.Relation)
	if tableName == "" {
		return
	}

	for _, cmd := range omniAlterTableCmds(n) {
		switch ast.AlterTableType(cmd.Subtype) {
		case ast.AT_AddColumn:
			colDef, ok := cmd.Def.(*ast.ColumnDef)
			if !ok || colDef == nil || colDef.TypeName == nil {
				continue
			}
			charLength := r.getCharLength(colDef.TypeName)
			if charLength > r.maximum {
				r.addAdvice(tableName, colDef.Colname, colDef.Colname)
				return
			}
		case ast.AT_AlterColumnType:
			typeName, ok := cmd.Def.(*ast.ColumnDef)
			if !ok || typeName == nil || typeName.TypeName == nil {
				continue
			}
			charLength := r.getCharLength(typeName.TypeName)
			if charLength > r.maximum {
				r.addAdvice(tableName, cmd.Name, cmd.Name)
				return
			}
		default:
		}
	}
}

// getCharLength returns the character length if the type is CHAR/CHARACTER/NCHAR (without VARYING).
// Returns 0 for non-character types or VARCHAR types.
func (*columnMaximumCharacterLengthRule) getCharLength(tn *ast.TypeName) int {
	name := strings.ToLower(omniTypeName(tn))

	// Only check fixed-length char types (char, character, bpchar, nchar)
	// Skip varchar, character varying, text
	switch name {
	case "char", "character", "bpchar", "nchar":
		// OK, continue
	default:
		return 0
	}

	// Get the length from Typmods
	if tn.Typmods == nil || len(tn.Typmods.Items) == 0 {
		return 0
	}

	// First typmod item is the length
	if intVal, ok := tn.Typmods.Items[0].(*ast.Integer); ok {
		return int(intVal.Ival)
	}

	return 0
}

func (r *columnMaximumCharacterLengthRule) addAdvice(tableName, columnName, searchName string) {
	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    code.CharLengthExceedsLimit.Int32(),
		Title:   r.Title,
		Content: fmt.Sprintf("The length of the CHAR column %q in table %s is bigger than %d, please use VARCHAR instead", columnName, tableName, r.maximum),
		StartPosition: &storepb.Position{
			Line:   r.FindLineByName(searchName),
			Column: 0,
		},
	})
}

// omniFormatTableName formats a RangeVar as a quoted table name string for display.
func omniFormatTableName(rv *ast.RangeVar) string {
	if rv == nil || rv.Relname == "" {
		return ""
	}
	if rv.Schemaname != "" {
		return fmt.Sprintf("%q.%q", rv.Schemaname, rv.Relname)
	}
	return fmt.Sprintf("%q", rv.Relname)
}
