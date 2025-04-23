package trino

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestDiagnose(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		expectErr bool
	}{
		{
			name:      "Valid SELECT statement",
			sql:       "SELECT * FROM users",
			expectErr: false,
		},
		{
			name:      "Valid CREATE TABLE statement",
			sql:       "CREATE TABLE users (id INT, name VARCHAR)",
			expectErr: false,
		},
		{
			name:      "Missing FROM clause",
			sql:       "SELECT * FFROM users",
			expectErr: true,
		},
		{
			name:      "Invalid SQL statement",
			sql:       "CREATE TABLEE users (id INT)",
			expectErr: true,
		},
		{
			name:      "Syntax error",
			sql:       "SELECT FROM users",
			expectErr: true,
		},
		{
			name:      "Empty statement",
			sql:       "",
			expectErr: true,
		},
	}

	ctx := context.Background()
	diagnoseCtx := base.DiagnoseContext{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics, err := Diagnose(ctx, diagnoseCtx, tt.sql)
			require.NoError(t, err, "Diagnose should not return an error")

			if tt.expectErr {
				assert.NotEmpty(t, diagnostics, "Expected diagnostics for invalid SQL")
			} else {
				assert.Empty(t, diagnostics, "Expected no diagnostics for valid SQL")
			}
		})
	}
}

func TestParseSyntaxError(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		expectErr bool
	}{
		{
			name:      "Valid statement with semicolon",
			sql:       "SELECT * FROM users;",
			expectErr: false,
		},
		{
			name:      "Valid statement without semicolon",
			sql:       "SELECT * FROM users",
			expectErr: false,
		},
		{
			name:      "Invalid statement",
			sql:       "SELECT * FFROM users",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseTrinoStatement(tt.sql)
			if tt.expectErr {
				assert.NotNil(t, err, "Expected syntax error")
			} else {
				assert.Nil(t, err, "Expected no syntax error")
			}
		})
	}
}
