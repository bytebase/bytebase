package plsql

import (
	"github.com/bytebase/omni/oracle/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_ORACLE, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
//
// It uses omni's lexical Oracle splitter, which handles strings, comments,
// SQL*Plus separators, and PL/SQL blocks without requiring valid SQL.
func SplitSQL(statement string) ([]base.Statement, error) {
	return convertStatements(statement, analysis.SplitStatements(statement)), nil
}

func SplitSQLForCompletion(statement string) ([]base.Statement, error) {
	return convertStatements(statement, analysis.SplitStatementsForCompletion(statement)), nil
}

func convertStatements(statement string, statements []analysis.Statement) []base.Statement {
	result := make([]base.Statement, 0, len(statements))
	positionMapper := base.NewByteOffsetPositionMapper(statement)
	for _, stmt := range statements {
		result = append(result, base.Statement{
			Text:  stmt.Text,
			Start: positionMapper.Position(stmt.Start),
			End:   positionMapper.Position(stmt.End),
			Empty: stmt.Empty,
			Range: &storepb.Range{
				Start: int32(stmt.Start),
				End:   int32(stmt.End),
			},
		})
	}
	return result
}
