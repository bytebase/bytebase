package tidb

import (
	"testing"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/model"
	"github.com/pingcap/tidb/pkg/parser/opcode"
	"github.com/pingcap/tidb/pkg/types"
	driver "github.com/pingcap/tidb/pkg/types/parser_driver"
	"github.com/stretchr/testify/require"
)

func TestIsKeyPartEqual(t *testing.T) {
	tests := []struct {
		old []*ast.IndexPartSpecification
		new []*ast.IndexPartSpecification
		eq  bool
	}{
		{
			old: []*ast.IndexPartSpecification{
				// `id` + 1
				{
					Expr: &ast.BinaryOperationExpr{
						Op: opcode.Plus,
						L: &ast.ColumnNameExpr{
							Name: &ast.ColumnName{
								Name: model.NewCIStr("id"),
							},
						},
						R: &driver.ValueExpr{
							Datum: types.NewDatum(1),
						},
					},
					Column: &ast.ColumnName{
						Name: model.NewCIStr("id"),
					},
				},
			},
			new: []*ast.IndexPartSpecification{
				// `id` * 2
				{
					Expr: &ast.BinaryOperationExpr{
						Op: opcode.Mul,
						L: &ast.ColumnNameExpr{
							Name: &ast.ColumnName{
								Name: model.NewCIStr("id"),
							},
						},
						R: &driver.ValueExpr{
							Datum: types.NewDatum(2),
						},
					},
					Column: &ast.ColumnName{
						Name: model.NewCIStr("id"),
					},
				},
			},
			eq: false,
		},
		{
			old: []*ast.IndexPartSpecification{
				// `id` + 1
				{
					Expr: &ast.BinaryOperationExpr{
						Op: opcode.Plus,
						L: &ast.ColumnNameExpr{
							Name: &ast.ColumnName{
								Name: model.NewCIStr("id"),
							},
						},
						R: &driver.ValueExpr{
							Datum: types.NewDatum(1),
						},
					},
					Column: &ast.ColumnName{
						Name: model.NewCIStr("id"),
					},
				},
			},
			new: []*ast.IndexPartSpecification{
				// `id` + 1
				{
					Expr: &ast.BinaryOperationExpr{
						Op: opcode.Plus,
						L: &ast.ColumnNameExpr{
							Name: &ast.ColumnName{
								Name: model.NewCIStr("id"),
							},
						},
						R: &driver.ValueExpr{
							Datum: types.NewDatum(1),
						},
					},
					Column: &ast.ColumnName{
						Name: model.NewCIStr("id"),
					},
				},
				// `id` * 2
				{
					Expr: &ast.BinaryOperationExpr{
						Op: opcode.Mul,
						L: &ast.ColumnNameExpr{
							Name: &ast.ColumnName{
								Name: model.NewCIStr("id"),
							},
						},
						R: &driver.ValueExpr{
							Datum: types.NewDatum(2),
						},
					},
					Column: &ast.ColumnName{
						Name: model.NewCIStr("id"),
					},
				},
			},
			eq: false,
		},
		{
			old: []*ast.IndexPartSpecification{
				// `id` + 1
				{
					Expr: &ast.BinaryOperationExpr{
						Op: opcode.Plus,
						L: &ast.ColumnNameExpr{
							Name: &ast.ColumnName{
								Name: model.NewCIStr("id"),
							},
						},
						R: &driver.ValueExpr{
							Datum: types.NewDatum(1),
						},
					},
					Column: &ast.ColumnName{
						Name: model.NewCIStr("id"),
					},
				},
			},
			new: []*ast.IndexPartSpecification{
				// `id` + 1
				{
					Expr: &ast.BinaryOperationExpr{
						Op: opcode.Plus,
						L: &ast.ColumnNameExpr{
							Name: &ast.ColumnName{
								Name: model.NewCIStr("id"),
							},
						},
						R: &driver.ValueExpr{
							Datum: types.NewDatum(1),
						},
					},
					Column: &ast.ColumnName{
						Name: model.NewCIStr("id"),
					},
				},
			},
			eq: true,
		},
	}
	a := require.New(t)
	for _, test := range tests {
		got := isKeyPartEqual(test.old, test.new)
		a.Equalf(test.eq, got, "old: %v, new: %v", test.old, test.new)
	}
}

func TestIsIndexOptionEqual(t *testing.T) {
	tests := []struct {
		old *ast.IndexOption
		new *ast.IndexOption
		eq  bool
	}{
		{
			old: nil,
			new: nil,
			eq:  true,
		},
		{
			old: &ast.IndexOption{
				KeyBlockSize: 1024,
			},
			new: nil,
			eq:  false,
		},
		{
			old: &ast.IndexOption{
				KeyBlockSize: 1024,
			},
			new: nil,
			eq:  false,
		},
		{
			old: &ast.IndexOption{
				KeyBlockSize: 1024,
				Tp:           model.IndexTypeBtree,
				ParserName:   model.NewCIStr("parser"),
				Comment:      "comment",
				Visibility:   ast.IndexVisibilityVisible,
			},
			new: &ast.IndexOption{
				KeyBlockSize: 1024,
				Tp:           model.IndexTypeHash,
				ParserName:   model.NewCIStr("parser"),
				Comment:      "commen_idx",
				Visibility:   ast.IndexVisibilityInvisible,
			},
			eq: false,
		},
		{
			old: &ast.IndexOption{
				KeyBlockSize: 1024,
				Tp:           model.IndexTypeBtree,
				ParserName:   model.NewCIStr("parser"),
				Comment:      "comment",
				Visibility:   ast.IndexVisibilityVisible,
			},
			new: &ast.IndexOption{
				KeyBlockSize: 1024,
				Tp:           model.IndexTypeBtree,
				ParserName:   model.NewCIStr("parser"),
				Comment:      "comment",
				Visibility:   ast.IndexVisibilityVisible,
			},
			eq: true,
		},
	}

	a := require.New(t)
	for _, test := range tests {
		got := isIndexOptionEqual(test.old, test.new)
		a.Equalf(test.eq, got, "old: %v, new: %v", test.old, test.new)
	}
}

func TestIndexType(t *testing.T) {
	tests := []testCase{
		{
			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE(name));`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx USING HASH(name));`,
			want: "DROP INDEX `book_idx` ON `book`;\n\n" +
				"CREATE INDEX `book_idx` ON `book` (`name`) USING HASH;\n\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE(name));`,
			new:  `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE(name));`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name));`,
			new:  `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name));`,
			want: "",
		},
	}

	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestIndexOption(t *testing.T) {
	tests := []testCase{
		// KEY_BLOCK_SIZE not match.
		{
			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name) KEY_BLOCK_SIZE=30);`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name) KEY_BLOCK_SIZE=50);`,
			want: "DROP INDEX `book_idx` ON `book`;\n\n" +
				"CREATE INDEX `book_idx` ON `book` (`name`) KEY_BLOCK_SIZE=50;\n\n",
		},
		{
			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY (name) KEY_BLOCK_SIZE=30);`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY (name) KEY_BLOCK_SIZE=50);`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY (`name`) KEY_BLOCK_SIZE=50;\n\n",
		},
		// WITH PARSER not match.
		// {
		// 	old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, FULLTEXT INDEX book_idx(name) WITH PARSER parser_a);`,
		// 	new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, FULLTEXT INDEX book_idx(name) WITH PARSER parser_b);`,
		// 	want: "DROP INDEX `book_idx` ON `book`;\n\n" +
		// 		"CREATE FULLTEXT INDEX `book_idx` ON `book` (`name`) WITH PARSER `parser_b`;\n\n",
		// },
		// {
		// 	old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY (name) WITH PARSER parser_a);`,
		// 	new: `CREATE TABLE book(name VARCHAR(50) NOT NULL,CONSTRAINT PRIMARY KEY (name) WITH PARSER parser_b);`,
		// 	want: "ALTER TABLE `book` DROP PRIMARY KEY;\n\n" +
		// 		"ALTER TABLE `book` ADD PRIMARY KEY (`name`) WITH PARSER `parser_b`;\n\n",
		// },
		// COMMENT not match.
		{
			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name) COMMENT 'comment_b');`,
			want: "DROP INDEX `book_idx` ON `book`;\n\n" +
				"CREATE INDEX `book_idx` ON `book` (`name`) COMMENT 'comment_b';\n\n",
		},
		{
			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(name) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(name) COMMENT 'comment_b');`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY (`name`) COMMENT 'comment_b';\n\n",
		},
		// VISIBILITY not match.
		{

			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name) VISIBLE);`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name) INVISIBLE);`,
			want: "DROP INDEX `book_idx` ON `book`;\n\n" +
				"CREATE INDEX `book_idx` ON `book` (`name`) INVISIBLE;\n\n",
		},
		{

			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(name) VISIBLE);`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(name) INVISIBLE);`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY (`name`) INVISIBLE;\n\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL, FULLTEXT INDEX book_idx(name) KEY_BLOCK_SIZE=30 COMMENT 'no difference!');`,
			new:  `CREATE TABLE book(name VARCHAR(50) NOT NULL, FULLTEXT INDEX book_idx(name) KEY_BLOCK_SIZE=30 COMMENT 'no difference!');`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(name) KEY_BLOCK_SIZE=30 COMMENT 'no difference!');`,
			new:  `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(name) KEY_BLOCK_SIZE=30 COMMENT 'no difference!');`,
			want: "",
		},
	}

	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestKeyPart(t *testing.T) {
	tests := []testCase{
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE (id, name) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE (id) COMMENT 'comment_a');`,
			want: "DROP INDEX `book_idx` ON `book`;\n\n" +
				"CREATE INDEX `book_idx` ON `book` (`id`) USING BTREE COMMENT 'comment_a';\n\n",
		},
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(id, name) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(id) COMMENT 'comment_a');`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY (`id`) COMMENT 'comment_a';\n\n",
		},
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE (id, name) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE ((id + 1)) COMMENT 'comment_a');`,
			want: "DROP INDEX `book_idx` ON `book`;\n\n" +
				"CREATE INDEX `book_idx` ON `book` ((`id`+1)) USING BTREE COMMENT 'comment_a';\n\n",
		},
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY (id, name) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY ((id + 1)) COMMENT 'comment_a');`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY ((`id`+1)) COMMENT 'comment_a';\n\n",
		},
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE ((id + 1)) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE ((id + 2)) COMMENT 'comment_a');`,
			want: "DROP INDEX `book_idx` ON `book`;\n\n" +
				"CREATE INDEX `book_idx` ON `book` ((`id`+2)) USING BTREE COMMENT 'comment_a';\n\n",
		},
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY ((id + 1)) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY ((id + 2)) COMMENT 'comment_a');`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY ((`id`+2)) COMMENT 'comment_a';\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE (id, name) COMMENT 'comment_a');`,
			new:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE (id, name) COMMENT 'comment_a');`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY (id, name) COMMENT 'comment_a');`,
			new:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY (id, name) COMMENT 'comment_a');`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE ((id + 1)) COMMENT 'comment_a');`,
			new:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE ((id + 1)) COMMENT 'comment_a');`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY ((id + 1)) COMMENT 'comment_a');`,
			new:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY ((id + 1)) COMMENT 'comment_a');`,
			want: "",
		},
	}

	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestForeignKeyDefination(t *testing.T) {
	tests := []testCase{
		{
			old: `CREATE TABLE department(id INT, name VARCHAR(50) NOT NULL, PRIMARY KEY(department));
			CREATE TABLE employee(id INT, name VARCHAR(50) NOT NULL, department_id INT, PRIMARY KEY(id), FOREIGN KEY employee_ibfk_1(department_id) REFERENCES department(id));`,
			new: `CREATE TABLE department(id INT, name VARCHAR(50) NOT NULL, PRIMARY KEY(department));
			CREATE TABLE employee(id INT, name VARCHAR(50) NOT NULL, department_id INT, PRIMARY KEY(id), FOREIGN KEY fk_2(department_id) REFERENCES department(id));`,
			want: "ALTER TABLE `employee` DROP FOREIGN KEY `employee_ibfk_1`;\n\nALTER TABLE `employee` ADD CONSTRAINT `fk_2` FOREIGN KEY (`department_id`) REFERENCES `department`(`id`);\n\n",
		},
		{
			old: "CREATE TABLE `department` (" +
				"	`id` int NOT NULL," +
				"	`name` varchar(50) NOT NULL," +
				"	PRIMARY KEY (`id`)," +
				"	KEY `id_name_idx` (`id`,`name`)" +
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;" +

				"CREATE TABLE `employee` (" +
				"	`id` int NOT NULL," +
				"	`name` varchar(50) NOT NULL," +
				"	`department_id` int DEFAULT NULL," +
				"	`department_name` varchar(50) DEFAULT NULL," +
				"	PRIMARY KEY (`id`)," +
				"	KEY `department_id_name_idx` (`department_id`,`department_name`)," +
				"	CONSTRAINT `employee_ibfk_1` FOREIGN KEY (`department_id`, `department_name`) REFERENCES `department` (`id`, `name`)" +
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;",

			new: "CREATE TABLE `department` (" +
				"	`id` int NOT NULL," +
				"	`name` varchar(50) NOT NULL," +
				"	PRIMARY KEY (`id`)," +
				"	KEY `id_idx` (`id`)" +
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;" +

				"CREATE TABLE `employee` (" +
				"	`id` int NOT NULL," +
				"	`name` varchar(50) NOT NULL," +
				"	`department_id` int DEFAULT NULL," +
				"	`department_name` varchar(50) DEFAULT NULL," +
				"	PRIMARY KEY (`id`)," +
				"	KEY `department_id_idx` (`department_id`)," +
				"	CONSTRAINT `employee_ibfk_1` FOREIGN KEY (`department_id`) REFERENCES `department` (`id`)" +
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;",

			want: "ALTER TABLE `employee` DROP FOREIGN KEY `employee_ibfk_1`;\n\n" +
				"DROP INDEX `id_name_idx` ON `department`;\n\n" +
				"DROP INDEX `department_id_name_idx` ON `employee`;\n\n" +
				"CREATE INDEX `id_idx` ON `department` (`id`);\n\n" +
				"CREATE INDEX `department_id_idx` ON `employee` (`department_id`);\n\n" +
				"ALTER TABLE `employee` ADD CONSTRAINT `employee_ibfk_1` FOREIGN KEY (`department_id`) REFERENCES `department`(`id`);\n\n",
		},
		// Reference itself.
		{
			old: "CREATE TABLE `employeee` (" +
				"	`id` int NOT NULL," +
				"	`name` varchar(50) NOT NULL," +
				"   `leader_id` int DEFAULT NULL," +
				"	PRIMARY KEY (`id`)," +
				"	CONSTRAINT `employee_ibfk_1` FOREIGN KEY (`leader_id`) REFERENCES `employeee` (`id`)" +
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;",

			new: "CREATE TABLE `employeee` (" +
				"	`id` int NOT NULL," +
				"	`name` varchar(50) NOT NULL," +
				"   `leader_id` int DEFAULT NULL," +
				"   `manager_id` int DEFAULT NULL," +
				"	PRIMARY KEY (`id`)," +
				"	CONSTRAINT `employee_ibfk_1` FOREIGN KEY (`manager_id`) REFERENCES `employeee` (`id`)" +
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;",

			want: "ALTER TABLE `employeee` DROP FOREIGN KEY `employee_ibfk_1`;\n\n" +
				"ALTER TABLE `employeee` ADD COLUMN `manager_id` INT DEFAULT NULL AFTER `leader_id`;\n\n" +
				"ALTER TABLE `employeee` ADD CONSTRAINT `employee_ibfk_1` FOREIGN KEY (`manager_id`) REFERENCES `employeee`(`id`);\n\n",
		},
	}

	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestCheckConstraint(t *testing.T) {
	tests := []testCase{
		{
			old:  `CREATE TABLE book(a INT, CONSTRAINT t1_chk_1 CHECK (a>0))`,
			new:  `CREATE TABLE book(a INT, CONSTRAINT t1_chk_1 CHECK ((a > 0)))`,
			want: "",
		},
		{
			old: "CREATE TABLE book(id INT, price INT, PRIMARY KEY(id), CONSTRAINT `check_price` CHECK (price > 0));",
			new: "CREATE TABLE book(id INT, price INT, PRIMARY KEY(id), CONSTRAINT `check_price` CHECK (price > 1));",
			want: "ALTER TABLE `book` DROP CHECK `check_price`;\n\n" +
				"ALTER TABLE `book` ADD CONSTRAINT `check_price` CHECK(`price`>1) ENFORCED;\n\n",
		},
		{
			old: "CREATE TABLE book(id INT, price INT, PRIMARY KEY(id), CONSTRAINT `check_price` CHECK (price > 0));",
			new: "CREATE TABLE book(id INT, price INT, PRIMARY KEY(id), CONSTRAINT `check_price` CHECK (price > 0) NOT ENFORCED);",
			want: "ALTER TABLE `book` DROP CHECK `check_price`;\n\n" +
				"ALTER TABLE `book` ADD CONSTRAINT `check_price` CHECK(`price`>0) NOT ENFORCED;\n\n",
		},
		{
			old: "CREATE TABLE book(id INT, price INT, PRIMARY KEY(id), CONSTRAINT `check_price` CHECK (price > 0), CONSTRAINT `check_price2` CHECK(price > 1));",
			new: "CREATE TABLE book(id INT, price INT, PRIMARY KEY(id), CONSTRAINT `check_price` CHECK (price > 0) NOT ENFORCED);",
			want: "ALTER TABLE `book` DROP CHECK `check_price`;\n\n" +
				"ALTER TABLE `book` DROP CHECK `check_price2`;\n\n" +
				"ALTER TABLE `book` ADD CONSTRAINT `check_price` CHECK(`price`>0) NOT ENFORCED;\n\n",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestConstraint(t *testing.T) {
	tests := []testCase{
		// ADD COLUMN -> DROP PRIMARY KEY -> ADD PRIMARY KEY
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50), CONSTRAINT PRIMARY KEY(id, name));`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50), address VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(id, address));`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n\n" +
				"ALTER TABLE `book` ADD COLUMN `address` VARCHAR(50) NOT NULL AFTER `name`;\n\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY (`id`, `address`);\n\n",
		},
		// ADD COLUMN -> ADD INDEX WITH ANOTHER NAME-> DROP INDEX
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50), INDEX id_name_idx (id, name));`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50), address VARCHAR(50) NOT NULL, INDEX id_address_idx (id, address));`,
			want: "DROP INDEX `id_name_idx` ON `book`;\n\n" +
				"ALTER TABLE `book` ADD COLUMN `address` VARCHAR(50) NOT NULL AFTER `name`;\n\n" +
				"CREATE INDEX `id_address_idx` ON `book` (`id`, `address`);\n\n",
		},
		// ADD COLUMN -> ADD INDEX WITH SAME NAME -> DROP INDEX
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50), INDEX idx (id, name));`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50), address VARCHAR(50) NOT NULL, INDEX idx (id, address));`,
			want: "DROP INDEX `idx` ON `book`;\n\n" +
				"ALTER TABLE `book` ADD COLUMN `address` VARCHAR(50) NOT NULL AFTER `name`;\n\n" +
				"CREATE INDEX `idx` ON `book` (`id`, `address`);\n\n",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}
