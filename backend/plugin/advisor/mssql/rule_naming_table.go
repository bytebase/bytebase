package mssql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mssql/ast"
	mssqlv1review "github.com/bytebase/omni/mssql/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*NamingTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableAdvisor{})
}

// NamingTableAdvisor is the advisor checking for table naming convention.
// The judgment lives in omni (mssql/v1review.NamingTable); this adapter
// converts the rule's payload into the omni config and renders the findings
// as advice.
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
		node, ok := tsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		findings, err := mssqlv1review.NamingTable([]ast.Node{node}, cfg)
		if err != nil {
			return nil, err
		}
		for _, f := range findings {
			var content string
			switch f.Code {
			case v1review.NameMismatch:
				content = fmt.Sprintf(`%s mismatches table naming convention, naming format should be %q`, f.Name, namingPayload.Format)
			case v1review.NameTooLong:
				content = fmt.Sprintf(`%s mismatches table naming convention, its length should be within %d characters`, f.Name, maxLength)
			default:
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NamingTableConventionMismatch.Int32(),
				Title:         title,
				Content:       content,
				StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + findingLine(stmt.Text, f)},
			})
		}
	}
	return adviceList, nil
}

// findingLine is the 1-based line of a finding within the statement text: the
// line its range starts on, or the statement's first content line when the
// AST kept no range.
func findingLine(text string, f v1review.Finding) int32 {
	offset := f.Start
	if f.Start == 0 && f.End == 0 {
		if i := strings.IndexFunc(text, func(c rune) bool {
			return c != ' ' && c != '\t' && c != '\n' && c != '\r'
		}); i > 0 {
			offset = i
		}
	}
	return tsqlparser.ByteOffsetToRunePosition(text, offset).Line
}
