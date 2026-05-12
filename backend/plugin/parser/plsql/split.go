package plsql

import (
	"github.com/antlr4-go/antlr/v4"
	oracleparser "github.com/bytebase/omni/oracle/parser"
	parser "github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_ORACLE, SplitSQL)
}

// consumeTrailingSemicolon walks forward from stopIdx through hidden-channel
// tokens (whitespace, comments) looking for a trailing ';' that belongs to the
// statement ending at stopIdx. Returns the index of the ';' if found before any
// default-channel token, otherwise stopIdx (no consumption). ParsePLSQL uses
// this to avoid BYT-9367's secondary AST-classification leak.
func consumeTrailingSemicolon(allTokens []antlr.Token, stopIdx int) int {
	for nextIdx := stopIdx + 1; nextIdx < len(allTokens); nextIdx++ {
		next := allTokens[nextIdx]
		if next.GetTokenType() == parser.PlSqlParserSEMICOLON {
			return nextIdx
		}
		if next.GetChannel() == antlr.TokenDefaultChannel {
			return stopIdx
		}
	}
	return stopIdx
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
//
// It uses omni's lexical Oracle splitter, which handles strings, comments,
// SQL*Plus separators, and PL/SQL blocks without requiring valid SQL.
func SplitSQL(statement string) ([]base.Statement, error) {
	segments := oracleparser.Split(statement)

	result := make([]base.Statement, 0, len(segments))
	for _, seg := range segments {
		if seg.Kind == oracleparser.SegmentSQLPlusCommand {
			continue
		}
		byteEnd := seg.ByteStart + trimOracleSegmentTrailingHidden(seg.Text)
		text := statement[seg.ByteStart:byteEnd]
		result = append(result, base.Statement{
			Text:  text,
			Start: ByteOffsetToRunePosition(statement, seg.ByteStart),
			End:   ByteOffsetToRunePosition(statement, byteEnd),
			Empty: seg.Empty(),
			Range: &storepb.Range{
				Start: int32(seg.ByteStart),
				End:   int32(byteEnd),
			},
		})
	}
	return result, nil
}

func trimOracleSegmentTrailingHidden(text string) int {
	lexer := oracleparser.NewLexer(text)
	lastTokenEnd := 0
	for {
		token := lexer.NextToken()
		if token.Type == 0 {
			break
		}
		lastTokenEnd = token.End
	}
	if lexer.Err != nil || lastTokenEnd == 0 {
		return len(text)
	}
	return lastTokenEnd
}

func SplitSQLForCompletion(statement string) ([]base.Statement, error) {
	tree, stream, err := ParsePLSQLForStringsManipulation(statement)
	if err != nil {
		return nil, err
	}
	if tree == nil {
		return nil, nil
	}
	tokens, ok := stream.(*antlr.CommonTokenStream)
	if !ok {
		return nil, nil
	}

	var result []base.Statement
	for _, item := range tree.GetChildren() {
		if stmt, ok := item.(parser.IUnit_statementContext); ok {
			if isCallStatement(item) && len(result) > 0 {
				lastResult := result[len(result)-1]
				stopIndex := stmt.GetStop().GetTokenIndex()
				lastToken := tokens.Get(stopIndex)
				result[len(result)-1] = base.Statement{
					Text: lastResult.Text + tokens.GetTextFromTokens(stmt.GetStart(), lastToken),
					End: common.ConvertANTLRTokenToExclusiveEndPosition(
						int32(lastToken.GetLine()),
						int32(lastToken.GetColumn()),
						lastToken.GetText(),
					),
					Empty: false,
				}
				continue
			}

			stopIndex := stmt.GetStop().GetTokenIndex()
			lastToken := tokens.Get(stopIndex)

			result = append(result, base.Statement{
				Text: tokens.GetTextFromTokens(stmt.GetStart(), lastToken),
				End: common.ConvertANTLRTokenToExclusiveEndPosition(
					int32(lastToken.GetLine()),
					int32(lastToken.GetColumn()),
					lastToken.GetText(),
				),
				Empty: base.IsEmpty(tokens.GetAllTokens()[stmt.GetStart().GetTokenIndex():stmt.GetStop().GetTokenIndex()+1], parser.PlSqlParserSEMICOLON),
			})
		}
	}
	return result, nil
}

func isCallStatement(item antlr.Tree) bool {
	unitStmt, ok := item.(parser.IUnit_statementContext)
	if !ok {
		return false
	}
	// BYT-8268: Changed from Call_statement to Sql_call_statement
	return unitStmt.Sql_call_statement() != nil
}
