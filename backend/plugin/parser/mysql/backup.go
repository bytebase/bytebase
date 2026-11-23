package mysql

import (
	"context"

	"github.com/bytebase/omni/mysql/analysis"
	mysqlparser "github.com/bytebase/omni/mysql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_MYSQL, TransformDMLToSelect)
}

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
		mysqlparser.Split(statement),
		func(s mysqlparser.Segment) (int, int) { return s.ByteStart, s.ByteEnd },
		backups,
		func(b analysis.BackupStatement) (string, string, string, int, int) {
			return b.Statement, b.SourceTable, b.BackupTable, b.ByteStart, b.ByteEnd
		},
	)
}

func wrapDatabase(d *base.UnnamedSchemaDatabase) analysis.Database { return d }

func newBackupCatalog(ctx context.Context, instanceID string, getDB base.GetDatabaseMetadataFunc, listDBs base.ListDatabaseNamesFunc) analysis.Catalog {
	list, get := base.BackupCatalogFuncs(ctx, instanceID, getDB, listDBs, wrapDatabase)
	return analysis.Catalog{ListDatabaseNames: list, GetDatabase: get}
}
