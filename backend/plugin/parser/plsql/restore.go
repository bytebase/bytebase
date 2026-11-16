package plsql

import (
	"context"
	"strings"

	"github.com/bytebase/omni/oracle/analysis"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_ORACLE, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractSQL(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	node, err := analysis.FindFirstDML(originalSQL)
	if err != nil {
		return "", err
	}
	if node == nil {
		return "", errors.New("no DML statement found in extracted SQL")
	}

	_, _, sourceDatabase, err := common.GetDatabaseResourceName(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}
	_, _, targetDatabase, err := common.GetDatabaseResourceName(backupItem.TargetTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get target database ID for %s", backupItem.TargetTable.Database)
	}

	if rCtx.GetDatabaseMetadataFunc == nil {
		return "", errors.Errorf("GetDatabaseMetadataFunc is required")
	}

	_, metadata, err := rCtx.GetDatabaseMetadataFunc(ctx, rCtx.InstanceID, sourceDatabase)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get database metadata for %s", sourceDatabase)
	}
	if metadata == nil {
		return "", errors.Errorf("database metadata for database %q not found (database not synced)", sourceDatabase)
	}

	schemaMetadata := metadata.GetSchemaMetadata("")
	if schemaMetadata == nil {
		return "", errors.Errorf("no schema metadata for %s", sourceDatabase)
	}

	tableMetadata := schemaMetadata.GetTable(backupItem.SourceTable.Table)
	if tableMetadata == nil {
		return "", errors.Errorf("no table metadata for %s.%s", sourceDatabase, backupItem.SourceTable.Table)
	}

	opts := analysis.RestoreOptions{
		SourceDatabase: sourceDatabase,
		SourceTable:    backupItem.SourceTable.Table,
		BackupDatabase: targetDatabase,
		BackupTable:    backupItem.TargetTable.Table,
		Table:          tableMetadata.GetProto(),
	}
	if pk := tableMetadata.GetPrimaryKey(); pk != nil {
		opts.PrimaryKey = pk.GetProto()
	}
	return analysis.RestoreSQL(originalSQL, node, opts)
}

// extractSQL returns the statements of the script between the backup item's
// positions that back up its source table.
func extractSQL(statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if backupItem == nil {
		return "", errors.New("backup item is nil")
	}

	list, err := SplitSQL(statement)
	if err != nil {
		return "", errors.Wrapf(err, "failed to split SQL")
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
			return "", errors.Wrap(err, "failed to prepare transformation")
		}
		if containsSourceTable {
			result = append(result, analysis.NormalizeRestoreSQL(list[i].Text))
		}
	}
	return strings.Join(result, "\n"), nil
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
