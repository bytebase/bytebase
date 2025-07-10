package doris

import (
	parser "github.com/bytebase/doris-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_STARROCKS, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_DORIS, SplitSQL)
}

func SplitSQL(statement string) ([]base.SingleSQL, error) {
	parseResult, err := ParseDorisSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.SingleSQL
	tree, ok := parseResult.Tree.(*parser.SqlStatementsContext)
	if !ok {
		return nil, errors.Errorf("failed to cast tree to SqlStatementsContext")
	}
	tokens := parseResult.Tokens.GetAllTokens()
	stream := parseResult.Tokens

	start := 0
	for _, singleStatement := range tree.AllSingleStatement() {
		var pos int
		if singleStatement.SEMICOLON() != nil {
			pos = singleStatement.SEMICOLON().GetSymbol().GetTokenIndex()
		} else {
			pos = singleStatement.GetStop().GetTokenIndex()
		}

		antlrPosition := base.FirstDefaultChannelTokenPosition(tokens[start : pos+1])
		// From antlr4, the line is ONE based, and the column is ZERO based.
		// So we should minus 1 for the line.
		result = append(result, base.SingleSQL{
			Text:     stream.GetTextFromTokens(tokens[start], tokens[pos]),
			BaseLine: tokens[start].GetLine() - 1,
			End: common.ConvertANTLRPositionToPosition(&common.ANTLRPosition{
				Line:   int32(tokens[pos].GetLine()),
				Column: int32(tokens[pos].GetColumn()),
			}, statement),
			Start: common.ConvertANTLRPositionToPosition(antlrPosition, statement),
			Empty: singleStatement.Statement() == nil,
		})
		start = pos + 1
	}
	return result, nil
}
