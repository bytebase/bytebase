package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*TableDropNamingConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, &TableDropNamingConventionAdvisor{})
}

// TableDropNamingConventionAdvisor is the advisor checking for table drop
// with naming convention. The judgment lives in omni
// (pg/v1review.TableDropNaming); this adapter converts the rule's payload
// into the omni config and renders the findings as advice.
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

	cfg := v1review.NamingConfig{Pattern: namingPayload.Format}
	if _, err := cfg.Compile(); err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		findings, err := pgv1review.TableDropNaming([]ast.Node{node}, cfg)
		if err != nil {
			return nil, err
		}
		// V1 anchored every finding on the statement text's first
		// non-whitespace line, which can be a leading comment above the
		// statement node the finding's range covers.
		line := (&BaseRule{StmtText: stmt.Text}).ContentStartLine() + int32(stmt.BaseLine())
		for _, f := range findings {
			if f.Code != v1review.NameMismatch {
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.TableDropNamingConventionMismatch.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", f.Name, namingPayload.Format),
				StartPosition: &storepb.Position{Line: line},
			})
		}
	}
	return adviceList, nil
}
