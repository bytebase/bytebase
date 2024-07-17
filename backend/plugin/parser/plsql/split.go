package plsql

import (
	"strings"

	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_ORACLE, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_DM, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_OCEANBASE_ORACLE, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.SingleSQL, error) {
	tree, tokens, err := ParsePLSQL(statement)
	if err != nil {
		return nil, err
	}

	byteOffsetStart := 0
	prevStopTokenIndex := -1
	var result []base.SingleSQL
	for _, item := range tree.GetChildren() {
		if stmt, ok := item.(parser.IUnit_statementContext); ok {
			text := ""
			lastLine := 0
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
				text = tokens.GetTextFromTokens(stmt.GetStart(), lastToken)
				text = strings.TrimRight(text, " \n\t;")
			}

			result = append(result, base.SingleSQL{
				Text:            text,
				LastLine:        lastLine,
				Empty:           base.IsEmpty(tokens.GetAllTokens()[stmt.GetStart().GetTokenIndex():stmt.GetStop().GetTokenIndex()+1], parser.PlSqlParserSEMICOLON),
				ByteOffsetStart: byteOffsetStart,
				ByteOffsetEnd:   byteOffsetEnd,
			})
			byteOffsetStart = byteOffsetEnd
			prevStopTokenIndex = stmt.GetStop().GetTokenIndex()
		}
	}
	return result, nil
}

func SplitSQLWithoutModify(statement string) ([]base.SingleSQL, error) {
	tree, tokens, err := ParsePLSQL(statement)
	if err != nil {
		return nil, err
	}

	byteOffsetStart := 0
	prevStopTokenIndex := -1
	var result []base.SingleSQL
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

			stopIndex := stmt.GetStop().GetTokenIndex()
			lastToken := tokens.Get(stopIndex)
			lastLine = lastToken.GetLine()
			lastColumn = lastToken.GetColumn()

			result = append(result, base.SingleSQL{
				Text:            text,
				LastLine:        lastLine,
				LastColumn:      lastColumn,
				Empty:           base.IsEmpty(tokens.GetAllTokens()[stmt.GetStart().GetTokenIndex():stmt.GetStop().GetTokenIndex()+1], parser.PlSqlParserSEMICOLON),
				ByteOffsetStart: byteOffsetStart,
				ByteOffsetEnd:   byteOffsetEnd,
			})
			byteOffsetStart = byteOffsetEnd
			prevStopTokenIndex = stmt.GetStop().GetTokenIndex()
		}
	}
	return result, nil
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
	}
	return false
}
