package mysql

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetTemporaryView(t *testing.T) {
	a := require.New(t)
	got := getTemporaryView("db1", []string{"col1", "col2"})
	want := "--\n-- Temporary view structure for `db1`\n--\nCREATE VIEW `db1` AS SELECT\n  1 AS `col1`,\n  1 AS `col2`;\n\n"
	a.Equal(want, got)
}

func TestExcludeSchemaAutoIncrementValue(t *testing.T) {
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
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci AUTO_INCREMENT=12345;`,
			`CREATE TABLE world (
				id int NOT NULL AUTO_INCREMENT,
				PRIMARY KEY (id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;`,
		},
	}

	for _, test := range tests {
		got := excludeSchemaAutoIncrementValue(test.stmt)
		require.Equal(t, test.want, got)
	}
}

func TestHH(t *testing.T) {
	u := url.UserPassword("special_password", `8F%f&eLxxx`)
	a := require.New(t)
	a.Equal("special_password", u.String())
}
