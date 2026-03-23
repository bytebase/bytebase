package pg

import (
	"context"
	"fmt"
	"regexp"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_COLUMN, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column naming convention.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (*NamingColumnConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for this rule")
	}

	format, err := regexp.Compile(namingPayload.Format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}

	rule := &namingColumnConventionRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		format:    format,
		maxLength: maxLength,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type namingColumnConventionRule struct {
	OmniBaseRule

	format    *regexp.Regexp
	maxLength int
}

func (*namingColumnConventionRule) Name() string {
	return "naming_column_convention"
}

func (r *namingColumnConventionRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	case *ast.RenameStmt:
		r.handleRenameStmt(n)
	default:
	}
}

func (r *namingColumnConventionRule) handleCreateStmt(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	cols, _ := omniTableElements(n)
	for _, col := range cols {
		r.checkColumnName(tableName, col.Colname)
	}
}

func (r *namingColumnConventionRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)
	for _, cmd := range omniAlterTableCmds(n) {
		if ast.AlterTableType(cmd.Subtype) == ast.AT_AddColumn {
			if colDef, ok := cmd.Def.(*ast.ColumnDef); ok {
				r.checkColumnName(tableName, colDef.Colname)
			}
		}
	}
}

func (r *namingColumnConventionRule) handleRenameStmt(n *ast.RenameStmt) {
	if n.RenameType == ast.OBJECT_COLUMN {
		tableName := omniTableName(n.Relation)
		r.checkColumnName(tableName, n.Newname)
	}
}

func (r *namingColumnConventionRule) checkColumnName(tableName, columnName string) {
	line := r.FindLineByName(columnName)

	if !r.format.MatchString(columnName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NamingColumnConventionMismatch.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, naming format should be %q", tableName, columnName, r.format),
			StartPosition: &storepb.Position{
				Line:   line,
				Column: 0,
			},
		})
	}

	if r.maxLength > 0 && len(columnName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NamingColumnConventionMismatch.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, its length should be within %d characters", tableName, columnName, r.maxLength),
			StartPosition: &storepb.Position{
				Line:   line,
				Column: 0,
			},
		})
	}
}
