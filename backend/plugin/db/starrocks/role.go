package starrocks

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// CreateRole creates the role.
func (*Driver) CreateRole(_ context.Context, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("not implemented")
}

// UpdateRole updates the role.
func (*Driver) UpdateRole(_ context.Context, _ string, _ *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("not implemented")
}

// FindRole finds the role by name.
func (*Driver) FindRole(_ context.Context, _ string) (*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("not implemented")
}

// ListRole lists the role.
func (*Driver) ListRole(_ context.Context) ([]*db.DatabaseRoleMessage, error) {
	return nil, errors.Errorf("not implemented")
}

// DeleteRole deletes the role by name.
func (*Driver) DeleteRole(_ context.Context, _ string) error {
	return errors.Errorf("not implemented")
}

func (driver *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRoleMetadata, error) {
	var instanceRoles []*storepb.InstanceRoleMetadata
	var users []string
	var err error
	users, err = driver.getUsersFromMySQLUser(ctx)
	if err != nil {
		slog.Info("failed to get users", log.BBError(err))
		users, err = driver.getUsersFromUserAttributes(ctx)
		if err != nil {
			slog.Info("failed to get users", log.BBError(err))
			return nil, nil
		}
	}

	// Uses single quote instead of backtick to escape because this is a string
	// instead of table (which should use backtick instead). MySQL actually works
	// in both ways. On the other hand, some other MySQL compatible engines might not (OceanBase in this case).
	for _, name := range users {
		grantList, err := driver.getGrantFromUser(ctx, name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get grants for %s", name)
		}
		instanceRoles = append(instanceRoles, &storepb.InstanceRoleMetadata{
			Name:  name,
			Grant: strings.Join(grantList, "\n"),
		})
	}
	return instanceRoles, nil
}

// getGrantFromUser reads grants for user with format "'<user>'@'<host>'".
func (driver *Driver) getGrantFromUser(ctx context.Context, name string) ([]string, error) {
	grantQuery := fmt.Sprintf("SHOW GRANTS FOR %s", name)
	grantRows, err := driver.db.QueryContext(ctx,
		grantQuery,
	)
	if err != nil {
		slog.Info("failed to get grants", slog.String("user", name), log.BBError(err))
		return nil, nil
	}
	defer grantRows.Close()

	grants := []string{}
	for grantRows.Next() {
		var grant string
		if err := grantRows.Scan(&grant); err != nil {
			return nil, err
		}
		grants = append(grants, grant)
	}
	if err := grantRows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to iterate grants for %s", name)
	}
	return grants, nil
}

// getUsersFromUserAttributes reads users from information_schema.user_attributes, returns the list of users with format "'<user>'@'<host>'".
func (driver *Driver) getUsersFromUserAttributes(ctx context.Context) ([]string, error) {
	var users []string
	query := `
	SELECT
		user,
		host
	FROM information_schema.user_attributes
	WHERE user NOT LIKE 'mysql.%'
	`
	roleRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query information_schema.user_attributes")
	}
	defer roleRows.Close()
	for roleRows.Next() {
		var user string
		var host string
		if err := roleRows.Scan(
			&user,
			&host,
		); err != nil {
			return nil, err
		}
		users = append(users, fmt.Sprintf("'%s'@'%s'", user, host))
	}
	if err := roleRows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to iterate information_schema.user_attributes")
	}
	return users, nil
}

// getUsersFromMySQLUser reads users from mysql.user, returns the list of users with format "'<user>'@'<host>'".
func (driver *Driver) getUsersFromMySQLUser(ctx context.Context) ([]string, error) {
	var users []string
	query := `
	SELECT
		user,
		host
	FROM mysql.user
	WHERE user NOT LIKE 'mysql.%'
	`
	roleRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query mysql.user")
	}
	defer roleRows.Close()
	for roleRows.Next() {
		var user string
		var host string
		if err := roleRows.Scan(
			&user,
			&host,
		); err != nil {
			return nil, err
		}
		users = append(users, fmt.Sprintf("'%s'@'%s'", user, host))
	}
	if err := roleRows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to iterate mysql.user")
	}
	return users, nil
}
