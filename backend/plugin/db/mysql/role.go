package mysql

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (d *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRole, error) {
	var instanceRoles []*storepb.InstanceRole
	var users []string
	var err error
	users, err = d.getUsersFromMySQLUser(ctx)
	if err != nil {
		slog.Info("failed to get users", log.BBError(err))
		users, err = d.getUsersFromUserAttributes(ctx)
		if err != nil {
			slog.Info("failed to get users", log.BBError(err))
			return nil, nil
		}
	}

	// Uses single quote instead of backtick to escape because this is a string
	// instead of table (which should use backtick instead). MySQL actually works
	// in both ways. On the other hand, some other MySQL compatible engines might not (OceanBase in this case).
	for _, name := range users {
		grantList, err := d.getGrantFromUser(ctx, name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get grants for %s", name)
		}
		attribute := strings.Join(grantList, "\n")
		instanceRoles = append(instanceRoles, &storepb.InstanceRole{
			Name:      name,
			Attribute: &attribute,
		})
	}
	return instanceRoles, nil
}

// getGrantFromUser reads grants for user with format "'<user>'@'<host>'".
func (d *Driver) getGrantFromUser(ctx context.Context, name string) ([]string, error) {
	grantQuery := fmt.Sprintf("SHOW GRANTS FOR %s", name)
	grantRows, err := d.db.QueryContext(ctx,
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
func (d *Driver) getUsersFromUserAttributes(ctx context.Context) ([]string, error) {
	var users []string
	query := `
	SELECT
		user,
		host
	FROM information_schema.user_attributes
	WHERE user NOT LIKE 'mysql.%'
	`
	roleRows, err := d.db.QueryContext(ctx, query)
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
func (d *Driver) getUsersFromMySQLUser(ctx context.Context) ([]string, error) {
	var users []string
	query := `
	SELECT
		user,
		host
	FROM mysql.user
	WHERE user NOT LIKE 'mysql.%'
	`
	roleRows, err := d.db.QueryContext(ctx, query)
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
