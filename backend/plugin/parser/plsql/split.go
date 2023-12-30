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

	var result []base.SingleSQL
	for _, item := range tree.GetChildren() {
		if stmt, ok := item.(parser.IUnit_statementContext); ok {
			text := ""
			lastLine := 0
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
				Text:     text,
				LastLine: lastLine,
				Empty:    false,
			})
		}
	}
	return result, nil
}

// needSemicolon returns true if the given statement needs a semicolon.
// The go-ora driver requires semicolon for anonymous block,
// but does not support semicolon for other statements.
func needSemicolon(stmt parser.IUnit_statementContext) bool {
	return stmt.Anonymous_block() != nil
}
