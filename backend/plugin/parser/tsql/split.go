package tsql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_MSSQL, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.SingleSQL, error) {
	r, err := splitByParser(statement)
	if err != nil {
		// Fall back to semi split.
		return splitBySemi(statement)
	}
	return r, err
}

func splitBySemi(statement string) ([]base.SingleSQL, error) {
	t := tokenizer.NewTokenizer(statement)
	list, err := t.SplitStandardMultiSQL()
	if err != nil {
		return nil, err
	}
	return list, nil
}

func splitByParser(statement string) ([]base.SingleSQL, error) {
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewTSqlLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewTSqlParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &base.ParseErrorListener{}
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Tsql_file()

	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	var result []base.SingleSQL
	tokens := stream.GetAllTokens()
	start := 0

	if len(tree.AllBatch_without_go()) == 0 {
		// Go statement only.
		for _, goStmt := range tree.AllGo_statement() {
			pos := goStmt.GetStop().GetTokenIndex()
			line, col := base.FirstDefaultChannelTokenPosition(tokens[start : pos+1])
			result = append(result, base.SingleSQL{
				Text:                 stream.GetTextFromTokens(tokens[start], tokens[pos]),
				BaseLine:             tokens[start].GetLine() - 1,
				LastLine:             tokens[pos].GetLine() - 1,
				LastColumn:           tokens[pos].GetColumn(),
				FirstStatementLine:   line,
				FirstStatementColumn: col,
				Empty:                false,
			})
			start = pos + 1
		}
		return result, nil
	}

	// First batch_without_go.
	b := tree.AllBatch_without_go()[0]
	var r []base.SingleSQL
	r, start = splitBatchWithoutGo(b, tokens, stream, start)
	result = append(result, r...)

	goIdx := 0
	if len(tree.AllBatch_without_go()) > 1 {
		bs := tree.AllBatch_without_go()[1:]
		for _, b := range bs {
			// Find all go statement before the current batch_without_go.
			var goStmts []parser.IGo_statementContext
			for _, goStmt := range tree.AllGo_statement()[goIdx:] {
				if goStmt.GetStop().GetTokenIndex() < b.GetStart().GetTokenIndex() {
					goStmts = append(goStmts, goStmt)
					goIdx++
					continue
				}
				break
			}

			for _, goStmt := range goStmts {
				pos := goStmt.GetStop().GetTokenIndex()
				line, col := base.FirstDefaultChannelTokenPosition(tokens[start : pos+1])
				result = append(result, base.SingleSQL{
					Text:                 stream.GetTextFromTokens(tokens[start], tokens[pos]),
					BaseLine:             tokens[start].GetLine() - 1,
					LastLine:             tokens[pos].GetLine() - 1,
					LastColumn:           tokens[pos].GetColumn(),
					FirstStatementLine:   line,
					FirstStatementColumn: col,
					Empty:                false,
				})
				start = pos + 1
			}

			r, start = splitBatchWithoutGo(b, tokens, stream, start)
			result = append(result, r...)
		}
	}

	if goIdx < len(tree.AllGo_statement()) {
		// Last go statement.
		for _, goStmt := range tree.AllGo_statement()[goIdx:] {
			pos := goStmt.GetStop().GetTokenIndex()
			line, col := base.FirstDefaultChannelTokenPosition(tokens[start : pos+1])
			result = append(result, base.SingleSQL{
				Text:                 stream.GetTextFromTokens(tokens[start], tokens[pos]),
				BaseLine:             tokens[start].GetLine() - 1,
				LastLine:             tokens[pos].GetLine() - 1,
				LastColumn:           tokens[pos].GetColumn(),
				FirstStatementLine:   line,
				FirstStatementColumn: col,
				Empty:                false,
			})
			start = pos + 1
		}
	}

	return result, nil
}

func splitBatchWithoutGo(b parser.IBatch_without_goContext, tokens []antlr.Token, stream *antlr.CommonTokenStream, start int) ([]base.SingleSQL, int) {
	var result []base.SingleSQL
	switch {
	case b.Batch_level_statement() == nil && b.Execute_body_batch() == nil:
		// All sql_clauses.
		for _, sqlClause := range b.AllSql_clauses() {
			pos := sqlClause.GetStop().GetTokenIndex()
			line, col := base.FirstDefaultChannelTokenPosition(tokens[start : pos+1])
			result = append(result, base.SingleSQL{
				Text:                 stream.GetTextFromTokens(tokens[start], tokens[pos]),
				BaseLine:             tokens[start].GetLine() - 1,
				LastLine:             tokens[pos].GetLine() - 1,
				LastColumn:           tokens[pos].GetColumn(),
				FirstStatementLine:   line,
				FirstStatementColumn: col,
				Empty:                false,
			})
			start = pos + 1
		}
	case b.Batch_level_statement() != nil:
		pos := b.Batch_level_statement().GetStop().GetTokenIndex()
		line, col := base.FirstDefaultChannelTokenPosition(tokens[start : pos+1])
		result = append(result, base.SingleSQL{
			Text:                 stream.GetTextFromTokens(tokens[start], tokens[pos]),
			BaseLine:             tokens[start].GetLine() - 1,
			LastLine:             tokens[pos].GetLine() - 1,
			LastColumn:           tokens[pos].GetColumn(),
			FirstStatementLine:   line,
			FirstStatementColumn: col,
			Empty:                false,
		})
		start = pos + 1
	case b.Execute_body_batch() != nil:
		pos := b.Execute_body_batch().GetStop().GetTokenIndex()
		line, col := base.FirstDefaultChannelTokenPosition(tokens[start : pos+1])
		result = append(result, base.SingleSQL{
			Text:                 stream.GetTextFromTokens(tokens[start], tokens[pos]),
			BaseLine:             tokens[start].GetLine() - 1,
			LastLine:             tokens[pos].GetLine() - 1,
			LastColumn:           tokens[pos].GetColumn(),
			FirstStatementLine:   line,
			FirstStatementColumn: col,
			Empty:                false,
		})
		start = pos + 1
		for _, sqlClause := range b.AllSql_clauses() {
			pos := sqlClause.GetStop().GetTokenIndex()
			line, col := base.FirstDefaultChannelTokenPosition(tokens[start : pos+1])
			result = append(result, base.SingleSQL{
				Text:                 stream.GetTextFromTokens(tokens[start], tokens[pos]),
				BaseLine:             tokens[start].GetLine() - 1,
				LastLine:             tokens[pos].GetLine() - 1,
				LastColumn:           tokens[pos].GetColumn(),
				FirstStatementLine:   line,
				FirstStatementColumn: col,
				Empty:                false,
			})
			start = pos + 1
		}
	}
	return result, start
}
