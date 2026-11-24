package pg

import (
	"context"
	"strings"

	omnipg "github.com/bytebase/omni/pg"
	"github.com/bytebase/omni/pg/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TransformDMLToSelect(ctx context.Context, tCtx base.TransformContext, statement string, _ string, targetSchema string, tablePrefix string) ([]base.BackupStatement, error) {
	stmts, err := omnipg.Parse(statement)
	if err != nil {
		return nil, errors.Wrapf(errors.Wrap(err, "failed to parse statement"), "failed to prepare transformation")
	}
	if tCtx.GetDatabaseMetadataFunc == nil {
		return nil, errors.Wrapf(errors.New("GetDatabaseMetadataFunc is not set in TransformContext"), "failed to prepare transformation")
	}
	_, metadata, err := tCtx.GetDatabaseMetadataFunc(ctx, tCtx.InstanceID, tCtx.DatabaseName)
	if err != nil {
		return nil, errors.Wrapf(errors.Wrap(err, "failed to get database metadata"), "failed to prepare transformation")
	}
	if metadata == nil {
		return nil, errors.Wrapf(errors.Errorf("database metadata for database %q not found (database not synced)", tCtx.DatabaseName), "failed to prepare transformation")
	}

	backups, err := analysis.TransformDMLToSelect(stmts, statement, analysis.BackupOptions{
		SearchPath: metadata.GetSearchPath(),
		SearchRelation: func(searchPath []string, name string) string {
			schema, _ := metadata.SearchRelation(searchPath, name)
			return schema
		},
		TargetSchema: targetSchema,
		TablePrefix:  tablePrefix,
	})
	if err != nil {
		return nil, err
	}
	return base.ConvertOffsetBackupStatements(statement, backups, func(b analysis.BackupStatement) (string, string, string, string, int, int) {
		return b.Statement, b.SourceSchema, b.SourceTableName, b.TargetTableName, b.StartOffset, b.EndOffset
	}, false), nil
}

// ExtractPrependStatement returns the first SET role or SET search_path
// statement in statement, trimmed and terminated with a semicolon, or "" if
// none is present.
func ExtractPrependStatement(statement string) (string, error) {
	stmts, err := omnipg.Parse(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}
	prepend := analysis.PrependStatements(stmts)
	if prepend == "" {
		return "", nil
	}
	return prepend + ";", nil
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	stmts, err := base.ParseForRestore(statement, backupItem, omnipg.Parse)
	if err != nil {
		return "", err
	}

	posMapper := base.NewByteOffsetPositionMapper(statement)
	dmls := base.FilterDMLsInRange(analysis.BackedUpDMLs(stmts), backupItem, func(dml analysis.BackedUpDML) (*storepb.Position, *storepb.Position) {
		startPos := posMapper.Position(dml.Start)
		startPos.Column-- // 0-based to match backup convention
		return startPos, posMapper.Position(dml.End)
	})

	var sqls []string
	for _, dml := range dmls {
		// Statement text without trailing semicolon/whitespace.
		sqls = append(sqls, strings.TrimRight(strings.TrimSpace(dml.Text), ";"))
	}
	originalSQL := strings.Join(sqls, ";\n")
	if len(originalSQL) == 0 {
		return "", errors.New("no original SQL")
	}
	if len(dmls) == 0 {
		return "", base.StatementNotFoundError(backupItem)
	}

	sourceDatabase, err := base.ParseBackupSourceDatabase(backupItem)
	if err != nil {
		return "", err
	}
	tableMeta, err := base.ResolveRestoreTableMetadata(ctx, rCtx, sourceDatabase, backupItem.SourceTable, "public", func(db string) error {
		return errors.Errorf("database metadata not found for %s", db)
	})
	if err != nil {
		return "", err
	}

	return analysis.RestoreSQL(dmls[0].Node, analysis.RestoreOptions{
		OriginalSchema:    tableMeta.Schema,
		OriginalTable:     backupItem.SourceTable.Table,
		BackupSchema:      backupItem.TargetTable.Schema,
		BackupTable:       backupItem.TargetTable.Table,
		Table:             tableMeta.Table,
		PrimaryKey:        tableMeta.PrimaryKey,
		OriginalSQL:       base.FormatRestoreComment(originalSQL),
		PrependStatements: analysis.PrependStatements(stmts),
	})
}
