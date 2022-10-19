package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrigger(t *testing.T) {
	tests := []struct {
		old  string
		new  string
		want string
	}{
		{
			old: `CREATE TABLE account(acct_num INT, amount DECIMAL(10,2));` +
				"CREATE DEFINER=`root`@`%` TRIGGER `ins_sum` BEFORE INSERT ON account FOR EACH ROW SET @sum = @sum + NEW.amount;",
			new: `CREATE TABLE account(acct_num INT, amount DECIMAL(10,2), price INT);` +
				"CREATE DEFINER=`root`@`%` TRIGGER `ins_sum` BEFORE INSERT ON account FOR EACH ROW SET @sum = sum + NEW.amount * NEW.price;",
			want: "ALTER TABLE `account` ADD COLUMN (`price` INT);\n" +
				"DROP TRIGGER IF EXISTS `ins_sum`;\n" +
				"CREATE DEFINER=`root`@`%` TRIGGER `ins_sum` BEFORE INSERT ON account FOR EACH ROW SET @sum = sum + NEW.amount * NEW.price;\n",
		},
	}
	a := require.New(t)
	mysqlDiffer := &SchemaDiffer{}
	for _, test := range tests {
		out, err := mysqlDiffer.SchemaDiff(test.old, test.new)
		a.NoError(err)
		a.Equalf(test.want, out, "old: %s\nnew: %s\n", test.old, test.new)
	}
}
