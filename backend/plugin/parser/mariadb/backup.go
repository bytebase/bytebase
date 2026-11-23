package mariadb

import (
	"context"

	"github.com/bytebase/omni/mariadb/analysis"
	"github.com/bytebase/omni/mariadb/ast"
	mariadbparser "github.com/bytebase/omni/mariadb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_MARIADB, TransformDMLToSelect)
}

type StatementType = analysis.DMLType

const (
	StatementTypeUnknown = analysis.DMLTypeUnknown
	StatementTypeUpdate  = analysis.DMLTypeUpdate
	StatementTypeInsert  = analysis.DMLTypeInsert
	StatementTypeDelete  = analysis.DMLTypeDelete
)

type TableReference = analysis.TableReference

type StatementInfo = analysis.DMLTarget

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

// ExtractTables extracts table references from a DML statement. dbMetadata,
// which may be nil, is databaseName's metadata.
func ExtractTables(databaseName string, node ast.Node, fullSQL string, offset int, dbMetadata *model.DatabaseMetadata) ([]StatementInfo, error) {
	return analysis.ExtractTables(databaseName, node, fullSQL, offset, analysis.Catalog{
		GetDatabase: base.SingleUnnamedSchemaGetter(databaseName, dbMetadata, wrapDatabase),
	})
}

func wrapDatabase(d *base.UnnamedSchemaDatabase) analysis.Database { return d }

func newBackupCatalog(ctx context.Context, instanceID string, getDB base.GetDatabaseMetadataFunc, listDBs base.ListDatabaseNamesFunc) analysis.Catalog {
	list, get := base.BackupCatalogFuncs(ctx, instanceID, getDB, listDBs, wrapDatabase)
	return analysis.Catalog{ListDatabaseNames: list, GetDatabase: get}
}
