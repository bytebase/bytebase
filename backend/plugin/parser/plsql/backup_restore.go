package plsql

import (
	"context"

	"github.com/bytebase/omni/oracle/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TransformDMLToSelect transforms DML statement to SELECT statement.
// For Oracle, we only consider the managed on schema mode.
func TransformDMLToSelect(_ context.Context, tCtx base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	opts := analysis.BackupOptions{
		SourceDatabase: sourceDatabase,
		TargetDatabase: targetDatabase,
		TablePrefix:    tablePrefix,
	}
	if version, ok := tCtx.Version.(*Version); ok {
		opts.Version = version
	}
	statements, err := analysis.TransformDMLToSelect(statement, opts)
	if err != nil {
		return nil, err
	}
	return base.ConvertOffsetBackupStatements(statement, statements, func(s analysis.BackupStatement) (string, string, string, string, int, int) {
		return s.Statement, s.SourceSchema, s.SourceTableName, s.TargetTableName, s.Start, s.End
	}, true), nil
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := base.ExtractSingleStatement(statement, backupItem, base.ExtractStatementOptions{
		SplitSQL:      SplitSQL,
		SplitErrMsg:   "failed to split SQL",
		Match:         analysis.ReferencesTable,
		MatchErrMsg:   "failed to prepare transformation",
		Normalize:     analysis.NormalizeRestoreSQL,
		JoinSeparator: "\n",
	})
	if err != nil {
		return "", err
	}

	node, err := analysis.FindFirstDML(originalSQL)
	if err != nil {
		return "", err
	}
	if node == nil {
		return "", errors.New("no DML statement found in extracted SQL")
	}

	sourceDatabase, targetDatabase, err := base.ParseBackupItemDatabases(backupItem)
	if err != nil {
		return "", err
	}
	tableMeta, err := base.ResolveRestoreTableMetadata(ctx, rCtx, sourceDatabase, backupItem.SourceTable, "", nil)
	if err != nil {
		return "", err
	}

	return analysis.RestoreSQL(originalSQL, node, analysis.RestoreOptions{
		SourceDatabase: sourceDatabase,
		SourceTable:    backupItem.SourceTable.Table,
		BackupDatabase: targetDatabase,
		BackupTable:    backupItem.TargetTable.Table,
		Table:          tableMeta.Table,
		PrimaryKey:     tableMeta.PrimaryKey,
	})
}
