package milvus

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var dmlActionRE = regexp.MustCompile(`(?is)^\s*(insert|upsert|delete)\b`)

func init() {
	advisor.Register(storepb.Engine_MILVUS, storepb.SQLReviewRule_TABLE_DISALLOW_DML, &TableDisallowDMLAdvisor{})
}

// TableDisallowDMLAdvisor blocks Milvus data mutation operations.
type TableDisallowDMLAdvisor struct{}

// Check checks whether the statement contains Milvus data mutation operations.
func (*TableDisallowDMLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.Empty {
			continue
		}
		trimmed := strings.TrimSpace(stmt.Text)
		if !dmlActionRE.MatchString(trimmed) {
			continue
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          advisorcode.TableDisallowDML.Int32(),
			Title:         checkCtx.Rule.Type.String(),
			Content:       fmt.Sprintf("Milvus DML operation is not allowed by policy: %q", compactStatement(trimmed)),
			StartPosition: stmt.Start,
		})
	}
	return adviceList, nil
}
