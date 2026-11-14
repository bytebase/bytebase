package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*IndexTotalNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, &IndexTotalNumberLimitAdvisor{})
}

// IndexTotalNumberLimitAdvisor is the advisor checking for index total number
// limit. The judgment lives in omni (pg/v1review.IndexTotalNumberLimit); this
// adapter passes the script and the simulated final schema and renders the
// findings as advice.
type IndexTotalNumberLimitAdvisor struct {
}

// Check checks for index total number limit.
func (*IndexTotalNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}
	limit := int(numberPayload.Number)

	stmts, nodes := scriptNodes(checkCtx.ParsedStatements)
	findings, err := pgv1review.IndexTotalNumberLimit(nodes, limit, schemaOf(checkCtx.FinalMetadata))
	if err != nil {
		readMissingSchema(checkCtx.FinalMetadata, err)
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range findings {
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Code:    code.IndexCountExceedsLimit.Int32(),
			Title:   title,
			Content: fmt.Sprintf("The count of index in table %q.%q should be no more than %d, but found %s", f.Schema, f.Table, limit, f.Value),
			StartPosition: &storepb.Position{
				Line: (&BaseRule{StmtText: stmt.Text}).ContentEndLine() + int32(stmt.BaseLine()),
			},
		})
	}
	return adviceList, nil
}

// schemaOf returns a snapshot as omni reads it; nil holds nothing.
func schemaOf(m *model.DatabaseMetadata) v1review.Schema {
	if m == nil {
		return v1review.Schema{}
	}
	return v1review.Schema{Metadata: m.GetProto(), ObjectCaseSensitive: m.GetIsObjectCaseSensitive()}
}

// readMissingSchema repeats the V1 advisors' unguarded read of a snapshot
// omni found missing, so a nil snapshot panics where it did and
// advisor.Check reports the panic as the rule's error.
func readMissingSchema(m *model.DatabaseMetadata, err error) {
	if errors.Is(err, v1review.ErrSchemaMissing) {
		m.GetSchemaMetadata("")
	}
}
