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
	_ advisor.Advisor = (*TableDropNamingConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, &TableDropNamingConventionAdvisor{})
}

// TableDropNamingConventionAdvisor is the advisor checking for table drop with naming convention.
type TableDropNamingConventionAdvisor struct {
}

// Check checks for table drop with naming convention.
func (*TableDropNamingConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for table drop naming convention rule")
	}

	format, err := regexp.Compile(namingPayload.Format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}

	checker := &tableDropNamingConventionChecker{
		level:      level,
		title:      checkCtx.Rule.Type.String(),
		format:     format,
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

type tableDropNamingConventionChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	format     *regexp.Regexp
	// text is the SQL text of the statement currently being checked.
	text string
	// baseLine is the 0-based line of the current statement in the whole script.
	baseLine int
}

// checkStmt checks DROP TABLE statements: the canonical (folded) table name
// must match the configured naming format. Other DROP kinds (VIEW, DYNAMIC
// TABLE, ...) are not checked, matching the legacy listener which only
// subscribed to the plain DROP TABLE grammar rule.
func (c *tableDropNamingConventionChecker) checkStmt(node snowflakeast.Node) {
	dropStmt, ok := node.(*snowflakeast.DropStmt)
	if !ok || dropStmt.Kind != snowflakeast.DropTable || dropStmt.Name == nil {
		return
	}

	normalizedObjectName := dropStmt.Name.Name.Normalize()
	if !c.format.MatchString(normalizedObjectName) {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.TableDropNamingConventionMismatch.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("%q mismatches drop table naming convention, naming format should be %q", normalizedObjectName, c.format),
			StartPosition: c.positionAtOffset(dropStmt.Name.Name.Loc.Start),
		})
	}
}

// positionAtOffset converts a byte offset within the current statement text to
// an advice position, reproducing the legacy baseLine + ANTLR-line arithmetic.
// A negative (unknown) offset degrades to the statement's first line.
func (c *tableDropNamingConventionChecker) positionAtOffset(offset int) *storepb.Position {
	line := 1
	if offset > 0 {
		if offset > len(c.text) {
			offset = len(c.text)
		}
		line += strings.Count(c.text[:offset], "\n")
	}
	return common.ConvertANTLRLineToPosition(c.baseLine + line)
}
