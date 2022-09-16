package mysql

import (
	"testing"

	"github.com/pingcap/tidb/parser"
	"github.com/stretchr/testify/require"
)

func TestColumn(t *testing.T) {
	tests := []struct {
		old  string
		new  string
		want string
	}{
		// Missing columns
		{
			old:  `CREATE TABLE book(id INT, PRIMARY KEY(id));`,
			new:  `CREATE TABLE book(id INT, price INT, PRIMARY KEY(id));`,
			want: "ALTER TABLE `book` ADD COLUMN (`price` INT);\n",
		},
		{
			old:  `CREATE TABLE book(id INT, PRIMARY KEY(id))`,
			new:  `CREATE TABLE book(id INT, price INT, code VARCHAR(50), PRIMARY KEY(id));`,
			want: "ALTER TABLE `book` ADD COLUMN (`price` INT), ADD COLUMN (`code` VARCHAR(50));\n",
		},
		{
			old:  ``,
			new:  `CREATE TABLE book(id INT, price INT, code VARCHAR(50), PRIMARY KEY(id));`,
			want: "CREATE TABLE IF NOT EXISTS `book` (`id` INT,`price` INT,`code` VARCHAR(50),PRIMARY KEY(`id`));\n",
		},
		{
			old:  `CREATE TABLE book(id INT, price INT, code VARCHAR(50), PRIMARY KEY(id));`,
			new:  `CREATE TABLE book(id INT, price INT, code VARCHAR(50), PRIMARY KEY(id));`,
			want: "",
		},

		// Different types.
		{
			old:  `CREATE TABLE book(id INT);`,
			new:  `CREATE TABLE book(id VARCHAR(50));`,
			want: "ALTER TABLE `book` MODIFY COLUMN `id` VARCHAR(50);\n",
		},

		{
			old:  `CREATE TABLE book(id INT, isbn VARCHAR(50));`,
			new:  `CREATE TABLE book(id VARCHAR(50), isbn VARCHAR(100));`,
			want: "ALTER TABLE `book` MODIFY COLUMN `id` VARCHAR(50), MODIFY COLUMN `isbn` VARCHAR(100);\n",
		},
		{
			old:  `CREATE TABLE book(id INT, isbn VARCHAR(50));`,
			new:  `CREATE TABLE book(id INT, isbn VARCHAR(50));`,
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
