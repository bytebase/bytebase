package mysql

import (
	"testing"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	_ "github.com/pingcap/tidb/types/parser_driver"
	"github.com/stretchr/testify/require"
)

func TestColumnExist(t *testing.T) {
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
	}
	a := require.New(t)
	for _, test := range tests {
		oldNodes := getStmtNodes(t, test.old)
		newNodes := getStmtNodes(t, test.new)
		out, err := SchemaDiff(oldNodes, newNodes)
		a.NoError(err)
		a.Equalf(test.want, out, "old: %s\nnew: %s\n", test.old, test.new)
	}
}

func TestColumnType(t *testing.T) {
	tests := []struct {
		old  string
		new  string
		want string
	}{
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
		oldNodes := getStmtNodes(t, test.old)
		newNodes := getStmtNodes(t, test.new)
		out, err := SchemaDiff(oldNodes, newNodes)
		a.NoError(err)
		a.Equalf(test.want, out, "old: %s\nnew: %s\n", test.old, test.new)
	}
}

func TestColumnOption(t *testing.T) {
	tests := []struct {
		old  string
		new  string
		want string
	}{
		// NULL option not match.
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) NULL;\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50));`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50);\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL DEFAULT 'Harry Potter');`,
			new:  `CREATE TABLE book(name VARCHAR(50) NULL DEFAULT 'Harry Potter');`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) NULL DEFAULT 'Harry Potter';\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL DEFAULT 'Harry Potter');`,
			new:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Harry Potter');`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) DEFAULT 'Harry Potter';\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50));`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50));`,
			new:  `CREATE TABLE book(name VARCHAR(50) NULL);`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) NULL);`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			want: "",
		},
	}
	a := require.New(t)
	for _, test := range tests {
		oldNodes := getStmtNodes(t, test.old)
		newNodes := getStmtNodes(t, test.new)
		out, err := SchemaDiff(oldNodes, newNodes)
		a.NoError(err)
		a.Equalf(test.want, out, "old: %s\nnew: %s\n", test.old, test.new)
	}
}

func TestColumnComment(t *testing.T) {
	tests := []struct {
		old  string
		new  string
		want string
	}{
		{
			old:  `CREATE TABLE book(name VARCHAR(50) COMMENT 'Author Name' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) COMMENT 'Book Name' NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) COMMENT 'Book Name' NOT NULL;\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) COMMENT 'Author Name' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) COMMENT 'AUTHOR NAME' NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) COMMENT 'AUTHOR NAME' NOT NULL;\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) COMMENT 'Book Name' NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) COMMENT 'Book Name' NOT NULL;\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) COMMENT 'Book Name' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) NOT NULL;\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) COMMENT 'Book Name' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) COMMENT 'Book Name' NOT NULL);`,
			want: "",
		},
	}
	a := require.New(t)
	for _, test := range tests {
		oldNodes := getStmtNodes(t, test.old)
		newNodes := getStmtNodes(t, test.new)
		out, err := SchemaDiff(oldNodes, newNodes)
		a.NoError(err)
		a.Equalf(test.want, out, "old: %s\nnew: %s\n", test.old, test.new)
	}
}

func TestColumnDefaultValue(t *testing.T) {
	tests := []struct {
		old  string
		new  string
		want string
	}{
		{
			old:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Harry Potter' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) NOT NULL;\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Harry Potter' NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) DEFAULT 'Harry Potter' NOT NULL;\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Holmes' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Harry Potter' NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) DEFAULT 'Harry Potter' NOT NULL;\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Holmes' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Holmes' NOT NULL);`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(id INT DEFAULT 0 NOT NULL);`,
			new:  `CREATE TABLE book(id INT NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `id` INT NOT NULL;\n",
		},
		{
			old:  `CREATE TABLE book(id INT NOT NULL);`,
			new:  `CREATE TABLE book(id INT DEFAULT 0 NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `id` INT DEFAULT 0 NOT NULL;\n",
		},
		{
			old:  `CREATE TABLE book(id INT DEFAULT 0 NOT NULL);`,
			new:  `CREATE TABLE book(id INT DEFAULT 1 NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `id` INT DEFAULT 1 NOT NULL;\n",
		},
		{
			old:  `CREATE TABLE book(id INT DEFAULT 0 NOT NULL);`,
			new:  `CREATE TABLE book(id INT DEFAULT 0 NOT NULL);`,
			want: "",
		},
	}
	a := require.New(t)
	for _, test := range tests {
		oldNodes := getStmtNodes(t, test.old)
		newNodes := getStmtNodes(t, test.new)
		out, err := SchemaDiff(oldNodes, newNodes)
		a.NoError(err)
		a.Equalf(test.want, out, "old: %s\nnew: %s\n", test.old, test.new)
	}
}

func getStmtNodes(t *testing.T, sql string) []ast.StmtNode {
	nodes, _, err := parser.New().Parse(sql, "", "")
	require.NoError(t, err)
	return nodes
}
