package pg

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnRequirementAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_REQUIRED, &ColumnRequirementAdvisor{})
}

type columnSet map[string]bool

// ColumnRequirementAdvisor is the advisor checking for column requirement.
type ColumnRequirementAdvisor struct {
}

// Check checks for the column requirement.
func (*ColumnRequirementAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for column required rule")
	}

	requiredColumnsMap := make(columnSet)
	for _, col := range stringArrayPayload.List {
		requiredColumnsMap[col] = true
	}

	rule := &columnRequirementRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		requiredColumnsMap: requiredColumnsMap,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnRequirementRule struct {
	OmniBaseRule

	requiredColumnsMap columnSet
}

func (*columnRequirementRule) Name() string {
	return "column_requirement"
}

func (r *columnRequirementRule) OnStatement(node ast.Node) {
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

func (r *columnRequirementRule) handleCreateStmt(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}

	// Copy required columns to check against
	required := make(columnSet)
	for col := range r.requiredColumnsMap {
		required[col] = true
	}

	cols, _ := omniTableElements(n)
	for _, col := range cols {
		delete(required, col.Colname)
	}

	if len(required) > 0 {
		var missingColumns []string
		for column := range required {
			missingColumns = append(missingColumns, column)
		}
		slices.Sort(missingColumns)

		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NoRequiredColumn.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Table %q requires columns: %s", tableName, strings.Join(missingColumns, ", ")),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}

func (r *columnRequirementRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}

	for _, cmd := range omniAlterTableCmds(n) {
		if ast.AlterTableType(cmd.Subtype) == ast.AT_DropColumn {
			colName := cmd.Name
			if r.requiredColumnsMap[colName] {
				r.AddAdvice(&storepb.Advice{
					Status:  r.Level,
					Code:    code.NoRequiredColumn.Int32(),
					Title:   r.Title,
					Content: fmt.Sprintf("Table %q requires columns: %s", tableName, colName),
					StartPosition: &storepb.Position{
						Line:   r.ContentStartLine(),
						Column: 0,
					},
				})
			}
		}
	}
}

func (r *columnRequirementRule) handleRenameStmt(n *ast.RenameStmt) {
	if n.RenameType != ast.OBJECT_COLUMN {
		return
	}

	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}

	oldName := n.Subname
	newName := n.Newname

	if r.requiredColumnsMap[oldName] && oldName != newName {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NoRequiredColumn.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Table %q requires columns: %s", tableName, oldName),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}
