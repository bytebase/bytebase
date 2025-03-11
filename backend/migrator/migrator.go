// Package migrator handles store schema migration.
package migrator

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
)

//go:embed migration
var migrationFS embed.FS

// MigrateSchema migrates the schema for metadata database.
func MigrateSchema(ctx context.Context, db *sql.DB) (*semver.Version, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := backfillSchemaObjectOwner(ctx, conn); err != nil {
		return nil, err
	}

	// Calculate prod cutoffSchemaVersion.
	cutoffSchemaVersion, err := getProdCutoffVersion()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cutoff version")
	}
	slog.Info(fmt.Sprintf("The prod cutoff schema version: %s", cutoffSchemaVersion))
	if err := initializeSchema(ctx, conn, cutoffSchemaVersion); err != nil {
		return nil, err
	}
	var c int
	err = conn.QueryRowContext(ctx, "SELECT count(1) FROM instance_change_history WHERE database_id IS NOT NULL").Scan(&c)
	if err == nil && c > 0 {
		return nil, errors.Errorf("Must upgrade to Bytebase 3.3.1 first")
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
		return nil, err
	}
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
		return nil, err
	}

	verBefore, err := getLatestMigrationVersion(ctx, conn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current schema version")
	}

	if _, err := migrate(ctx, conn, *verBefore); err != nil {
		return nil, errors.Wrap(err, "failed to migrate")
	}

	verAfter, err := getLatestMigrationVersion(ctx, conn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current schema version")
	}
	slog.Info(fmt.Sprintf("Current schema version after migration: %s", verAfter))

	return verAfter, nil
}

func initializeSchema(ctx context.Context, conn *sql.Conn, cutoffSchemaVersion semver.Version) error {
	// We use environment table to determine whether we've initialized the schema.
	var exists bool
	if err := conn.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'environment')`,
	).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	slog.Info("The database schema has not been setup.")

	latestSchemaPath := fmt.Sprintf("migration/%s", latestSchemaFile)
	buf, err := migrationFS.ReadFile(latestSchemaPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read latest schema %q", latestSchemaPath)
	}

	version := cutoffSchemaVersion.String()
	// Set role to database owner so that the schema owner and database owner are consistent.
	owner, err := getCurrentDatabaseOwner(ctx, conn)
	if err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET ROLE '%s'", owner)); err != nil {
		return err
	}

	if err := executeMigration(ctx, conn, string(buf), version); err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Completed database initial migration with version %s.", cutoffSchemaVersion))
	return nil
}

// getCurrentDatabaseOwner gets the role of the current database.
func getCurrentDatabaseOwner(ctx context.Context, conn *sql.Conn) (string, error) {
	const query = `
		SELECT
			u.rolname
		FROM
			pg_roles AS u JOIN pg_database AS d ON (d.datdba = u.oid)
		WHERE
			d.datname = current_database();
		`
	var owner string
	if err := conn.QueryRowContext(ctx, query).Scan(&owner); err != nil {
		return "", err
	}
	return owner, nil
}

func getCurrentUser(ctx context.Context, conn *sql.Conn) (string, error) {
	row := conn.QueryRowContext(ctx, "SELECT current_user;")
	var user string
	if err := row.Scan(&user); err != nil {
		return "", err
	}
	return user, nil
}

func backfillSchemaObjectOwner(ctx context.Context, conn *sql.Conn) error {
	currentUser, err := getCurrentUser(ctx, conn)
	if err != nil {
		return err
	}
	databaseOwner, err := getCurrentDatabaseOwner(ctx, conn)
	if err != nil {
		return err
	}
	if currentUser == databaseOwner {
		return nil
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("reassign owned by %s to %s;", currentUser, databaseOwner)); err != nil {
		return err
	}
	return nil
}

const (
	latestSchemaFile = "LATEST.sql"
)

func migrate(ctx context.Context, conn *sql.Conn, curVer semver.Version) (bool, error) {
	slog.Info("Apply database migration if needed...")
	slog.Info(fmt.Sprintf("Current schema version before migration: %s", curVer))

	// Apply migrations if needed.
	retVersion := curVer
	names, err := fs.Glob(migrationFS, "migration/*")
	if err != nil {
		return false, err
	}

	minorVersions, err := getMinorMigrationVersions(names, curVer)
	if err != nil {
		return false, err
	}

	for _, minorVersion := range minorVersions {
		slog.Info(fmt.Sprintf("Starting minor version migration cycle from %s ...", minorVersion))
		names, err := fs.Glob(migrationFS, fmt.Sprintf("migration/%d.%d/*.sql", minorVersion.Major, minorVersion.Minor))
		if err != nil {
			return false, err
		}
		patchVersions, err := getPatchVersions(minorVersion, curVer, names)
		if err != nil {
			return false, err
		}

		for _, pv := range patchVersions {
			buf, err := fs.ReadFile(migrationFS, pv.filename)
			if err != nil {
				return false, errors.Wrapf(err, "failed to read migration file %q", pv.filename)
			}
			slog.Info(fmt.Sprintf("Migrating %s...", pv.version))
			version := pv.version.String()
			if err := executeMigration(ctx, conn, string(buf), version); err != nil {
				return false, err
			}
			retVersion = pv.version
		}
		if retVersion.EQ(curVer) {
			slog.Info(fmt.Sprintf("Database schema is at version %s; nothing to migrate.", curVer))
		} else {
			slog.Info(fmt.Sprintf("Completed database migration from version %s to %s.", curVer, retVersion))
		}
	}

	return len(minorVersions) > 0, nil
}

func getProdCutoffVersion() (semver.Version, error) {
	minorPathPrefix := "migration/*"
	names, err := fs.Glob(migrationFS, minorPathPrefix)
	if err != nil {
		return semver.Version{}, err
	}

	versions, err := getMinorVersions(names)
	if err != nil {
		return semver.Version{}, err
	}
	if len(versions) == 0 {
		return semver.Version{}, errors.Errorf("migration path %s has no minor version", minorPathPrefix)
	}
	minorVersion := versions[len(versions)-1]

	patchPathPrefix := fmt.Sprintf("migration/%d.%d", minorVersion.Major, minorVersion.Minor)
	names, err = fs.Glob(migrationFS, fmt.Sprintf("%s/*.sql", patchPathPrefix))
	if err != nil {
		return semver.Version{}, err
	}
	patchVersions, err := getPatchVersions(minorVersion, semver.Version{} /* currentVersion */, names)
	if err != nil {
		return semver.Version{}, err
	}
	if len(patchVersions) == 0 {
		return semver.Version{}, errors.Errorf("migration path %s has no patch version", patchPathPrefix)
	}
	return patchVersions[len(patchVersions)-1].version, nil
}

type patchVersion struct {
	version  semver.Version
	filename string
}

// getPatchVersions gets the patch versions above the current version in a minor version directory.
func getPatchVersions(minorVersion semver.Version, currentVersion semver.Version, names []string) ([]patchVersion, error) {
	var patchVersions []patchVersion
	for _, name := range names {
		baseName := filepath.Base(name)
		parts := strings.Split(baseName, "##")
		if len(parts) != 2 {
			return nil, errors.Errorf("migration filename %q should include '##'", name)
		}
		patch, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "migration filename prefix %q should be four digits integer such as '0000'", parts[0])
		}
		version := minorVersion
		version.Patch = uint64(patch)
		if version.LE(currentVersion) {
			continue
		}

		patchVersions = append(patchVersions,
			patchVersion{
				version:  version,
				filename: name,
			},
		)
	}
	if len(patchVersions) == 0 {
		return nil, nil
	}
	// Sort patch version in ascending order.
	sort.Slice(patchVersions, func(i, j int) bool {
		return patchVersions[i].version.LT(patchVersions[j].version)
	})
	return patchVersions, nil
}

// getMinorMigrationVersions gets all the prod minor versions since currentVersion (included).
func getMinorMigrationVersions(names []string, currentVersion semver.Version) ([]semver.Version, error) {
	versions, err := getMinorVersions(names)
	if err != nil {
		return nil, err
	}

	// We should still include the version with the same minor version with currentVersion in case we have missed some patches.
	currentVersion.Patch = 0

	var migrateVersions []semver.Version
	for _, version := range versions {
		// If the migration version is less than to the current version, we will skip the migration since it's already applied.
		// We should still double check the current version in case there's any patch needed.
		if version.LT(currentVersion) {
			slog.Debug(fmt.Sprintf("Skip migration %s; the current schema version %s is higher.", version, currentVersion))
			continue
		}
		migrateVersions = append(migrateVersions, version)
	}
	return migrateVersions, nil
}

// getMinorVersions returns the minor versions based on file names in the prod directory.
func getMinorVersions(names []string) ([]semver.Version, error) {
	var versions []semver.Version
	for _, name := range names {
		baseName := filepath.Base(name)
		if baseName == latestSchemaFile {
			continue
		}
		// Convert minor version to semantic version format, e.g. "1.12" will be "1.12.0".
		s := fmt.Sprintf("%s.0", baseName)
		v, err := semver.Make(s)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid migration file path %q", name)
		}
		versions = append(versions, v)
	}
	// Sort the migration semantic version in ascending order.
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].LT(versions[j])
	})
	return versions, nil
}

func executeMigration(ctx context.Context, conn *sql.Conn, statement string, version string) error {
	startedNs := time.Now().UnixNano()

	txn, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	if _, err := txn.ExecContext(ctx, statement); err != nil {
		return err
	}
	durationNano := time.Now().UnixNano() - startedNs
	if _, err := txn.ExecContext(ctx,
		`INSERT INTO instance_change_history (status, version, execution_duration_ns) VALUES ($1, $2, $3)`, "DONE",
		version,
		durationNano,
	); err != nil {
		return err
	}

	return txn.Commit()
}

func getLatestMigrationVersion(ctx context.Context, conn *sql.Conn) (*semver.Version, error) {
	query := `SELECT version FROM instance_change_history WHERE status = 'DONE' ORDER BY id DESC`

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
