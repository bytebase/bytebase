package mariadb

import (
	"context"

	"github.com/bytebase/omni/mariadb/analysis"
	mariadbparser "github.com/bytebase/omni/mariadb/parser"

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
		mariadbparser.Split(statement),
		func(s mariadbparser.Segment) (int, int) { return s.ByteStart, s.ByteEnd },
		backups,
		func(b analysis.BackupStatement) (string, string, string, int, int) {
			return b.Statement, b.SourceTable, b.BackupTable, b.ByteStart, b.ByteEnd
		},
	)
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

func wrapDatabase(d *base.UnnamedSchemaDatabase) analysis.Database { return d }

func newBackupCatalog(ctx context.Context, instanceID string, getDB base.GetDatabaseMetadataFunc, listDBs base.ListDatabaseNamesFunc) analysis.Catalog {
	list, get := base.BackupCatalogFuncs(ctx, instanceID, getDB, listDBs, wrapDatabase)
	return analysis.Catalog{ListDatabaseNames: list, GetDatabase: get}
}
