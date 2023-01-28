package clickhouse

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// CreateRole creates the role.
func (*Driver) CreateRole(_ context.Context, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("create role for ClickHouse is not implemented yet")
}

// UpdateRole updates the role.
func (*Driver) UpdateRole(_ context.Context, _ string, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("update role for ClickHouse is not implemented yet")
}

// FindRole finds the role by name.
func (*Driver) FindRole(_ context.Context, _ string) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("find role for ClickHouse is not implemented yet")
}

// ListRole lists the role.
func (*Driver) ListRole(_ context.Context) ([]*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("list role for ClickHouse is not implemented yet")
}

// DeleteRole deletes the role by name.
func (*Driver) DeleteRole(_ context.Context, _ string) error {
	return errors.Errorf("delete role for ClickHouse is not implemented yet")
}

func (driver *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRoleMetadata, error) {
	// host_ip isn't used for user identifier.
	query := `
	  SELECT
			name
		FROM system.users
	`
	roleRows, err := driver.db.QueryContext(ctx, query)

	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer roleRows.Close()

	var instanceRoles []*storepb.InstanceRoleMetadata
	for roleRows.Next() {
		var user string
		if err := roleRows.Scan(
			&user,
		); err != nil {
			return nil, err
		}

		if err := func() error {
			// Uses single quote instead of backtick to escape because this is a string
			// instead of table (which should use backtick instead). MySQL actually works
			// in both ways. On the other hand, some other MySQL compatible engines might not (OceanBase in this case).
			grantQuery := fmt.Sprintf("SHOW GRANTS FOR %s", user)
			grantRows, err := driver.db.QueryContext(ctx,
				grantQuery,
			)
			if err != nil {
				return util.FormatErrorWithQuery(err, grantQuery)
			}
			defer grantRows.Close()

			grantList := []string{}
			for grantRows.Next() {
				var grant string
				if err := grantRows.Scan(&grant); err != nil {
					return err
				}
				grantList = append(grantList, grant)
			}
			if err := grantRows.Err(); err != nil {
				return util.FormatErrorWithQuery(err, grantQuery)
			}

			instanceRoles = append(instanceRoles, &storepb.InstanceRoleMetadata{
				Name:  user,
				Grant: strings.Join(grantList, "\n"),
			})
			return nil
		}(); err != nil {
			return nil, err
		}
	}
	if err := roleRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	return instanceRoles, nil
}
