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
			stmts: "CREATE TABLE t1(id INT, name VARCHAR(50), price DECIMAL(10,2), CONSTRAINT PRIMARY KEY(50), INDEX idx_name(name);\n" +
				"DELIMITER ;;\n" +
				"CREATE DEFINER=`root`@`%` TRIGGER `ins_sum` BEFORE INSERT ON `account` FOR EACH SET @sum=@sum + NEW.price;;\n" +
				"DELIMITER ;\n",
			wantUnsupport: []string{
				"DELIMITER ;;",
				"CREATE DEFINER=`root`@`%` TRIGGER `ins_sum` BEFORE INSERT ON `account` FOR EACH SET @sum=@sum + NEW.price;;",
				"DELIMITER ;",
			},
			wantSupport: "CREATE TABLE t1(id INT, name VARCHAR(50), price DECIMAL(10,2), CONSTRAINT PRIMARY KEY(50), INDEX idx_name(name);\n",
			wantErr:     false,
		},
	}
	a := require.New(t)
	for _, test := range tests {
		gotUnsupport, gotSupport, err := ExtractTiDBUnsupportStmts(test.stmts)
		if test.wantErr {
			a.Error(err)
		} else {
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
			stmt: "CREATE TABLE t1(id INT, name VARCHAR(50), price DECIMAL(10,2), CONSTRAINT PRIMARY KEY(50), INDEX idx_name(name);",
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
		}, {
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

func TestExtractDatabaseList(t *testing.T) {
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
	}

	for _, test := range tests {
		res, err := ExtractDatabaseList(MySQL, test.stmt)
		require.NoError(t, err)
		require.Equal(t, test.want, res)
	}
}
