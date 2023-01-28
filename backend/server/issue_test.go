package server

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestCheckCharacterSetCollationOwner(t *testing.T) {
	tests := []struct {
		dbType       db.Type
		characterSet string
		collation    string
		owner        string
		expectError  bool
	}{
		/* ClickHouse */
		// With character set or collation
		{
			dbType:       db.ClickHouse,
			characterSet: "utf8mb4",
			expectError:  true,
		},
		{
			dbType:      db.ClickHouse,
			collation:   "utf8mb4_0900_ai_ci",
			expectError: true,
		},
		// Normal
		{
			dbType:      db.ClickHouse,
			expectError: false,
		},

		/* Snowflake */
		// With character set or collation
		{
			dbType:       db.Snowflake,
			characterSet: "utf8mb4",
			expectError:  true,
		},
		{
			dbType:      db.Snowflake,
			collation:   "utf8mb4_0900_ai_ci",
			expectError: true,
		},
		// Normal
		{
			dbType:      db.Snowflake,
			expectError: false,
		},

		/* PostgreSQL */
		// Without owner
		{
			dbType:      db.Postgres,
			owner:       "",
			expectError: true,
		},
		// Without character set
		{
			dbType:      db.Postgres,
			owner:       "bytebase",
			collation:   "fr_FR",
			expectError: false,
		},
		// Without collation
		{
			dbType:       db.Postgres,
			owner:        "bytebase",
			characterSet: "UTF8",
			expectError:  false,
		},

		/* MySQL */
		// With character set or collation
		{
			dbType:       db.MySQL,
			characterSet: "utf8mb4",
			expectError:  true,
		},
		{
			dbType:      db.MySQL,
			collation:   "utf8mb4_0900_ai_ci",
			expectError: true,
		},
		// Normal
		{
			dbType:       db.MySQL,
			characterSet: "utf8mb4",
			collation:    "utf8mb4_0900_ai_ci",
			expectError:  false,
		},
	}
	for _, test := range tests {
		err := checkCharacterSetCollationOwner(test.dbType, test.characterSet, test.collation, test.owner)
		if test.expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}
