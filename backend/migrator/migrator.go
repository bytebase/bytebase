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

	if err := backfill(ctx, conn); err != nil {
		return err
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

func backfill(ctx context.Context, conn *sql.Conn) error {
	var c int
	err := conn.QueryRowContext(ctx, "SELECT count(1) FROM instance_change_history WHERE database_id IS NOT NULL").Scan(&c)
	if err == nil && c > 0 {
		return errors.Errorf("Must upgrade to Bytebase 3.3.1 first")
	}
	if _, err := conn.ExecContext(ctx, `
	ALTER TABLE instance_change_history
	DROP COLUMN IF EXISTS row_status,
	DROP COLUMN IF EXISTS creator_id,
	DROP COLUMN IF EXISTS updater_id,
	DROP COLUMN IF EXISTS created_ts,
	DROP COLUMN IF EXISTS updated_ts,
	DROP COLUMN IF EXISTS instance_id,
	DROP COLUMN IF EXISTS database_id,
	DROP COLUMN IF EXISTS project_id,
	DROP COLUMN IF EXISTS issue_id,
	DROP COLUMN IF EXISTS release_version,
	DROP COLUMN IF EXISTS sequence,
	DROP COLUMN IF EXISTS source,
	DROP COLUMN IF EXISTS type,
	DROP COLUMN IF EXISTS description,
	DROP COLUMN IF EXISTS sheet_id,
	DROP COLUMN IF EXISTS statement,
	DROP COLUMN IF EXISTS schema,
	DROP COLUMN IF EXISTS schema_prev,
	DROP COLUMN IF EXISTS payload;
	CREATE UNIQUE INDEX IF NOT EXISTS idx_instance_change_history_unique_version ON instance_change_history (version);`); err != nil {
		return err
	}
	var hasStatus bool
	if err := conn.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'instance_change_history' AND column_name = 'status')").Scan(&hasStatus); err != nil {
		return err
	}
	if hasStatus {
		if _, err := conn.ExecContext(ctx, `
		DELETE FROM instance_change_history WHERE status = 'FAILED';
		UPDATE instance_change_history
		SET
			version = ARRAY_TO_STRING(
				(STRING_TO_ARRAY(
					SUBSTRING(version, 0, 15),
					'.'
				)::integer[])::text[],
				'.'
			)
		WHERE version LIKE '%-%';`); err != nil {
			return err
		}
	}
	if _, err := conn.ExecContext(ctx, `
	ALTER TABLE instance_change_history
	DROP COLUMN IF EXISTS status,
	DROP COLUMN IF EXISTS execution_duration_ns;`); err != nil {
		return err
	}
	return nil
}
