package mysql

import (
	"testing"

	"github.com/pingcap/tidb/parser"
	_ "github.com/pingcap/tidb/parser/test_driver"
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
	for _, test := range tests {
		oldNodes, _, err := parser.New().Parse(test.old, "", "")
		a.NoError(err)
		newNodes, _, err := parser.New().Parse(test.new, "", "")
		a.NoError(err)
		out, err := SchemaDiff(oldNodes, newNodes)
		a.NoError(err)
		a.Equal(test.want, out)
	}
}
