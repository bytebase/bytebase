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

func TestNormalize(t *testing.T) {
	input := `
	create table t(a int DEFAULT NULL, b varchar(20) COLLATE utf8mb4_general_ci) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
	create index idx_a on t(a);
	create unique index uk_t_a on t(a);
	create index idx_xxx on t(a);
	create table t2(a int) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
	create table t3(a int);
	`
	standard := `
	create table t4(a int);
	create table t2(a int) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
	create table t(a int, b varchar(20));
	create unique index uk_t_a on t(a);
	create index idx_a on t(a);
	create index idx_yyy on t(a);
	`
	want := "" +
		"CREATE TABLE `t2` (\n" +
		"  `a` INT\n" +
		") ENGINE=InnoDB DEFAULT CHARACTER SET=UTF8MB4 DEFAULT COLLATE=UTF8MB4_GENERAL_CI;\n\n" +
		"CREATE TABLE `t` (\n" +
		"  `a` INT,\n" +
		"  `b` VARCHAR(20)\n" +
		");\n\n" +
		"CREATE INDEX `idx_xxx` ON `t` (`a`);\n\n" +
		"CREATE UNIQUE INDEX `uk_t_a` ON `t` (`a`);\n\n" +
		"CREATE INDEX `idx_a` ON `t` (`a`);\n\n" +
		"CREATE TABLE `t3` (\n" +
		"  `a` INT\n" +
		");\n\n"
	a := require.New(t)
	mysqlTransformer := &SchemaTransformer{}
	got, err := mysqlTransformer.Normalize(input, standard)
	a.NoError(err)
	a.Equal(want, got)
}

func TestCheck(t *testing.T) {
	type testData struct {
		input string
		err   string
	}
	tests := []testData{
		{
			input: `create table t(a int, FOREIGN KEY (a) references t1(a));`,
			err:   "The constraint name is required for SDL format",
		},
		{
			input: `create table t(a int, CHECK (a > 0));`,
			err:   "The constraint name is required for SDL format",
		},
	}
	a := require.New(t)
	mysqlTransformer := &SchemaTransformer{}
	for _, test := range tests {
		_, err := mysqlTransformer.Check(test.input)
		a.Equal(test.err, err.Error())
	}
}
