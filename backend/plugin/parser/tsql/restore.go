package tsql

import (
	"context"
	"strings"

	"github.com/bytebase/omni/mssql/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_MSSQL, GenerateRestoreSQL)
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
