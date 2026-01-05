package tsql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_MSSQL, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	r, err := splitByParser(statement)
	if err != nil {
		// Fall back to semi split.
		return splitBySemi(statement)
	}
	return r, err
}

func splitBySemi(statement string) ([]base.Statement, error) {
	t := tokenizer.NewTokenizer(statement)
	list, err := t.SplitStandardMultiSQL()
	if err != nil {
		return nil, err
	}
	return list, nil
}

func splitByParser(statement string) ([]base.Statement, error) {
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewTSqlLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewTSqlParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Tsql_file()

	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	var result []base.Statement
	tokens := stream.GetAllTokens()
	start := 0
	byteOffset := 0

	if len(tree.AllBatch_without_go()) == 0 {
		// Go statement only.
		for _, goStmt := range tree.AllGo_statement() {
			pos := goStmt.GetStop().GetTokenIndex()
			stmtText := stream.GetTextFromTokens(tokens[start], tokens[pos])
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
					int32(tokens[pos].GetLine()),
					int32(tokens[pos].GetColumn()),
					tokens[pos].GetText(),
				),
				Start: &storepb.Position{
					Line:   int32(startLine + 1),
					Column: int32(startColumn + 1),
				},
				Empty: false,
			})
			byteOffset += stmtByteLength
			start = pos + 1
		}
		return result, nil
	}

	// First batch_without_go.
	b := tree.AllBatch_without_go()[0]
	var r []base.Statement
	r, start, byteOffset = splitBatchWithoutGo(b, tokens, stream, start, byteOffset, statement)
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
				stmtText := stream.GetTextFromTokens(tokens[start], tokens[pos])
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
						int32(tokens[pos].GetLine()),
						int32(tokens[pos].GetColumn()),
						tokens[pos].GetText(),
					),
					Start: &storepb.Position{
						Line:   int32(startLine + 1),
						Column: int32(startColumn + 1),
					},
					Empty: false,
				})
				byteOffset += stmtByteLength
				start = pos + 1
			}

			r, start, byteOffset = splitBatchWithoutGo(b, tokens, stream, start, byteOffset, statement)
			result = append(result, r...)
		}
	}

	if goIdx < len(tree.AllGo_statement()) {
		// Last go statement.
		for _, goStmt := range tree.AllGo_statement()[goIdx:] {
			pos := goStmt.GetStop().GetTokenIndex()
			stmtText := stream.GetTextFromTokens(tokens[start], tokens[pos])
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
					int32(tokens[pos].GetLine()),
					int32(tokens[pos].GetColumn()),
					tokens[pos].GetText(),
				),
				Start: &storepb.Position{
					Line:   int32(startLine + 1),
					Column: int32(startColumn + 1),
				},
				Empty: false,
			})
			byteOffset += stmtByteLength
			start = pos + 1
		}
	}

	return result, nil
}

func splitBatchWithoutGo(b parser.IBatch_without_goContext, tokens []antlr.Token, stream *antlr.CommonTokenStream, start int, byteOffset int, statement string) ([]base.Statement, int, int) {
	var result []base.Statement
	switch {
	case b.Batch_level_statement() == nil && b.Execute_body_batch() == nil:
		// All sql_clauses.
		for _, sqlClause := range b.AllSql_clauses() {
			pos := sqlClause.GetStop().GetTokenIndex()
			stmtText := stream.GetTextFromTokens(tokens[start], tokens[pos])
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
					int32(tokens[pos].GetLine()),
					int32(tokens[pos].GetColumn()),
					tokens[pos].GetText(),
				),
				Start: &storepb.Position{
					Line:   int32(startLine + 1),
					Column: int32(startColumn + 1),
				},
				Empty: false,
			})
			byteOffset += stmtByteLength
			start = pos + 1
		}
	case b.Batch_level_statement() != nil:
		pos := b.Batch_level_statement().GetStop().GetTokenIndex()
		stmtText := stream.GetTextFromTokens(tokens[start], tokens[pos])
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
				int32(tokens[pos].GetLine()),
				int32(tokens[pos].GetColumn()),
				tokens[pos].GetText(),
			),
			Start: &storepb.Position{
				Line:   int32(startLine + 1),
				Column: int32(startColumn + 1),
			},
			Empty: false,
		})
		byteOffset += stmtByteLength
		start = pos + 1
	case b.Execute_body_batch() != nil:
		pos := b.Execute_body_batch().GetStop().GetTokenIndex()
		stmtText := stream.GetTextFromTokens(tokens[start], tokens[pos])
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
				int32(tokens[pos].GetLine()),
				int32(tokens[pos].GetColumn()),
				tokens[pos].GetText(),
			),
			Start: &storepb.Position{
				Line:   int32(startLine + 1),
				Column: int32(startColumn + 1),
			},
			Empty: false,
		})
		byteOffset += stmtByteLength
		start = pos + 1
		for _, sqlClause := range b.AllSql_clauses() {
			pos := sqlClause.GetStop().GetTokenIndex()
			stmtText := stream.GetTextFromTokens(tokens[start], tokens[pos])
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
					int32(tokens[pos].GetLine()),
					int32(tokens[pos].GetColumn()),
					tokens[pos].GetText(),
				),
				Start: &storepb.Position{
					Line:   int32(startLine + 1),
					Column: int32(startColumn + 1),
				},
				Empty: false,
			})
			byteOffset += stmtByteLength
			start = pos + 1
		}
	default:
		// No statements found in this batch
	}
	return result, start, byteOffset
}
