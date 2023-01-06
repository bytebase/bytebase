package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
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

func TestValidateDatabaseLabelList(t *testing.T) {
	tests := []struct {
		name            string
		labelList       []*api.DatabaseLabel
		environmentName string
		wantErr         bool
	}{
		{
			name: "valid label list",
			labelList: []*api.DatabaseLabel{
				{
					Key:   "bb.location",
					Value: "earth",
				},
				{
					Key:   api.EnvironmentLabelKey,
					Value: "Dev",
				},
			},
			environmentName: "Dev",
			wantErr:         false,
		},
		{
			name: "environment label not present",
			labelList: []*api.DatabaseLabel{
				{
					Key:   "bb.location",
					Value: "earth",
				},
			},
			environmentName: "Dev",
			wantErr:         true,
		},
		{
			name: "cannot mutate environment label",
			labelList: []*api.DatabaseLabel{
				{
					Key:   "bb.location",
					Value: "earth",
				},
				{
					Key:   api.EnvironmentLabelKey,
					Value: "Prod",
				},
			},
			environmentName: "Dev",
			wantErr:         true,
		},
	}

	for _, test := range tests {
		err := validateDatabaseLabelList(test.labelList, test.environmentName)
		if test.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}
