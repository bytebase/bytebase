package mariadb

import (
	"context"

	"github.com/bytebase/omni/mariadb/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_MARIADB, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := base.ExtractSingleStatement(statement, backupItem, base.ExtractStatementOptions{
		SplitSQL: SplitSQL,
		Match:    analysis.ReferencesTable,
	})
	if err != nil {
		return "", err
	}

	matchingNodes, err := base.CheckRestoreDMLs(analysis.DMLs(originalSQL))
	if err != nil {
		return "", err
	}

	sourceDatabase, targetDatabase, err := base.ParseBackupItemDatabases(backupItem)
	if err != nil {
		return "", err
	}
	catalog := newBackupCatalog(ctx, rCtx.InstanceID, rCtx.GetDatabaseMetadataFunc, rCtx.ListDatabaseNamesFunc)
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
		CaseSensitive:  rCtx.IsCaseSensitive,
	})
}
