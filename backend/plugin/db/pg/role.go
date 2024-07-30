package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (driver *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRole, error) {
	query := `
		SELECT r.rolname, r.rolsuper, r.rolinherit, r.rolcreaterole, r.rolcreatedb, r.rolcanlogin, r.rolreplication, r.rolvaliduntil, r.rolbypassrls
		FROM pg_catalog.pg_roles r
		WHERE r.rolname !~ '^pg_';
	`
	var instanceRoles []*storepb.InstanceRole
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		var super, inherit, createRole, createDB, canLogin, replication, bypassRLS bool
		var rolValidUntil sql.NullString
		if err := rows.Scan(
			&role,
			&super,
			&inherit,
			&createRole,
			&createDB,
			&canLogin,
			&replication,
			&rolValidUntil,
			&bypassRLS,
		); err != nil {
			return nil, err
		}
		if pgparser.IsSystemUser(role) {
			continue
		}

		var attributes []string
		if super {
			attributes = append(attributes, "Superuser")
		}
		if !inherit {
			attributes = append(attributes, "No inheritance")
		}
		if createRole {
			attributes = append(attributes, "Create role")
		}
		if createDB {
			attributes = append(attributes, "Create DB")
		}
		if !canLogin {
			attributes = append(attributes, "Cannot login")
		}
		if replication {
			attributes = append(attributes, "Replication")
		}
		if rolValidUntil.Valid {
			attributes = append(attributes, fmt.Sprintf("Password valid until %s", rolValidUntil.String))
		}
		if bypassRLS {
			attributes = append(attributes, "Bypass RLS+")
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
