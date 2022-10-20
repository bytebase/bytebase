package mysql

import (
	"testing"

	_ "github.com/pingcap/tidb/types/parser_driver"
)

func TestColumnExist(t *testing.T) {
	tests := []testCase{
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
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestColumnType(t *testing.T) {
	tests := []testCase{
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
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestColumnOption(t *testing.T) {
	tests := []testCase{
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
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestColumnComment(t *testing.T) {
	tests := []testCase{
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
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestColumnDefaultValue(t *testing.T) {
	tests := []testCase{
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
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestColumnCollate(t *testing.T) {
	tests := []testCase{
		{
			old:  `CREATE TABLE book(name VARCHAR(50) COLLATE utf8mb4_bin DEFAULT 'Harry Potter' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) COLLATE utf8mb4_polish_ci DEFAULT 'Harry Potter' NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) COLLATE utf8mb4_polish_ci DEFAULT 'Harry Potter' NOT NULL;\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Harry Potter' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) COLLATE utf8mb4_polish_ci DEFAULT 'Harry Potter' NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) COLLATE utf8mb4_polish_ci DEFAULT 'Harry Potter' NOT NULL;\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Holmes' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Holmes' NOT NULL);`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) COLLATE utf8mb4_bin DEFAULT 'Holmes' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) COLLATE utf8mb4_bin DEFAULT 'Holmes' NOT NULL);`,
			want: "",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}
