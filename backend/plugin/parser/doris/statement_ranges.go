package doris

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	parser "github.com/bytebase/doris-parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_DORIS, GetStatementRanges)
	base.RegisterStatementRangesFunc(storepb.Engine_STARROCKS, GetStatementRanges)
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	createLexer := func(input antlr.CharStream) antlr.Lexer {
		return parser.NewDorisSQLLexer(input)
	}
	stream := base.PrepareANTLRTokenStream(statement, createLexer)
	ranges := base.GetANTLRStatementRangesUTF16Position(stream, parser.DorisSQLParserEOF, parser.DorisSQLParserSEMICOLON)
	return ranges, nil
}
