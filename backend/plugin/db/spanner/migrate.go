package spanner

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
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
		zap.String("environment", d.connCtx.EnvironmentID),
		zap.String("instance", d.connCtx.InstanceID),
	)
	statements, err := sanitizeSQL(migrationSchema)
	if err != nil {
		return err
	}
	return d.creataDatabase(ctx, createBytebaseDatabaseStatement, statements)
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
// Returns the created migration history id and the updated schema on success.
func (d *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (migrationHistoryID string, updatedSchema string, resErr error) {
	var prevSchemaBuf bytes.Buffer
	// Don't record schema if the database hasn't existed yet or is schemaless (e.g. Mongo).
	if !m.CreateDatabase {
		// For baseline migration, we also record the live schema to detect the schema drift.
		// See https://bytebase.com/blog/what-is-database-schema-drift
		if _, err := d.Dump(ctx, m.Database, &prevSchemaBuf, true /*schemaOnly*/); err != nil {
			return "", "", util.FormatError(err)
		}
	}

	// Switch to the database where the migration_history table resides.
	if err := d.switchDatabase(ctx, db.BytebaseDatabase); err != nil {
		return "", "", err
	}

	// Phase 1 - Pre-check before executing migration
	// Phase 2 - Record migration history as PENDING
	insertedID, err := d.beginMigration(ctx, m, prevSchemaBuf.String(), statement)
	if err != nil {
		if common.ErrorCode(err) == common.MigrationAlreadyApplied {
			return insertedID, prevSchemaBuf.String(), nil
		}
		return "", "", errors.Wrapf(err, "failed to begin migration for issue %s", m.IssueID)
	}

	startedNs := time.Now().UnixNano()

	defer func() {
		if err := d.endMigration(ctx, startedNs, insertedID, updatedSchema, db.BytebaseDatabase, resErr == nil /*isDone*/); err != nil {
			log.Error("Failed to update migration history record",
				zap.Error(err),
				zap.String("migration_id", migrationHistoryID),
			)
		}
	}()

	// Phase 3 - Executing migration
	// Branch migration type always has empty sql.
	// Baseline migration type could has non-empty sql but will not execute.
	// https://github.com/bytebase/bytebase/issues/394
	doMigrate := true
	if statement == "" {
		doMigrate = false
	}
	if m.Type == db.Baseline {
		doMigrate = false
	}
	if doMigrate {
		// Switch back to the target database.
		if !m.CreateDatabase {
			if err := d.switchDatabase(ctx, m.Database); err != nil {
				return "", "", err
			}
			if _, err := d.Execute(ctx, statement, m.CreateDatabase); err != nil {
				return "", "", util.FormatError(err)
			}
		} else {
			if err := d.creataDatabase(ctx, statement, nil); err != nil {
				return "", "", err
			}
		}
	}

	// Phase 4 - Dump the schema after migration
	var afterSchemaBuf bytes.Buffer
	if _, err := d.Dump(ctx, m.Database, &afterSchemaBuf, true /*schemaOnly*/); err != nil {
		return "", "", util.FormatError(err)
	}

	return insertedID, afterSchemaBuf.String(), nil
}

// BeginMigration checks before executing migration and inserts a migration history record with pending status.
func (d *Driver) beginMigration(ctx context.Context, m *db.MigrationInfo, prevSchema string, statement string) (insertedID string, err error) {
	// Convert version to stored version.
	storedVersion, err := util.ToStoredVersion(m.UseSemanticVersion, m.Version, m.SemanticVersionSuffix)
	if err != nil {
		return "", errors.Wrap(err, "failed to convert to stored version")
	}
	// Phase 1 - Pre-check before executing migration
	// Check if the same migration version has already been applied.
	if list, err := d.FindMigrationHistoryList(ctx, &db.MigrationHistoryFind{
		Database: &m.Namespace,
		Version:  &m.Version,
	}); err != nil {
		return "", errors.Wrap(err, "failed to check duplicate version")
	} else if len(list) > 0 {
		migrationHistory := list[0]
		switch migrationHistory.Status {
		case db.Done:
			if migrationHistory.IssueID != m.IssueID {
				return migrationHistory.ID, common.Errorf(common.MigrationFailed, "database %q has already applied version %s by issue %s", m.Database, m.Version, migrationHistory.IssueID)
			}
			return migrationHistory.ID, common.Errorf(common.MigrationAlreadyApplied, "database %q has already applied version %s", m.Database, m.Version)
		case db.Pending:
			err := errors.Errorf("database %q version %s migration is already in progress", m.Database, m.Version)
			log.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			if m.Force {
				return migrationHistory.ID, nil
			}
			return "", common.Wrap(err, common.MigrationPending)
		case db.Failed:
			err := errors.Errorf("database %q version %s migration has failed, please check your database to make sure things are fine and then start a new migration using a new version ", m.Database, m.Version)
			log.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			if m.Force {
				return migrationHistory.ID, nil
			}
			return "", common.Wrap(err, common.MigrationFailed)
		}
	}

	// Get largestSequence, largestVersionSinceBaseline in a transaction.
	tx := d.client.ReadOnlyTransaction()
	defer tx.Close()

	largestSequence, err := d.findLargestSequence(ctx, tx, m.Namespace, false /* baseline */)
	if err != nil {
		return "", err
	}

	// Check if there is any higher version already been applied since the last baseline or branch.
	if version, err := d.findLargestVersionSinceBaseline(ctx, tx, m.Namespace); err != nil {
		return "", err
	} else if version != nil && len(*version) > 0 && *version >= m.Version {
		// len(*version) > 0 is used because Clickhouse will always return non-nil version with empty string.
		return "", common.Errorf(common.MigrationOutOfOrder, "database %q has already applied version %s which >= %s", m.Database, *version, m.Version)
	}

	// Phase 2 - Record migration history as PENDING.
	// MySQL runs DDL in its own transaction, so we can't commit migration history together with DDL in a single transaction.
	// Thus we sort of doing a 2-phase commit, where we first write a PENDING migration record, and after migration completes, we then
	// update the record to DONE together with the updated schema.
	statementRecord, _ := common.TruncateString(statement, common.MaxSheetSize)
	if insertedID, err = d.insertPendingHistory(ctx, largestSequence+1, prevSchema, m, storedVersion, statementRecord); err != nil {
		return "", err
	}

	return insertedID, nil
}

// EndMigration updates the migration history record to DONE or FAILED depending on migration is done or not.
func (d *Driver) endMigration(ctx context.Context, startedNs int64, migrationHistoryID string, updatedSchema string, databaseName string, isDone bool) (err error) {
	migrationDurationNs := time.Now().UnixNano() - startedNs
	if err := d.switchDatabase(ctx, databaseName); err != nil {
		return err
	}

	if isDone {
		// Upon success, update the migration history as 'DONE', execution_duration_ns, updated schema.
		err = d.updateHistoryAsDone(ctx, migrationDurationNs, updatedSchema, migrationHistoryID)
	} else {
		// Otherwise, update the migration history as 'FAILED', execution_duration.
		err = d.updateHistoryAsFailed(ctx, migrationDurationNs, migrationHistoryID)
	}
	if err != nil {
		return err
	}
	return nil
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
