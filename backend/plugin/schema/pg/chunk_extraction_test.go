package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestChunkSDLTextExtraction(t *testing.T) {
	testCases := []struct {
		name            string
		sdlText         string
		expectedResults map[string]string // chunk type:identifier -> expected text
	}{
		{
			name: "Single table extraction",
			sdlText: `CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);`,
			expectedResults: map[string]string{
				"TABLE:users": `CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
)`,
			},
		},
		{
			name: "Multiple statements extraction",
			sdlText: `CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE INDEX idx_users_name ON users(name);

CREATE SEQUENCE user_seq START 1;`,
			expectedResults: map[string]string{
				"TABLE:users": `CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
)`,
				"INDEX:idx_users_name": `CREATE INDEX idx_users_name ON users(name)`,
				"SEQUENCE:user_seq":    `CREATE SEQUENCE user_seq START 1`,
			},
		},
		{
			name: "Function extraction",
			sdlText: `CREATE FUNCTION get_user_count() RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql;`,
			expectedResults: map[string]string{
				"FUNCTION:get_user_count()": `CREATE FUNCTION get_user_count() RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql`,
			},
		},
		{
			name: "View with qualified name",
			sdlText: `CREATE VIEW public.active_users AS 
SELECT * FROM users WHERE active = true;`,
			expectedResults: map[string]string{
				"VIEW:public.active_users": `CREATE VIEW public.active_users AS 
SELECT * FROM users WHERE active = true`,
			},
		},
		{
			name: "Complex mixed statements",
			sdlText: `CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2)
);

CREATE VIEW product_summary AS
SELECT id, name, price
FROM products
WHERE price > 0;

CREATE INDEX idx_products_price ON products(price);

CREATE FUNCTION calculate_discount(amount DECIMAL) RETURNS DECIMAL AS $$
BEGIN
    RETURN amount * 0.9;
END;
$$ LANGUAGE plpgsql;

CREATE SEQUENCE product_seq START 100;`,
			expectedResults: map[string]string{
				"TABLE:products": `CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2)
)`,
				"VIEW:product_summary": `CREATE VIEW product_summary AS
SELECT id, name, price
FROM products
WHERE price > 0`,
				"INDEX:idx_products_price": `CREATE INDEX idx_products_price ON products(price)`,
				"FUNCTION:calculate_discount(amount numeric)": `CREATE FUNCTION calculate_discount(amount DECIMAL) RETURNS DECIMAL AS $$
BEGIN
    RETURN amount * 0.9;
END;
$$ LANGUAGE plpgsql`,
				"SEQUENCE:product_seq": `CREATE SEQUENCE product_seq START 100`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Extract chunks from SDL text
			chunks, err := ChunkSDLText(tc.sdlText)
			require.NoError(t, err, "Should successfully chunk SDL text")
			require.NotNil(t, chunks, "Chunks should not be nil")
			require.NotNil(t, chunks.Tokens, "Token stream should not be nil")

			// Build a map of all chunks with type:identifier keys
			allChunks := make(map[string]*schema.SDLChunk)
			for id, chunk := range chunks.Tables {
				allChunks["TABLE:"+id] = chunk
			}
			for id, chunk := range chunks.Views {
				allChunks["VIEW:"+id] = chunk
			}
			for id, chunk := range chunks.Functions {
				allChunks["FUNCTION:"+id] = chunk
			}
			for id, chunk := range chunks.Indexes {
				allChunks["INDEX:"+id] = chunk
			}
			for id, chunk := range chunks.Sequences {
				allChunks["SEQUENCE:"+id] = chunk
			}

			// Verify we have the expected number of chunks
			assert.Equal(t, len(tc.expectedResults), len(allChunks),
				"Expected %d chunks, got %d", len(tc.expectedResults), len(allChunks))

			// Verify each expected chunk
			for expectedKey, expectedText := range tc.expectedResults {
				chunk, exists := allChunks[expectedKey]
				require.True(t, exists, "Expected chunk with key %s not found", expectedKey)

				// Verify chunk has required fields
				assert.NotEmpty(t, chunk.Identifier, "Chunk identifier should not be empty")
				assert.NotNil(t, chunk.ASTNode, "Chunk AST node should not be nil")

				// Extract text using GetText method and compare
				actualText := chunk.GetText(chunks.Tokens)
				require.NotEmpty(t, actualText, "GetText should return non-empty text")

				// Normalize whitespace for comparison
				actualTextNormalized := strings.TrimSpace(actualText)
				expectedTextNormalized := strings.TrimSpace(expectedText)

				assert.Equal(t, expectedTextNormalized, actualTextNormalized,
					"Text mismatch for chunk %s.\nExpected:\n%s\nActual:\n%s",
					expectedKey, expectedTextNormalized, actualTextNormalized)

				// Verify AST node type matches chunk type
				switch {
				case strings.HasPrefix(expectedKey, "TABLE:"):
					_, ok := chunk.ASTNode.(*parser.CreatestmtContext)
					assert.True(t, ok, "TABLE chunk should have CreatestmtContext AST node")
				case strings.HasPrefix(expectedKey, "INDEX:"):
					_, ok := chunk.ASTNode.(*parser.IndexstmtContext)
					assert.True(t, ok, "INDEX chunk should have IndexstmtContext AST node")
				case strings.HasPrefix(expectedKey, "SEQUENCE:"):
					_, ok := chunk.ASTNode.(*parser.CreateseqstmtContext)
					assert.True(t, ok, "SEQUENCE chunk should have CreateseqstmtContext AST node")
				case strings.HasPrefix(expectedKey, "FUNCTION:"):
					_, ok := chunk.ASTNode.(*parser.CreatefunctionstmtContext)
					assert.True(t, ok, "FUNCTION chunk should have CreatefunctionstmtContext AST node")
				case strings.HasPrefix(expectedKey, "VIEW:"):
					_, ok := chunk.ASTNode.(*parser.ViewstmtContext)
					assert.True(t, ok, "VIEW chunk should have ViewstmtContext AST node")
				default:
					assert.Fail(t, "Unknown chunk type: %s", expectedKey)
				}
			}
		})
	}
}

func TestChunkSDLTextEdgeCases(t *testing.T) {
	testCases := []struct {
		name           string
		sdlText        string
		expectedChunks int
		shouldError    bool
	}{
		{
			name:           "Empty SDL text",
			sdlText:        "",
			expectedChunks: 0,
			shouldError:    false,
		},
		{
			name:           "Whitespace only",
			sdlText:        "   \n\t   ",
			expectedChunks: 0,
			shouldError:    false,
		},
		{
			name: "CREATE INDEX without name (should be skipped)",
			sdlText: `CREATE TABLE users (id SERIAL);
CREATE INDEX ON users(id);`,
			expectedChunks: 1, // Only table, index without name is skipped
			shouldError:    false,
		},
		{
			name: "Schema-qualified identifiers",
			sdlText: `CREATE TABLE public.users (id SERIAL);
CREATE TABLE admin.settings (key VARCHAR);`,
			expectedChunks: 2,
			shouldError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chunks, err := ChunkSDLText(tc.sdlText)

			if tc.shouldError {
				assert.Error(t, err, "Should return error")
				return
			}

			require.NoError(t, err, "Should not return error")
			require.NotNil(t, chunks, "Chunks should not be nil")

			// Count total chunks
			totalChunks := len(chunks.Tables) + len(chunks.Views) + len(chunks.Functions) +
				len(chunks.Indexes) + len(chunks.Sequences)

			assert.Equal(t, tc.expectedChunks, totalChunks,
				"Expected %d chunks, got %d", tc.expectedChunks, totalChunks)
		})
	}
}

func TestChunkGetTextMethod(t *testing.T) {
	sdlText := `CREATE TABLE test_table (
    id INTEGER PRIMARY KEY,
    data TEXT
);`

	chunks, err := ChunkSDLText(sdlText)
	require.NoError(t, err)
	require.Contains(t, chunks.Tables, "test_table")

	chunk := chunks.Tables["test_table"]

	// Test GetText method
	text := chunk.GetText(chunks.Tokens)
	assert.NotEmpty(t, text, "GetText should return non-empty text")
	assert.Contains(t, text, "CREATE TABLE test_table", "Text should contain table creation")
	assert.Contains(t, text, "id INTEGER PRIMARY KEY", "Text should contain column definition")

	// Test with nil tokens (should handle gracefully)
	emptyText := chunk.GetText(nil)
	assert.Empty(t, emptyText, "GetText with nil tokens should return empty string")
}
