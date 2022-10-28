package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateTableSeparateIndex(t *testing.T) {
	input := "" +
		"CREATE TABLE `t1` (\n" +
		"	`id` int DEFAULT NULL,\n" +
		"	`c1` int DEFAULT NULL,\n" +
		"	`c2` int DEFAULT NULL,\n" +
		"	UNIQUE KEY `c1` (`c1`,`c2`),\n" +
		"	UNIQUE KEY `haha` (`c2`),\n" +
		"	KEY `t1` (`c1`)\n" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;"
	want := "CREATE TABLE `t1` (\n" +
		"  `id` INT DEFAULT NULL,\n" +
		"  `c1` INT DEFAULT NULL,\n" +
		"  `c2` INT DEFAULT NULL\n" +
		") ENGINE=InnoDB DEFAULT CHARACTER SET=UTF8MB4 DEFAULT COLLATE=UTF8MB4_0900_AI_CI;\n" +
		"CREATE UNIQUE INDEX `c1` ON `t1` (`c1`, `c2`);\n" +
		"CREATE UNIQUE INDEX `haha` ON `t1` (`c2`);\n" +
		"CREATE INDEX `t1` ON `t1` (`c1`);\n"

	a := require.New(t)
	mysqlTransformer := &SchemaTransformer{}
	got, err := mysqlTransformer.Transform(input)
	a.NoError(err)
	a.Equal(want, got)
}
