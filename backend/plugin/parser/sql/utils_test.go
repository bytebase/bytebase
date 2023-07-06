package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractTiDBUnsupportStmts(t *testing.T) {
	tests := []struct {
		stmts         string
		wantUnsupport []string
		wantSupport   string
		wantErr       bool
	}{
		{
			stmts: "CREATE TABLE t1(id INT);\n" +
				`CREATE TRIGGER order_insert_audit 
				AFTER INSERT ON orders
				FOR EACH ROW 
				BEGIN
					INSERT INTO orders_audit(order_id, order_date, customer_id, order_amount)
					VALUES (NEW.order_id, NEW.order_date, NEW.customer_id, NEW.order_amount);
				END;`,
			wantUnsupport: []string{
				`
CREATE TRIGGER order_insert_audit 
				AFTER INSERT ON orders
				FOR EACH ROW 
				BEGIN
					INSERT INTO orders_audit(order_id, order_date, customer_id, order_amount)
					VALUES (NEW.order_id, NEW.order_date, NEW.customer_id, NEW.order_amount);
				END
;`,
			},
			wantSupport: "CREATE TABLE t1(id INT);\n",
			wantErr:     false,
		},
	}
	a := require.New(t)
	for _, test := range tests {
		gotUnsupport, gotSupport, err := ExtractTiDBUnsupportStmts(test.stmts)
		if test.wantErr {
			a.Error(err)
		} else {
			a.NoError(err)
			a.Equal(test.wantUnsupport, gotUnsupport)
			a.Equal(test.wantSupport, gotSupport)
		}
	}
}

func TestIsTiDBUnsupportStmt(t *testing.T) {
	tests := []struct {
		stmt string
		want bool
	}{
		{
			stmt: "DELIMITER ;;",
			want: true,
		},
		{
			stmt: "delimiter ;;",
			want: true,
		},
		{
			stmt: "CREATE DEFINER=`root`@`%` TRIGGER `ins_sum` BEFORE INSERT ON `account` FOR EACH SET @sum=@sum + NEW.price;;",
			want: true,
		},
		{
			stmt: "create trigger `ins_sum` BEFORE INSERT ON `account` FOR EACH SET @sum=@sum + NEW.price;;",
			want: true,
		},
		{
			stmt: "DROP TRIGGER `ins_sum`;",
			want: true,
		},
		{
			stmt: "DROP TRIGGER IF EXISTS `ins_sum`;",
			want: true,
		},
		{
			stmt: "CREATE TABLE t1(id INT, name VARCHAR(50));",
			want: false,
		},
		{
			stmt: "CREATE DEFINER=`root`@`%` PROCEDURE `citycount` (IN `country` CHAR(3), OUT `cities` INT)\n" +
				"BEGIN\n" +
				"	SELECT COUNT(*) INTO cities FROM world.city\n" +
				"WHERE CountryCode = country;\n" +
				"END//",
			want: true,
		},
		{
			stmt: "create procedure `citycount` (IN `country` CHAR(3), OUT `cities` INT)\n" +
				"BEGIN\n" +
				"	SELECT COUNT(*) INTO cities FROM world.city\n" +
				"WHERE CountryCode = country;\n" +
				"END//",
			want: true,
		},
		{
			stmt: "CREATE DEFINER=`root`@`%` FUNCTION `hello`(s CHAR(20)) RETURNS CHAR(50) DETERMINISTIC\n" +
				"RETURN CONCAT('Hello, ',s,'!');",
			want: true,
		},
		{
			stmt: "CREATE DEFINER=root@%% FUNCTION `hello`(s CHAR(20)) RETURNS CHAR(50) DETERMINISTIC\n" +
				"RETURN CONCAT('Hello, ',s,'!');",
			want: true,
		},
		{
			stmt: "create function `hello` (s CHAR(20)) RETURNS CHAR(50) DETERMINISTIC\n" +
				"RETURN CONCAT('Hello, ',s,'!');",
			want: true,
		},
		{
			stmt: "CREATE DEFINER=`root`@`%` EVENT `test_event_01` ON SCHEDULE AT CURRENT_TIMESTAMP \n" +
				"DO\n" +
				"	INSERT INTO message(message, created_at)\n" +
				"	VALUES('test event', NOW());",
			want: true,
		},
		{
			stmt: "create event `test_event_01` ON SCHEDULE AT CURRENT_TIMESTAMP \n" +
				"DO\n" +
				"	INSERT INTO message(message, created_at)\n" +
				"	VALUES('test event', NOW());",
			want: true,
		},
	}
	a := require.New(t)
	for _, test := range tests {
		got := isTiDBUnsupportStmt(test.stmt)
		a.Equalf(test.want, got, "statement: %s\n", test.stmt)
	}
}

func TestExtractDelimiter(t *testing.T) {
	tests := []struct {
		stmt    string
		want    string
		wantErr bool
	}{
		{
			stmt:    "DELIMITER ;;",
			want:    ";;",
			wantErr: false,
		},
		{
			stmt:    "DELIMITER //",
			want:    "//",
			wantErr: false,
		},
		{
			stmt:    "DELIMITER $$",
			want:    "$$",
			wantErr: false,
		},
		{
			stmt:    "DELIMITER    @@   ",
			want:    "@@",
			wantErr: false,
		},
		{
			stmt:    "DELIMITER    @@//",
			want:    "@@//",
			wantErr: false,
		},
		{
			stmt:    "DELIMITER    @@//",
			want:    "@@//",
			wantErr: false,
		},
		// DELIMITER cannot contain a backslash character
		{
			stmt:    "DELIMITER    \\",
			wantErr: true,
		},
	}
	a := require.New(t)
	for _, test := range tests {
		got, err := ExtractDelimiter(test.stmt)
		if test.wantErr {
			a.Error(err)
		} else {
			a.NoError(err)
			a.Equal(test.want, got)
		}
	}
}

func TestMySQLExtractDatabaseList(t *testing.T) {
	tests := []struct {
		stmt string
		want []string
	}{
		{
			stmt: `
				SELECT * FROM t;
				SELECT * FROM db1.t;
				SELECT * FROM db2.t;
				SELECT * FROM db1.t;
			`,
			want: []string{"", "db1", "db2"},
		},
		{
			stmt: `
				SELECT 1;
			`,
			want: nil,
		},
		{
			stmt: `SELECT * FROM t, db1.t1;`,
			want: []string{"", "db1"},
		},
		{
			stmt: `select * from t join db1.t as t1 where t.a != t1.a;`,
			want: []string{"", "db1"},
		},
	}

	for _, test := range tests {
		res, err := ExtractDatabaseList(MySQL, test.stmt, "")
		require.NoError(t, err)
		require.Equal(t, test.want, res)
	}
}

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
		res, err := GetSQLFingerprint(MySQL, test.stmt)
		require.NoError(t, err, test.stmt)
		require.Equal(t, test.want, res, test.stmt)
	}
}

func TestExtractPostgresResourceList(t *testing.T) {
	tests := []struct {
		statement string
		want      []SchemaResource
	}{
		{
			statement: `SELECT * FROM t;SELECT * FROM t1;`,
			want: []SchemaResource{
				{
					Database: "db",
					Schema:   "public",
					Table:    "t",
				},
				{
					Database: "db",
					Schema:   "public",
					Table:    "t1",
				},
			},
		},
		{
			statement: "SELECT * FROM schema1.t1 JOIN schema2.t2 ON t1.c1 = t2.c1;",
			want: []SchemaResource{
				{
					Database: "db",
					Schema:   "schema1",
					Table:    "t1",
				},
				{
					Database: "db",
					Schema:   "schema2",
					Table:    "t2",
				},
			},
		},
		{
			statement: "SELECT a > (select max(a) from t1) FROM t2;",
			want: []SchemaResource{
				{
					Database: "db",
					Schema:   "public",
					Table:    "t1",
				},
				{
					Database: "db",
					Schema:   "public",
					Table:    "t2",
				},
			},
		},
	}

	for _, test := range tests {
		res, err := ExtractResourceList(Postgres, "db", "public", test.statement)
		require.NoError(t, err)
		require.Equal(t, test.want, res)
	}
}
