package pg

import (
	"context"
	"strings"

	"github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/analysis"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

const (
	maxCommentLength = 1000
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_POSTGRES, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if backupItem == nil {
		return "", errors.Errorf("failed to extract single SQL: %v", errors.Errorf("backup item is nil"))
	}
	stmts, err := ParsePg(statement)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", errors.Wrapf(err, "failed to parse statement"))
	}
	dmls := backedUpDMLs(statement, analysis.BackedUpDMLs(stmts), backupItem)

	var sqls []string
	for _, dml := range dmls {
		// Statement text without trailing semicolon/whitespace.
		sqls = append(sqls, strings.TrimRight(strings.TrimSpace(dml.Text), ";"))
	}
	originalSQL := strings.Join(sqls, ";\n")
	if len(originalSQL) == 0 {
		return "", errors.Errorf("no original SQL")
	}
	if len(dmls) == 0 {
		return "", errors.Errorf("could not find statement at position (line %d:%d - %d:%d)",
			backupItem.StartPosition.Line, backupItem.StartPosition.Column,
			backupItem.EndPosition.Line, backupItem.EndPosition.Column)
	}

	sqlForComment, truncated := common.TruncateString(originalSQL, maxCommentLength)
	if truncated {
		sqlForComment += "..."
	}

	_, _, sourceDatabase, err := common.GetDatabaseResourceName(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}
	if rCtx.GetDatabaseMetadataFunc == nil {
		return "", errors.Errorf("GetDatabaseMetadataFunc is required")
	}
	_, dbMetadata, err := rCtx.GetDatabaseMetadataFunc(ctx, rCtx.InstanceID, sourceDatabase)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get database metadata for %s", sourceDatabase)
	}
	if dbMetadata == nil {
		return "", errors.Errorf("database metadata not found for %s", sourceDatabase)
	}
	schema := backupItem.SourceTable.Schema
	if schema == "" {
		schema = "public"
	}
	schemaMetadata := dbMetadata.GetSchemaMetadata(schema)
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
	return analysis.RestoreSQL(dmls[0].Node, analysis.RestoreOptions{
		OriginalSchema:    schema,
		OriginalTable:     backupItem.SourceTable.Table,
		BackupSchema:      backupItem.TargetTable.Schema,
		BackupTable:       backupItem.TargetTable.Table,
		Table:             tableMetadata.GetProto(),
		PrimaryKey:        primaryKey,
		OriginalSQL:       sqlForComment,
		PrependStatements: analysis.PrependStatements(stmts),
	})
}

// backedUpDMLs returns the UPDATE and DELETE statements within the backup
// item's range.
func backedUpDMLs(statement string, dmls []analysis.BackedUpDML, backupItem *storepb.PriorBackupDetail_Item) []analysis.BackedUpDML {
	var result []analysis.BackedUpDML
	posMapper := base.NewByteOffsetPositionMapper(statement)
	for _, dml := range dmls {
		startPos := posMapper.Position(dml.Start)
		startPos.Column-- // 0-based to match backup convention
		endPos := posMapper.Position(dml.End)
		if inRange(startPos, endPos, backupItem.StartPosition, backupItem.EndPosition) {
			result = append(result, dml)
		}
	}
	return result
}

func inRange(start, end, targetStart, targetEnd *storepb.Position) bool {
	if start.Line < targetStart.Line || (start.Line == targetStart.Line && start.Column < targetStart.Column) {
		return false
	}
	if end.Line > targetEnd.Line || (end.Line == targetEnd.Line && end.Column > targetEnd.Column) {
		return false
	}
	return true
}
