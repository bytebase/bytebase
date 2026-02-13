package milvus

import (
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// SplitSQL splits statements by semicolon.
// It keeps statement ranges/positions stable for editor features.
func SplitSQL(statement string) ([]base.Statement, error) {
	var (
		stmts      []base.Statement
		startByte  int
		currentPos int
	)

	for currentPos < len(statement) {
		if statement[currentPos] == ';' {
			endByte := currentPos + 1
			stmts = append(stmts, makeStatement(statement, startByte, endByte))
			startByte = endByte
		}
		currentPos++
	}

	if startByte < len(statement) {
		stmts = append(stmts, makeStatement(statement, startByte, len(statement)))
	}
	return stmts, nil
}

func makeStatement(full string, startByte, endByte int) base.Statement {
	text := full[startByte:endByte]
	startLine, startColumn := base.CalculateLineAndColumn(full, startByte)
	endLine, endColumn := base.CalculateLineAndColumn(full, endByte)
	return base.Statement{
		Text:  text,
		Empty: strings.TrimSpace(strings.TrimSuffix(text, ";")) == "",
		Start: &storepb.Position{
			Line:   int32(startLine + 1),
			Column: int32(startColumn + 1),
		},
		End: &storepb.Position{
			Line:   int32(endLine + 1),
			Column: int32(endColumn + 1),
		},
		Range: &storepb.Range{
			Start: int32(startByte),
			End:   int32(endByte),
		},
	}
}
