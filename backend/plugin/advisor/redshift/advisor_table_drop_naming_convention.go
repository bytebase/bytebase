package redshift

import (
	"context"
	"fmt"

	redshiftast "github.com/bytebase/omni/redshift/ast"
	redshiftv1review "github.com/bytebase/omni/redshift/v1review"
	"github.com/bytebase/omni/v1review"
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
// The judgment lives in omni (redshift/v1review.TableDropNaming); this
// adapter converts the rule's payload into the omni config and renders the
// findings as advice.
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
	adviceList := []*storepb.Advice{}
	for _, stmt := range checkCtx.ParsedStatements {
		node, ok := parserredshift.GetNode(stmt.AST)
		if !ok {
			continue
		}
		findings, err := redshiftv1review.TableDropNaming([]redshiftast.Node{node}, cfg)
		if err != nil {
			return nil, err
		}
		for _, f := range findings {
			if f.Code != v1review.NameMismatch {
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Code:    code.TableDropNamingConventionMismatch.Int32(),
				Title:   title,
				Content: fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", f.Name, namingPayload.Format),
				// V1 anchored every finding on where the splitter starts the
				// statement, which can be whitespace or a comment before the
				// statement node the finding's range covers.
				StartPosition: stmt.Start,
			})
		}
	}
	return adviceList, nil
}
