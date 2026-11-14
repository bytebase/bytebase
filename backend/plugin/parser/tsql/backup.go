package tsql

import (
	"context"

	"github.com/bytebase/omni/mssql/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_MSSQL, TransformDMLToSelect)
}

func TransformDMLToSelect(_ context.Context, tCtx base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	stmts, err := parseScript(statement)
	if err != nil {
		return nil, errors.Wrap(errors.Wrap(err, "failed to parse statement"), "failed to prepare transformation")
	}
	backups, err := analysis.TransformDMLToSelect(stmts, sourceDatabase, targetDatabase, tablePrefix, tCtx.IsCaseSensitive)
	if err != nil {
		return nil, err
	}
	var result []base.BackupStatement
	for _, backup := range backups {
		result = append(result, base.BackupStatement{
			Statement:       backup.Statement,
			SourceSchema:    backup.SourceSchema,
			SourceTableName: backup.SourceTableName,
			TargetTableName: backup.TargetTableName,
			StartPosition:   convertPosition(backup.StartPosition),
			EndPosition:     convertPosition(backup.EndPosition),
		})
	}
	return result, nil
}

// parseScript splits and parses a script for omni, with the line each
// statement starts on.
func parseScript(statement string) ([]analysis.ParsedStatement, error) {
	parsedStatements, err := parseTSQLStatements(statement)
	if err != nil {
		return nil, err
	}
	var stmts []analysis.ParsedStatement
	for _, parsedStatement := range parsedStatements {
		node, _ := GetNode(parsedStatement.AST)
		startLine := 1
		if parsedStatement.Start != nil {
			startLine = int(parsedStatement.Start.Line)
		}
		stmts = append(stmts, analysis.ParsedStatement{Text: parsedStatement.Text, Node: node, StartLine: startLine})
	}
	return stmts, nil
}

func convertPosition(p analysis.Position) *storepb.Position {
	return &storepb.Position{Line: int32(p.Line), Column: int32(p.Column)}
}
