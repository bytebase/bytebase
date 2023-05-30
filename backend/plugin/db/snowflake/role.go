package snowflake

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// CreateRole creates the role.
func (*Driver) CreateRole(_ context.Context, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("create role for Snowflake is not implemented yet")
}

// UpdateRole updates the role.
func (*Driver) UpdateRole(_ context.Context, _ string, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("update role for Snowflake is not implemented yet")
}

// FindRole finds the role by name.
func (*Driver) FindRole(_ context.Context, _ string) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("find role for Snowflake is not implemented yet")
}

// ListRole lists the role.
func (*Driver) ListRole(_ context.Context) ([]*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("list role for Snowflake is not implemented yet")
}

// DeleteRole deletes the role by name.
func (*Driver) DeleteRole(_ context.Context, _ string) error {
	return errors.Errorf("delete role for Snowflake is not implemented yet")
}

func (driver *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRoleMetadata, error) {
	grantQuery := `
		SELECT
			GRANTEE_NAME,
			ROLE
		FROM SNOWFLAKE.ACCOUNT_USAGE.GRANTS_TO_USERS
	`
	grants := make(map[string][]string)
	grantRows, err := driver.db.QueryContext(ctx, grantQuery)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, grantQuery)
	}
	defer grantRows.Close()
	for grantRows.Next() {
		var name, role string
		if err := grantRows.Scan(
			&name,
			&role,
		); err != nil {
			return nil, err
		}
		grants[name] = append(grants[name], role)
	}
	if err := grantRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, grantQuery)
	}

	// Query user info
	// The same user could have multiple entires in the usage if it's been deleted and recreated,
	// so we need to use DELETED_ON IS NULL to retrieve the active user.
	userQuery := `
	  SELECT
			name
		FROM SNOWFLAKE.ACCOUNT_USAGE.USERS
		WHERE DELETED_ON IS NULL
		ORDER BY name ASC
	`
	var instanceRoles []*storepb.InstanceRoleMetadata
	rows, err := driver.db.QueryContext(ctx, userQuery)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, userQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(
			&name,
		); err != nil {
			return nil, err
		}

		instanceRoles = append(instanceRoles, &storepb.InstanceRoleMetadata{
			Name:  name,
			Grant: strings.Join(grants[name], ", "),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, userQuery)
	}

	return instanceRoles, nil
}
