package pg

import (
	"testing"
)

func TestPostgreSQLIndexComparer_ExtractWhereClauseFromIndexDef(t *testing.T) {
	comparer := &PostgreSQLIndexComparer{}

	tests := []struct {
		name          string
		indexDef      string
		expectedWhere string
	}{
		{
			name:          "Index with simple WHERE clause",
			indexDef:      "CREATE INDEX idx_users_active ON users (email) WHERE active = true",
			expectedWhere: "active = true",
		},
		{
			name:          "Index with complex WHERE clause",
			indexDef:      "CREATE INDEX idx_orders_status ON orders (customer_id) WHERE status IN ('pending', 'processing')",
			expectedWhere: "status IN ('pending', 'processing')",
		},
		{
			name:          "Index with function in WHERE clause",
			indexDef:      `CREATE INDEX idx_text_search ON articles (title) WHERE "length"(content) > 100`,
			expectedWhere: `"length"(content) > 100`,
		},
		{
			name:          "Index without WHERE clause",
			indexDef:      "CREATE INDEX idx_users_email ON users (email)",
			expectedWhere: "",
		},
		{
			name:          "UNIQUE index with WHERE clause",
			indexDef:      "CREATE UNIQUE INDEX idx_users_email_active ON users (email) WHERE active = true",
			expectedWhere: "active = true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := comparer.ExtractWhereClauseFromIndexDef(tt.indexDef)
			if result != tt.expectedWhere {
				t.Errorf("ExtractWhereClauseFromIndexDef() = %q, want %q", result, tt.expectedWhere)
			}
		})
	}
}

func TestPostgreSQLIndexComparer_CompareIndexWhereConditions(t *testing.T) {
	comparer := &PostgreSQLIndexComparer{}

	tests := []struct {
		name     string
		def1     string
		def2     string
		expected bool
	}{
		{
			name:     "Same WHERE conditions",
			def1:     "CREATE INDEX idx1 ON table1 (col1) WHERE active = true",
			def2:     "CREATE INDEX idx2 ON table2 (col2) WHERE active = true",
			expected: true,
		},
		{
			name:     "Different WHERE conditions",
			def1:     "CREATE INDEX idx1 ON table1 (col1) WHERE active = true",
			def2:     "CREATE INDEX idx2 ON table2 (col2) WHERE status = 'pending'",
			expected: false,
		},
		{
			name:     "Both without WHERE clause",
			def1:     "CREATE INDEX idx1 ON table1 (col1)",
			def2:     "CREATE INDEX idx2 ON table2 (col2)",
			expected: true,
		},
		{
			name:     "One with WHERE, one without",
			def1:     "CREATE INDEX idx1 ON table1 (col1) WHERE active = true",
			def2:     "CREATE INDEX idx2 ON table2 (col2)",
			expected: false,
		},
		{
			name:     "Function names with different quoting should be equal",
			def1:     `CREATE INDEX idx1 ON table1 (LEFT(text_col, 50)) WHERE active = true`,
			def2:     `CREATE INDEX idx2 ON table2 ("left"(text_col, 50)) WHERE active = true`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := comparer.CompareIndexWhereConditions(tt.def1, tt.def2)
			if result != tt.expected {
				t.Errorf("CompareIndexWhereConditions() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPostgreSQLIndexComparer_ParseIndexDefinition(t *testing.T) {
	comparer := &PostgreSQLIndexComparer{}

	tests := []struct {
		name        string
		indexDef    string
		expectError bool
		expected    *IndexDefinition
	}{
		{
			name:        "Simple index",
			indexDef:    "CREATE INDEX idx_users_email ON users (email)",
			expectError: false,
			expected: &IndexDefinition{
				IndexName:   "idx_users_email",
				TableName:   "users",
				Unique:      false,
				WhereClause: "",
			},
		},
		{
			name:        "UNIQUE index with WHERE clause",
			indexDef:    "CREATE UNIQUE INDEX idx_users_email_active ON users (email) WHERE active = true",
			expectError: false,
			expected: &IndexDefinition{
				IndexName:   "idx_users_email_active",
				TableName:   "users",
				Unique:      true,
				WhereClause: "active = true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := comparer.ParseIndexDefinition(tt.indexDef)

			if tt.expectError && err == nil {
				t.Error("ParseIndexDefinition() expected error, but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("ParseIndexDefinition() unexpected error: %v", err)
				return
			}

			if tt.expected != nil {
				if result.IndexName != tt.expected.IndexName {
					t.Errorf("ParseIndexDefinition() IndexName = %q, want %q", result.IndexName, tt.expected.IndexName)
				}
				if result.TableName != tt.expected.TableName {
					t.Errorf("ParseIndexDefinition() TableName = %q, want %q", result.TableName, tt.expected.TableName)
				}
				if result.Unique != tt.expected.Unique {
					t.Errorf("ParseIndexDefinition() Unique = %v, want %v", result.Unique, tt.expected.Unique)
				}
				if result.WhereClause != tt.expected.WhereClause {
					t.Errorf("ParseIndexDefinition() WhereClause = %q, want %q", result.WhereClause, tt.expected.WhereClause)
				}
			}
		})
	}
}
