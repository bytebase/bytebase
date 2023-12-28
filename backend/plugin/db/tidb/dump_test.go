package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetTemporaryView(t *testing.T) {
	a := require.New(t)
	got := getTemporaryView("db1", []string{"col1", "col2"})
	want := "--\n-- Temporary view structure for `db1`\n--\nCREATE VIEW `db1` AS SELECT\n  1 AS `col1`,\n  1 AS `col2`;\n\n"
	a.Equal(want, got)
}

func TestExcludeSchemaAutoValue(t *testing.T) {
	tests := []struct {
		stmt string
		want string
	}{
		{
			`CREATE TABLE world (
				id int NOT NULL AUTO_INCREMENT,
				PRIMARY KEY (id)
			) ENGINE=InnoDB AUTO_INCREMENT=12345 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;`,
			`CREATE TABLE world (
				id int NOT NULL AUTO_INCREMENT,
				PRIMARY KEY (id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;`,
		},
		{
			`CREATE TABLE world (
				id int NOT NULL AUTO_INCREMENT,
				PRIMARY KEY (id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci /*T![auto_rand_base] AUTO_RANDOM_BASE=39456621 */;`,
			`CREATE TABLE world (
				id int NOT NULL AUTO_INCREMENT,
				PRIMARY KEY (id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci /*T![auto_rand_base] */;`,
		},
		{
			`CREATE TABLE world (
				id int NOT NULL AUTO_INCREMENT,
				PRIMARY KEY (id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci AUTO_INCREMENT=12345;`,
			`CREATE TABLE world (
				id int NOT NULL AUTO_INCREMENT,
				PRIMARY KEY (id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;`,
		},
	}

	for _, test := range tests {
		got := excludeSchemaAutoValues(test.stmt)
		require.Equal(t, test.want, got)
	}
}
