package pg

import (
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// statementFindingLine returns the line the V1 statement advisors placed a
// finding on: the statement text's first line that holds more than white
// space, which can be a comment above the statement.
func statementFindingLine(stmt base.ParsedStatement) int32 {
	return int32(stmt.BaseLine()) + (&BaseRule{StmtText: stmt.Text}).ContentStartLine()
}

// statementTrimmedText returns the statement's text as the V1 statement
// advisors printed it: without surrounding white space and trailing
// semicolons.
func statementTrimmedText(stmt base.ParsedStatement) string {
	return (&BaseRule{StmtText: stmt.Text}).TrimmedStmtText()
}
