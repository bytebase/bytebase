package tidb

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
			stmts:       "CREATE TABLE `actor` (\n  `actor_id` smallint unsigned NOT NULL AUTO_INCREMENT,\n  `first_name` varchar(45) NOT NULL,\n  `last_name` varchar(45) NOT NULL,\n  `last_update` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,\n  PRIMARY KEY (`actor_id`),\n  KEY `idx_actor_last_name` (`last_name`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;",
			wantSupport: "CREATE TABLE `actor` (\n  `actor_id` smallint unsigned NOT NULL AUTO_INCREMENT,\n  `first_name` varchar(45) NOT NULL,\n  `last_name` varchar(45) NOT NULL,\n  `last_update` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,\n  PRIMARY KEY (`actor_id`),\n  KEY `idx_actor_last_name` (`last_name`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;\n",
			wantErr:     false,
		},
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
				END;`,
			},
			wantSupport: "CREATE TABLE t1(id INT);\n",
			wantErr:     false,
		},
	}
	a := require.New(t)
	for _, test := range tests {
		gotUnsupport, gotSupport, err := ExtractTiDBUnsupportedStmts(test.stmts)
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
