package plsql

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_ORACLE, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
// TODO(zp): Consolidate with split logic in ParsePLSQL?
func SplitSQL(statement string) ([]base.Statement, error) {
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

	byteOffsetStart := 0
	prevStopTokenIndex := -1
	var result []base.Statement
	for _, item := range tree.GetChildren() {
		if stmt, ok := item.(parser.IUnit_statementContext); ok {
			text := ""
			lastLine := 0
			lastColumn := 0
			// tokens looks like
			if startTokenIndex := stmt.GetStart().GetTokenIndex(); startTokenIndex-1 >= 0 && prevStopTokenIndex+1 <= startTokenIndex-1 {
				byteOffsetStart += len(tokens.GetTextFromTokens(tokens.Get(prevStopTokenIndex+1), tokens.Get(stmt.GetStart().GetTokenIndex()-1)))
			}
			byteOffsetEnd := byteOffsetStart + len(tokens.GetTextFromTokens(stmt.GetStart(), stmt.GetStop()))

			// The go-ora driver requires semicolon for anonymous block,
			// but does not support semicolon for other statements.
			if needSemicolon(stmt) {
				lastToken := tokens.Get(stmt.GetStop().GetTokenIndex())
				lastLine = lastToken.GetLine()
				lastColumn = lastToken.GetColumn()
				text = tokens.GetTextFromTokens(stmt.GetStart(), lastToken)
				if lastToken.GetTokenType() != parser.PlSqlParserSEMICOLON {
					text += ";"
				}
			} else {
				stopIndex := stmt.GetStop().GetTokenIndex()
				if stmt.GetStop().GetTokenType() == parser.PlSqlParserSEMICOLON {
					stopIndex--
				}
				lastToken := tokens.Get(stopIndex)
				lastLine = lastToken.GetLine()
				lastColumn = lastToken.GetColumn()
				text = tokens.GetTextFromTokens(stmt.GetStart(), lastToken)
				text = strings.TrimRightFunc(text, utils.IsSpaceOrSemicolon)
			}

			result = append(result, base.Statement{
				Text: text,
				// BaseLine is 0-based line offset of this statement in the original SQL
				Start: common.ConvertANTLRPositionToPosition(
					&common.ANTLRPosition{
						Line:   int32(stmt.GetStart().GetLine()),
						Column: int32(stmt.GetStart().GetColumn()),
					},
					statement,
				),
				End: common.ConvertANTLRPositionToPosition(
					&common.ANTLRPosition{
						Line:   int32(lastLine),
						Column: int32(lastColumn),
					},
					statement,
				),
				Empty: base.IsEmpty(tokens.GetAllTokens()[stmt.GetStart().GetTokenIndex():stmt.GetStop().GetTokenIndex()+1], parser.PlSqlParserSEMICOLON),
				Range: &storepb.Range{
					Start: int32(byteOffsetStart),
					End:   int32(byteOffsetEnd),
				},
			})
			byteOffsetStart = byteOffsetEnd
			prevStopTokenIndex = stmt.GetStop().GetTokenIndex()
		}
	}
	return result, nil
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
					End: common.ConvertANTLRPositionToPosition(
						&common.ANTLRPosition{
							Line:   int32(lastToken.GetLine()),
							Column: int32(lastToken.GetColumn()),
						},
						statement,
					),
					Empty: false,
				}
				continue
			}
			lastLine := 0
			lastColumn := 0

			stopIndex := stmt.GetStop().GetTokenIndex()
			lastToken := tokens.Get(stopIndex)
			lastLine = lastToken.GetLine()
			lastColumn = lastToken.GetColumn()

			result = append(result, base.Statement{
				Text: tokens.GetTextFromTokens(stmt.GetStart(), lastToken),
				End: common.ConvertANTLRPositionToPosition(
					&common.ANTLRPosition{
						Line:   int32(lastLine),
						Column: int32(lastColumn),
					},
					statement,
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

// needSemicolon returns true if the given statement needs a semicolon.
// The go-ora driver requires semicolon for anonymous block and create procedure/function/package/trigger type of statements,
// but does not support semicolon for other statements.
func needSemicolon(stmt parser.IUnit_statementContext) bool {
	switch {
	case stmt.Anonymous_block() != nil,
		stmt.Create_procedure_body() != nil,
		stmt.Create_function_body() != nil,
		stmt.Create_package() != nil,
		stmt.Create_package_body() != nil,
		stmt.Create_trigger() != nil:
		return true
	default:
		return false
	}
}
