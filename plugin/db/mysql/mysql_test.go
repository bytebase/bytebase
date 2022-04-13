package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRestoreFuncWithGhostTable(t *testing.T) {
	a := require.New(t)
	var modifiedSQL string
	fnCopyModifiedSQL := func(query string) error {
		modifiedSQL = query
		return nil
	}
	fnExecuteStmt := genRestoreFuncWithGhostTable(fnCopyModifiedSQL)

	testCases := [][]string{
		{
			"CREATE TABLE `table` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT `table_chk_1` CHECK ((`id` >= 0))",

			"CREATE TABLE `table_ghost` (\n" +
				"  `id` int DEFAULT NULL,\n" +
				"  CONSTRAINT CHECK ((`id` >= 0))",
		},
		{
			// We currently DO NOT support dbPrefix in CREATE TABLE statement, so this case should fail
			"CREATE TABLE `database`.`table`",
			"",
		},
		{
			"INSERT INTO `table` VALUES (1, 2, 3);",
			"INSERT INTO `table_ghost` VALUES (1, 2, 3);",
		},
		{
			"INSERT INTO `database`.`table` VALUES (1, 2, 3);",
			"INSERT INTO `database`.`table_ghost` VALUES (1, 2, 3);",
		},
	}

	for _, testCase := range testCases {
		originalSQL := testCase[0]
		expectedSQL := testCase[1]
		a.NoError(fnExecuteStmt(originalSQL))
		a.Equal(expectedSQL, modifiedSQL)
	}
}
