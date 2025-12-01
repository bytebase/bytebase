package doris

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	parser "github.com/bytebase/parser/doris"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_DORIS, GetStatementRanges)
	base.RegisterStatementRangesFunc(storepb.Engine_STARROCKS, GetStatementRanges)
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	createLexer := func(input antlr.CharStream) antlr.Lexer {
		return parser.NewDorisLexer(input)
	}
	stream := base.PrepareANTLRTokenStream(statement, createLexer)
	ranges := base.GetANTLRStatementRangesUTF16Position(stream, parser.DorisParserEOF, parser.DorisParserSEMICOLON)
	return ranges, nil
}
