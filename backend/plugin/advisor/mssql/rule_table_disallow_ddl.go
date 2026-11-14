package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mssql/ast"
	mssqlv1review "github.com/bytebase/omni/mssql/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_DISALLOW_DDL, &TableDisallowDDLAdvisor{})
}

// TableDisallowDDLAdvisor checks for DDL on the tables the rule lists. The
// judgment lives in omni (mssql/v1review.TableDisallowDDL); this adapter
// renders the findings as advice.
type TableDisallowDDLAdvisor struct{}

func (*TableDisallowDDLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for table disallow DDL rule")
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := tsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range mssqlv1review.TableDisallowDDL([]ast.Node{node}, stringArrayPayload.List) {
			if f.Code != v1review.TableDisallowed {
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.TableDisallowDDL.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("DDL is disallowed on table %s.", f.Name),
				StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + findingLine(stmt.Text, f)},
			})
		}
	}
	return adviceList, nil
}
