package spanner

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/googlesql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_SPANNER, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements using ANTLR parser.
// This handles BEGIN/END blocks, CASE, IF, LOOP, etc. correctly.
func SplitSQL(statement string) ([]base.Statement, error) {
	lexer := parser.NewGoogleSQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewGoogleSQLParser(stream)
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Root()
	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	if tree == nil || tree.Stmts() == nil {
		return nil, errors.New("failed to split multiple statements")
	}

	var result []base.Statement
	tokens := stream.GetAllTokens()

	// Get all statement-terminating semicolons from the parse tree
	allStatements := tree.Stmts().AllUnterminated_sql_statement()
	if len(allStatements) == 0 {
		return result, nil
	}

	byteOffset := 0
	start := 0
	for i, stmt := range allStatements {
		// Find the semicolon after this statement
		var endPos int
		if i < len(allStatements)-1 {
			// Not the last statement - find the semicolon between this and next statement
			nextStmt := allStatements[i+1]
			nextStmtStart := nextStmt.GetStart().GetTokenIndex()
			// Find the semicolon just before the next statement
			endPos = nextStmtStart - 1
			// Skip back to find the actual semicolon
			for endPos >= start && tokens[endPos].GetTokenType() != parser.GoogleSQLLexerSEMI_SYMBOL {
				endPos--
			}
		} else {
			// Last statement - may or may not have semicolon
			stmtStop := stmt.GetStop().GetTokenIndex()
			endPos = stmtStop
			// Check if there's a semicolon after the statement
			for endPos < len(tokens)-1 {
				endPos++
				if tokens[endPos].GetTokenType() == parser.GoogleSQLLexerSEMI_SYMBOL {
					break
				}
			}
		}

		if endPos < start {
			continue
		}

		stmtText := stream.GetTextFromTokens(tokens[start], tokens[endPos])
		stmtByteLength := len(stmtText)

		// Calculate start position from byte offset (first character of Text)
		startLine, startColumn := base.CalculateLineAndColumn(statement, byteOffset)
		result = append(result, base.Statement{
			Text: stmtText,
			Range: &storepb.Range{
				Start: int32(byteOffset),
				End:   int32(byteOffset + stmtByteLength),
			},
			End: common.ConvertANTLRTokenToExclusiveEndPosition(
				int32(tokens[endPos].GetLine()),
				int32(tokens[endPos].GetColumn()),
				tokens[endPos].GetText(),
			),
			Start: &storepb.Position{
				Line:   int32(startLine + 1),
				Column: int32(startColumn + 1),
			},
			Empty: base.IsEmpty(tokens[start:endPos+1], parser.GoogleSQLLexerSEMI_SYMBOL),
		})
		byteOffset += stmtByteLength
		start = endPos + 1
	}

	return result, nil
}
