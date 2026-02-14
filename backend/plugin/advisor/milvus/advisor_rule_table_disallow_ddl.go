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

var (
	ddlActionRE = regexp.MustCompile(`(?is)^\s*(create|drop|alter|load|release)\s+(collection|partition|index|alias)\b`)

	destructiveDDLRE = regexp.MustCompile(`(?is)^\s*(drop|release)\s+(collection|partition|index|alias)\b`)
)

func init() {
	advisor.Register(storepb.Engine_MILVUS, storepb.SQLReviewRule_TABLE_DISALLOW_DDL, &TableDisallowDDLAdvisor{})
}

// TableDisallowDDLAdvisor blocks Milvus DDL/resource lifecycle operations for strict environments.
type TableDisallowDDLAdvisor struct{}

// Check checks whether the statement contains Milvus DDL/resource lifecycle operations.
func (*TableDisallowDDLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		if !ddlActionRE.MatchString(trimmed) {
			continue
		}
		content := fmt.Sprintf("Milvus DDL operation is not allowed by policy: %q", compactStatement(trimmed))
		if destructiveDDLRE.MatchString(trimmed) {
			content = fmt.Sprintf("Milvus destructive/disruptive operation is not allowed by policy: %q", compactStatement(trimmed))
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          advisorcode.TableDisallowDDL.Int32(),
			Title:         checkCtx.Rule.Type.String(),
			Content:       content,
			StartPosition: stmt.Start,
		})
	}
	return adviceList, nil
}
