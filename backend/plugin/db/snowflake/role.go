package snowflake

import (
	"context"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (d *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRole, error) {
	grantQuery := `
		SELECT
			GRANTEE_NAME,
			ROLE
		FROM SNOWFLAKE.ACCOUNT_USAGE.GRANTS_TO_USERS
	`
	grants := make(map[string][]string)
	grantRows, err := d.db.QueryContext(ctx, grantQuery)
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
	var instanceRoles []*storepb.InstanceRole
	rows, err := d.db.QueryContext(ctx, userQuery)
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

		attribute := strings.Join(grants[name], ", ")
		instanceRoles = append(instanceRoles, &storepb.InstanceRole{
			Name:      name,
			Attribute: &attribute,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, userQuery)
	}

	return instanceRoles, nil
}
