package tidb

import (
	"context"

	"github.com/bytebase/omni/tidb/analysis"
	omniparser "github.com/bytebase/omni/tidb/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TransformDMLToSelect(ctx context.Context, tCtx base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	catalog := newBackupCatalog(ctx, tCtx.InstanceID, tCtx.GetDatabaseMetadataFunc, tCtx.ListDatabaseNamesFunc)
	backups, err := analysis.TransformDMLToSelect(statement, analysis.BackupOptions{
		DefaultDatabase:      sourceDatabase,
		BackupDatabase:       targetDatabase,
		TablePrefix:          tablePrefix,
		CaseSensitive:        tCtx.IsCaseSensitive,
		Catalog:              catalog,
		ReferencingDatabases: tCtx.ReferencingDatabases(ctx),
		Columns:              base.NewColumnClassifier(tCtx.GetDatabaseMetadataFunc, tCtx.InstanceID, catalog, tCtx.IsCaseSensitive, analysis.ClassifyColumns),
	})
	if err != nil || backups == nil {
		return nil, err
	}
	return base.ConvertBackupStatements(
		statement,
		SplitSQL,
		omniparser.Split(statement),
		func(s omniparser.Segment) (int, int) { return s.ByteStart, s.ByteEnd },
		backups,
		func(b analysis.BackupStatement) (string, string, string, int, int) {
			return b.Statement, b.SourceTable, b.BackupTable, b.ByteStart, b.ByteEnd
		},
	)
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

func wrapDatabase(d *base.UnnamedSchemaDatabase) analysis.Database { return d }

func newBackupCatalog(ctx context.Context, instanceID string, getDB base.GetDatabaseMetadataFunc, listDBs base.ListDatabaseNamesFunc) analysis.Catalog {
	list, get := base.BackupCatalogFuncs(ctx, instanceID, getDB, listDBs, wrapDatabase)
	return analysis.Catalog{ListDatabaseNames: list, GetDatabase: get}
}
