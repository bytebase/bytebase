package redshift

import (
	"context"
	"fmt"
	"regexp"

	redshiftast "github.com/bytebase/omni/redshift/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	parserredshift "github.com/bytebase/bytebase/backend/plugin/parser/redshift"
)

var (
	_ advisor.Advisor = (*TableDropNamingConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_REDSHIFT, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, &TableDropNamingConventionAdvisor{})
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
		node, ok := parserredshift.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		checker.checkStmt(node, stmt.Start)
	}

	return checker.adviceList, nil
}

type tableDropNamingConventionChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	format     *regexp.Regexp
}

func (c *tableDropNamingConventionChecker) checkStmt(node redshiftast.Node, startPosition *storepb.Position) {
	dropStmt, ok := node.(*redshiftast.DropStmt)
	if !ok {
		return
	}
	if redshiftast.ObjectType(dropStmt.RemoveType) != redshiftast.OBJECT_TABLE {
		return
	}
	for _, tableName := range omniDropTableNames(dropStmt) {
		if tableName == "" || c.format.MatchString(tableName) {
			continue
		}
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.TableDropNamingConventionMismatch.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", tableName, c.format),
			StartPosition: startPosition,
		})
	}
}

func omniDropTableNames(dropStmt *redshiftast.DropStmt) []string {
	if dropStmt.Objects == nil {
		return nil
	}

	var result []string
	for _, item := range dropStmt.Objects.Items {
		var parts []string
		switch n := item.(type) {
		case *redshiftast.List:
			for _, nameItem := range n.Items {
				if s, ok := nameItem.(*redshiftast.String); ok {
					parts = append(parts, s.Str)
				}
			}
		case *redshiftast.RangeVar:
			parts = append(parts, n.Relname)
		default:
		}
		if len(parts) > 0 {
			result = append(result, parts[len(parts)-1])
		}
	}
	return result
}
