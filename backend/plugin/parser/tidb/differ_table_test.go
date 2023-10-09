package tidb

import (
	"testing"
)

func TestTable(t *testing.T) {
	tests := []testCase{
		{
			old: ``,
			new: `CREATE TABLE book(id INT, price INT, PRIMARY KEY(id));
			CREATE TABLE author(id INT, name VARCHAR(255), PRIMARY KEY(id));
			`,
			want: "" +
				"CREATE TABLE IF NOT EXISTS `author` (\n" +
				"  `id` INT,\n" +
				"  `name` VARCHAR(255),\n" +
				"  PRIMARY KEY (`id`)\n" +
				");\n\n" +
				"CREATE TABLE IF NOT EXISTS `book` (\n" +
				"  `id` INT,\n" +
				"  `price` INT,\n" +
				"  PRIMARY KEY (`id`)\n" +
				");\n\n",
		},
		{
			old: `CREATE TABLE author(id INT, name VARCHAR(255), PRIMARY KEY(id))`,
			new: `CREATE TABLE book(id INT, price INT, PRIMARY KEY(id));
			CREATE TABLE author(id INT, name VARCHAR(255), PRIMARY KEY(id));
			`,
			want: "" +
				"CREATE TABLE IF NOT EXISTS `book` (\n" +
				"  `id` INT,\n" +
				"  `price` INT,\n" +
				"  PRIMARY KEY (`id`)\n" +
				");\n\n",
		},
		{
			old: `CREATE TABLE book(id INT, price INT, PRIMARY KEY(id));
			CREATE TABLE author(id INT, name VARCHAR(255), PRIMARY KEY(id))`,
			new: `CREATE TABLE book(id INT, price INT, PRIMARY KEY(id));
			CREATE TABLE author(id INT, name VARCHAR(255), PRIMARY KEY(id));
			`,
			want: "",
		},
		// Drop excess table
		{
			old: `CREATE TABLE book(id INT, price_id INT, PRIMARY KEY(id), CONSTRAINT fk_price_id FOREIGN KEY (price_id) REFERENCES price(id));
			CREATE TABLE price(id INT, PRIMARY KEY(id));`,
			new: `CREATE TABLE book(id INT, PRIMARY KEY(id));`,
			want: "ALTER TABLE `book` DROP FOREIGN KEY `fk_price_id`;\n\n" +
				"DROP TABLE IF EXISTS `price`;\n\n" +
				"ALTER TABLE `book` DROP COLUMN `price_id`;\n\n",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestTableOption(t *testing.T) {
	tests := []testCase{
		// AUTO_INCREMENT
		{
			old:  `CREATE TABLE book(id INT AUTO_INCREMENT, CONSTRAINT PRIMARY KEY(id)) AUTO_INCREMENT = 4;`,
			new:  `CREATE TABLE book(id INT AUTO_INCREMENT, CONSTRAINT PRIMARY KEY(id)) AUTO_INCREMENT = 10;`,
			want: "ALTER TABLE `book` AUTO_INCREMENT=10;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT AUTO_INCREMENT, CONSTRAINT PRIMARY KEY(id)) AUTO_INCREMENT = 4;`,
			new:  `CREATE TABLE book(id INT AUTO_INCREMENT, CONSTRAINT PRIMARY KEY(id));`,
			want: "ALTER TABLE `book` AUTO_INCREMENT=0;\n\n",
		},
		// AVG_ROW_LENGTH
		{
			old:  `CREATE TABLE book(id INT) AVG_ROW_LENGTH = 1;`,
			new:  `CREATE TABLE book(id INT) AVG_ROW_LENGTH = 2;`,
			want: "ALTER TABLE `book` AVG_ROW_LENGTH=2;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT) AVG_ROW_LENGTH = 1;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` AVG_ROW_LENGTH=0;\n\n",
		},
		// DEFAULT CHARSET
		{
			old:  `CREATE TABLE book(id INT) DEFAULT CHARACTER SET = utf8;`,
			new:  `CREATE TABLE book(id INT) DEFAULT CHARACTER SET = utf8mb4;`,
			want: "ALTER TABLE `book` DEFAULT CHARACTER SET=UTF8MB4;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT) DEFAULT CHARACTER SET = utf8;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "",
		},
		// DEFAULT COLLATE
		{
			old:  `CREATE TABLE book(id INT) DEFAULT COLLATE = latin1_swedish_ci;`,
			new:  `CREATE TABLE book(id INT) DEFAULT COLLATE = utf8mb4_general_ci;`,
			want: "ALTER TABLE `book` DEFAULT COLLATE=UTF8MB4_GENERAL_CI;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT) DEFAULT COLLATE = latin1_swedish_ci;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "",
		},
		// CHECKSUM
		{
			old:  `CREATE TABLE book(id INT) CHECKSUM = 1;`,
			new:  `CREATE TABLE book(id INT) CHECKSUM = 0;`,
			want: "ALTER TABLE `book` CHECKSUM=0;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT) CHECKSUM = 1;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` CHECKSUM=0;\n\n",
		},
		// COMMENT
		{
			old:  `CREATE TABLE book(id INT) COMMENT = 'old';`,
			new:  `CREATE TABLE book(id INT) COMMENT = 'new';`,
			want: "ALTER TABLE `book` COMMENT='new';\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT) COMMENT = 'old';`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` COMMENT='';\n\n",
		},
		// TODO(zp): handle drop COMPRESSION
		{
			old:  `CREATE TABLE book(id INT) COMPRESSION = 'ZLIB';`,
			new:  `CREATE TABLE book(id INT) COMPRESSION = 'LZ4';`,
			want: "ALTER TABLE `book` COMPRESSION='LZ4';\n\n",
		},
		// CONNECTION
		{
			old:  `CREATE TABLE book(id INT) CONNECTION = 'old';`,
			new:  `CREATE TABLE book(id INT) CONNECTION = 'new';`,
			want: "ALTER TABLE `book` CONNECTION='new';\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT) CONNECTION = 'old';`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` CONNECTION='';\n\n",
		},
		// TODO(zp): handle drop DATA DIRECTORY
		{
			old:  `CREATE TABLE book(id INT) DATA DIRECTORY = 'old';`,
			new:  `CREATE TABLE book(id INT) DATA DIRECTORY = 'new';`,
			want: "ALTER TABLE `book` DATA DIRECTORY='new';\n\n",
		},
		// TODO(zp): handle drop INDEX DIRECTORY
		{
			old:  `CREATE TABLE book(id INT) INDEX DIRECTORY = 'old';`,
			new:  `CREATE TABLE book(id INT) INDEX DIRECTORY = 'new';`,
			want: "ALTER TABLE `book` INDEX DIRECTORY='new';\n\n",
		},
		// DELAY_KEY_WRITE
		{
			old:  `CREATE TABLE book(id INT) DELAY_KEY_WRITE = 1;`,
			new:  `CREATE TABLE book(id INT) DELAY_KEY_WRITE = 0;`,
			want: "ALTER TABLE `book` DELAY_KEY_WRITE=0;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT) DELAY_KEY_WRITE = 1;`,
			new:  `CREATE TABLE book(id INT)`,
			want: "ALTER TABLE `book` DELAY_KEY_WRITE=0;\n\n",
		},
		// ENCRYPTION
		{
			old:  `CREATE TABLE book(id INT) ENCRYPTION = 'Y';`,
			new:  `CREATE TABLE book(id INT) ENCRYPTION = 'N';`,
			want: "ALTER TABLE `book` ENCRYPTION='N';\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT) ENCRYPTION = 'Y';`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` ENCRYPTION='N';\n\n",
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
			want: "ALTER TABLE `book` MAX_ROWS=200;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT) MAX_ROWS = 100;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` MAX_ROWS=0;\n\n",
		},
		// MIN_ROWS
		{
			old:  `CREATE TABLE book(id INT) MIN_ROWS = 100;`,
			new:  `CREATE TABLE book(id INT) MIN_ROWS = 200;`,
			want: "ALTER TABLE `book` MIN_ROWS=200;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT) MIN_ROWS = 100;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` MIN_ROWS=0;\n\n",
		},
		// PACK_KEYS
		// TODO(zp): alter table pack_keys
		{
			old:  `CREATE TABLE book(id INT) PACK_KEYS = 1;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` PACK_KEYS=DEFAULT /* TableOptionPackKeys is not supported */ ;\n\n",
		},
		// PACK_KEYS
		// TODO(zp): alter table pack_keys
		{
			old:  `CREATE TABLE book(id INT) PACK_KEYS = 1;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` PACK_KEYS=DEFAULT /* TableOptionPackKeys is not supported */ ;\n\n",
		},
		// PASSWORD
		{
			old:  `CREATE TABLE book(id INT) PASSWORD = 'old';`,
			new:  `CREATE TABLE book(id INT) PASSWORD = 'new';`,
			want: "ALTER TABLE `book` PASSWORD='new';\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT) PASSWORD = 'old';`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` PASSWORD='';\n\n",
		},
		// ROW_FORMAT
		{
			old:  `CREATE TABLE book(id INT) ROW_FORMAT = DYNAMIC;`,
			new:  `CREATE TABLE book(id INT) ROW_FORMAT = COMPRESSED;`,
			want: "ALTER TABLE `book` ROW_FORMAT=COMPRESSED;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT) ROW_FORMAT = DYNAMIC;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` ROW_FORMAT=DEFAULT;\n\n",
		},
		// STATS_AUTO_RECALC
		{
			old:  `CREATE TABLE book(id INT) STATS_AUTO_RECALC = 1;`,
			new:  `CREATE TABLE book(id INT) STATS_AUTO_RECALC = 0;`,
			want: "ALTER TABLE `book` STATS_AUTO_RECALC=0;\n\n",
		},
		{
			old:  `CREATE TABLE book(id INT) STATS_AUTO_RECALC = 1;`,
			new:  `CREATE TABLE book(id INT);`,
			want: "ALTER TABLE `book` STATS_AUTO_RECALC=DEFAULT;\n\n",
		},
		// TODO(zp): STATS_PERSISTENT

		// UNION
		{
			old:  `CREATE TABLE book(id INT) UNION = (book2);`,
			new:  `CREATE TABLE book(id INT) UNION = (book2, book3);`,
			want: "ALTER TABLE `book` UNION=(`book2`,`book3`);\n\n",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}

func TestView(t *testing.T) {
	tests := []testCase{
		{
			old:  `CREATE VIEW book AS SELECT * FROM book;`,
			new:  `CREATE VIEW book AS SELECT * FROM book2;`,
			want: "CREATE OR REPLACE ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `book` AS SELECT * FROM `book2`;\n\n",
		},
		{
			old: `CREATE VIEW order_incomes AS
			SELECT
				order_id,
				customer_name,
				SUM(ordered_quantity * product_price) total
			FROM
				order_details
			INNER JOIN orders USING (order_id)
			INNER JOIN customers USING (customer_name)
			GROUP BY order_id;`,

			new: `CREATE VIEW order_incomes AS
			SELECT
				order_id,
				customer_name,
				SUM(ordered_quantity * product_price) + 1 total
			FROM
				order_details
			INNER JOIN orders USING (order_id)
			INNER JOIN customers USING (customer_name)
			GROUP BY order_id;`,

			want: "CREATE OR REPLACE ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `order_incomes` AS " +
				"SELECT `order_id`,`customer_name`,SUM(`ordered_quantity`*`product_price`)+1 AS `total` " +
				"FROM (`order_details` JOIN `orders` USING (`order_id`)) JOIN `customers` USING (`customer_name`) GROUP BY `order_id`;\n\n",
		},
		// mysqldump temporary view
		{
			old: `CREATE VIEW a AS SELECT 1 AS id, 1 AS name;
				CREATE VIEW a AS SELECT id, name FROM book;
			`,

			new: `CREATE VIEW a AS SELECT 1 AS id;
			CREATE VIEW a AS SELECT id FROM book;
			`,

			want: "CREATE OR REPLACE ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `a` AS SELECT `id` FROM `book`;\n\n",
		},
		// SDL for gitops dependency view
		{
			old: `CREATE VIEW a AS SELECT id, name FROM book`,
			new: `CREATE VIEW a AS SELECT id FROM book;
				CREATE VIEW b AS SELECT id FROM a;
			`,
			want: "CREATE OR REPLACE ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `b` AS SELECT 1 AS `id`;\n\n" +
				"CREATE OR REPLACE ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `a` AS SELECT `id` FROM `book`;\n\n" +
				"CREATE OR REPLACE ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `b` AS SELECT `id` FROM `a`;\n\n",
		},
		{
			old: `CREATE VIEW a AS SELECT id, name FROM book`,
			new: `CREATE VIEW a AS SELECT id AS a_id FROM book;
				CREATE VIEW b AS SELECT a_id AS b_id FROM a;
			`,
			want: "CREATE OR REPLACE ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `b` AS SELECT 1 AS `b_id`;\n\n" +
				"CREATE OR REPLACE ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `a` AS SELECT `id` AS `a_id` FROM `book`;\n\n" +
				"CREATE OR REPLACE ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `b` AS SELECT `a_id` AS `b_id` FROM `a`;\n\n",
		},
		{
			old: ``,
			new: `CREATE VIEW a AS WITH cte AS (SELECT id, name FROM book) SELECT id, name FROM cte UNION SELECT c, d FROM e;`,
			want: "CREATE OR REPLACE ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `a` AS SELECT 1 AS `id`,1 AS `name`;\n\n" +
				"CREATE OR REPLACE ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `a` AS WITH `cte` AS (SELECT `id`,`name` FROM `book`) SELECT `id`,`name` FROM `cte` UNION SELECT `c`,`d` FROM `e`;\n\n",
		},
	}
	testDiffWithoutDisableForeignKeyCheck(t, tests)
}
