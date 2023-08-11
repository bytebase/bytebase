package risingwave

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// RoleAttribute is the attribute string for role.
type RoleAttribute string

const (
	// SUPERUSER is the role attribute for rolsuper.
	SUPERUSER RoleAttribute = "SUPERUSER"
	// NOSUPERUSER is the role attribute for rolsuper.
	NOSUPERUSER RoleAttribute = "NOSUPERUSER"
	// LOGIN is the role attribute for rolcanlogin.
	LOGIN RoleAttribute = "LOGIN"
	// NOLOGIN is the role attribute for rolcanlogin.
	NOLOGIN RoleAttribute = "NOLOGIN"
	// INHERIT is the role attribute for rolinherit.
	INHERIT RoleAttribute = "INHERIT"
	// NOINHERIT is the role attribute for rolinherit.
	NOINHERIT RoleAttribute = "NOINHERIT"
	// CREATEDB is the role attribute for rolcreatedb.
	CREATEDB RoleAttribute = "CREATEDB"
	// NOCREATEDB is the role attribute for rolcreatedb.
	NOCREATEDB RoleAttribute = "NOCREATEDB"
	// CREATEROLE is the role attribute for rolcreaterole.
	CREATEROLE RoleAttribute = "CREATEROLE"
	// NOCREATEROLE is the role attribute for rolcreaterole.
	NOCREATEROLE RoleAttribute = "NOCREATEROLE"
	// REPLICATION is the role attribute for rolreplication.
	REPLICATION RoleAttribute = "REPLICATION"
	// NOREPLICATION is the role attribute for rolreplication.
	NOREPLICATION RoleAttribute = "NOREPLICATION"
	// BYPASSRLS is the role attribute for rolbypassrls.
	BYPASSRLS RoleAttribute = "BYPASSRLS"
	// NOBYPASSRLS is the role attribute for rolbypassrls.
	NOBYPASSRLS RoleAttribute = "NOBYPASSRLS"
)

// ToString returns the string value for role attribute.
func (a RoleAttribute) ToString() string {
	return string(a)
}

type roleFind struct {
	Name *string
}

// CreateRole will create the PG role.
func (driver *Driver) CreateRole(ctx context.Context, upsert *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	if err := createRoleImpl(ctx, txn, upsert); err != nil {
		return nil, err
	}

	roles, err := findRoleImpl(ctx, txn, &roleFind{
		Name: &upsert.Name,
	})
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, common.Errorf(common.NotFound, fmt.Sprintf("cannot find the role %s", upsert.Name))
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return roles[0], nil
}

// UpdateRole will alter the PG role.
func (driver *Driver) UpdateRole(ctx context.Context, roleName string, upsert *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	if err := alterRoleImpl(ctx, txn, roleName, upsert); err != nil {
		return nil, err
	}

	roles, err := findRoleImpl(ctx, txn, &roleFind{
		Name: &upsert.Name,
	})
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, common.Errorf(common.NotFound, fmt.Sprintf("cannot find the role %s", upsert.Name))
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return roles[0], nil
}

// FindRole will find the PG role by name.
func (driver *Driver) FindRole(ctx context.Context, roleName string) (*db.DatabaseRoleMessage, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	roles, err := findRoleImpl(ctx, txn, &roleFind{
		Name: &roleName,
	})
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, common.Errorf(common.NotFound, fmt.Sprintf("cannot find the role %s", roleName))
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return roles[0], nil
}

// ListRole lists the role.
func (driver *Driver) ListRole(ctx context.Context) ([]*db.DatabaseRoleMessage, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	roles, err := findRoleImpl(ctx, txn, &roleFind{})
	if err != nil {
		return nil, err
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return roles, nil
}

// DeleteRole will drop the PG role by name.
func (driver *Driver) DeleteRole(ctx context.Context, roleName string) error {
	statement := fmt.Sprintf(`DROP ROLE IF EXISTS "%s"`, roleName)
	if _, err := driver.db.ExecContext(ctx, statement); err != nil {
		return util.FormatErrorWithQuery(err, statement)
	}

	return nil
}

func findRoleImpl(ctx context.Context, txn *sql.Tx, find *roleFind) ([]*db.DatabaseRoleMessage, error) {
	where := []string{}
	if v := find.Name; v != nil {
		where = append(where, fmt.Sprintf("r.rolname = '%s'", *v))
	}
	if len(where) == 0 {
		where = append(where, "r.rolname !~ '^pg_'")
	}

	statement := fmt.Sprintf(`
		SELECT
			r.rolname,
			r.rolsuper,
			r.rolinherit,
			r.rolcreaterole,
			r.rolcreatedb,
			r.rolcanlogin,
			r.rolreplication,
			r.rolbypassrls,
			r.rolvaliduntil,
			r.rolconnlimit
		FROM pg_catalog.pg_roles r
		WHERE %s;
	`, strings.Join(where, " AND "))

	rows, err := txn.QueryContext(ctx, statement)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, statement)
	}
	defer rows.Close()

	var roleList []*db.DatabaseRoleMessage

	for rows.Next() {
		var name string
		var super, inherit, createRole, createDB, canLogin, replication, bypassRLS bool
		var validUntil sql.NullString
		var connectionLimit int32
		if err := rows.Scan(
			&name,
			&super,
			&inherit,
			&createRole,
			&createDB,
			&canLogin,
			&replication,
			&bypassRLS,
			&validUntil,
			&connectionLimit,
		); err != nil {
			return nil, util.FormatErrorWithQuery(err, statement)
		}

		var attributes []string
		if super {
			attributes = append(attributes, SUPERUSER.ToString())
		} else {
			attributes = append(attributes, NOSUPERUSER.ToString())
		}
		if inherit {
			attributes = append(attributes, INHERIT.ToString())
		} else {
			attributes = append(attributes, NOINHERIT.ToString())
		}
		if createRole {
			attributes = append(attributes, CREATEROLE.ToString())
		} else {
			attributes = append(attributes, NOCREATEROLE.ToString())
		}
		if createDB {
			attributes = append(attributes, CREATEDB.ToString())
		} else {
			attributes = append(attributes, NOCREATEDB.ToString())
		}
		if canLogin {
			attributes = append(attributes, LOGIN.ToString())
		} else {
			attributes = append(attributes, NOLOGIN.ToString())
		}
		if replication {
			attributes = append(attributes, REPLICATION.ToString())
		} else {
			attributes = append(attributes, NOREPLICATION.ToString())
		}
		if bypassRLS {
			attributes = append(attributes, BYPASSRLS.ToString())
		} else {
			attributes = append(attributes, NOBYPASSRLS.ToString())
		}

		attribute := strings.Join(attributes, " ")
		role := &db.DatabaseRoleMessage{
			Name:            name,
			ConnectionLimit: connectionLimit,
			Attribute:       &attribute,
		}

		if validUntil.Valid {
			role.ValidUntil = &validUntil.String
		}

		roleList = append(roleList, role)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roleList, nil
}

func createRoleImpl(ctx context.Context, txn *sql.Tx, upsert *db.DatabaseRoleUpsertMessage) error {
	statement := fmt.Sprintf(`CREATE ROLE "%s" %s;`, upsert.Name, convertToAttributeStatement(upsert))
	if _, err := txn.ExecContext(ctx, statement); err != nil {
		return util.FormatErrorWithQuery(err, statement)
	}

	return nil
}

func alterRoleImpl(ctx context.Context, txn *sql.Tx, roleName string, upsert *db.DatabaseRoleUpsertMessage) error {
	if roleName != upsert.Name {
		renameStatement := fmt.Sprintf(`ALTER ROLE "%s" RENAME TO "%s";`, roleName, upsert.Name)
		if _, err := txn.ExecContext(ctx, renameStatement); err != nil {
			return util.FormatErrorWithQuery(err, renameStatement)
		}
	}

	attributeStatement := convertToAttributeStatement(upsert)
	if attributeStatement == "" {
		return nil
	}

	statement := fmt.Sprintf(`ALTER ROLE "%s" %s;`, upsert.Name, attributeStatement)
	if _, err := txn.ExecContext(ctx, statement); err != nil {
		return util.FormatErrorWithQuery(err, statement)
	}

	return nil
}

func convertToAttributeStatement(r *db.DatabaseRoleUpsertMessage) string {
	attributeList := []string{}

	if r.Attribute != nil {
		attributeList = append(attributeList, *r.Attribute)
	}

	if v := r.Password; v != nil {
		attributeList = append(attributeList, fmt.Sprintf("ENCRYPTED PASSWORD '%s'", *v))
	}
	if v := r.ValidUntil; v != nil {
		attributeList = append(attributeList, fmt.Sprintf("VALID UNTIL '%s'", *v))
	}
	if v := r.ConnectionLimit; v != nil {
		attributeList = append(attributeList, fmt.Sprintf("CONNECTION LIMIT %d", *v))
	}

	attribute := ""
	if len(attributeList) > 0 {
		attribute = fmt.Sprintf("WITH %s", strings.Join(attributeList, " "))
	}

	return attribute
}

func (driver *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRoleMetadata, error) {
	query := `
		SELECT r.rolname, r.rolsuper, r.rolinherit, r.rolcreaterole, r.rolcreatedb, r.rolcanlogin, r.rolreplication, r.rolvaliduntil, r.rolbypassrls
		FROM pg_catalog.pg_roles r
		WHERE r.rolname !~ '^pg_';
	`
	var instanceRoles []*storepb.InstanceRoleMetadata
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
