package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniSequenceSDLDiffAndMigration(t *testing.T) {
	tests := []struct {
		name        string
		fromSDL     string
		toSDL       string
		contains    []string
		notContains []string
	}{
		{
			name:    "Create new sequence",
			fromSDL: ``,
			toSDL: `
				CREATE SEQUENCE my_sequence
				START WITH 1
				INCREMENT BY 1
				MINVALUE 1
				MAXVALUE 9223372036854775807
				CACHE 1;
			`,
			contains: []string{"CREATE SEQUENCE", "my_sequence"},
		},
		{
			name: "Drop sequence",
			fromSDL: `
				CREATE SEQUENCE user_id_seq
				START WITH 1
				INCREMENT BY 1
				MINVALUE 1
				MAXVALUE 2147483647
				CACHE 1;
			`,
			toSDL:    ``,
			contains: []string{"DROP SEQUENCE", "user_id_seq"},
		},
		{
			name: "Modify sequence",
			fromSDL: `
				CREATE SEQUENCE order_seq
				START WITH 1
				INCREMENT BY 1
				CACHE 1;
			`,
			toSDL: `
				CREATE SEQUENCE order_seq
				START WITH 100
				INCREMENT BY 5
				CACHE 10;
			`,
			contains: []string{"order_seq"},
		},
		{
			name:    "Create sequence with data type",
			fromSDL: ``,
			toSDL: `
				CREATE SEQUENCE bigint_sequence AS bigint
				START WITH 1
				INCREMENT BY 1;
			`,
			contains: []string{"CREATE SEQUENCE", "bigint_sequence"},
		},
		{
			name: "Multiple sequences with different operations",
			fromSDL: `
				CREATE SEQUENCE seq_a START WITH 1;
				CREATE SEQUENCE seq_b START WITH 1;
			`,
			toSDL: `
				CREATE SEQUENCE seq_a START WITH 1;
				CREATE SEQUENCE seq_c START WITH 1;
			`,
			contains: []string{"DROP SEQUENCE", "seq_b", "CREATE SEQUENCE", "seq_c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			for _, s := range tt.contains {
				require.Contains(t, sql, s)
			}
			for _, s := range tt.notContains {
				require.NotContains(t, sql, s)
			}
		})
	}
}

func TestOmniSequenceASTOnlyModeValidation(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE SEQUENCE test_sequence
		START WITH 100
		INCREMENT BY 2
		CACHE 5;
	`)
	require.Contains(t, sql, "CREATE SEQUENCE")
	require.Contains(t, sql, "test_sequence")
}

func TestOmniMultipleSequencesHandling(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE SEQUENCE seq1 START WITH 1;
	`, `
		CREATE SEQUENCE seq1 START WITH 1;
		CREATE SEQUENCE seq2 START WITH 10 INCREMENT BY 2;
		CREATE SEQUENCE seq3 AS bigint START WITH 1000;
	`)
	require.Contains(t, sql, "seq2")
	require.Contains(t, sql, "seq3")
	sequenceCount := strings.Count(sql, "CREATE SEQUENCE")
	require.Equal(t, 2, sequenceCount)
}

func TestOmniSequenceOwnershipMigration(t *testing.T) {
	tests := []struct {
		name        string
		fromSDL     string
		toSDL       string
		contains    []string
		expectEmpty bool
	}{
		{
			name: "Add ALTER SEQUENCE OWNED BY",
			fromSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					order_number BIGINT NOT NULL
				);

				CREATE SEQUENCE custom_seq AS bigint START WITH 100 INCREMENT BY 5 MINVALUE 100 MAXVALUE 9223372036854775807 NO CYCLE CACHE 10;
			`,
			toSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					order_number BIGINT NOT NULL
				);

				CREATE SEQUENCE custom_seq AS bigint START WITH 100 INCREMENT BY 5 MINVALUE 100 MAXVALUE 9223372036854775807 NO CYCLE CACHE 10;

				ALTER SEQUENCE custom_seq OWNED BY orders.order_number;
			`,
			// Omni handles ownership changes via DROP/CREATE of the sequence
			contains: []string{"custom_seq"},
		},
		{
			name: "Modify ALTER SEQUENCE OWNED BY (change owner)",
			fromSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					order_number BIGINT NOT NULL
				);
				CREATE TABLE products (
					id BIGINT PRIMARY KEY,
					product_code BIGINT NOT NULL
				);

				CREATE SEQUENCE custom_seq AS bigint START WITH 100 INCREMENT BY 5 NO CYCLE;

				ALTER SEQUENCE custom_seq OWNED BY orders.order_number;
			`,
			toSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					order_number BIGINT NOT NULL
				);
				CREATE TABLE products (
					id BIGINT PRIMARY KEY,
					product_code BIGINT NOT NULL
				);

				CREATE SEQUENCE custom_seq AS bigint START WITH 100 INCREMENT BY 5 NO CYCLE;

				ALTER SEQUENCE custom_seq OWNED BY products.product_code;
			`,
			// Omni absorbs owned sequences into the table definition, so changing
			// the ownership target between two owned columns produces an empty diff.
			expectEmpty: true,
		},
		{
			name: "Remove ALTER SEQUENCE OWNED BY",
			fromSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					order_number BIGINT NOT NULL
				);

				CREATE SEQUENCE custom_seq AS bigint START WITH 100 INCREMENT BY 5 NO CYCLE;

				ALTER SEQUENCE custom_seq OWNED BY orders.order_number;
			`,
			toSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					order_number BIGINT NOT NULL
				);

				CREATE SEQUENCE custom_seq AS bigint START WITH 100 INCREMENT BY 5 NO CYCLE;
			`,
			contains: []string{"custom_seq"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			if tt.expectEmpty {
				require.Empty(t, sql, "expected empty migration SQL")
			}
			for _, s := range tt.contains {
				require.Contains(t, sql, s)
			}
		})
	}
}
