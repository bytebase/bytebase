package tsql

import (
	"context"
	"strings"

	"github.com/bytebase/omni/mssql/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

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

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	stmts, err := base.ParseForRestore(statement, backupItem, parseScript)
	if err != nil {
		return "", err
	}

	dmls := base.FilterDMLsInRange(analysis.BackedUpDMLs(stmts), backupItem, func(dml analysis.BackedUpDML) (*storepb.Position, *storepb.Position) {
		return convertPosition(dml.StartPosition), convertPosition(dml.EndPosition)
	})
	if len(dmls) == 0 {
		return "", errors.New("no original SQL")
	}

	sqls := make([]string, 0, len(dmls))
	for _, dml := range dmls {
		sqls = append(sqls, dml.SQL)
	}
	// Join with semicolons and add a trailing semicolon.
	originalSQL := strings.Join(sqls, ";\n") + ";"
	if dmls[0].Node == nil {
		return "", base.StatementNotFoundError(backupItem)
	}

	sourceDatabase, targetDatabase, err := base.ParseBackupItemDatabases(backupItem)
	if err != nil {
		return "", err
	}
	tableMeta, err := base.ResolveRestoreTableMetadata(ctx, rCtx, sourceDatabase, backupItem.SourceTable, "dbo", func(string) error {
		return errors.New("database metadata is nil")
	})
	if err != nil {
		return "", err
	}

	return analysis.RestoreSQL(dmls[0].Node, analysis.RestoreOptions{
		OriginalDatabase: sourceDatabase,
		OriginalSchema:   tableMeta.Schema,
		OriginalTable:    backupItem.SourceTable.Table,
		BackupDatabase:   targetDatabase,
		BackupTable:      backupItem.TargetTable.Table,
		Table:            tableMeta.Table,
		PrimaryKey:       tableMeta.PrimaryKey,
		OriginalSQL:      originalSQL,
	})
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
