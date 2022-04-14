package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRestoreFuncWithGhostTable(t *testing.T) {
	type testCase struct {
		name        string
		originalSQL string
		expectedSQL string
		successful  bool
	}
	testCasesSuccess := []testCase{
		{
			name: "multiple constraints",
			originalSQL: "CREATE TABLE `table` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  `fk_id` int DEFAULT NULL,\n" +
				"  UNIQUE KEY `fk_id` (`fk_id`),\n" +
				"  CONSTRAINT `chk_table_1` CHECK ((`id` >= 0))\n" +
				"  CONSTRAINT `fk_table_1` FOREIGN KEY (fk_id) REFERENCES another_table (id)",
			expectedSQL: "CREATE TABLE `table_ghost` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  `fk_id` int DEFAULT NULL,\n" +
				"  UNIQUE KEY `fk_id` (`fk_id`),\n" +
				"  CONSTRAINT `chk_table_1_20060102150405` CHECK ((`id` >= 0))\n" +
				"  CONSTRAINT `fk_table_1_20060102150405` FOREIGN KEY (fk_id) REFERENCES another_table (id)",
			successful: true,
		},
		{
			name: "constraint name has timestamp suffix with 14 digits (unix timestamp to the second)",
			originalSQL: "CREATE TABLE `table` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT `chk_table_1_12345678901234` CHECK ((`id` >= 0))",
			expectedSQL: "CREATE TABLE `table_ghost` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT `chk_table_1_20060102150405` CHECK ((`id` >= 0))",
			successful: true,
		},
		{
			name: "constraint name has wrong timestamp suffix digits",
			originalSQL: "CREATE TABLE `table` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT `chk_table_1_1234567890` CHECK ((`id` >= 0))",
			expectedSQL: "CREATE TABLE `table_ghost` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT `chk_table_1_1234567890_20060102150405` CHECK ((`id` >= 0))",
			successful: true,
		},
		{
			name: "constraint name is longer than 64 after appending timestamp suffix",
			originalSQL: "CREATE TABLE `table` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT `chk_table_1_12345678901234567890123456789012345678901234567890` CHECK ((`id` >= 0))",
			expectedSQL: "CREATE TABLE `table_ghost` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT `chk_table_1_1234567890123456789012345678901234567_20060102150405` CHECK ((`id` >= 0))",
			successful: true,
		},
		{
			name:        "should not match any rule",
			originalSQL: "CREATE TABLE `database`.`table`",
			expectedSQL: "CREATE TABLE `database`.`table`",
			successful:  true,
		},
		{
			name:        "insert without database prefix",
			originalSQL: "INSERT INTO `table` VALUES (1, 2, 3);",
			expectedSQL: "INSERT INTO `table_ghost` VALUES (1, 2, 3);",
			successful:  true,
		},
		{
			name:        "insert with database prefix",
			originalSQL: "INSERT INTO `database`.`table` VALUES (1, 2, 3);",
			expectedSQL: "",
			successful:  false,
		},
	}

	for _, tc := range testCasesSuccess {
		t.Run(tc.name, func(t *testing.T) {
			a := require.New(t)
			rewrittenSQL, err := rewriteToGhostTableForRestore(tc.originalSQL, "20060102150405")
			if tc.successful {
				a.NoError(err)
			} else {
				a.Error(err)
			}
			a.Equal(tc.expectedSQL, rewrittenSQL)
		})
	}

}
