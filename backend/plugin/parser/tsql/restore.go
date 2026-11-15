package tsql

import (
	"context"
	"strings"

	"github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/mssql/analysis"
	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_MSSQL, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractStatement(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	if len(originalSQL) == 0 {
		return "", errors.Errorf("no original SQL")
	}

	stmts, err := parseScript(statement)
	if err != nil {
		return "", err
	}

	// Find the AST that contains the statement at the backup position
	var targetResult ast.Node
	for _, dml := range analysis.BackedUpDMLs(stmts) {
		if inRange(dml, backupItem) {
			targetResult = dml.Node
			break
		}
	}

	if targetResult == nil {
		return "", errors.Errorf("could not find statement at position (line %d:%d - %d:%d)",
			backupItem.StartPosition.Line, backupItem.StartPosition.Column,
			backupItem.EndPosition.Line, backupItem.EndPosition.Column)
	}

	return doGenerate(ctx, rCtx, originalSQL, targetResult, backupItem)
}

func doGenerate(ctx context.Context, rCtx base.RestoreContext, originalSQL string, node ast.Node, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
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

	_, databaseMetadata, err := rCtx.GetDatabaseMetadataFunc(ctx, rCtx.InstanceID, sourceDatabase)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get database metadata for %s", sourceDatabase)
	}

	if databaseMetadata == nil {
		return "", errors.Errorf("database metadata is nil")
	}

	schema := backupItem.SourceTable.Schema
	if schema == "" {
		schema = "dbo"
	}
	schemaMetadata := databaseMetadata.GetSchemaMetadata(schema)
	if schemaMetadata == nil {
		return "", errors.Errorf("schema metadata not found for %s", schema)
	}

	tableMetadata := schemaMetadata.GetTable(backupItem.SourceTable.Table)
	if tableMetadata == nil {
		return "", errors.Errorf("table metadata not found for %s.%s", schema, backupItem.SourceTable.Table)
	}

	var primaryKey *metadata.IndexMetadata
	if pk := tableMetadata.GetPrimaryKey(); pk != nil {
		primaryKey = pk.GetProto()
	}
	return analysis.RestoreSQL(node, analysis.RestoreOptions{
		OriginalDatabase: sourceDatabase,
		OriginalSchema:   schema,
		OriginalTable:    backupItem.SourceTable.Table,
		BackupDatabase:   targetDatabase,
		BackupTable:      backupItem.TargetTable.Table,
		Table:            tableMetadata.GetProto(),
		PrimaryKey:       primaryKey,
		OriginalSQL:      originalSQL,
	})
}

func extractStatement(statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if backupItem == nil {
		return "", errors.Errorf("backup item is nil")
	}

	stmts, err := parseScript(statement)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse statement")
	}

	var originalSQL []string
	for _, dml := range analysis.BackedUpDMLs(stmts) {
		if inRange(dml, backupItem) {
			originalSQL = append(originalSQL, dml.SQL)
		}
	}
	if len(originalSQL) == 0 {
		return "", nil
	}

	// Join with semicolons and add a trailing semicolon
	return strings.Join(originalSQL, ";\n") + ";", nil
}

// inRange checks if a statement's position range is within the backup item's range.
func inRange(dml analysis.BackedUpDML, backupItem *storepb.PriorBackupDetail_Item) bool {
	start, end := convertPosition(dml.StartPosition), convertPosition(dml.EndPosition)
	targetStart, targetEnd := backupItem.StartPosition, backupItem.EndPosition
	if start.Line < targetStart.Line || (start.Line == targetStart.Line && start.Column < targetStart.Column) {
		return false
	}
	if end.Line > targetEnd.Line || (end.Line == targetEnd.Line && end.Column > targetEnd.Column) {
		return false
	}
	return true
}
