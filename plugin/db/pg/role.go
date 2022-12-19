package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db/util"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// RoleAttribute is the attribute string for role.
type RoleAttribute string

const (
	// SUPERUSER is the role attribute for rolsuper.
	SUPERUSER RoleAttribute = "SUPERUSER"
	// LOGIN is the role attribute for rolcanlogin.
	LOGIN RoleAttribute = "LOGIN"
	// NOINHERIT is the role attribute for rolinherit.
	// INHERIT is the default value for rolinherit, so we need to use the NOINHERIT.
	NOINHERIT RoleAttribute = "NOINHERIT"
	// CREATEDB is the role attribute for rolcreatedb.
	CREATEDB RoleAttribute = "CREATEDB"
	// CREATEROLE is the role attribute for rolcreaterole.
	CREATEROLE RoleAttribute = "CREATEROLE"
	// REPLICATION is the role attribute for rolreplication.
	REPLICATION RoleAttribute = "REPLICATION"
	// BYPASSRLS is the role attribute for rolbypassrls.
	BYPASSRLS RoleAttribute = "BYPASSRLS"
)

// ToString returns the string value for role attribute.
func (a RoleAttribute) ToString() string {
	return string(a)
}

// CreateRole will create the PG role.
func (driver *Driver) CreateRole(ctx context.Context, upsert *v1pb.DatabaseRoleUpsert) (*v1pb.DatabaseRole, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	if err := createRoleImpl(ctx, txn, upsert); err != nil {
		return nil, err
	}

	role, err := getRoleImpl(ctx, txn, upsert.Name)
	if err != nil {
		return nil, err
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return role, nil
}

// UpdateRole will alter the PG role.
func (driver *Driver) UpdateRole(ctx context.Context, roleName string, upsert *v1pb.DatabaseRoleUpsert) (*v1pb.DatabaseRole, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	if err := alterRoleImpl(ctx, txn, roleName, upsert); err != nil {
		return nil, err
	}

	role, err := getRoleImpl(ctx, txn, upsert.Name)
	if err != nil {
		return nil, err
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return role, nil
}

// FindRole will find the PG role by name.
func (driver *Driver) FindRole(ctx context.Context, roleName string) (*v1pb.DatabaseRole, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	role, err := getRoleImpl(ctx, txn, roleName)
	if err != nil {
		return nil, err
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return role, nil
}

// DeleteRole will drop the PG role by name.
func (driver *Driver) DeleteRole(ctx context.Context, roleName string) error {
	statement := fmt.Sprintf(`DROP ROLE IF EXISTS "%s"`, roleName)
	if _, err := driver.db.ExecContext(ctx, statement); err != nil {
		return util.FormatErrorWithQuery(err, statement)
	}

	return nil
}

func getRoleImpl(ctx context.Context, txn *sql.Tx, roleName string) (*v1pb.DatabaseRole, error) {
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
		WHERE r.rolname = '%s';
	`, roleName)

	role := &v1pb.DatabaseRole{
		Name:      roleName,
		Attribute: &v1pb.DatabaseRoleAttribute{},
	}

	inherit := false

	if err := txn.QueryRowContext(ctx, statement).Scan(
		&role.Name,
		&role.Attribute.SuperUser,
		&inherit,
		&role.Attribute.CreateRole,
		&role.Attribute.CreateDB,
		&role.Attribute.CanLogin,
		&role.Attribute.Replication,
		&role.Attribute.ByPassRLS,
		&role.ValidUntil,
		&role.ConnectionLimit,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.Errorf(common.NotFound, fmt.Sprintf("query %q returned empty row", statement))
		}
		return nil, util.FormatErrorWithQuery(err, statement)
	}

	role.Attribute.NoInherit = !inherit

	return role, nil
}

func createRoleImpl(ctx context.Context, txn *sql.Tx, upsert *v1pb.DatabaseRoleUpsert) error {
	statement := fmt.Sprintf(`CREATE ROLE "%s" %s;`, upsert.Name, convertToAttributeStatement(upsert))
	if _, err := txn.ExecContext(ctx, statement); err != nil {
		return util.FormatErrorWithQuery(err, statement)
	}

	return nil
}

func alterRoleImpl(ctx context.Context, txn *sql.Tx, roleName string, upsert *v1pb.DatabaseRoleUpsert) error {
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

func convertToAttributeStatement(r *v1pb.DatabaseRoleUpsert) string {
	attributeList := []string{}

	if r.Attribute != nil {
		if r.Attribute.SuperUser {
			attributeList = append(attributeList, SUPERUSER.ToString())
		}
		if r.Attribute.NoInherit {
			attributeList = append(attributeList, NOINHERIT.ToString())
		}
		if r.Attribute.CanLogin {
			attributeList = append(attributeList, LOGIN.ToString())
		}
		if r.Attribute.CreateRole {
			attributeList = append(attributeList, CREATEROLE.ToString())
		}
		if r.Attribute.CreateDB {
			attributeList = append(attributeList, CREATEDB.ToString())
		}
		if r.Attribute.Replication {
			attributeList = append(attributeList, REPLICATION.ToString())
		}
		if r.Attribute.ByPassRLS {
			attributeList = append(attributeList, BYPASSRLS.ToString())
		}
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
