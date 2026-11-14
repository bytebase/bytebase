package mariadb

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/mariadb/analysis"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_MARIADB, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractStatement(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	// Parse the filtered SQL and find the first DML node.
	matchingNode, err := analysis.FirstDML(originalSQL)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}
	if matchingNode == nil {
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
	_, normalColumns, err := classifyColumns(ctx, rCtx.GetDatabaseMetadataFunc, rCtx.ListDatabaseNamesFunc, rCtx.IsCaseSensitive, rCtx.InstanceID, sourceDatabase, backupItem.SourceTable.Table)
	if err != nil {
		return "", errors.Wrapf(err, "failed to classify columns for %s.%s", backupItem.SourceTable.Database, backupItem.SourceTable.Table)
	}

	return analysis.RestoreSQL(matchingNode, analysis.RestoreOptions{
		Statement:      originalSQL,
		Database:       sourceDatabase,
		Table:          backupItem.SourceTable.Table,
		BackupDatabase: targetDatabase,
		BackupTable:    backupItem.TargetTable.Table,
		Columns:        normalColumns,
		Catalog:        newBackupCatalog(ctx, rCtx.InstanceID, rCtx.GetDatabaseMetadataFunc, rCtx.ListDatabaseNamesFunc),
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

	start := 0
	end := len(list) - 1
	for i, item := range list {
		if equalOrLess(item.Start, backupItem.StartPosition) {
			start = i
		}
	}

	for i := len(list) - 1; i >= 0; i-- {
		if equalOrGreater(list[i].Start, backupItem.EndPosition) {
			end = i
		}
	}

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

func equalOrLess(a, b *storepb.Position) bool {
	if a.Line < b.Line {
		return true
	}
	if a.Line == b.Line && a.Column <= b.Column {
		return true
	}
	return false
}

func equalOrGreater(a, b *storepb.Position) bool {
	if a.Line > b.Line {
		return true
	}
	if a.Line == b.Line && a.Column >= b.Column {
		return true
	}
	return false
}
