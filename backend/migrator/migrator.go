// Package migrator handles store schema migration.
package migrator

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

const latestSchemaFileName = "migration/LATEST.sql"

//go:embed migration
var migrationFS embed.FS

// GoMigrationFunc is a function that performs data migration in Go code.
// It receives a context and database connection, and should handle its own
// transaction management for optimal performance (e.g., batching large updates).
type GoMigrationFunc func(ctx context.Context, conn *sql.Conn) error

// goMigrations is a registry of version-specific Go migrations that run
// in addition to or instead of SQL migrations. These are useful for:
// - Large data migrations that need batching
// - Complex transformations better expressed in Go
// - Operations that benefit from programmatic control
var goMigrations = map[string]GoMigrationFunc{
	"3.13.21": migrate3_13_21,
}

// MigrateSchema migrates the schema for metadata database.
func MigrateSchema(ctx context.Context, db *sql.DB) error {
	files, err := getSortedVersionedFiles()
	if err != nil {
		return err
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	var ok bool
	if err := conn.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'principal')`,
	).Scan(&ok); err != nil {
		return err
	}

	latestVersion := files[len(files)-1].version.String()
	// Initialize the latest schema.
	if !ok {
		buf, err := migrationFS.ReadFile(latestSchemaFileName)
		if err != nil {
			return errors.Wrapf(err, "failed to read latest schema %q", latestSchemaFileName)
		}
		if err := executeMigration(ctx, conn, string(buf), latestVersion); err != nil {
			return err
		}
		slog.Info(fmt.Sprintf("Initialized database schema with version %s.", latestVersion))
		return nil
	}

	latestDatabaseVersion, err := getLatestDatabaseVersion(ctx, conn)
	if err != nil {
		return err
	}
	if latestDatabaseVersion == nil {
		return errors.New("the latest database version is not found")
	}

	for _, f := range files {
		if f.version.LE(*latestDatabaseVersion) {
			continue
		}

		buf, err := fs.ReadFile(migrationFS, f.path)
		if err != nil {
			return errors.Wrapf(err, "failed to read file %q", f.path)
		}
		version := f.version.String()
		slog.Info(fmt.Sprintf("Migrating %s.", version))

		// Run Go migration FIRST if one exists for this version
		// This ensures that if Go migration fails, the version is not recorded
		// and both SQL and Go migrations will retry on next startup
		if goMigration, exists := goMigrations[version]; exists {
			slog.Info(fmt.Sprintf("Running Go migration for %s.", version))
			if err := goMigration(ctx, conn); err != nil {
				return errors.Wrapf(err, "Go migration %s failed", version)
			}
		}

		// Run SQL migration, which records the version upon success
		if err := executeMigration(ctx, conn, string(buf), version); err != nil {
			return err
		}
	}

	slog.Info(fmt.Sprintf("Current schema version: %s", latestVersion))
	return nil
}

type versionedFile struct {
	version *semver.Version
	path    string
}

func getSortedVersionedFiles() ([]versionedFile, error) {
	var files []versionedFile
	if err := fs.WalkDir(migrationFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if path == latestSchemaFileName {
			return nil
		}

		v, err := getVersionFromPath(path)
		if err != nil {
			return err
		}
		files = append(files, versionedFile{
			version: v,
			path:    path,
		})
		return nil
	}); err != nil {
		return nil, err
	}
	slices.SortFunc(files, func(a, b versionedFile) int {
		if a.version.LT(*b.version) {
			return -1
		} else if a.version.GT(*b.version) {
			return 1
		}
		return 0
	})
	return files, nil
}

func getVersionFromPath(path string) (*semver.Version, error) {
	// migration/3.5/0000##vcs.sql
	s := strings.TrimPrefix(path, "migration/")
	splits := strings.Split(s, "/")
	if len(splits) != 2 {
		return nil, errors.Errorf("invalid migration path %q", path)
	}
	splits2 := strings.Split(splits[1], "##")
	if len(splits2) != 2 {
		return nil, errors.Errorf("invalid migration path %q", path)
	}
	patch, err := strconv.ParseInt(splits2[0], 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "migration filename prefix %q should be four digits integer such as '0000'", splits2[0])
	}

	v := fmt.Sprintf("%s.%d", splits[0], patch)
	version, err := semver.Parse(v)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid version %q", v)
	}
	return &version, nil
}

func executeMigration(ctx context.Context, conn *sql.Conn, statement string, version string) error {
	// Get current database context for error reporting
	var currentUser, currentDatabase string
	_ = conn.QueryRowContext(ctx, "SELECT current_user, current_database()").Scan(&currentUser, &currentDatabase)

	txn, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	if _, err := txn.ExecContext(ctx, statement); err != nil {
		// Extract SQLSTATE and provide contextual information
		var sqlState string
		if pqErr, ok := err.(*pq.Error); ok {
			sqlState = string(pqErr.Code)
		}

		// Truncate statement for readability in error message
		stmtPreview := statement
		if len(stmtPreview) > 100 {
			stmtPreview = stmtPreview[:100] + "..."
		}

		return errors.Errorf("migration %s failed\n"+
			"Statement: %s\n"+
			"User: %s\n"+
			"Database: %s\n"+
			"Error: %v\n"+
			"SQLSTATE: %s",
			version, stmtPreview, currentUser, currentDatabase, err, sqlState)
	}
	if _, err := txn.ExecContext(ctx,
		`INSERT INTO instance_change_history (version) VALUES ($1)`,
		version,
	); err != nil {
		return err
	}

	return txn.Commit()
}

func getLatestDatabaseVersion(ctx context.Context, conn *sql.Conn) (*semver.Version, error) {
	query := `SELECT version FROM instance_change_history ORDER BY id DESC`

	var v string
	if err := conn.QueryRowContext(ctx, query).Scan(&v); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	version, err := semver.Make(v)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid version %q", v)
	}
	return &version, nil
}

// migrate3_13_21 migrates audit_log user references from users/{id} to users/{email} format.
// This is done in Go with batching to handle large audit_log tables efficiently.
func migrate3_13_21(ctx context.Context, conn *sql.Conn) error {
	const batchSize = 10000

	// Build user ID to email lookup map
	userMap, err := buildUserIDToEmailMap(ctx, conn)
	if err != nil {
		return errors.Wrap(err, "failed to build user ID to email map")
	}

	var lastID int64
	totalUpdated := 0
	batchCount := 0

	for {
		rowsUpdated, maxID, done, err := migrateAuditLogBatch(ctx, conn, userMap, lastID, batchSize)
		if err != nil {
			return err
		}
		if done {
			break
		}

		totalUpdated += rowsUpdated
		batchCount++
		lastID = maxID

		// Log progress every 10 batches (100k rows)
		if batchCount%10 == 0 {
			slog.Info(fmt.Sprintf("Updated %d audit_log rows...", totalUpdated))
		}
	}

	slog.Info(fmt.Sprintf("Completed audit_log migration. Total rows updated: %d", totalUpdated))
	return nil
}

// migrateAuditLogBatch processes a single batch of audit_log updates.
func migrateAuditLogBatch(ctx context.Context, conn *sql.Conn, userMap map[int]string, lastID int64, batchSize int) (rowsUpdated int, maxID int64, done bool, err error) {
	txn, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, false, errors.Wrap(err, "failed to begin transaction")
	}
	defer txn.Rollback()

	// Get batch of rows to update
	rows, err := txn.QueryContext(ctx, `
		SELECT id, payload->>'user' AS user_ref
		FROM audit_log
		WHERE id > $1
		  AND payload->>'user' LIKE 'users/%'
		  AND payload->>'user' NOT LIKE 'users/%@%'
		ORDER BY id
		LIMIT $2
	`, lastID, batchSize)
	if err != nil {
		return 0, 0, false, errors.Wrap(err, "failed to query batch")
	}
	defer rows.Close()

	// Build VALUES clause for bulk update
	var valueStrings []string
	var valueArgs []any
	argPos := 1

	for rows.Next() {
		var id int64
		var userRef string
		if err := rows.Scan(&id, &userRef); err != nil {
			return 0, 0, false, errors.Wrap(err, "failed to scan row")
		}

		newUserRef := convertUserIDToEmail(userRef, userMap)
		valueStrings = append(valueStrings, fmt.Sprintf("($%d::bigint, $%d::text)", argPos, argPos+1))
		valueArgs = append(valueArgs, id, newUserRef)
		argPos += 2
		maxID = id
	}
	if err := rows.Err(); err != nil {
		return 0, 0, false, errors.Wrap(err, "failed to iterate rows")
	}

	// If no rows, we're done
	if len(valueStrings) == 0 {
		return 0, 0, true, nil
	}

	// Execute bulk update: UPDATE audit_log SET ... FROM (VALUES ...) AS v(id, new_user) WHERE audit_log.id = v.id
	updateSQL := fmt.Sprintf(`
		UPDATE audit_log
		SET payload = jsonb_set(payload, '{user}', to_jsonb(v.new_user))
		FROM (VALUES %s) AS v(id, new_user)
		WHERE audit_log.id = v.id
	`, strings.Join(valueStrings, ", "))

	result, err := txn.ExecContext(ctx, updateSQL, valueArgs...)
	if err != nil {
		return 0, 0, false, errors.Wrap(err, "failed to execute batch update")
	}

	rowsAffected, _ := result.RowsAffected()

	// Commit this batch
	if err := txn.Commit(); err != nil {
		return 0, 0, false, errors.Wrap(err, "failed to commit batch")
	}

	return int(rowsAffected), maxID, false, nil
}

// buildUserIDToEmailMap creates a lookup map from user ID to email format.
func buildUserIDToEmailMap(ctx context.Context, conn *sql.Conn) (map[int]string, error) {
	rows, err := conn.QueryContext(ctx, `SELECT id, email FROM principal WHERE email IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userMap := make(map[int]string)
	for rows.Next() {
		var id int
		var email string
		if err := rows.Scan(&id, &email); err != nil {
			return nil, err
		}
		userMap[id] = email
	}
	return userMap, rows.Err()
}

// convertUserIDToEmail converts a user reference from users/{id} to users/{email} format.
func convertUserIDToEmail(userRef string, userMap map[int]string) string {
	// Already in email format
	if !strings.HasPrefix(userRef, "users/") || strings.Contains(userRef, "@") {
		return userRef
	}

	// Extract ID from users/{id}
	idStr := strings.TrimPrefix(userRef, "users/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		// Can't parse as int, return original
		return userRef
	}

	// Look up email
	if email, found := userMap[id]; found {
		return fmt.Sprintf("users/%s", email)
	}

	// Not found, return original
	return userRef
}
