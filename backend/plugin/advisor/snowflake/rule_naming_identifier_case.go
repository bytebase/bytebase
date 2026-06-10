// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"
	"strings"

	snowflakeast "github.com/bytebase/omni/snowflake/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*NamingIdentifierCaseAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE, &NamingIdentifierCaseAdvisor{})
}

// NamingIdentifierCaseAdvisor is the advisor checking for identifier case.
type NamingIdentifierCaseAdvisor struct {
}

// Check checks for identifier case.
func (*NamingIdentifierCaseAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &namingIdentifierCaseChecker{
		level:      level,
		title:      checkCtx.Rule.Type.String(),
		adviceList: []*storepb.Advice{},
	}

	for _, stmt := range checkCtx.ParsedStatements {
		node, ok := snowsqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		checker.text = stmt.Text
		checker.baseLine = stmt.BaseLine()
		checker.checkStmt(node)
	}

	return checker.adviceList, nil
}

type namingIdentifierCaseChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	// text is the SQL text of the statement currently being checked.
	text string
	// baseLine is the 0-based line of the current statement in the whole script.
	baseLine int
}

func (c *namingIdentifierCaseChecker) checkStmt(node snowflakeast.Node) {
	switch n := node.(type) {
	case *snowflakeast.CreateTableStmt:
		c.checkCreateTable(n)
	case *snowflakeast.AlterTableStmt:
		c.checkAlterTable(n)
	default:
	}
}

// checkCreateTable checks the declared column names of CREATE TABLE and
// CREATE TABLE ... AS SELECT: a column whose canonical (folded) name is not
// upper case — i.e. a double-quoted identifier containing lower-case
// characters — is reported. Mirroring the legacy listener, every advice is
// anchored on the start of the column declaration list, so all violations in
// one statement share that line.
func (c *namingIdentifierCaseChecker) checkCreateTable(stmt *snowflakeast.CreateTableStmt) {
	if len(stmt.Columns) == 0 {
		return
	}

	listPosition := c.columnDeclListPosition(stmt)
	for _, column := range stmt.Columns {
		// Snowflake folding: unquoted identifiers are stored upper case, quoted
		// identifiers verbatim — so only quoted lower-case names can violate.
		originalColName := column.Name.Normalize()
		if strings.ToUpper(originalColName) != originalColName {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          code.NamingCaseMismatch.Int32(),
				Title:         c.title,
				Content:       fmt.Sprintf("Identifier %q should be upper case", originalColName),
				StartPosition: listPosition,
			})
		}
	}
}

// checkAlterTable checks ALTER TABLE ... RENAME COLUMN old TO new: the new
// column name must fold to upper case. Other ALTER TABLE actions are not
// checked, matching the legacy listener.
func (c *namingIdentifierCaseChecker) checkAlterTable(stmt *snowflakeast.AlterTableStmt) {
	for _, action := range stmt.Actions {
		if action.Kind != snowflakeast.AlterTableRenameColumn {
			continue
		}
		renameToColName := action.NewColName.Normalize()
		if strings.ToUpper(renameToColName) != renameToColName {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          code.NamingCaseMismatch.Int32(),
				Title:         c.title,
				Content:       fmt.Sprintf("Identifier %q should be upper case", renameToColName),
				StartPosition: c.positionAtOffset(action.NewColName.Loc.Start),
			})
		}
	}
}

// columnDeclListPosition returns the position of the first item of the column
// declaration list — columns, out-of-line constraints and inline indexes in
// source order — mirroring the legacy rule's anchor on the ANTLR
// Column_decl_item_list start token.
func (c *namingIdentifierCaseChecker) columnDeclListPosition(stmt *snowflakeast.CreateTableStmt) *storepb.Position {
	start := -1
	merge := func(loc snowflakeast.Loc) {
		if loc.Start >= 0 && (start < 0 || loc.Start < start) {
			start = loc.Start
		}
	}
	for _, column := range stmt.Columns {
		merge(column.Loc)
	}
	for _, constraint := range stmt.Constraints {
		merge(constraint.Loc)
	}
	for _, index := range stmt.Indexes {
		merge(index.Loc)
	}
	return c.positionAtOffset(start)
}

// positionAtOffset converts a byte offset within the current statement text to
// an advice position, reproducing the legacy baseLine + ANTLR-line arithmetic.
// A negative (unknown) offset degrades to the statement's first line.
func (c *namingIdentifierCaseChecker) positionAtOffset(offset int) *storepb.Position {
	line := 1
	if offset > 0 {
		if offset > len(c.text) {
			offset = len(c.text)
		}
		line += strings.Count(c.text[:offset], "\n")
	}
	return common.ConvertANTLRLineToPosition(c.baseLine + line)
}
