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

func TestTableOption(t *testing.T) {
	tests := []struct {
		old  string
		new  string
		want string
	}{
		// AUTO_INCREMENT
		{
			old:  `CREATE TABLE book(id INT PRIMARY KEY AUTO_INCREMENT) AUTO_INCREMENT = 4;`,
			new:  `CREATE TABLE book(id INT PRIMARY KEY AUTO_INCREMENT) AUTO_INCREMENT = 10;`,
			want: "ALTER TABLE `book` AUTO_INCREMENT = 10;\n",
		},
		{
			old:  `CREATE TABLE book(id INT PRIMARY KEY AUTO_INCREMENT) AUTO_INCREMENT = 4;`,
			new:  `CREATE TABLE book(id INT PRIMARY KEY AUTO_INCREMENT);`,
			want: "ALTER TABLE `book` AUTO_INCREMENT = 0;\n",
		},
		// AVG_ROW_LENGTH
		{
			old:  `CREATE TABLE book(id INT) AVG_ROW_LENGTH = 1;`,
			new:  `CREATE TABLE book(id INT) AVG_ROW_LENGTH = 2;`,
			want: "ALTER TABLE `book` AVG_ROW_LENGTH = 2;\n",
		},
		{
			old:  `CREATE TABLE book(id INT) AVG_ROW_LENGTH = 1;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` AVG_ROW_LENGTH = 0;\n",
		},
		// DEFAULT CHARSET
		{
			old:  `CREATE TABLE book(id INT) DEFAULT CHARACTER SET = utf8;`,
			new:  `CREATE TABLE book(id INT) DEFAULT CHARACTER SET = utf8mb4;`,
			want: "ALTER TABLE `book` DEFAULT CHARACTER SET = UTF8MB4;\n",
		},
		{
			old:  `CREATE TABLE book(id INT) DEFAULT CHARACTER SET = utf8;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` DEFAULT CHARACTER SET = UTF8MB4;\n",
		},
		// DEFAULT COLLATE
		{
			old:  `CREATE TABLE book(id INT) DEFAULT COLLATE = latin1_swedish_ci;`,
			new:  `CREATE TABLE book(id INT) DEFAULT COLLATE = utf8mb4_general_ci;`,
			want: "ALTER TABLE `book` DEFAULT COLLATE = UTF8MB4_GENERAL_CI;\n",
		},
		{
			old:  `CREATE TABLE book(id INT) DEFAULT COLLATE = latin1_swedish_ci;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` DEFAULT COLLATE = UTF8MB4_GENERAL_CI;\n",
		},
		// CHECKSUM
		{
			old:  `CREATE TABLE book(id INT) CHECKSUM = 1;`,
			new:  `CREATE TABLE book(id INT) CHECKSUM = 0;`,
			want: "ALTER TABLE `book` CHECKSUM = 0;\n",
		},
		{
			old:  `CREATE TABLE book(id INT) CHECKSUM = 1;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` CHECKSUM = 0;\n",
		},
		// COMMENT
		{
			old:  `CREATE TABLE book(id INT) COMMENT = 'old';`,
			new:  `CREATE TABLE book(id INT) COMMENT = 'new';`,
			want: "ALTER TABLE `book` COMMENT = 'new';\n",
		},
		{
			old:  `CREATE TABLE book(id INT) COMMENT = 'old';`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` COMMENT = '';\n",
		},
		// TODO(zp): handle drop COMPRESSION
		{
			old:  `CREATE TABLE book(id INT) COMPRESSION = 'ZLIB';`,
			new:  `CREATE TABLE book(id INT) COMPRESSION = 'LZ4';`,
			want: "ALTER TABLE `book` COMPRESSION = 'LZ4';\n",
		},
		// CONNECTION
		{
			old:  `CREATE TABLE book(id INT) CONNECTION = 'old';`,
			new:  `CREATE TABLE book(id INT) CONNECTION = 'new';`,
			want: "ALTER TABLE `book` CONNECTION = 'new';\n",
		},
		{
			old:  `CREATE TABLE book(id INT) CONNECTION = 'old';`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` CONNECTION = '';\n",
		},
		// TODO(zp): handle drop DATA DIRECTORY
		{
			old:  `CREATE TABLE book(id INT) DATA DIRECTORY = 'old';`,
			new:  `CREATE TABLE book(id INT) DATA DIRECTORY = 'new';`,
			want: "ALTER TABLE `book` DATA DIRECTORY = 'new';\n",
		},
		// TODO(zp): handle drop INDEX DIRECTORY
		{
			old:  `CREATE TABLE book(id INT) INDEX DIRECTORY = 'old';`,
			new:  `CREATE TABLE book(id INT) INDEX DIRECTORY = 'new';`,
			want: "ALTER TABLE `book` INDEX DIRECTORY = 'new';\n",
		},
		// DELAY_KEY_WRITE
		{
			old:  `CREATE TABLE book(id INT) DELAY_KEY_WRITE = 1;`,
			new:  `CREATE TABLE book(id INT) DELAY_KEY_WRITE = 0;`,
			want: "ALTER TABLE `book` DELAY_KEY_WRITE = 0;\n",
		},
		{
			old:  `CREATE TABLE book(id INT) DELAY_KEY_WRITE = 1;`,
			new:  `CREATE TABLE book(id INT)`,
			want: "ALTER TABLE `book` DELAY_KEY_WRITE = 0;\n",
		},
		// ENCRYPTION
		{
			old:  `CREATE TABLE book(id INT) ENCRYPTION = 'Y';`,
			new:  `CREATE TABLE book(id INT) ENCRYPTION = 'N';`,
			want: "ALTER TABLE `book` ENCRYPTION = 'N';\n",
		},
		{
			old:  `CREATE TABLE book(id INT) ENCRYPTION = 'Y';`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` ENCRYPTION = 'N';\n",
		},
		// INSERT_METHOD
		// TODO(zp): enable this test if the upstream repo fix it.
		// https://github.com/pingcap/tidb/pull/38355
		// {
		// 	old:  `CREATE TABLE book(id INT) INSERT_METHOD = LAST;`,
		// 	new:  `CREATE TABLE book(id INT) INSERT_METHOD = FIRST;`,
		// 	want: "ALTER TABLE `book` INSERT_METHOD = FIRST;\n",
		// },
		// {
		// 	old:  `CREATE TABLE book(id INT) INSERT_METHOD = LAST;`,
		// 	new:  `CREATE TABLE book(id INT);`,
		// 	want: "ALTER TABLE `book` INSERT_METHOD = NO;\n",
		// },
		// TODO(zp): KEY_BLOCK_SIZE

		// MAX_ROWS
		{
			old:  `CREATE TABLE book(id INT) MAX_ROWS = 100;`,
			new:  `CREATE TABLE book(id INT) MAX_ROWS = 200;`,
			want: "ALTER TABLE `book` MAX_ROWS = 200;\n",
		},
		{
			old:  `CREATE TABLE book(id INT) MAX_ROWS = 100;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` MAX_ROWS = 0;\n",
		},
		// MIN_ROWS
		{
			old:  `CREATE TABLE book(id INT) MIN_ROWS = 100;`,
			new:  `CREATE TABLE book(id INT) MIN_ROWS = 200;`,
			want: "ALTER TABLE `book` MIN_ROWS = 200;\n",
		},
		{
			old:  `CREATE TABLE book(id INT) MIN_ROWS = 100;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` MIN_ROWS = 0;\n",
		},
		// PACK_KEYS
		// TODO(zp): alter table pack_keys
		{
			old:  `CREATE TABLE book(id INT) PACK_KEYS = 1;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` PACK_KEYS = DEFAULT /* TableOptionPackKeys is not supported */ ;\n",
		},
		// PACK_KEYS
		// TODO(zp): alter table pack_keys
		{
			old:  `CREATE TABLE book(id INT) PACK_KEYS = 1;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` PACK_KEYS = DEFAULT /* TableOptionPackKeys is not supported */ ;\n",
		},
		// PASSWORD
		{
			old:  `CREATE TABLE book(id INT) PASSWORD = 'old';`,
			new:  `CREATE TABLE book(id INT) PASSWORD = 'new';`,
			want: "ALTER TABLE `book` PASSWORD = 'new';\n",
		},
		{
			old:  `CREATE TABLE book(id INT) PASSWORD = 'old';`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` PASSWORD = '';\n",
		},
		// ROW_FORMAT
		{
			old:  `CREATE TABLE book(id INT) ROW_FORMAT = DYNAMIC;`,
			new:  `CREATE TABLE book(id INT) ROW_FORMAT = COMPRESSED;`,
			want: "ALTER TABLE `book` ROW_FORMAT = COMPRESSED;\n",
		},
		{
			old:  `CREATE TABLE book(id INT) ROW_FORMAT = DYNAMIC;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` ROW_FORMAT = DEFAULT;\n",
		},
		// STATS_AUTO_RECALC
		{
			old:  `CREATE TABLE book(id INT) STATS_AUTO_RECALC = 1;`,
			new:  `CREATE TABLE book(id INT) STATS_AUTO_RECALC = 0;`,
			want: "ALTER TABLE `book` STATS_AUTO_RECALC = 0;\n",
		},
		{
			old:  `CREATE TABLE book(id INT) STATS_AUTO_RECALC = 1;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` STATS_AUTO_RECALC = DEFAULT;\n",
		},
		// TODO(zp): STATS_PERSISTENT

		// UNION
		{
			old:  `CREATE TABLE book(id INT) UNION = (book2);`,
			new:  `CREATE TABLE book(id INT) UNION = (book2, book3);`,
			want: "ALTER TABLE `book` UNION = (`book2`,`book3`);\n",
		},
	}
	t.Parallel()
	a := require.New(t)
	mysqlDiffer := &SchemaDiffer{}
	for _, test := range tests {
		out, err := mysqlDiffer.SchemaDiff(test.old, test.new)
		a.NoError(err)
		a.Equalf(test.want, out, "old: %s\nnew: %s\n", test.old, test.new)
	}
}
