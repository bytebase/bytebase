package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*TableMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_TEXT_FIELDS_TOTAL_LENGTH, &TableMaximumVarcharLengthAdvisor{})
}

// TableMaximumVarcharLengthAdvisor checks the total text length of a table's
// columns. The judgment lives in omni
// (mysql/v1review.TableTextFieldsTotalLength); this adapter passes the schema
// after the change and renders the findings as advice.
type TableMaximumVarcharLengthAdvisor struct {
}

func (*TableMaximumVarcharLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}
	maximum := int(numberPayload.Number)
	// A nil schema makes omni panic where V1 did with no final metadata.
	var final *v1review.Schema
	if checkCtx.FinalMetadata != nil {
		final = &v1review.Schema{
			Metadata:            checkCtx.FinalMetadata.GetProto(),
			ObjectCaseSensitive: checkCtx.FinalMetadata.GetIsObjectCaseSensitive(),
		}
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		// V1 reported a CREATE TABLE under the index count code.
		adviceCode := code.TotalTextLengthExceedsLimit
		if _, ok := node.(*ast.CreateTableStmt); ok {
			adviceCode = code.IndexCountExceedsLimit
		}
		for _, f := range mysqlv1review.TableTextFieldsTotalLength([]ast.Node{node}, final, maximum) {
			if f.Code != v1review.TableTextTooLong {
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          adviceCode.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Table %q total text column length (%s) exceeds the limit (%d).", f.Name, f.Value, maximum),
				StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + findingLine(stmt.Text, f)},
			})
		}
	}
	return adviceList, nil
}
