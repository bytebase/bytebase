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
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*NamingTableConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableConventionAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming
// convention. The judgment lives in omni (pg/v1review.NamingTable); this
// adapter converts the rule's payload into the omni config and renders the
// findings as advice.
type NamingTableConventionAdvisor struct {
}

// Check checks for table naming convention.
func (*NamingTableConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for naming table rule")
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}
	cfg := v1review.NamingConfig{Pattern: namingPayload.Format, MaxLength: maxLength}
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
		findings, err := pgv1review.NamingTable([]ast.Node{node}, cfg)
		if err != nil {
			return nil, err
		}
		// V1 anchored every finding on the statement text's first
		// non-whitespace line, which can be a leading comment above the
		// statement node the finding's range covers.
		line := stmt.BaseLine() + int((&BaseRule{StmtText: stmt.Text}).ContentStartLine())
		for _, f := range findings {
			var content string
			switch f.Code {
			case v1review.NameMismatch:
				content = fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, f.Name, namingPayload.Format)
			case v1review.NameTooLong:
				content = fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", f.Name, maxLength)
			default:
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NamingTableConventionMismatch.Int32(),
				Title:         title,
				Content:       content,
				StartPosition: base.ConvertANTLRLineToPosition(line),
			})
		}
	}
	return adviceList, nil
}
