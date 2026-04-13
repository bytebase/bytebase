package tsql

import (
	"strings"

	omnimssql "github.com/bytebase/omni/mssql"
	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_MSSQL, SplitSQL)
}

// SplitSQL splits the given T-SQL script into statements.
//
// Strategy: use omni's mssql parser to obtain per-statement byte ranges, then
// emit one base.Statement per omni statement with strict byte-contiguity and
// position adjacency:
//
//  1. stmts[0].Range.Start == 0
//     stmts[i].Range.Start == stmts[i-1].Range.End          (byte contiguity)
//     stmts[0].Start       == {Line: 1, Column: 1}
//     stmts[i].Start       == stmts[i-1].End                (position adjacency)
//
//  2. `GO` batch separators become base.Statement entries with Empty=true.
//     The GO batch semantics live in the tsqlbatch.Batcher (which strips GO
//     before SplitSQL is called on the execution path); this function only
//     reports them so that span analysis paths (which consume raw user SQL
//     without the Batcher) can filter them cleanly via FilterEmptyStatements.
//
// On parser failure, falls back to the semicolon-based tokenizer splitter.
func SplitSQL(statement string) ([]base.Statement, error) {
	omniStmts, err := omnimssql.Parse(statement)
	if err != nil {
		return splitBySemi(statement)
	}
	return toBaseStatements(statement, omniStmts), nil
}

func splitBySemi(statement string) ([]base.Statement, error) {
	t := tokenizer.NewTokenizer(statement)
	return t.SplitStandardMultiSQL()
}

// toBaseStatements converts omni's statement list into []base.Statement while
// enforcing byte-contiguity and position adjacency. Any bytes that omni did
// not attribute to a statement (gaps) are absorbed into the following
// statement so the output remains contiguous.
func toBaseStatements(sql string, omniStmts []omnimssql.Statement) []base.Statement {
	if len(omniStmts) == 0 {
		return nil
	}
	result := make([]base.Statement, 0, len(omniStmts))
	prevEnd := &storepb.Position{Line: 1, Column: 1}
	prevByte := 0
	for _, os := range omniStmts {
		startByte := prevByte
		endByte := os.ByteEnd
		if endByte < startByte {
			endByte = startByte
		}
		text := sql[startByte:endByte]
		endLine, endCol := base.CalculateLineAndColumn(sql, endByte)
		endPos := &storepb.Position{
			Line:   int32(endLine + 1),
			Column: int32(endCol + 1),
		}

		_, isGo := os.AST.(*ast.GoStmt)
		empty := isGo || os.Empty() || strings.TrimSpace(text) == ""

		result = append(result, base.Statement{
			Text: text,
			Range: &storepb.Range{
				Start: int32(startByte),
				End:   int32(endByte),
			},
			Start: prevEnd,
			End:   endPos,
			Empty: empty,
		})
		prevEnd = endPos
		prevByte = endByte
	}
	return result
}
