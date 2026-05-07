package tidb

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingTableConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableConventionAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming convention.
type NamingTableConventionAdvisor struct {
}

// Check checks for table naming convention.
func (*NamingTableConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for naming table rule")
	}

	format, err := regexp.Compile(namingPayload.Format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}
	checker := &namingTableConventionChecker{
		level:     level,
		title:     checkCtx.Rule.Type.String(),
		format:    format,
		maxLength: maxLength,
	}

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

type namingTableConventionChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	format     *regexp.Regexp
	maxLength  int
}

func (c *namingTableConventionChecker) checkStmt(ostmt OmniStmt) {
	var tableNames []string
	var line int
	switch n := ostmt.Node.(type) {
	case *ast.CreateTableStmt:
		if n.Table == nil {
			return
		}
		tableNames = append(tableNames, n.Table.Name)
		line = ostmt.AbsoluteLine(n.Loc.Start)
	case *ast.AlterTableStmt:
		for _, cmd := range n.Commands {
			if cmd == nil {
				continue
			}
			if cmd.Type == ast.ATRenameTable && cmd.NewName != "" {
				tableNames = append(tableNames, cmd.NewName)
			}
		}
		line = ostmt.AbsoluteLine(n.Loc.Start)
	case *ast.RenameTableStmt:
		for _, pair := range n.Pairs {
			if pair == nil || pair.New == nil {
				continue
			}
			tableNames = append(tableNames, pair.New.Name)
		}
		line = ostmt.AbsoluteLine(n.Loc.Start)
	default:
		return
	}

	for _, tableName := range tableNames {
		if !c.format.MatchString(tableName) {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          code.NamingTableConventionMismatch.Int32(),
				Title:         c.title,
				Content:       fmt.Sprintf("`%s` mismatches table naming convention, naming format should be %q", tableName, c.format),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
		}
		if c.maxLength > 0 && len(tableName) > c.maxLength {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          code.NamingTableConventionMismatch.Int32(),
				Title:         c.title,
				Content:       fmt.Sprintf("`%s` mismatches table naming convention, its length should be within %d characters", tableName, c.maxLength),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
		}
	}
}
