package mysql

import (
	"context"
	"errors"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func newRoleTestDriver(t *testing.T) (*Driver, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectClose()
	t.Cleanup(func() {
		require.NoError(t, db.Close())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	return &Driver{db: db}, mock
}

func TestGetUsersFiltersAnonymousUsers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		query     string
		getUsers  func(*Driver, context.Context) ([]string, error)
		queryRows *sqlmock.Rows
	}{
		{
			name:  "mysql.user",
			query: `(?s)SELECT\s+user,\s+host\s+FROM mysql\.user\s+WHERE user <> ''\s+AND user NOT LIKE 'mysql\.%'`,
			getUsers: func(d *Driver, ctx context.Context) ([]string, error) {
				return d.getUsersFromMySQLUser(ctx)
			},
			queryRows: sqlmock.NewRows([]string{"user", "host"}).AddRow("app", "%"),
		},
		{
			name:  "information_schema.user_attributes",
			query: `(?s)SELECT\s+user,\s+host\s+FROM information_schema\.user_attributes\s+WHERE user <> ''\s+AND user NOT LIKE 'mysql\.%'`,
			getUsers: func(d *Driver, ctx context.Context) ([]string, error) {
				return d.getUsersFromUserAttributes(ctx)
			},
			queryRows: sqlmock.NewRows([]string{"user", "host"}).AddRow("app", "%"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			driver, mock := newRoleTestDriver(t)
			mock.ExpectQuery(tc.query).WillReturnRows(tc.queryRows)

			got, err := tc.getUsers(driver, context.Background())
			require.NoError(t, err)
			require.Equal(t, []string{"'app'@'%'"}, got)
		})
	}
}

func TestGetInstanceRolesSkipsGrantLookupFailure(t *testing.T) {
	t.Parallel()

	driver, mock := newRoleTestDriver(t)
	mock.ExpectQuery(`(?s)SELECT\s+user,\s+host\s+FROM mysql\.user\s+WHERE user <> ''\s+AND user NOT LIKE 'mysql\.%'`).WillReturnRows(
		sqlmock.NewRows([]string{"user", "host"}).
			AddRow("app", "%").
			AddRow("broken", "localhost"),
	)
	mock.ExpectQuery(regexp.QuoteMeta("SHOW GRANTS FOR 'app'@'%'")).WillReturnRows(
		sqlmock.NewRows([]string{"Grants for app@%"}).AddRow("GRANT SELECT ON *.* TO 'app'@'%'"),
	)
	mock.ExpectQuery(regexp.QuoteMeta("SHOW GRANTS FOR 'broken'@'localhost'")).WillReturnError(errors.New("show grants failed"))

	got := driver.getInstanceRoles(context.Background())
	require.Len(t, got, 1)
	require.Equal(t, "'app'@'%'", got[0].Name)
	require.Equal(t, "GRANT SELECT ON *.* TO 'app'@'%'", got[0].GetAttribute())
}
