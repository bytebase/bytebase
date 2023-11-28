package tidb

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
			old: "CREATE DEFINER=`root`@`%` FUNCTION `AddOne`(v INT) RETURNS int\n" +
				"BEGIN   DECLARE a INT;   SET a = v;   SET a = a + 1;   RETURN a; END ;\n",
			new: "CREATE DEFINER=`root`@`%` FUNCTION `AddOne`(v INT) RETURNS int\n" +
				"BEGIN   DECLARE a INT;   SET a = v;   SET a = a * 1 + 1;   RETURN a; END ;\n",
			want: "DROP FUNCTION IF EXISTS `AddOne`;\n\n" +
				"CREATE DEFINER=`root`@`%` FUNCTION `AddOne`(v INT) RETURNS int\n" +
				"BEGIN   DECLARE a INT;   SET a = v;   SET a = a * 1 + 1;   RETURN a; END ;\n\n",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestProcedure(t *testing.T) {
	tests := []testCase{
		{
			old: "CREATE DEFINER=`admin`@`localhost` PROCEDURE `account_count`()\n" +
				"SQL SECURITY INVOKER\n" +
				"BEGIN\n" +
				"SELECT 'Number of accounts:', COUNT(*) FROM mysql.user;\n" +
				"END ;\n",
			new: "CREATE DEFINER=`admin`@`localhost` PROCEDURE `account_count`()\n" +
				"SQL SECURITY INVOKER\n" +
				"BEGIN\n" +
				"SELECT 'Number of accounts:', (COUNT(*)-1) FROM mysql.user;\n" +
				"END ;\n",
			want: "DROP PROCEDURE IF EXISTS `account_count`;\n\n" +
				"CREATE DEFINER=`admin`@`localhost` PROCEDURE `account_count`()\n" +
				"SQL SECURITY INVOKER\n" +
				"BEGIN\n" +
				"SELECT 'Number of accounts:', (COUNT(*)-1) FROM mysql.user;\n" +
				"END ;\n\n",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestEvent(t *testing.T) {
	tests := []testCase{
		{
			old: `CREATE EVENT purge_old_users
			ON SCHEDULE EVERY 1 DAY STARTS '2023-07-01 00:00:00'
			DO 
			DELETE FROM users WHERE age > 100;
			`,
			new: `CREATE EVENT purge_old_users
			ON SCHEDULE EVERY 1 DAY STARTS '2023-07-01 00:00:00'
			DO 
			DELETE FROM users WHERE age > 110;`,
			want: "DROP EVENT IF EXISTS `purge_old_users`;\n\n" +
				`CREATE EVENT purge_old_users
			ON SCHEDULE EVERY 1 DAY STARTS '2023-07-01 00:00:00'
			DO 
			DELETE FROM users WHERE age > 110;

`,
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}
