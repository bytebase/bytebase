package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetMySQLFingerprint(t *testing.T) {
	tests := []struct {
		stmt string
		want string
	}{
		{
			stmt: "-- this is comment\nSELECT * FROM `mytable`",
			want: "select * from `mytable`",
		},
		// Test mysqldump query.
		{
			stmt: "SELECT /*!40001 SQL_NO_CACHE */ * FROM `mytable`",
			want: "mysqldump",
		},
		// Test Percona Toolkit query.
		{
			stmt: "/*foo.bar:1/2*/ SELECT * FROM `mytable`",
			want: "percona-toolkit",
		},
		// Test administrator command.
		{
			stmt: "administrator command: SHOW STATUS",
			want: "administrator command: SHOW STATUS",
		},
		// Test stored procedure call statement.
		{
			stmt: "CALL my_stored_procedure(?, ?)",
			want: "call my_stored_procedure(?, ?)",
		},
		// Test INSERT INTO statement.
		{
			stmt: "INSERT INTO `mytable` (`id`, `name`) VALUES (1, 'John'), (2, 'Doe')",
			want: "insert into `mytable` (`id`, `name`) values(?+)",
		},
		// Test REPLACE INTO statement.
		{
			stmt: "REPLACE INTO `mytable` (`id`, `name`) VALUES (1, 'John'), (2, 'Doe')",
			want: "replace into `mytable` (`id`, `name`) values(?+)",
		},
		// Test multi-line comment.
		{
			stmt: "SELECT * FROM `mytable` /* WHERE `id` = 1 */",
			want: "select * from `mytable`",
		},
		// Test single-line comment.
		{
			stmt: "SELECT * FROM `mytable` -- WHERE `id` = 1",
			want: "select * from `mytable`",
		},
		// Test USE statement.
		{
			stmt: "USE `mydatabase`",
			want: "use ?",
		},
		// Test escape characters in SQL query.
		{
			stmt: "SELECT 'It\\'s raining' FROM `mytable`",
			want: "select ? from `mytable`",
		},
		// Test special characters in SQL query.
		{
			stmt: "SELECT 'Hello, \"world\"!' FROM `mytable`",
			want: "select ? from `mytable`",
		},
		// Test boolean values in SQL query.
		{
			stmt: "SELECT * FROM `mytable` WHERE `is_active` = true",
			want: "select * from `mytable` where `is_active` = ?",
		},
		// Test MD5 values in SQL query.
		{
			stmt: "SELECT * FROM `mytable` WHERE `password` = '5f4dcc3b5aa765d61d8327deb882cf99'",
			want: "select * from `mytable` where `password` = ?",
		},
		// Test numbers in SQL query.
		{
			stmt: "SELECT * FROM `mytable` WHERE `id` = 123",
			want: "select * from `mytable` where `id` = ?",
		},
		// Test special characters in SQL query.
		{
			stmt: "SELECT * FROM `mytable` WHERE `id` IN (1, 2, 3)",
			want: "select * from `mytable` where `id` in(?+)",
		},
		// Test repeated clauses in SQL query.
		{
			stmt: "SELECT * FROM `mytable` WHERE `id` = 1 UNION SELECT * FROM `mytable` WHERE `id` = 2 UNION ALL SELECT * FROM `mytable` WHERE `id` = 3",
			want: "select * from `mytable` where `id` = ? /*repeat union all */",
		},
		// Test LIMIT clause in SQL query.
		{
			stmt: "SELECT * FROM `mytable` LIMIT 10",
			want: "select * from `mytable` limit ?",
		},
		// Test ASC sorting in SQL query.
		{
			stmt: "SELECT * FROM `mytable` ORDER BY `id` ASC, `name` DESC",
			want: "select * from `mytable` order by `id`, `name` desc",
		},
	}

	for _, test := range tests {
		res, err := GetFingerprint(test.stmt)
		require.NoError(t, err, test.stmt)
		require.Equal(t, test.want, res, test.stmt)
	}
}
