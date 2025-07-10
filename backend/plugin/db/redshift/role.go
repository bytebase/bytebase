package redshift

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func (d *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRole, error) {
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
	var instanceRoles []*storepb.InstanceRole
	rows, err := d.db.QueryContext(ctx, query)
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
			switch connectionLimit {
			case 0:
				attributes = append(attributes, "No connections")
			case 1:
				attributes = append(attributes, "1 connection")
			default:
				attributes = append(attributes, fmt.Sprintf("%d connections", connectionLimit))
			}
		}
		if validUntil.Valid {
			attributes = append(attributes, fmt.Sprintf("Password valid until %s", validUntil.String))
		}
		attribute := strings.Join(attributes, " ")
		instanceRoles = append(instanceRoles, &storepb.InstanceRole{
			Name:      role,
			Attribute: &attribute,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return instanceRoles, nil
}
