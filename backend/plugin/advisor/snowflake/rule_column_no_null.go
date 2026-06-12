package snowflake

import (
	"context"
	"fmt"
	"strings"

	omniast "github.com/bytebase/omni/snowflake/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_COLUMN_NO_NULL, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewColumnNoNullRule(level, checkCtx.Rule.Type.String())

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := snowsqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		rule.checkStatement(node, stmt.Text)
	}

	return rule.GetAdviceList(), nil
}

// ColumnNoNullRule checks for column no NULL value.
type ColumnNoNullRule struct {
	BaseRule
	// stmtText is the SQL text of the statement currently being checked; node
	// Loc offsets are relative to it.
	stmtText string
}

// NewColumnNoNullRule creates a new ColumnNoNullRule.
func NewColumnNoNullRule(level storepb.Advice_Status, title string) *ColumnNoNullRule {
	return &ColumnNoNullRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*ColumnNoNullRule) Name() string {
	return "ColumnNoNullRule"
}

// checkStatement checks one statement's omni AST node.
func (r *ColumnNoNullRule) checkStatement(node omniast.Node, stmtText string) {
	r.stmtText = stmtText
	switch n := node.(type) {
	case *omniast.CreateTableStmt:
		r.checkCreateTable(n)
	case *omniast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *ColumnNoNullRule) checkCreateTable(stmt *omniast.CreateTableStmt) {
	// The legacy advisor only fired on plain CREATE TABLE (the Create_table
	// context); CREATE TABLE ... AS SELECT / LIKE / CLONE parsed as different
	// contexts and were never checked.
	if stmt.AsSelect != nil || stmt.Like != nil || stmt.Clone != nil {
		return
	}

	// columnNullable maps the normalized column name to the line number causing
	// the column to be nullable. Mirrors the legacy add-then-delete walk so
	// duplicate column names behave identically.
	columnNullable := make(map[string]int)
	for _, column := range stmt.Columns {
		normalizedColumnName := column.Name.Normalize()
		columnNullable[normalizedColumnName] = r.line(column.Loc.Start)
		if column.NotNull {
			delete(columnNullable, normalizedColumnName)
		}
		if column.InlineConstraint != nil && column.InlineConstraint.Type == omniast.ConstrPrimaryKey {
			delete(columnNullable, normalizedColumnName)
		}
	}
	// Out-of-line PRIMARY KEY constraints make their columns non-nullable.
	for _, constraint := range stmt.Constraints {
		if constraint.Type != omniast.ConstrPrimaryKey {
			continue
		}
		for _, column := range constraint.Columns {
			delete(columnNullable, column.Normalize())
		}
	}

	r.addNullableAdvice(columnNullable)
}

func (r *ColumnNoNullRule) checkAlterTable(stmt *omniast.AlterTableStmt) {
	columnNullable := make(map[string]int)
	for _, action := range stmt.Actions {
		switch action.Kind {
		case omniast.AlterTableAddColumn:
			for _, column := range action.Columns {
				normalizedColumnName := column.Name.Normalize()
				columnNullable[normalizedColumnName] = r.line(column.Loc.Start)
				if column.NotNull {
					delete(columnNullable, normalizedColumnName)
				}
				if column.InlineConstraint != nil && column.InlineConstraint.Type == omniast.ConstrPrimaryKey {
					delete(columnNullable, normalizedColumnName)
				}
			}
		case omniast.AlterTableAlterColumn:
			for _, columnAlter := range action.ColumnAlters {
				if columnAlter.Kind == omniast.ColumnAlterDropNotNull {
					columnNullable[columnAlter.Column.Normalize()] = r.line(columnAlter.Column.Loc.Start)
				}
			}
		case omniast.AlterTableAddConstraint:
			// Mirrors the legacy walk where an out-of-line PRIMARY KEY removed
			// its columns from the nullable set.
			if action.Constraint != nil && action.Constraint.Type == omniast.ConstrPrimaryKey {
				for _, column := range action.Constraint.Columns {
					delete(columnNullable, column.Normalize())
				}
			}
		default:
		}
	}

	r.addNullableAdvice(columnNullable)
}

func (r *ColumnNoNullRule) addNullableAdvice(columnNullable map[string]int) {
	for normalizedColumnName, columnNullableLine := range columnNullable {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.ColumnCannotNull.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Column %s is nullable, which is not allowed.", normalizedColumnName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + columnNullableLine),
		})
	}
}

// line converts a byte offset within the current statement text to a 1-based
// line number, mirroring the ANTLR token line the legacy advisor reported.
func (r *ColumnNoNullRule) line(offset int) int {
	if offset < 0 {
		return 1
	}
	if offset > len(r.stmtText) {
		offset = len(r.stmtText)
	}
	return 1 + strings.Count(r.stmtText[:offset], "\n")
}
