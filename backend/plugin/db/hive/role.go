package hive

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func (d *Driver) CreateRole(ctx context.Context, roleUpsertMessage *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	_, err := d.Execute(ctx, fmt.Sprintf("CREATE ROLE %s", roleUpsertMessage.Name), db.ExecuteOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create role: %s", roleUpsertMessage.Name)
	}
	return &db.DatabaseRoleMessage{
		Name: roleUpsertMessage.Name,
	}, nil
}

func (*Driver) UpdateRole(_ context.Context, _ string, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("UpdateRole() is not applicable to Hive")
}

func (*Driver) FindRole(_ context.Context, _ string) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("FindRole() is not applicable to Hive")
}

func (d *Driver) ListRole(ctx context.Context) ([]*db.DatabaseRoleMessage, error) {
	roleResults, err := d.QueryConn(ctx, nil, "SHOW ROLES", nil)
	var roleMessages []*db.DatabaseRoleMessage
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get roles")
	}

	for _, row := range roleResults[0].Rows {
		roleMessages = append(roleMessages, &db.DatabaseRoleMessage{
			Name: row.Values[0].GetStringValue(),
		})
	}

	return roleMessages, nil
}

func (d *Driver) DeleteRole(ctx context.Context, roleName string) error {
	_, err := d.Execute(ctx, fmt.Sprintf("DROP ROLE %s", roleName), db.ExecuteOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to drop role %s", roleName)
	}
	return nil
}

// some DMLs need admin role.
func (d *Driver) SetRole(ctx context.Context, roleName string) error {
	_, err := d.Execute(ctx, fmt.Sprintf("SET ROLE %s", roleName), db.ExecuteOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to set role to %s", roleName)
	}
	return nil
}

func (d *Driver) GetRoleGrant(ctx context.Context, roleName string) (string, error) {
	grantResults, err := d.QueryConn(ctx, nil, fmt.Sprintf("SHOW GRANT ROLE %s", roleName), nil)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get grant from %s", roleName)
	}
	var grantStrings []string
	// get grant-info rows from a certain role and combine them into a string.
	for _, row := range grantResults[0].Rows {
		databaseName := row.Values[0].GetStringValue()
		tableName := row.Values[1].GetStringValue()
		privilege := row.Values[6].GetStringValue()
		// TODO(tommy): the format of this string should be carefully considered.
		grantStrings = append(grantStrings, fmt.Sprintf("%s/%s: %s", databaseName, tableName, privilege))
	}
	grantString := strings.Join(grantStrings, "\n")

	return grantString, nil
}
