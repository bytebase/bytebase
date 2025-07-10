package clickhouse

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func (d *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRole, error) {
	// host_ip isn't used for user identifier.
	query := `
	  SELECT
			name
		FROM system.users
	`
	roleRows, err := d.db.QueryContext(ctx, query)

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
