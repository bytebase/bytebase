package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// CreateRole creates the role.
func (driver *Driver) CreateRole(ctx context.Context, upsert *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	if err := createRoleImpl(ctx, txn, upsert); err != nil {
		return nil, err
	}

	roles, err := driver.findRoleImpl(ctx, &upsert.Name)
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

// UpdateRole updates the role.
func (driver *Driver) UpdateRole(ctx context.Context, roleName string, upsert *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	if err := alterRoleImpl(ctx, txn, roleName, upsert); err != nil {
		return nil, err
	}

	roles, err := driver.findRoleImpl(ctx, &upsert.Name)
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

// FindRole finds the role by name.
func (driver *Driver) FindRole(ctx context.Context, roleName string) (*db.DatabaseRoleMessage, error) {
	roles, err := driver.findRoleImpl(ctx, &roleName)
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, common.Errorf(common.NotFound, fmt.Sprintf("cannot find the role %s", roleName))
	}

	return roles[0], nil
}

// ListRole lists the role.
func (driver *Driver) ListRole(ctx context.Context) ([]*db.DatabaseRoleMessage, error) {
	roles, err := driver.findRoleImpl(ctx, nil)
	if err != nil {
		return nil, err
	}

	return roles, nil
}

// DeleteRole deletes the role by name.
func (driver *Driver) DeleteRole(ctx context.Context, roleName string) error {
	statement := fmt.Sprintf(`DROP USER IF EXISTS %s`, roleName)
	if _, err := driver.db.ExecContext(ctx, statement); err != nil {
		return util.FormatErrorWithQuery(err, statement)
	}

	return nil
}

func (driver *Driver) findRoleImpl(ctx context.Context, name *string) ([]*db.DatabaseRoleMessage, error) {
	if name != nil {
		attribute, err := driver.findRoleGrant(ctx, *name)
		if err != nil {
			return nil, err
		}

		user, host := parseNameAndHost(*name)
		maxUserConnection, lifetime, err := driver.findConnectionLimitAndExpiration(ctx, user, host)
		if err != nil {
			return nil, err
		}

		var lifetimeString *string
		if lifetime != nil {
			day := fmt.Sprintf("%d", *lifetime)
			lifetimeString = &day
		}
		return []*db.DatabaseRoleMessage{
			{
				Name:            *name,
				Attribute:       &attribute,
				ConnectionLimit: maxUserConnection,
				ValidUntil:      lifetimeString,
			},
		}, nil
	}

	query := `
	  SELECT
			user,
			host
		FROM mysql.user
		WHERE user NOT LIKE 'mysql.%'
	`
	roleRows, err := driver.db.QueryContext(ctx, query)

	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer roleRows.Close()

	var result []*db.DatabaseRoleMessage

	for roleRows.Next() {
		var user string
		var host string
		if err := roleRows.Scan(
			&user,
			&host,
		); err != nil {
			return nil, err
		}

		name := fmt.Sprintf("'%s'@'%s'", user, host)
		attribute, err := driver.findRoleGrant(ctx, name)
		if err != nil {
			return nil, err
		}

		maxUserConnection, lifetime, err := driver.findConnectionLimitAndExpiration(ctx, user, host)
		if err != nil {
			return nil, err
		}

		var lifetimeString *string
		if lifetime != nil {
			day := fmt.Sprintf("%d", *lifetime)
			lifetimeString = &day
		}

		result = append(result, &db.DatabaseRoleMessage{
			Name:            name,
			Attribute:       &attribute,
			ConnectionLimit: maxUserConnection,
			ValidUntil:      lifetimeString,
		})
	}
	if err := roleRows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (driver *Driver) findConnectionLimitAndExpiration(ctx context.Context, user string, host string) (int32, *int32, error) {
	statement := fmt.Sprintf(`SELECT max_user_connections, password_lifetime FROM mysql.user WHERE user = '%s' AND host = '%s'`, user, host)
	rows, err := driver.db.QueryContext(ctx, statement)
	if err != nil {
		return 0, nil, util.FormatErrorWithQuery(err, statement)
	}
	defer rows.Close()

	type result struct {
		maxUserConnection int32
		lifetime          sql.NullInt32
	}
	var list []result

	for rows.Next() {
		var row result
		if err := rows.Scan(
			&row.maxUserConnection,
			&row.lifetime,
		); err != nil {
			return 0, nil, util.FormatErrorWithQuery(err, statement)
		}
		list = append(list, row)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}

	if len(list) == 0 {
		return 0, nil, common.Errorf(common.NotFound, "role '%s'@'%s' not found", user, host)
	}

	// never catch
	if len(list) > 1 {
		return 0, nil, common.Errorf(common.Internal, "role '%s'@'%s' is not unique", user, host)
	}

	lifetime := (*int32)(nil)
	if list[0].lifetime.Valid {
		lifetime = &list[0].lifetime.Int32
	}
	return list[0].maxUserConnection, lifetime, nil
}

// parseNameAndHost parses the MySQL role name, there are two cases:
// 1. user@host: specify host
// 2. user: do not specify host, the host will be `%`
// user and host can be surrounded by quotes('), double-quotes("), black-quotes(`) or not.
func parseNameAndHost(in string) (string, string) {
	reg := regexp.MustCompile("(.*)@(.*)")
	list := reg.FindStringSubmatch(in)
	trimFunc := func(r rune) bool {
		return r == '`' || r == '"' || r == '\''
	}
	if len(list) != 3 {
		// Case Two: do not specify host
		return strings.TrimFunc(in, trimFunc), "%"
	}
	// Case One: specify host
	return strings.TrimFunc(list[1], trimFunc), strings.TrimFunc(list[2], trimFunc)
}

func alterRoleImpl(ctx context.Context, txn *sql.Tx, roleName string, upsert *db.DatabaseRoleUpsertMessage) error {
	if roleName != upsert.Name {
		renameStatement := fmt.Sprintf(`RENAME USER %s to %s`, roleName, upsert.Name)
		if _, err := txn.ExecContext(ctx, renameStatement); err != nil {
			return util.FormatErrorWithQuery(err, renameStatement)
		}
	}

	content, err := convertToUserContent(upsert)
	if err != nil {
		return err
	}

	if content != "" {
		alterStatement := fmt.Sprintf(`ALTER USER IF EXISTS %s %s`, upsert.Name, content)
		if _, err := txn.ExecContext(ctx, alterStatement); err != nil {
			return util.FormatErrorWithQuery(err, alterStatement)
		}
	}

	if upsert.Attribute != nil {
		revokeStatement := fmt.Sprintf(`REVOKE ALL PRIVILEGES, GRANT OPTION FROM %s`, upsert.Name)
		if _, err := txn.ExecContext(ctx, revokeStatement); err != nil {
			return util.FormatErrorWithQuery(err, revokeStatement)
		}

		if len(*upsert.Attribute) > 0 {
			list, err := splitGrantStatement(*upsert.Attribute)
			if err != nil {
				return err
			}
			for _, sql := range list {
				if _, err := txn.ExecContext(ctx, sql.Text); err != nil {
					return util.FormatErrorWithQuery(err, sql.Text)
				}
			}
		}
	}

	return nil
}

func createRoleImpl(ctx context.Context, txn *sql.Tx, upsert *db.DatabaseRoleUpsertMessage) error {
	content, err := convertToUserContent(upsert)
	if err != nil {
		return err
	}
	statement := fmt.Sprintf(`CREATE USER IF NOT EXISTS %s %s`, upsert.Name, content)

	if _, err := txn.ExecContext(ctx, statement); err != nil {
		return util.FormatErrorWithQuery(err, statement)
	}

	if upsert.Attribute != nil && len(*upsert.Attribute) > 0 {
		list, err := splitGrantStatement(*upsert.Attribute)
		if err != nil {
			return err
		}
		for _, sql := range list {
			if _, err := txn.ExecContext(ctx, sql.Text); err != nil {
				return util.FormatErrorWithQuery(err, sql.Text)
			}
		}
	}

	return nil
}

func convertToUserContent(upsert *db.DatabaseRoleUpsertMessage) (string, error) {
	var contentList []string
	if upsert.Password != nil {
		contentList = append(contentList, fmt.Sprintf(`IDENTIFIED BY '%s'`, *upsert.Password))
	}
	if upsert.ConnectionLimit != nil {
		contentList = append(contentList, fmt.Sprintf(`WITH MAX_USER_CONNECTIONS %d`, *upsert.ConnectionLimit))
	}
	if upsert.ValidUntil != nil {
		interval, err := strconv.Atoi(*upsert.ValidUntil)
		if err != nil || interval < 0 {
			return "", common.Wrapf(err, common.Invalid, "invalid MySQL expiration")
		}
		if interval == 0 {
			contentList = append(contentList, "PASSWORD EXPIRE NEVER")
		} else {
			contentList = append(contentList, fmt.Sprintf("PASSWORD EXPIRE INTERVAL %d DAY", interval))
		}
	}
	return strings.Join(contentList, " "), nil
}

func splitGrantStatement(stmts string) ([]base.SingleSQL, error) {
	list, err := base.SplitMultiSQL(storepb.Engine_MYSQL, stmts)
	if err != nil {
		return nil, common.Wrapf(err, common.Invalid, "failed to split grant statement")
	}

	grantReg := regexp.MustCompile("(?i)^GRANT ")
	for _, sql := range list {
		if len(grantReg.FindString(sql.Text)) == 0 {
			return nil, common.Wrapf(err, common.Invalid, "\"%s\" is not the GRANT statement", sql.Text)
		}
	}

	return list, nil
}

func (driver *Driver) findRoleGrant(ctx context.Context, name string) (string, error) {
	grantQuery := fmt.Sprintf("SHOW GRANTS FOR %s", name)
	grantRows, err := driver.db.QueryContext(ctx, grantQuery)
	if err != nil {
		return "", util.FormatErrorWithQuery(err, grantQuery)
	}
	defer grantRows.Close()

	grantList := []string{}
	for grantRows.Next() {
		var grant string
		if err := grantRows.Scan(&grant); err != nil {
			return "", err
		}
		grantList = append(grantList, grant)
	}
	if err := grantRows.Err(); err != nil {
		return "", util.FormatErrorWithQuery(err, grantQuery)
	}

	return strings.Join(grantList, ";\n"), nil
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
