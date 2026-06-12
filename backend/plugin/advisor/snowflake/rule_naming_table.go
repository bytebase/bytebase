// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	snowflakeast "github.com/bytebase/omni/snowflake/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*NamingTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableAdvisor{})
}

// NamingTableAdvisor is the advisor checking for table naming convention.
type NamingTableAdvisor struct {
}

// Check checks for table naming convention.
func (*NamingTableAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	checker := &namingTableChecker{
		level:      level,
		title:      checkCtx.Rule.Type.String(),
		format:     format,
		maxLength:  maxLength,
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

type namingTableChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	format     *regexp.Regexp
	maxLength  int
	// text is the SQL text of the statement currently being checked.
	text string
	// baseLine is the 0-based line of the current statement in the whole script.
	baseLine int
}

func (c *namingTableChecker) checkStmt(node snowflakeast.Node) {
	switch n := node.(type) {
	case *snowflakeast.CreateTableStmt:
		// The legacy listener only fired on the plain CREATE TABLE grammar rule;
		// CREATE TABLE ... AS SELECT / LIKE / CLONE were separate rules it did
		// not subscribe to. Mirror that scope exactly.
		if n.AsSelect != nil || n.Like != nil || n.Clone != nil {
			return
		}
		if n.Name == nil {
			return
		}
		c.checkTableName(n.Name.Name, n.Loc)
	case *snowflakeast.AlterTableStmt:
		for _, action := range n.Actions {
			if action.Kind != snowflakeast.AlterTableRename || action.NewName == nil {
				continue
			}
			c.checkTableName(action.NewName.Name, n.Loc)
		}
	default:
	}
}

// checkTableName applies the format and max-length checks to the table part of
// an object name, anchoring the advice on the statement start (stmtLoc), as
// the legacy listener did. The legacy rule took the raw source text of the
// name and stripped only the surrounding double quotes — no case folding and
// no unescaping of doubled quotes — which Ident.String() (the re-quoted
// source form) reproduces exactly.
func (c *namingTableChecker) checkTableName(name snowflakeast.Ident, stmtLoc snowflakeast.Loc) {
	objectName := name.String()
	tableName := strings.TrimPrefix(strings.TrimSuffix(objectName, `"`), `"`)
	position := c.positionAtOffset(stmtLoc.Start)

	if !c.format.MatchString(tableName) {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, c.format),
			StartPosition: position,
		})
	}
	if c.maxLength > 0 && len(tableName) > c.maxLength {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, c.maxLength),
			StartPosition: position,
		})
	}
}

// positionAtOffset converts a byte offset within the current statement text to
// an advice position, reproducing the legacy baseLine + ANTLR-line arithmetic.
// A negative (unknown) offset degrades to the statement's first line.
func (c *namingTableChecker) positionAtOffset(offset int) *storepb.Position {
	line := 1
	if offset > 0 {
		if offset > len(c.text) {
			offset = len(c.text)
		}
		line += strings.Count(c.text[:offset], "\n")
	}
	return common.ConvertANTLRLineToPosition(c.baseLine + line)
}
