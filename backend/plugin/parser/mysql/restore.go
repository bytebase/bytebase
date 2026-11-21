package mysql

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/mysql/analysis"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_MYSQL, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractStatement(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	// Parse the filtered SQL and find its DML nodes; the first decides.
	matchingNodes, err := analysis.DMLs(originalSQL)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}
	if len(matchingNodes) == 0 {
		return "", errors.Errorf("no DML statement found in extracted SQL")
	}

	_, _, sourceDatabase, err := common.GetDatabaseResourceName(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}
	_, _, targetDatabase, err := common.GetDatabaseResourceName(backupItem.TargetTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get target database ID for %s", backupItem.TargetTable.Database)
	}
	columns, err := classifyColumns(ctx, rCtx.GetDatabaseMetadataFunc, rCtx.ListDatabaseNamesFunc, rCtx.IsCaseSensitive, rCtx.InstanceID, sourceDatabase, backupItem.SourceTable.Table)
	if err != nil {
		return "", errors.Wrapf(err, "failed to classify columns for %s.%s", backupItem.SourceTable.Database, backupItem.SourceTable.Table)
	}

	return analysis.RestoreSQL(matchingNodes, analysis.RestoreOptions{
		Statement:      originalSQL,
		Database:       sourceDatabase,
		Table:          backupItem.SourceTable.Table,
		BackupDatabase: targetDatabase,
		BackupTable:    backupItem.TargetTable.Table,
		Columns:        columns.Normal,
		Catalog:        newBackupCatalog(ctx, rCtx.InstanceID, rCtx.GetDatabaseMetadataFunc, rCtx.ListDatabaseNamesFunc),
		CaseSensitive:  rCtx.IsCaseSensitive,
	})
}

// extractStatement returns the statements between the backup item's positions
// that change its source table.
func extractStatement(statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if backupItem == nil {
		return "", errors.Errorf("backup item is nil")
	}

	list, err := SplitSQL(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to split sql")
	}

	start, end := base.StatementRangeIndices(list, backupItem.StartPosition, backupItem.EndPosition, false)

	_, _, sourceDatabase, err := common.GetDatabaseResourceName(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}

	var result []string
	for i := start; i <= end; i++ {
		containsSourceTable, err := analysis.ReferencesTable(list[i].Text, sourceDatabase, backupItem.SourceTable.Table)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse sql")
		}
		if containsSourceTable {
			result = append(result, list[i].Text)
		}
	}
	return strings.Join(result, ""), nil
}
