package pg

import (
	"context"

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
	return base.ConvertOffsetBackupStatements(statement, backups, func(b analysis.BackupStatement) (string, string, string, string, int, int) {
		return b.Statement, b.SourceSchema, b.SourceTableName, b.TargetTableName, b.StartOffset, b.EndOffset
	}, false), nil
}
