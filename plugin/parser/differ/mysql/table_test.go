package mysql

import (
	"testing"

	_ "github.com/pingcap/tidb/types/parser_driver"
	"github.com/stretchr/testify/require"
)

func TestTable(t *testing.T) {
	tests := []struct {
		old  string
		new  string
		want string
	}{
		{
			old: ``,
			new: `CREATE TABLE book(id INT, price INT, PRIMARY KEY(id));
			CREATE TABLE author(id INT, name VARCHAR(255), PRIMARY KEY(id));
			`,
			want: "CREATE TABLE IF NOT EXISTS `book` (`id` INT,`price` INT,PRIMARY KEY(`id`));\nCREATE TABLE IF NOT EXISTS `author` (`id` INT,`name` VARCHAR(255),PRIMARY KEY(`id`));\n",
		},
		{
			old: `CREATE TABLE author(id INT, name VARCHAR(255), PRIMARY KEY(id))`,
			new: `CREATE TABLE book(id INT, price INT, PRIMARY KEY(id));
			CREATE TABLE author(id INT, name VARCHAR(255), PRIMARY KEY(id));
			`,
			want: "CREATE TABLE IF NOT EXISTS `book` (`id` INT,`price` INT,PRIMARY KEY(`id`));\n",
		},
		{
			old: `CREATE TABLE book(id INT, price INT, PRIMARY KEY(id));
			CREATE TABLE author(id INT, name VARCHAR(255), PRIMARY KEY(id))`,
			new: `CREATE TABLE book(id INT, price INT, PRIMARY KEY(id));
			CREATE TABLE author(id INT, name VARCHAR(255), PRIMARY KEY(id));
			`,
			want: "",
		},
	}
	a := require.New(t)
	mysqlDiffer := &SchemaDiffer{}
	for _, test := range tests {
		out, err := mysqlDiffer.SchemaDiff(test.old, test.new)
		a.NoError(err)
		a.Equal(test.want, out)
	}
}
