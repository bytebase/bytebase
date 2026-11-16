package mariadb

import (
	"context"
	"fmt"

	mariadbv1review "github.com/bytebase/omni/mariadb/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*IndexTypeNoBlobAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, &IndexTypeNoBlobAdvisor{})
}

// IndexTypeNoBlobAdvisor is the advisor checking for index type no blob. The
// judgment lives in omni (mariadb/v1review.IndexTypeNoBlob); this adapter
// passes the script and the pre-change schema and renders the findings as
// advice.
type IndexTypeNoBlobAdvisor struct {
}

// Check checks for index type no blob.
func (*IndexTypeNoBlobAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := scriptNodes(checkCtx.ParsedStatements)
	findings, err := mariadbv1review.IndexTypeNoBlob(nodes, schemaOf(checkCtx.OriginalMetadata))
	if err != nil {
		readMissingSchema(checkCtx.OriginalMetadata, err)
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range findings {
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.IndexTypeNoBlob.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("Columns in index must not be BLOB but `%s`.`%s` is %s", f.Table, f.Name, f.Value),
			StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
		})
	}
	return adviceList, nil
}
