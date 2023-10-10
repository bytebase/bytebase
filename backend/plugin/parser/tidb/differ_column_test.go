package tidb

import (
	"testing"
)

func TestColumnExist(t *testing.T) {
	tests := []testCase{
		// Missing columns
		{
			old:  `CREATE TABLE book(id INT, PRIMARY KEY(id));`,
			new:  `CREATE TABLE book(id INT, price INT, PRIMARY KEY(id));`,
			want: "ALTER TABLE `book` ADD COLUMN `price` INT AFTER `id`;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT, PRIMARY KEY(id))`,
			new:  `CREATE TABLE book(id INT, price INT, code VARCHAR(50), PRIMARY KEY(id));`,
			want: "ALTER TABLE `book` ADD COLUMN `price` INT AFTER `id`, ADD COLUMN `code` VARCHAR(50) AFTER `price`;\n\n",
		},
		{
			old: ``,
			new: `CREATE TABLE book(id INT, price INT, code VARCHAR(50), PRIMARY KEY(id));`,
			want: "" +
				"CREATE TABLE IF NOT EXISTS `book` (\n" +
				"  `id` INT,\n" +
				"  `price` INT,\n" +
				"  `code` VARCHAR(50),\n" +
				"  PRIMARY KEY (`id`)\n" +
				");\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT, price INT, code VARCHAR(50), PRIMARY KEY(id));`,
			new:  `CREATE TABLE book(id INT, price INT, code VARCHAR(50), PRIMARY KEY(id));`,
			want: "",
		},
		// excess columns
		{
			old: `CREATE TABLE book(id INT, price INT, code VARCHAR(50), PRIMARY KEY(id));`,
			new: `CREATE TABLE book(price INT, code VARCHAR(50));`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n\n" +
				"ALTER TABLE `book` DROP COLUMN `id`;\n\n",
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
			want: "ALTER TABLE `book` MODIFY COLUMN `id` VARCHAR(50);\n\n",
		},

		{
			old:  `CREATE TABLE book(id INT, isbn VARCHAR(50));`,
			new:  `CREATE TABLE book(id VARCHAR(50), isbn VARCHAR(100));`,
			want: "ALTER TABLE `book` MODIFY COLUMN `id` VARCHAR(50), MODIFY COLUMN `isbn` VARCHAR(100);\n\n",
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
		{
			old:  `CREATE TABLE book(id INT COLUMN_FORMAT FIXED, PRIMARY KEY(id));`,
			new:  `CREATE TABLE book(id INT COLUMN_FORMAT DYNAMIC, PRIMARY KEY(id));`,
			want: "ALTER TABLE `book` MODIFY COLUMN `id` INT COLUMN_FORMAT DYNAMIC;\n\n",
		},
		// NULL option not match.
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) NULL;\n\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50));`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50);\n\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL DEFAULT 'Harry Potter');`,
			new:  `CREATE TABLE book(name VARCHAR(50) NULL DEFAULT 'Harry Potter');`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) NULL DEFAULT 'Harry Potter';\n\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL DEFAULT 'Harry Potter');`,
			new:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Harry Potter');`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) DEFAULT 'Harry Potter';\n\n",
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
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) COMMENT 'Book Name' NOT NULL;\n\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) COMMENT 'Author Name' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) COMMENT 'AUTHOR NAME' NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) COMMENT 'AUTHOR NAME' NOT NULL;\n\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) COMMENT 'Book Name' NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) COMMENT 'Book Name' NOT NULL;\n\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) COMMENT 'Book Name' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) NOT NULL;\n\n",
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
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) NOT NULL;\n\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Harry Potter' NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) DEFAULT 'Harry Potter' NOT NULL;\n\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Holmes' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Harry Potter' NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) DEFAULT 'Harry Potter' NOT NULL;\n\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Holmes' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Holmes' NOT NULL);`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(id INT DEFAULT 0 NOT NULL);`,
			new:  `CREATE TABLE book(id INT NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `id` INT NOT NULL;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT NOT NULL);`,
			new:  `CREATE TABLE book(id INT DEFAULT 0 NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `id` INT DEFAULT 0 NOT NULL;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT DEFAULT 0 NOT NULL);`,
			new:  `CREATE TABLE book(id INT DEFAULT 1 NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `id` INT DEFAULT 1 NOT NULL;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT DEFAULT 0 NOT NULL);`,
			new:  `CREATE TABLE book(id INT DEFAULT 0 NOT NULL);`,
			want: "",
		},
		// Function Call
		{
			old: "CREATE TABLE action(action_id smallint(5) unsigned NOT NULL AUTO_INCREMENT," +
				"`last_update` timestamp not null default current_timestamp);",
			new: "CREATE TABLE action(action_id smallint(5) unsigned NOT NULL AUTO_INCREMENT," +
				"`last_update` timestamp not null default current_timestamp(1));",
			want: "ALTER TABLE `action` MODIFY COLUMN `last_update` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(1);\n\n",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestColumnCollate(t *testing.T) {
	tests := []testCase{
		{
			old:  `CREATE TABLE book(name VARCHAR(50) COLLATE utf8mb4_bin DEFAULT 'Harry Potter' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) COLLATE utf8mb4_polish_ci DEFAULT 'Harry Potter' NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) COLLATE utf8mb4_polish_ci DEFAULT 'Harry Potter' NOT NULL;\n\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Harry Potter' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) COLLATE utf8mb4_polish_ci DEFAULT 'Harry Potter' NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `name` VARCHAR(50) COLLATE utf8mb4_polish_ci DEFAULT 'Harry Potter' NOT NULL;\n\n",
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
		{
			old:  `CREATE TABLE book(name VARCHAR(50) COLLATE utf8mb4_bin DEFAULT 'Holmes' NOT NULL);`,
			new:  `CREATE TABLE book(name VARCHAR(50) DEFAULT 'Holmes' NOT NULL);`,
			want: "",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestColumnOrder(t *testing.T) {
	tests := []testCase{
		// Append the column to the end of the table.
		{
			old:  `CREATE TABLE book(id INT PRIMARY KEY);`,
			new:  `CREATE TABLE book(id INT PRIMARY KEY, name VARCHAR(50) NOT NULL, author VARCHAR(50) NOT NULL);`,
			want: "ALTER TABLE `book` ADD COLUMN `name` VARCHAR(50) NOT NULL AFTER `id`, ADD COLUMN `author` VARCHAR(50) NOT NULL AFTER `name`;\n\n",
		},
		// Add the column at the beginning of the table.
		{
			old:  `CREATE TABLE book(author VARCHAR(50) NOT NULL);`,
			new:  `CREATE TABLE book(id INT PRIMARY KEY, name VARCHAR(50) NOT NULL, author VARCHAR(50) NOT NULL);`,
			want: "ALTER TABLE `book` ADD COLUMN `id` INT PRIMARY KEY FIRST, ADD COLUMN `name` VARCHAR(50) NOT NULL AFTER `id`;\n\n",
		},
		// Add the column in the middle of the table.
		{
			old:  `CREATE TABLE book(id INT PRIMARY KEY, author VARCHAR(50) NOT NULL);`,
			new:  `CREATE TABLE book(id INT PRIMARY KEY, name VARCHAR(50) NOT NULL, author VARCHAR(50) NOT NULL);`,
			want: "ALTER TABLE `book` ADD COLUMN `name` VARCHAR(50) NOT NULL AFTER `id`;\n\n",
		},
		// Modify the existing column order.
		{
			old:  `CREATE TABLE book(author VARCHAR(50) NOT NULL, id INT PRIMARY KEY, name VARCHAR(50) NOT NULL);`,
			new:  `CREATE TABLE book(id INT PRIMARY KEY, name VARCHAR(50) NOT NULL, author VARCHAR(50) NOT NULL);`,
			want: "ALTER TABLE `book` MODIFY COLUMN `id` INT PRIMARY KEY FIRST, MODIFY COLUMN `name` VARCHAR(50) NOT NULL AFTER `id`;\n\n",
		},
		// Complicated case.
		{
			old: `CREATE TABLE book(c1 INT, c2 INT, c3 INT, c4 INT, c5 INT);`,
			new: `CREATE TABLE book(c6 INT, c2 INT, c3 INT, c7 INT, c8 INT, c4 INT, c5 INT);`,
			want: "ALTER TABLE `book` ADD COLUMN `c6` INT FIRST, " + // c6, c1, c2, c3, c4, c5
				"ADD COLUMN `c7` INT AFTER `c3`, " + // c6, c1, c2, c3, c7, c4, c5
				"ADD COLUMN `c8` INT AFTER `c7`;\n\n" + // c6, c1, c2, c3, c7, c8, c4, c5
				"ALTER TABLE `book` DROP COLUMN `c1`;\n\n", // c6, c2, c3, c7, c8, c4, c5

		},
		{
			old: `CREATE TABLE book(c1 INT, c2 INT, c3 INT, c4 INT, c8 INT);`,
			new: `CREATE TABLE book(c9 INT, c8 VARCHAR(10), c4 INT, c2 VARCHAR(10), c9 INT);`,
			want: "ALTER TABLE `book` ADD COLUMN `c9` INT FIRST, " +
				"MODIFY COLUMN `c8` VARCHAR(10) AFTER `c9`, " +
				"MODIFY COLUMN `c4` INT AFTER `c8`, " +
				"MODIFY COLUMN `c2` VARCHAR(10), " +
				"ADD COLUMN `c9` INT AFTER `c2`;\n\n" +
				"ALTER TABLE `book` DROP COLUMN `c1`;\n\n" +
				"ALTER TABLE `book` DROP COLUMN `c3`;\n\n",
		},
		{
			old: `CREATE TABLE t(a int);`,
			new: `CREATE TABLE t(b int);`,
			want: "ALTER TABLE `t` ADD COLUMN `b` INT FIRST;\n\n" +
				"ALTER TABLE `t` DROP COLUMN `a`;\n\n",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}
