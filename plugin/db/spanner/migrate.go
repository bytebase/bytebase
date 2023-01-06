package spanner

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"

	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *Driver) notFoundDatabase(ctx context.Context, databaseName string) (bool, error) {
	dsn := getDSN(d.config.Host, databaseName)
	_, err := d.dbClient.GetDatabase(ctx, &databasepb.GetDatabaseRequest{Name: dsn})
	if status.Code(err) == codes.NotFound {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

// NeedsSetupMigration checks if it needs to set up migration.
func (d *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	notFound, err := d.notFoundDatabase(ctx, db.BytebaseDatabase)
	if err != nil {
		return false, err
	}
	return notFound, nil
}

// SetupMigrationIfNeeded sets up migration if needed.
func (d *Driver) SetupMigrationIfNeeded(ctx context.Context) error {
	setup, err := d.NeedsSetupMigration(ctx)
	if err != nil {
		return err
	}
	if !setup {
		return nil
	}
	log.Info("Bytebase migration schema not found, creating schema...",
		zap.String("environment", d.connCtx.EnvironmentName),
		zap.String("instance", d.connCtx.InstanceName),
	)
	statements := splitStatement(migrationSchema)
	op, err := d.dbClient.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          d.config.Host,
		CreateStatement: createBytebaseDatabaseStatement,
		ExtraStatements: statements,
	})
	if err != nil {
		return err
	}
	if _, err := op.Wait(ctx); err != nil {
		return err
	}
	return nil
}

// ExecuteMigration executes a migration.
func (*Driver) ExecuteMigration(_ context.Context, _ *db.MigrationInfo, _ string) (string, string, error) {
	panic("not implemented")
}

// FindMigrationHistoryList finds the migration history list.
func (d *Driver) FindMigrationHistoryList(ctx context.Context, find *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	defer func(db string) {
		if err := d.switchDatabase(ctx, db); err != nil {
			log.Error("failed to switch back database for spanner driver", zap.String("database", db), zap.Error(err))
		}
	}(d.dbName)
	if err := d.switchDatabase(ctx, db.BytebaseDatabase); err != nil {
		return nil, err
	}
	query := `
	SELECT
		id,
		created_by,
		created_ts,
		updated_by,
		updated_ts,
		release_version,
		namespace,
		sequence,
		source,
		type,
		status,
		version,
		description,
		statement,
		schema,
		schema_prev,
		execution_duration_ns,
		issue_id,
		payload
		FROM migration_history
  `
	params := make(map[string]interface{})
	var where []string

	if v := find.ID; v != nil {
		where = append(where, `id = @id`)
		params["id"] = *v
	}
	if v := find.Database; v != nil {
		where = append(where, `namespace = @namespace`)
		params["namespace"] = *v
	}
	if v := find.Source; v != nil {
		where = append(where, `source = @source`)
		params["source"] = *v
	}
	if v := find.Version; v != nil {
		// TODO(d): support semantic versioning.
		storedVersion, err := util.ToStoredVersion(false, *v, "")
		if err != nil {
			return nil, err
		}
		where = append(where, "version = @version")
		params["version"] = storedVersion
	}
	query = fmt.Sprintf("%s WHERE %s ORDER BY namespace, sequence DESC", query, strings.Join(where, " AND "))
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}

	stmt := spanner.Statement{
		SQL:    query,
		Params: params,
	}

	var migrationHistoryList []*db.MigrationHistory
	iter := d.client.Single().Query(ctx, stmt)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var history db.MigrationHistory
		var storedVersion string
		var sequence int64
		if err := row.Columns(
			&history.ID,
			&history.Creator,
			&history.CreatedTs,
			&history.Updater,
			&history.UpdatedTs,
			&history.ReleaseVersion,
			&history.Namespace,
			&sequence,
			&history.Source,
			&history.Type,
			&history.Status,
			&storedVersion,
			&history.Description,
			&history.Statement,
			&history.Schema,
			&history.SchemaPrev,
			&history.ExecutionDurationNs,
			&history.IssueID,
			&history.Payload,
		); err != nil {
			return nil, err
		}
		history.Sequence = int(sequence)
		useSemanticVersion, version, semanticVersionSuffix, err := util.FromStoredVersion(storedVersion)
		if err != nil {
			return nil, err
		}
		history.UseSemanticVersion, history.Version, history.SemanticVersionSuffix = useSemanticVersion, version, semanticVersionSuffix
		migrationHistoryList = append(migrationHistoryList, &history)
	}

	return migrationHistoryList, nil
}
