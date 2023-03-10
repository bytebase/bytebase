package spanner

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"

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
func (*Driver) NeedsSetupMigration(context.Context) (bool, error) {
	return false, nil
}

// SetupMigrationIfNeeded sets up migration if needed.
func (*Driver) SetupMigrationIfNeeded(context.Context) error {
	return nil
}

func (d *Driver) creataDatabase(ctx context.Context, createStatement string, extraStatement []string) error {
	op, err := d.dbClient.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          d.config.Host,
		CreateStatement: createStatement,
		ExtraStatements: extraStatement,
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
// ExecuteMigration will execute the database migration.
func (d *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (migrationHistoryID string, updatedSchema string, resErr error) {
	if statement == "" || m.Type == db.Baseline {
		return "", "", nil
	}

	if !m.CreateDatabase {
		if _, err := d.Execute(ctx, statement, m.CreateDatabase); err != nil {
			return "", "", util.FormatError(err)
		}
	} else {
		stmts, err := sanitizeSQL(statement)
		if err != nil {
			return "", "", errors.Wrapf(err, "failed to sanitize %v", statement)
		}
		if len(stmts) == 0 {
			return "", "", errors.Errorf("expect sanitized SQLs to have at least one entry, original statement: %v", statement)
		}
		if !strings.HasPrefix(stmts[0], "CREATE DATABASE") {
			return "", "", errors.Errorf("expect the first entry of the sanitized SQLs to start with 'CREATE DATABASE', sql %v", stmts[0])
		}
		if err := d.creataDatabase(ctx, stmts[0], stmts[1:]); err != nil {
			return "", "", errors.Wrap(err, "failed to create database")
		}
	}

	return "", "", nil
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
	if len(where) != 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(where, " AND "))
	}
	query = fmt.Sprintf("%s ORDER BY namespace, sequence DESC", query)
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

func (d *Driver) findLargestVersionSinceBaseline(ctx context.Context, tx *spanner.ReadOnlyTransaction, namespace string) (*string, error) {
	largestBaselineSequence, err := d.findLargestSequence(ctx, tx, namespace, true /* baseline */)
	if err != nil {
		return nil, err
	}
	query := `
    SELECT
      MAX(version)
    FROM migration_history
    WHERE namespace = @namespace AND sequence >= @sequence
  `
	params := map[string]interface{}{
		"namespace": namespace,
		"sequence":  largestBaselineSequence,
	}
	stmt := spanner.Statement{SQL: query, Params: params}
	iter := tx.Query(ctx, stmt)
	var versions []spanner.NullString
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var version spanner.NullString
		if err := row.Columns(&version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	if len(versions) != 1 {
		return nil, errors.New("expect to get 1 row")
	}
	if versions[0].Valid {
		version := versions[0].StringVal
		return &version, nil
	}
	return nil, nil
}

func (*Driver) findLargestSequence(ctx context.Context, tx *spanner.ReadOnlyTransaction, namespace string, baseline bool) (int, error) {
	query := `
    SELECT
      MAX(sequence)
    FROM migration_history
    WHERE namespace = @namespace
  `
	if baseline {
		query = fmt.Sprintf("%s AND (type = '%s' OR type = '%s')", query, db.Baseline, db.Branch)
	}
	params := map[string]interface{}{"namespace": namespace}
	stmt := spanner.Statement{SQL: query, Params: params}
	iter := tx.Query(ctx, stmt)
	var sequences []spanner.NullInt64
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		var sequence spanner.NullInt64
		if err := row.Columns(&sequence); err != nil {
			return 0, err
		}
		sequences = append(sequences, sequence)
	}
	if len(sequences) != 1 {
		return 0, errors.New("expect to get 1 row")
	}
	if sequences[0].Valid {
		return int(sequences[0].Int64), nil
	}

	return 0, nil
}

func (d *Driver) insertPendingHistory(ctx context.Context, sequence int, prevSchema string, m *db.MigrationInfo, storedVersion, statement string) (string, error) {
	id := uuid.NewString()
	query := `
        INSERT INTO migration_history (
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
        )
        VALUES (
        @id,
        @creator,
        UNIX_SECONDS(CURRENT_TIMESTAMP()),
        @creator,
        UNIX_SECONDS(CURRENT_TIMESTAMP()),
        @release_version,
        @namespace,
        @sequence,
        @source,
        @type,
        @status,
        @version,
        @description,
        @statement,
        @schema,
        @schema_prev,
        0,
        @issue_id,
        @payload)
  `
	params := map[string]interface{}{
		"id":              id,
		"creator":         m.Creator,
		"release_version": m.ReleaseVersion,
		"namespace":       m.Namespace,
		"sequence":        sequence,
		"source":          m.Source,
		"type":            m.Type,
		"status":          db.Pending,
		"version":         storedVersion,
		"description":     m.Description,
		"statement":       statement,
		"schema":          prevSchema,
		"schema_prev":     prevSchema,
		"issue_id":        m.IssueID,
		"payload":         m.Payload,
	}
	stmt := spanner.Statement{SQL: query, Params: params}
	if _, err := d.client.ReadWriteTransaction(ctx, func(ctx context.Context, rwt *spanner.ReadWriteTransaction) error {
		if _, err := rwt.Update(ctx, stmt); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return "", err
	}

	return id, nil
}

func (d *Driver) updateHistoryAsDone(ctx context.Context, migrationDurationNs int64, updatedSchema string, insertedID string) error {
	query := `
    UPDATE
      migration_history
    SET
      status = @status,
      execution_duration_ns = @execution_duration_ns,
      schema = @schema
    WHERE id = @id
  `
	params := map[string]interface{}{
		"status":                db.Done,
		"execution_duration_ns": migrationDurationNs,
		"schema":                updatedSchema,
		"id":                    insertedID,
	}
	stmt := spanner.Statement{SQL: query, Params: params}

	if _, err := d.client.ReadWriteTransaction(ctx, func(ctx context.Context, rwt *spanner.ReadWriteTransaction) error {
		if _, err := rwt.Update(ctx, stmt); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (d *Driver) updateHistoryAsFailed(ctx context.Context, migrationDurationNs int64, insertedID string) error {
	query := `
    UPDATE
      migration_history
    SET
      status = @status,
      execution_duration_ns = @execution_duration_ns
    WHERE id = @id
  `
	params := map[string]interface{}{
		"status":                db.Failed,
		"execution_duration_ns": migrationDurationNs,
		"id":                    insertedID,
	}
	stmt := spanner.Statement{SQL: query, Params: params}

	if _, err := d.client.ReadWriteTransaction(ctx, func(ctx context.Context, rwt *spanner.ReadWriteTransaction) error {
		if _, err := rwt.Update(ctx, stmt); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
