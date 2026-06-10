// Package snowflake is the advisor for snowflake database.
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
	_ advisor.Advisor = (*TableNoForeignKeyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoForeignKeyAdvisor{})
}

// TableNoForeignKeyAdvisor is the advisor checking for table disallow foreign key.
type TableNoForeignKeyAdvisor struct {
}

// Check checks for table disallow foreign key.
func (*TableNoForeignKeyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewTableNoForeignKeyRule(level, checkCtx.Rule.Type.String())

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

// TableNoForeignKeyRule checks for table disallow foreign key.
type TableNoForeignKeyRule struct {
	BaseRule

	// tableForeignKeyTimes is a map of normalized table name to the times of FOREIGN KEY.
	tableForeignKeyTimes map[string]int
	// tableOriginalName is a map of normalized table name to original table name.
	// The key of the tableOriginalName is the superset of the key of the tableForeignKeyTimes.
	tableOriginalName map[string]string
	// tableLine is a map of normalized table name to the line number of the table.
	// The key of the tableLine is the superset of the key of the tableForeignKeyTimes.
	tableLine map[string]int

	// stmtText is the SQL text of the statement currently being checked; node
	// Loc offsets are relative to it.
	stmtText string
}

// NewTableNoForeignKeyRule creates a new TableNoForeignKeyRule.
func NewTableNoForeignKeyRule(level storepb.Advice_Status, title string) *TableNoForeignKeyRule {
	return &TableNoForeignKeyRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		tableForeignKeyTimes: make(map[string]int),
		tableOriginalName:    make(map[string]string),
		tableLine:            make(map[string]int),
	}
}

// Name returns the rule name.
func (*TableNoForeignKeyRule) Name() string {
	return "TableNoForeignKeyRule"
}

// GetAdviceList returns the accumulated advice list, generating final advice for tables with FK.
func (r *TableNoForeignKeyRule) GetAdviceList() []*storepb.Advice {
	for tableName, times := range r.tableForeignKeyTimes {
		if times > 0 {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.TableHasFK.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("FOREIGN KEY is not allowed in the table %s.", r.tableOriginalName[tableName]),
				StartPosition: common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
			})
		}
	}
	return r.adviceList
}

// checkStatement checks one statement's omni AST node.
func (r *TableNoForeignKeyRule) checkStatement(node omniast.Node, stmtText string) {
	r.stmtText = stmtText
	switch n := node.(type) {
	case *omniast.CreateTableStmt:
		r.checkCreateTable(n)
	case *omniast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *TableNoForeignKeyRule) checkCreateTable(stmt *omniast.CreateTableStmt) {
	// The legacy advisor only fired on plain CREATE TABLE (the Create_table
	// context); CREATE TABLE ... AS SELECT / LIKE / CLONE parsed as different
	// contexts and were never checked.
	if stmt.AsSelect != nil || stmt.Like != nil || stmt.Clone != nil {
		return
	}

	normalizedTableName := r.normalizedTableName(stmt.Name)

	r.tableForeignKeyTimes[normalizedTableName] = 0
	r.tableOriginalName[normalizedTableName] = stmt.Name.String()
	r.tableLine[normalizedTableName] = r.baseLine + r.line(stmt.Loc.Start)

	// Inline REFERENCES constraints count but keep the table line.
	for _, column := range stmt.Columns {
		if column.InlineConstraint != nil && column.InlineConstraint.Type == omniast.ConstrForeignKey {
			r.tableForeignKeyTimes[normalizedTableName]++
		}
	}
	// Out-of-line FOREIGN KEY constraints count and move the line to the
	// constraint, matching the legacy advisor.
	for _, constraint := range stmt.Constraints {
		if constraint.Type == omniast.ConstrForeignKey {
			r.tableForeignKeyTimes[normalizedTableName]++
			r.tableLine[normalizedTableName] = r.baseLine + r.line(constraint.Loc.Start)
		}
	}
}

func (r *TableNoForeignKeyRule) checkAlterTable(stmt *omniast.AlterTableStmt) {
	// The legacy advisor only tracked ALTER TABLE statements with a constraint
	// action (ADD/DROP/RENAME CONSTRAINT); column actions were ignored.
	if !alterTableHasConstraintActionForFk(stmt) {
		return
	}

	normalizedTableName := r.normalizedTableName(stmt.Name)
	r.tableOriginalName[normalizedTableName] = stmt.Name.String()

	for _, action := range stmt.Actions {
		switch action.Kind {
		case omniast.AlterTableAddConstraint:
			if action.Constraint != nil && action.Constraint.Type == omniast.ConstrForeignKey {
				r.tableForeignKeyTimes[normalizedTableName]++
				r.tableLine[normalizedTableName] = r.baseLine + r.line(action.Constraint.Loc.Start)
			}
		case omniast.AlterTableDropConstraint:
			// Only an explicit DROP FOREIGN KEY decrements; DROP CONSTRAINT
			// <name> does not reveal the constraint type, matching the legacy
			// advisor.
			if action.DropForeignKey {
				if times, ok := r.tableForeignKeyTimes[normalizedTableName]; ok && times > 0 {
					r.tableForeignKeyTimes[normalizedTableName]--
				}
			}
		default:
		}
	}
}

func alterTableHasConstraintActionForFk(stmt *omniast.AlterTableStmt) bool {
	for _, action := range stmt.Actions {
		switch action.Kind {
		case omniast.AlterTableAddConstraint, omniast.AlterTableDropConstraint, omniast.AlterTableRenameConstraint:
			return true
		default:
		}
	}
	return false
}

// normalizedTableName normalizes an object name into the same
// "database.schema.table" key the legacy advisor produced via
// NormalizeSnowSQLObjectName(name, "", "PUBLIC").
func (*TableNoForeignKeyRule) normalizedTableName(name *omniast.ObjectName) string {
	if name == nil {
		return ""
	}
	database := ""
	if d := name.Database.Normalize(); d != "" {
		database = d
	}
	schema := "PUBLIC"
	if s := name.Schema.Normalize(); s != "" {
		schema = s
	}
	parts := []string{database, schema}
	if o := name.Name.Normalize(); o != "" {
		parts = append(parts, o)
	}
	return strings.Join(parts, ".")
}

// line converts a byte offset within the current statement text to a 1-based
// line number, mirroring the ANTLR token line the legacy advisor reported.
func (r *TableNoForeignKeyRule) line(offset int) int {
	if offset < 0 {
		return 1
	}
	if offset > len(r.stmtText) {
		offset = len(r.stmtText)
	}
	return 1 + strings.Count(r.stmtText[:offset], "\n")
}
