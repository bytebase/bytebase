package tidb

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/tidb/analysis"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// errMsgFailedToGetSourceDB is the wrap-error format string used at every
// common.GetDatabaseResourceName(SourceTable.Database) call site.
const errMsgFailedToGetSourceDB = "failed to get source database ID for %s"

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_TIDB, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	// backupItem and its SourceTable/TargetTable are dereferenced
	// unconditionally below.
	if backupItem == nil {
		return "", errors.Errorf("backup item is nil")
	}
	if backupItem.SourceTable == nil {
		return "", errors.Errorf("backup item source table is nil")
	}
	if backupItem.TargetTable == nil {
		return "", errors.Errorf("backup item target table is nil")
	}

	_, _, sourceDatabase, err := common.GetDatabaseResourceName(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, errMsgFailedToGetSourceDB, backupItem.SourceTable.Database)
	}

	// The target table's regular columns decide whether an unqualified SET
	// column changes it.
	if rCtx.GetDatabaseMetadataFunc == nil {
		return "", errors.Wrap(errors.Errorf("GetDatabaseMetadataFunc is nil"), "failed to get target table columns")
	}
	catalog := newBackupCatalog(ctx, rCtx.InstanceID, rCtx.GetDatabaseMetadataFunc, rCtx.ListDatabaseNamesFunc)
	targetCols, err := analysis.NormalColumnSet(catalog, sourceDatabase, backupItem.SourceTable.Table)
	if err != nil {
		return "", errors.Wrap(err, "failed to get target table columns")
	}

	originalSQL, err := extractStatement(statement, backupItem, sourceDatabase, targetCols)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	// One backup item can bundle several same-table DMLs, so the restore
	// covers every one of them.
	matchingNodes, err := analysis.MatchingDMLs(originalSQL, sourceDatabase, backupItem.SourceTable.Table, targetCols)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}
	if len(matchingNodes) == 0 {
		return "", errors.Errorf("no DML statement found in extracted SQL")
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
		Catalog:        catalog,
	})
}

// extractStatement returns the statements between the backup item's positions
// that change its source table.
func extractStatement(statement string, backupItem *storepb.PriorBackupDetail_Item, sourceDatabase string, targetCols map[string]bool) (string, error) {
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
		// EndPosition is the exclusive end of the range, so a statement
		// starting at or after it belongs to the next backup item.
		if equalOrGreater(list[i].Start, backupItem.EndPosition) {
			end = i - 1
		}
	}

	var result []string
	for i := start; i <= end; i++ {
		containsSourceTable, err := analysis.MutatesTable(list[i].Text, sourceDatabase, backupItem.SourceTable.Table, targetCols)
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
