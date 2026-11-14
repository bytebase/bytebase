package tidb

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/tidb/analysis"
	omniparser "github.com/bytebase/omni/tidb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_TIDB, TransformDMLToSelect)
}

func TransformDMLToSelect(ctx context.Context, tCtx base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	backups, err := analysis.TransformDMLToSelect(statement, analysis.BackupOptions{
		DefaultDatabase: sourceDatabase,
		BackupDatabase:  targetDatabase,
		TablePrefix:     tablePrefix,
		CaseSensitive:   tCtx.IsCaseSensitive,
		Catalog:         newBackupCatalog(ctx, tCtx.InstanceID, tCtx.GetDatabaseMetadataFunc, tCtx.ListDatabaseNamesFunc),
		Columns: func(database, table string) (analysis.TableColumns, error) {
			return classifyColumns(ctx, tCtx.GetDatabaseMetadataFunc, tCtx.ListDatabaseNamesFunc, tCtx.IsCaseSensitive, tCtx.InstanceID, database, table)
		},
	})
	if err != nil || backups == nil {
		return nil, err
	}
	return convertBackupStatements(statement, backups)
}

// convertBackupStatements gives each backup the positions SplitSQL gives the
// first and last segment it covers.
func convertBackupStatements(statement string, backups []analysis.BackupStatement) ([]base.BackupStatement, error) {
	segments := omniparser.Split(statement)
	list, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	starts := make(map[int]*storepb.Position, len(segments))
	ends := make(map[int]*storepb.Position, len(segments))
	for i, segment := range segments {
		starts[segment.ByteStart] = list[i].Start
		ends[segment.ByteEnd] = list[i].End
	}
	result := make([]base.BackupStatement, 0, len(backups))
	for _, backup := range backups {
		result = append(result, base.BackupStatement{
			Statement:       backup.Statement,
			SourceTableName: backup.SourceTable,
			TargetTableName: backup.BackupTable,
			StartPosition:   starts[backup.ByteStart],
			EndPosition:     ends[backup.ByteEnd],
		})
	}
	return result, nil
}

// newBackupCatalog reads the instance's metadata for prior backup. Unlike the
// query span's catalog, it returns lookup errors unchanged.
func newBackupCatalog(ctx context.Context, instanceID string, getDatabaseMetadataFunc base.GetDatabaseMetadataFunc, listDatabaseNamesFunc base.ListDatabaseNamesFunc) analysis.Catalog {
	var catalog analysis.Catalog
	if listDatabaseNamesFunc != nil {
		catalog.ListDatabaseNames = func() ([]string, error) {
			return listDatabaseNamesFunc(ctx, instanceID)
		}
	}
	if getDatabaseMetadataFunc != nil {
		catalog.GetDatabase = func(name string) (analysis.Database, error) {
			_, database, err := getDatabaseMetadataFunc(ctx, instanceID, name)
			if err != nil {
				return nil, err
			}
			if database == nil {
				return nil, nil
			}
			return &catalogDatabase{database: database}, nil
		}
	}
	return catalog
}

// classifyColumns returns what prior backup needs of a table's columns and
// keys, and reports a missing callback or database in Bytebase's words.
func classifyColumns(ctx context.Context, getDatabaseMetadataFunc base.GetDatabaseMetadataFunc, listDatabaseNamesFunc base.ListDatabaseNamesFunc, isCaseSensitive bool, instanceID string, database, table string) (analysis.TableColumns, error) {
	if getDatabaseMetadataFunc == nil {
		return analysis.TableColumns{}, errors.New("GetDatabaseMetadataFunc is not set")
	}
	columns, err := analysis.ClassifyColumns(newBackupCatalog(ctx, instanceID, getDatabaseMetadataFunc, listDatabaseNamesFunc), isCaseSensitive, database, table)
	var notFound *analysis.NotFoundError
	if errors.As(err, &notFound) {
		slog.Debug("failed to get database metadata", slog.String("instanceID", instanceID), slog.String("database", database))
		return analysis.TableColumns{}, errors.Errorf("failed to get database metadata for InstanceID %q, Database %q", instanceID, database)
	}
	return columns, err
}
