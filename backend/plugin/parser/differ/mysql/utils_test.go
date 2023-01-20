package mysql

import (
	"testing"
)

func TestTrigger(t *testing.T) {
	tests := []testCase{
		{
			old: `CREATE TABLE account(acct_num INT, amount DECIMAL(10,2));` +
				"CREATE DEFINER=`root`@`%` TRIGGER `ins_sum` BEFORE INSERT ON account FOR EACH ROW SET @sum = @sum + NEW.amount;",
			new: `CREATE TABLE account(acct_num INT, amount DECIMAL(10,2), price INT);` +
				"CREATE DEFINER=`root`@`%` TRIGGER `ins_sum` BEFORE INSERT ON account FOR EACH ROW SET @sum = sum + NEW.amount * NEW.price;",
			want: "ALTER TABLE `account` ADD COLUMN `price` INT AFTER `amount`;\n\n" +
				"DROP TRIGGER IF EXISTS `ins_sum`;\n\n" +
				"CREATE DEFINER=`root`@`%` TRIGGER `ins_sum` BEFORE INSERT ON account FOR EACH ROW SET @sum = sum + NEW.amount * NEW.price;\n\n",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestFunction(t *testing.T) {
	tests := []testCase{
		{
			old: "DELIMITER ;;\n" +
				"CREATE DEFINER=`root`@`%` FUNCTION `AddOne`(v INT) RETURNS int\n" +
				"BEGIN   DECLARE a INT;   SET a = v;   SET a = a + 1;   RETURN a; END ;;\n" +
				"DELIMITER ;\n",
			new: "DELIMITER ;;\n" +
				"CREATE DEFINER=`root`@`%` FUNCTION `AddOne`(v INT) RETURNS int\n" +
				"BEGIN   DECLARE a INT;   SET a = v;   SET a = a * 1 + 1;   RETURN a; END ;;\n" +
				"DELIMITER ;\n",
			want: "DROP FUNCTION IF EXISTS `AddOne`;\n\n" +
				"CREATE DEFINER=`root`@`%` FUNCTION `AddOne`(v INT) RETURNS int\n" +
				"BEGIN   DECLARE a INT;   SET a = v;   SET a = a * 1 + 1;   RETURN a; END ;;\n\n",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestProcedure(t *testing.T) {
	tests := []testCase{
		{
			old: "DELIMITER ;;\n" +
				"CREATE DEFINER=`admin`@`localhost` PROCEDURE `account_count`()\n" +
				"SQL SECURITY INVOKER\n" +
				"BEGIN\n" +
				"SELECT 'Number of accounts:', COUNT(*) FROM mysql.user;\n" +
				"END ;;\n" +
				"DELIMITER ;\n",
			new: "DELIMITER ;;\n" +
				"CREATE DEFINER=`admin`@`localhost` PROCEDURE `account_count`()\n" +
				"SQL SECURITY INVOKER\n" +
				"BEGIN\n" +
				"SELECT 'Number of accounts:', (COUNT(*)-1) FROM mysql.user;\n" +
				"END ;;\n" +
				"DELIMITER ;\n",
			want: "DROP PROCEDURE IF EXISTS `account_count`;\n\n" +
				"CREATE DEFINER=`admin`@`localhost` PROCEDURE `account_count`()\n" +
				"SQL SECURITY INVOKER\n" +
				"BEGIN\n" +
				"SELECT 'Number of accounts:', (COUNT(*)-1) FROM mysql.user;\n" +
				"END ;;\n\n",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestEvent(t *testing.T) {
	tests := []testCase{
		{
			old: "DELIMITER ;;\n" +
				"CREATE DEFINER=`root`@`%` EVENT `e_daily` ON SCHEDULE EVERY 1 DAY STARTS '2022-10-19 10:10:42' ON COMPLETION NOT PRESERVE ENABLE COMMENT 'Saves total number of sessions then clears the table each day' DO BEGIN\n" +
				"INSERT INTO site_activity.totals (time, total)\n" +
				"FROM site_activity.sessions;\n" +
				"END ;;\n" +
				"DELIMITER ;\n",
			new: "DELIMITER ;;\n" +
				"CREATE DEFINER=`root`@`%` EVENT `e_daily` ON SCHEDULE EVERY 1 DAY STARTS '2022-10-19 10:10:42' ON COMPLETION NOT PRESERVE ENABLE COMMENT 'Saves total number of sessions then clears the table each day' DO BEGIN\n" +
				"INSERT INTO site_activity.totals (time, total)\n" +
				"FROM site_activity.sessions;\n" +
				"DELITE FROM site_activity.sessions;\n" +
				"END ;;\n" +
				"DELIMITER ;\n",
			want: "DROP EVENT IF EXISTS `e_daily`;\n\n" +
				"CREATE DEFINER=`root`@`%` EVENT `e_daily` ON SCHEDULE EVERY 1 DAY STARTS '2022-10-19 10:10:42' ON COMPLETION NOT PRESERVE ENABLE COMMENT 'Saves total number of sessions then clears the table each day' DO BEGIN\n" +
				"INSERT INTO site_activity.totals (time, total)\n" +
				"FROM site_activity.sessions;\n" +
				"DELITE FROM site_activity.sessions;\n" +
				"END ;;\n\n",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}
