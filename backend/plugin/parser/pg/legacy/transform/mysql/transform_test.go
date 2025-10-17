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
		") ENGINE=InnoDB DEFAULT CHARACTER SET=UTF8MB4 DEFAULT COLLATE=UTF8MB4_0900_AI_CI;\n\n" +
		"CREATE UNIQUE INDEX `c1` ON `t1` (`c1`, `c2`);\n\n" +
		"CREATE UNIQUE INDEX `haha` ON `t1` (`c2`);\n\n" +
		"CREATE INDEX `t1` ON `t1` (`c1`);\n\n"

	a := require.New(t)
	mysqlTransformer := &SchemaTransformer{}
	got, err := mysqlTransformer.Transform(input)
	a.NoError(err)
	a.Equal(want, got)
}

func TestTransform(t *testing.T) {
	input := `
		CREATE TABLE t (
			id int NOT NULL,
			name varchar(60),
			uuid varchar(40),
			PRIMARY KEY (id),
			KEY idx_name (name)
		);
		
		CREATE PROCEDURE p1()
		BEGIN
			SELECT * FROM t;
		END;`

	want := "CREATE TABLE `t` (" + "\n" +
		"  `id` INT NOT NULL," + "\n" +
		"  `name` VARCHAR(60)," + "\n" +
		"  `uuid` VARCHAR(40)," + "\n" +
		"  PRIMARY KEY (`id`)" + "\n" +
		");" + "\n" +
		"\n" +
		"CREATE INDEX `idx_name` ON `t` (`name`);" + "\n" +
		"\n\n\t\t\n\t\t" +
		"CREATE PROCEDURE p1()" + "\n" +
		"		BEGIN" + "\n" +
		"			SELECT * FROM t;" + "\n" +
		"		END;" + "\n" +
		"\n"

	a := require.New(t)
	mysqlTransformer := &SchemaTransformer{}
	got, err := mysqlTransformer.Transform(input)
	a.NoError(err)
	a.Equal(want, got)
}
