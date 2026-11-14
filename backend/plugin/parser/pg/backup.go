package pg

import (
	"context"
	"slices"

	"github.com/bytebase/omni/pg/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_POSTGRES, TransformDMLToSelect)
}

func TransformDMLToSelect(ctx context.Context, tCtx base.TransformContext, statement string, _ string, targetSchema string, tablePrefix string) ([]base.BackupStatement, error) {
	stmts, err := ParsePg(statement)
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
	positions := mapPositions(statement, backups)
	// The backups' positions match the restore's: the start column is 0-based.
	var result []base.BackupStatement
	for _, backup := range backups {
		start, end := positions[backup.StartOffset], positions[backup.EndOffset]
		result = append(result, base.BackupStatement{
			Statement:       backup.Statement,
			SourceSchema:    backup.SourceSchema,
			SourceTableName: backup.SourceTableName,
			TargetTableName: backup.TargetTableName,
			StartPosition:   &storepb.Position{Line: start.Line, Column: start.Column - 1},
			EndPosition:     &storepb.Position{Line: end.Line, Column: end.Column},
		})
	}
	return result, nil
}

// mapPositions maps every backup offset in ascending order, so one mapper scans
// the script once however many backups it yields.
func mapPositions(statement string, backups []analysis.BackupStatement) map[int]*storepb.Position {
	var offsets []int
	for _, backup := range backups {
		offsets = append(offsets, backup.StartOffset, backup.EndOffset)
	}
	slices.Sort(offsets)
	mapper := base.NewByteOffsetPositionMapper(statement)
	positions := make(map[int]*storepb.Position, len(offsets))
	for _, offset := range slices.Compact(offsets) {
		positions[offset] = mapper.Position(offset)
	}
	return positions
}
