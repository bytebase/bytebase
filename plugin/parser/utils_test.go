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
