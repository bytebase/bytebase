package tidb

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/tidb/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_TIDB, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if err := base.ValidateBackupItem(backupItem); err != nil {
		return "", err
	}
	sourceDatabase, err := base.ParseBackupSourceDatabase(backupItem)
	if err != nil {
		return "", err
	}

	// The target table's regular columns decide whether an unqualified SET
	// column changes it.
	if rCtx.GetDatabaseMetadataFunc == nil {
		return "", errors.Wrap(errors.New("GetDatabaseMetadataFunc is nil"), "failed to get target table columns")
	}
	catalog := newBackupCatalog(ctx, rCtx.InstanceID, rCtx.GetDatabaseMetadataFunc, rCtx.ListDatabaseNamesFunc)
	targetCols, err := analysis.NormalColumnSet(catalog, sourceDatabase, backupItem.SourceTable.Table)
	if err != nil {
		return "", errors.Wrap(err, "failed to get target table columns")
	}

	originalSQL, err := base.ExtractSingleStatement(statement, backupItem, base.ExtractStatementOptions{
		SplitSQL:       SplitSQL,
		ExclusiveEnd:   true,
		SourceDatabase: sourceDatabase,
		Match: func(stmtText, sourceDatabase, sourceTable string) (bool, error) {
			return analysis.MutatesTable(stmtText, sourceDatabase, sourceTable, targetCols)
		},
	})
	if err != nil {
		return "", err
	}

	// One backup item can bundle several same-table DMLs, so the restore
	// covers every one of them.
	matchingNodes, err := base.CheckRestoreDMLs(analysis.MatchingDMLs(originalSQL, sourceDatabase, backupItem.SourceTable.Table, targetCols))
	if err != nil {
		return "", err
	}

	targetDatabase, err := base.ParseBackupTargetDatabase(backupItem)
	if err != nil {
		return "", err
	}
	columns, err := base.ClassifyRestoreColumns(rCtx, catalog, backupItem.SourceTable, sourceDatabase, analysis.ClassifyColumns)
	if err != nil {
		return "", err
	}

	return analysis.RestoreSQL(matchingNodes, analysis.RestoreOptions{
		Statement:      originalSQL,
		Database:       sourceDatabase,
		Table:          backupItem.SourceTable.Table,
		BackupDatabase: targetDatabase,
		BackupTable:    backupItem.TargetTable.Table,
		Columns:        columns.Normal,
		Catalog:        catalog,
	})
}
