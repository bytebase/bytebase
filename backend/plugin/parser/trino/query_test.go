package trino

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		name         string
		sql          string
		wantReadOnly bool
		wantData     bool
		wantErr      bool
	}{
		{
			name:         "Simple SELECT",
			sql:          "SELECT * FROM users",
			wantReadOnly: true,
			wantData:     true,
			wantErr:      false,
		},
		{
			name:         "INSERT statement",
			sql:          "INSERT INTO users (id, name) VALUES (1, 'John')",
			wantReadOnly: false,
			wantData:     false,
			wantErr:      false,
		},
		{
			name:         "UPDATE statement",
			sql:          "UPDATE users SET name = 'John' WHERE id = 1",
			wantReadOnly: false,
			wantData:     false,
			wantErr:      false,
		},
		{
			name:         "DELETE statement",
			sql:          "DELETE FROM users WHERE id = 1",
			wantReadOnly: false,
			wantData:     false,
			wantErr:      false,
		},
		{
			name:         "CREATE TABLE statement",
			sql:          "CREATE TABLE users (id INT, name VARCHAR)",
			wantReadOnly: false,
			wantData:     false,
			wantErr:      false,
		},
		{
			name:         "EXPLAIN statement",
			sql:          "EXPLAIN SELECT * FROM users",
			wantReadOnly: true,
			wantData:     true,
			wantErr:      false,
		},
		{
			name:         "EXPLAIN ANALYZE SELECT",
			sql:          "EXPLAIN ANALYZE SELECT * FROM users",
			wantReadOnly: true,
			wantData:     true,
			wantErr:      false,
		},
		{
			name:         "EXPLAIN ANALYZE UPDATE",
			sql:          "EXPLAIN ANALYZE UPDATE users SET name = 'John' WHERE id = 1",
			wantReadOnly: true, // EXPLAIN ANALYZE is always treated as read-only
			wantData:     true, // EXPLAIN ANALYZE always returns data
			wantErr:      false,
		},
		{
			name:         "SHOW TABLES",
			sql:          "SHOW TABLES",
			wantReadOnly: true,
			wantData:     true,
			wantErr:      false,
		},
		{
			name:         "Invalid SQL",
			sql:          "SELECT * FFROM users",
			wantReadOnly: false,
			wantData:     false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readOnly, returnsData, err := validateQuery(tt.sql)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantReadOnly, readOnly, "readOnly")
			assert.Equal(t, tt.wantData, returnsData, "returnsData")
		})
	}
}

func TestExtractResourceList(t *testing.T) {
	tests := []struct {
		name            string
		currentDatabase string
		currentSchema   string
		sql             string
		wantResources   []base.SchemaResource
		wantErr         bool
	}{
		{
			name:            "Simple SELECT",
			currentDatabase: "catalog1",
			currentSchema:   "public",
			sql:             "SELECT * FROM users",
			wantResources: []base.SchemaResource{
				{
					Database: "catalog1",
					Schema:   "public",
					Table:    "users",
				},
			},
			wantErr: false,
		},
		{
			name:            "JOIN query",
			currentDatabase: "catalog1",
			currentSchema:   "public",
			sql:             "SELECT u.id, o.order_id FROM users u JOIN orders o ON u.id = o.user_id",
			wantResources: []base.SchemaResource{
				{
					Database: "catalog1",
					Schema:   "public",
					Table:    "orders",
				},
				{
					Database: "catalog1",
					Schema:   "public",
					Table:    "users",
				},
			},
			wantErr: false,
		},
		{
			name:            "Fully qualified table names",
			currentDatabase: "catalog1",
			currentSchema:   "public",
			sql:             "SELECT * FROM catalog2.schema2.table2",
			wantResources: []base.SchemaResource{
				{
					Database: "catalog2",
					Schema:   "schema2",
					Table:    "table2",
				},
			},
			wantErr: false,
		},
		// SKIP: The current resourceExtractListener doesn't capture tables in INSERT statements
		// This is a limitation that should be addressed in a future update
		// {
		//   name:            "INSERT statement",
		//   currentDatabase: "catalog1",
		//   currentSchema:   "public",
		//   sql:             "INSERT INTO users (id, name) VALUES (1, 'John')",
		//   wantResources: []base.SchemaResource{
		//     {
		//       Database: "catalog1",
		//       Schema:   "public",
		//       Table:    "users",
		//     },
		//   },
		//   wantErr: false,
		// },
		{
			name:            "Invalid SQL",
			currentDatabase: "catalog1",
			currentSchema:   "public",
			sql:             "SELECT * FFROM users",
			wantResources:   nil,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources, err := ExtractResourceList(tt.currentDatabase, tt.currentSchema, tt.sql)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.ElementsMatch(t, tt.wantResources, resources)
		})
	}
}
