package clickhouse

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (driver *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRole, error) {
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

	var instanceRoles []*storepb.InstanceRole
	for roleRows.Next() {
		var user string
		if err := roleRows.Scan(
			&user,
		); err != nil {
			return nil, err
		}
		instanceRoles = append(instanceRoles, &storepb.InstanceRole{
			Name: user,
		})
	}
	if err := roleRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	return instanceRoles, nil
}
