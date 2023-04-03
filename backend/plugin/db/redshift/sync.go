// Package redshift is the plugin for RedShift driver.
package redshift

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := driver.getVersion(ctx)
	if err != nil {
		return nil, err
	}
	instanceRoles, err := driver.getInstanceRoles(ctx)
	if err != nil {
		return nil, err
	}

	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases")
	}

	var filteredDatabases []*storepb.DatabaseMetadata
	for _, database := range databases {
		// Skip all system databases
		if _, ok := excludedDatabaseList[database.Name]; ok {
			continue
		}
		filteredDatabases = append(filteredDatabases, database)
	}

	return &db.InstanceMetadata{
		Version:       version,
		InstanceRoles: instanceRoles,
		Databases:     filteredDatabases,
	}, nil
}

func (driver *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRoleMetadata, error) {
	// Reference: https://sourcegraph.com/github.com/postgres/postgres@REL_14_0/-/blob/src/bin/psql/describe.c?L3792
	query := `
	SELECT
		u.usename AS rolename,
		u.usesuper AS rolsuper,
		true AS rolinherit,
		false AS rolcreaterole,
		u.usecreatedb AS rolcreatedb,
		true AS rolcanlogin,
		-1 AS rolconnlimit,
		u.valuntil as rolvaliduntil
	FROM pg_catalog.pg_user u
	ORDER BY 1;
	`
	var instanceRoles []*storepb.InstanceRoleMetadata
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		var super, inherit, createrole, createdb, canLogin bool
		var connectionLimit int32
		var validUntil sql.NullString
		if err := rows.Scan(
			&role,
			&super,
			&inherit,
			&createrole,
			&createdb,
			&canLogin,
			&connectionLimit,
			&validUntil,
		); err != nil {
			return nil, err
		}

		var attributes []string
		if super {
			attributes = append(attributes, "Superuser")
		}
		if !inherit {
			attributes = append(attributes, "No inheritance")
		}
		if createrole {
			attributes = append(attributes, "Create role")
		}
		if createdb {
			attributes = append(attributes, "Create DB")
		}
		if !canLogin {
			attributes = append(attributes, "Cannot login")
		}
		if connectionLimit >= 0 {
			if connectionLimit == 0 {
				attributes = append(attributes, "No connections")
			} else if connectionLimit == 1 {
				attributes = append(attributes, "1 connection")
			} else {
				attributes = append(attributes, fmt.Sprintf("%d connections", connectionLimit))
			}
		}
		if validUntil.Valid {
			attributes = append(attributes, fmt.Sprintf("Password valid until %s", validUntil.String))
		}
		instanceRoles = append(instanceRoles, &storepb.InstanceRoleMetadata{
			Name:  role,
			Grant: strings.Join(attributes, ", "),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return instanceRoles, nil
}

// SyncDBSchema syncs a single database schema.
func (*Driver) SyncDBSchema(_ context.Context, _ string) (*storepb.DatabaseMetadata, error) {
	// TODO(zp): implement it.
	return &storepb.DatabaseMetadata{}, nil
}

func (driver *Driver) getVersion(ctx context.Context) (string, error) {
	// Redshift doesn't support SHOW server_version to retrieve the clean version number.
	// We can parse the output of `SELECT version()` to get the PostgreSQL version and the
	// Redshift version because Redshift is based on PostgreSQL.
	// For example, the output of `SELECT version()` is:
	// PostgreSQL 8.0.2 on i686-pc-linux-gnu, compiled by GCC gcc (GCC) 3.4.2 20041017 (Red Hat 3.4.2-6.fc3), Redshift 1.0.48042
	// We will return the 'Redshift 1.0.48042 based on PostgreSQL 8.0.2'.
	rows, err := driver.db.QueryContext(ctx, "SELECT version()")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var version string
	for rows.Next() {
		if err := rows.Scan(&version); err != nil {
			return "", err
		}
	}
	// We try to parse the version string to get the PostgreSQL version and the Redshift version, but it's not a big deal if we fail.
	// We will just return the version string as is.
	pgVersion, redshiftVersion, err := getPgVersionAndRedshiftVersion(version)
	if err != nil {
		log.Debug("Failed to parse version string", zap.String("version", version))
		// nolint
		return version, nil
	}
	return buildRedshiftVersionString(redshiftVersion, pgVersion), nil
}

// parseVersionRegex is a regex to parse the output from Redshift's `SELECT version()`, captures the PostgreSQL version and the Redshift version.
var parseVersionRegex = regexp.MustCompile(`(?i)^PostgreSQL (?P<pgVersion>\d+\.\d+\.\d+) on .*, Redshift (?P<redshiftVersion>\d+\.\d+\.\d+)`)

// getPgVersionAndRedshiftVersion parses the output from Redshift's `SELECT version()` to get the PostgreSQL version and the Redshift version.
func getPgVersionAndRedshiftVersion(version string) (string, string, error) {
	matches := parseVersionRegex.FindStringSubmatch(version)
	if len(matches) == 0 {
		return "", "", errors.Errorf("unable to parse version string: %s", version)
	}

	pgVersion := ""
	redshiftVersion := ""
	for i, name := range parseVersionRegex.SubexpNames() {
		if i != 0 && name != "" {
			switch name {
			case "pgVersion":
				pgVersion = matches[i]
			case "redshiftVersion":
				redshiftVersion = matches[i]
			}
		}
	}

	return pgVersion, redshiftVersion, nil
}

// buildRedshiftVersionString builds the Redshift version string, format is "Redshift <redshiftVersion> based on PostgreSQL <postgresVersion>".
func buildRedshiftVersionString(redshiftVersion, postgresVersion string) string {
	return "Redshift " + redshiftVersion + " based on PostgreSQL " + postgresVersion
}

// getDatabases gets all databases of an instance.
func (driver *Driver) getDatabases(ctx context.Context) ([]*storepb.DatabaseMetadata, error) {
	var databases []*storepb.DatabaseMetadata
	rows, err := driver.db.QueryContext(ctx, `
		SELECT datname,
		pg_encoding_to_char(encoding)
		FROM pg_database;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		database := storepb.DatabaseMetadata{}
		if err := rows.Scan(&database.Name, &database.CharacterSet); err != nil {
			return nil, err
		}
		databases = append(databases, &database)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return databases, nil
}
