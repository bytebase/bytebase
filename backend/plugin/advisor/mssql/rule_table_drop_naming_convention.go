package mssql

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, &TableDropNamingConventionAdvisor{})
}

type TableDropNamingConventionAdvisor struct{}

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

	rule := &tableDropNamingConventionRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
		format:       format,
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type tableDropNamingConventionRule struct {
	OmniBaseRule
	format *regexp.Regexp
}

func (*tableDropNamingConventionRule) Name() string {
	return "TableDropNamingConventionRule"
}

func (r *tableDropNamingConventionRule) OnStatement(node ast.Node) {
	drop, ok := node.(*ast.DropStmt)
	if !ok || drop.ObjectType != ast.DropTable {
		return
	}
	if drop.Names == nil {
		return
	}
	for _, item := range drop.Names.Items {
		ref, ok := item.(*ast.TableRef)
		if !ok || ref == nil {
			continue
		}
		tableName := strings.ToLower(ref.Object)
		if tableName == "" {
			continue
		}
		if !r.format.MatchString(tableName) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.TableDropNamingConventionMismatch.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("[%s] mismatches drop table naming convention, naming format should be %q", tableName, r.format),
				StartPosition: &storepb.Position{Line: r.LocToLine(drop.Loc)},
			})
		}
	}
}
