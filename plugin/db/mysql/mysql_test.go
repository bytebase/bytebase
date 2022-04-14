package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRestoreFuncWithGhostTable(t *testing.T) {
	testCases := [][]string{
		{"multiple constraints",

			"CREATE TABLE `table` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  `fk_id` int DEFAULT NULL,\n" +
				"  UNIQUE KEY `fk_id` (`fk_id`),\n" +
				"  CONSTRAINT `chk_table_1` CHECK ((`id` >= 0))\n" +
				"  CONSTRAINT `fk_table_1` FOREIGN KEY (fk_id) REFERENCES another_table (id)",

			"CREATE TABLE `table_ghost` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  `fk_id` int DEFAULT NULL,\n" +
				"  UNIQUE KEY `fk_id` (`fk_id`),\n" +
				"  CONSTRAINT `chk_table_1_20060102150405` CHECK ((`id` >= 0))\n" +
				"  CONSTRAINT `fk_table_1_20060102150405` FOREIGN KEY (fk_id) REFERENCES another_table (id)",
		},

		{"constraint name has timestamp postfix with 14 digits (unix timestamp to the second)",

			"CREATE TABLE `table` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT `chk_table_1_12345678901234` CHECK ((`id` >= 0))",

			"CREATE TABLE `table_ghost` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT `chk_table_1_20060102150405` CHECK ((`id` >= 0))",
		},

		{"constraint name has wrong timestamp postfix digits",

			"CREATE TABLE `table` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT `chk_table_1_1234567890` CHECK ((`id` >= 0))",

			"CREATE TABLE `table_ghost` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT `chk_table_1_1234567890_20060102150405` CHECK ((`id` >= 0))",
		},

		{"constraint name is longer than 64 after appending timestamp postfix",

			"CREATE TABLE `table` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT `chk_table_1_12345678901234567890123456789012345678901234567890` CHECK ((`id` >= 0))",

			"CREATE TABLE `table_ghost` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT `chk_table_1_1234567890123456789012345678901234567_20060102150405` CHECK ((`id` >= 0))",
		},

		{"should fail with database prefix",

			"CREATE TABLE `database`.`table`",
			"",
		},

		{"insert without database prefix",

			"INSERT INTO `table` VALUES (1, 2, 3);",
			"INSERT INTO `table_ghost` VALUES (1, 2, 3);",
		},

		{"insert with database prefix",

			"INSERT INTO `database`.`table` VALUES (1, 2, 3);",
			"INSERT INTO `database`.`table_ghost` VALUES (1, 2, 3);",
		},
	}

	for _, testCase := range testCases {
		caseName := testCase[0]
		originalSQL := testCase[1]
		expectedSQL := testCase[2]
		t.Run(caseName, func(t *testing.T) {
			rewrittenSQL := rewriteToGhostTableForRestore(originalSQL, "20060102150405")
			require.Equal(t, expectedSQL, rewrittenSQL)
		})
	}
}
