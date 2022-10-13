package mysql

import (
	"testing"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/parser/opcode"
	"github.com/pingcap/tidb/types"
	driver "github.com/pingcap/tidb/types/parser_driver"
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

func TestGetIndexName(t *testing.T) {
	tests := []struct {
		constraint *ast.Constraint
		want       string
	}{
		{
			constraint: &ast.Constraint{
				Name: "id_idx",
				Keys: []*ast.IndexPartSpecification{
					{
						Column: &ast.ColumnName{
							Name: model.NewCIStr("id"),
						},
					},
					{
						Column: &ast.ColumnName{
							Name: model.NewCIStr("name"),
						},
					},
				},
			},
			want: "id_idx",
		},
		{
			constraint: &ast.Constraint{
				Keys: []*ast.IndexPartSpecification{
					{
						Column: &ast.ColumnName{
							Name: model.NewCIStr("id"),
						},
					},
					{
						Column: &ast.ColumnName{
							Name: model.NewCIStr("name"),
						},
					},
				},
			},
			want: "id",
		},
	}
	a := require.New(t)
	for _, test := range tests {
		got := getIndexName(test.constraint)
		a.Equalf(test.want, got, "constraint: %v", test.constraint)
	}
}

func TestIndexType(t *testing.T) {
	tests := []struct {
		old  string
		new  string
		want string
	}{
		{
			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE(name));`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx USING HASH(name));`,
			want: "ALTER TABLE `book` DROP INDEX `book_idx`;\n" +
				"ALTER TABLE `book` ADD INDEX `book_idx`(`name`) USING HASH;\n",
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

	a := require.New(t)
	mysqlDiffer := &SchemaDiffer{}
	for _, test := range tests {
		out, err := mysqlDiffer.SchemaDiff(test.old, test.new)
		a.NoError(err)
		a.Equalf(test.want, out, "old: %s\nnew: %s\n", test.old, test.new)
	}
}

func TestIndexOption(t *testing.T) {
	tests := []struct {
		old  string
		new  string
		want string
	}{
		// KEY_BLOCK_SIZE not match.
		{
			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name) KEY_BLOCK_SIZE=30);`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name) KEY_BLOCK_SIZE=50);`,
			want: "ALTER TABLE `book` DROP INDEX `book_idx`;\n" +
				"ALTER TABLE `book` ADD INDEX `book_idx`(`name`) KEY_BLOCK_SIZE=50;\n",
		},
		{
			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY (name) KEY_BLOCK_SIZE=30);`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY (name) KEY_BLOCK_SIZE=50);`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY(`name`) KEY_BLOCK_SIZE=50;\n",
		},
		// WITH PARSER not match.
		{
			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, FULLTEXT INDEX book_idx(name) WITH PARSER parser_a);`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, FULLTEXT INDEX book_idx(name) WITH PARSER parser_b);`,
			want: "ALTER TABLE `book` DROP INDEX `book_idx`;\n" +
				"ALTER TABLE `book` ADD FULLTEXT `book_idx`(`name`) WITH PARSER `parser_b`;\n",
		},
		{
			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY (name) WITH PARSER parser_a);`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL,CONSTRAINT PRIMARY KEY (name) WITH PARSER parser_b);`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY(`name`) WITH PARSER `parser_b`;\n",
		},
		// COMMENT not match.
		{
			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name) COMMENT 'comment_b');`,
			want: "ALTER TABLE `book` DROP INDEX `book_idx`;\n" +
				"ALTER TABLE `book` ADD INDEX `book_idx`(`name`) COMMENT 'comment_b';\n",
		},
		{
			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(name) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(name) COMMENT 'comment_b');`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY(`name`) COMMENT 'comment_b';\n",
		},
		// VISIBILITY not match.
		{

			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name) VISIBLE);`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, INDEX book_idx(name) INVISIBLE);`,
			want: "ALTER TABLE `book` DROP INDEX `book_idx`;\n" +
				"ALTER TABLE `book` ADD INDEX `book_idx`(`name`) INVISIBLE;\n",
		},
		{

			old: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(name) VISIBLE);`,
			new: `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(name) INVISIBLE);`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY(`name`) INVISIBLE;\n",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL, FULLTEXT INDEX book_idx(name) KEY_BLOCK_SIZE=30 WITH PARSER parser_a COMMENT 'no difference!');`,
			new:  `CREATE TABLE book(name VARCHAR(50) NOT NULL, FULLTEXT INDEX book_idx(name) KEY_BLOCK_SIZE=30 WITH PARSER parser_a COMMENT 'no difference!');`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIAMRY KEY(name) KEY_BLOCK_SIZE=30 WITH PARSER parser_a COMMENT 'no difference!');`,
			new:  `CREATE TABLE book(name VARCHAR(50) NOT NULL, CONSTRAINT PRIAMRY KEY(name) KEY_BLOCK_SIZE=30 WITH PARSER parser_a COMMENT 'no difference!');`,
			want: "",
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

func TestKeyPart(t *testing.T) {
	tests := []struct {
		old  string
		new  string
		want string
	}{
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE (id, name) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE (id) COMMENT 'comment_a');`,
			want: "ALTER TABLE `book` DROP INDEX `book_idx`;\n" +
				"ALTER TABLE `book` ADD INDEX `book_idx`(`id`) USING BTREE COMMENT 'comment_a';\n",
		},
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(id, name) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(id) COMMENT 'comment_a');`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY(`id`) COMMENT 'comment_a';\n",
		},
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE (id, name) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE ((id + 1)) COMMENT 'comment_a');`,
			want: "ALTER TABLE `book` DROP INDEX `book_idx`;\n" +
				"ALTER TABLE `book` ADD INDEX `book_idx`((`id`+1)) USING BTREE COMMENT 'comment_a';\n",
		},
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY (id, name) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY ((id + 1)) COMMENT 'comment_a');`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY((`id`+1)) COMMENT 'comment_a';\n",
		},
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE ((id + 1)) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE ((id + 2)) COMMENT 'comment_a');`,
			want: "ALTER TABLE `book` DROP INDEX `book_idx`;\n" +
				"ALTER TABLE `book` ADD INDEX `book_idx`((`id`+2)) USING BTREE COMMENT 'comment_a';\n",
		},
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY ((id + 1)) COMMENT 'comment_a');`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY ((id + 2)) COMMENT 'comment_a');`,
			want: "ALTER TABLE `book` DROP PRIMARY KEY;\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY((`id`+2)) COMMENT 'comment_a';\n",
		},
		{
			old:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE (id, name) COMMENT 'comment_a');`,
			new:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE (id, name) COMMENT 'comment_a');`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIAMRY KEY (id, name) COMMENT 'comment_a');`,
			new:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIAMRY KEY (id, name) COMMENT 'comment_a');`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE ((id + 1)) COMMENT 'comment_a');`,
			new:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, INDEX book_idx USING BTREE ((id + 1)) COMMENT 'comment_a');`,
			want: "",
		},
		{
			old:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIAMRY KEY ((id + 1)) COMMENT 'comment_a');`,
			new:  `CREATE TABLE book(id INT, name VARCHAR(50) NOT NULL, CONSTRAINT PRIAMRY KEY ((id + 1)) COMMENT 'comment_a');`,
			want: "",
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

func TestIndex(t *testing.T) {
	tests := []struct {
		old  string
		new  string
		want string
	}{
		// ADD COLUMN -> DROP PRIMARY KEY -> ADD PRIMARY KEY
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50), CONSTRAINT PRIMARY KEY(id, name));`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50), address VARCHAR(50) NOT NULL, CONSTRAINT PRIMARY KEY(id, address));`,
			want: "ALTER TABLE `book` ADD COLUMN (`address` VARCHAR(50) NOT NULL);\n" +
				"ALTER TABLE `book` DROP PRIMARY KEY;\n" +
				"ALTER TABLE `book` ADD PRIMARY KEY(`id`, `address`);\n",
		},
		// ADD COLUMN -> DROP INDEX -> ADD INDEX
		{
			old: `CREATE TABLE book(id INT, name VARCHAR(50), INDEX id_name_idx (id, name));`,
			new: `CREATE TABLE book(id INT, name VARCHAR(50), address VARCHAR(50) NOT NULL, INDEX id_address_idx (id, address));`,
			want: "ALTER TABLE `book` ADD COLUMN (`address` VARCHAR(50) NOT NULL);\n" +
				"ALTER TABLE `book` DROP INDEX `id_name_idx`;\n" +
				"ALTER TABLE `book` ADD INDEX `id_address_idx`(`id`, `address`);\n",
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
