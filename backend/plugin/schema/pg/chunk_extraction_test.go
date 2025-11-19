package pg

import (
	"strings"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/parser/postgresql"

	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
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
				"TABLE:public.users": `CREATE TABLE users (
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
				"TABLE:public.users": `CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
)`,
				"INDEX:public.idx_users_name": `CREATE INDEX idx_users_name ON users(name)`,
				"SEQUENCE:public.user_seq":    `CREATE SEQUENCE user_seq START 1`,
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
				"FUNCTION:public.get_user_count()": `CREATE FUNCTION get_user_count() RETURNS INTEGER AS $$
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
				"TABLE:public.products": `CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2)
)`,
				"VIEW:public.product_summary": `CREATE VIEW product_summary AS
SELECT id, name, price
FROM products
WHERE price > 0`,
				"INDEX:public.idx_products_price": `CREATE INDEX idx_products_price ON products(price)`,
				"FUNCTION:public.calculate_discount(amount numeric)": `CREATE FUNCTION calculate_discount(amount DECIMAL) RETURNS DECIMAL AS $$
BEGIN
    RETURN amount * 0.9;
END;
$$ LANGUAGE plpgsql`,
				"SEQUENCE:public.product_seq": `CREATE SEQUENCE product_seq START 100`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Extract chunks from SDL text
			chunks, err := ChunkSDLText(tc.sdlText)
			require.NoError(t, err, "Should successfully chunk SDL text")
			require.NotNil(t, chunks, "Chunks should not be nil")
			// Token stream is no longer stored in chunks, obtained from AST nodes directly

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
				actualText := chunk.GetText()
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
	require.Contains(t, chunks.Tables, "public.test_table")

	chunk := chunks.Tables["public.test_table"]

	// Test GetText method
	text := chunk.GetText()
	assert.NotEmpty(t, text, "GetText should return non-empty text")
	assert.Contains(t, text, "CREATE TABLE test_table", "Text should contain table creation")
	assert.Contains(t, text, "id INTEGER PRIMARY KEY", "Text should contain column definition")

	// Test that GetText works with the new token-less approach
	// (This test is no longer relevant since we get tokens from AST directly)
}

func TestChunkKeyFormats(t *testing.T) {
	sdlText := `CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT);
CREATE TABLE admin.users (id SERIAL, role TEXT);
CREATE SEQUENCE user_seq;
CREATE SEQUENCE admin.admin_seq;
CREATE VIEW user_view AS SELECT * FROM users;
CREATE VIEW admin.admin_view AS SELECT * FROM admin.users;
CREATE INDEX idx_users_name ON users(name);`

	chunks, err := ChunkSDLText(sdlText)
	require.NoError(t, err)
	require.NotNil(t, chunks)

	// Check that all chunk keys use schema.objectName format
	expectedTableKeys := []string{"public.users", "admin.users"}
	expectedSequenceKeys := []string{"public.user_seq", "admin.admin_seq"}
	expectedViewKeys := []string{"public.user_view", "admin.admin_view"}
	expectedIndexKeys := []string{"public.idx_users_name"}

	// Verify table chunk keys
	require.Equal(t, len(expectedTableKeys), len(chunks.Tables))
	for _, key := range expectedTableKeys {
		_, exists := chunks.Tables[key]
		require.True(t, exists, "Expected table key %s not found", key)
	}

	// Verify sequence chunk keys
	require.Equal(t, len(expectedSequenceKeys), len(chunks.Sequences))
	for _, key := range expectedSequenceKeys {
		_, exists := chunks.Sequences[key]
		require.True(t, exists, "Expected sequence key %s not found", key)
	}

	// Verify view chunk keys
	require.Equal(t, len(expectedViewKeys), len(chunks.Views))
	for _, key := range expectedViewKeys {
		_, exists := chunks.Views[key]
		require.True(t, exists, "Expected view key %s not found", key)
	}

	// Verify index chunk keys
	require.Equal(t, len(expectedIndexKeys), len(chunks.Indexes))
	for _, key := range expectedIndexKeys {
		_, exists := chunks.Indexes[key]
		require.True(t, exists, "Expected index key %s not found", key)
	}
}

// TestNewTableASTParsing tests that new tables created during drift detection
// have proper AST nodes that can generate correct text
func TestNewTableASTParsing(t *testing.T) {
	// Test the createTableExtractor functionality
	tableSDL := `CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text NOT NULL
);`

	parseResults, err := pgparser.ParsePostgreSQL(tableSDL)
	require.NoError(t, err)
	require.Len(t, parseResults, 1, "Should parse single statement")

	// Extract the CREATE TABLE AST node
	var createTableNode *parser.CreatestmtContext
	antlr.ParseTreeWalkerDefault.Walk(&createTableExtractor{
		result: &createTableNode,
	}, parseResults[0].Tree)

	require.NotNil(t, createTableNode, "Should extract CREATE TABLE AST node")

	// Create chunk with the AST node
	chunk := &schema.SDLChunk{
		Identifier: "public.users",
		ASTNode:    createTableNode,
	}

	// Verify we can extract text from the AST node
	text := chunk.GetText()
	require.NotEmpty(t, text)
	require.Contains(t, strings.ToUpper(text), "CREATE TABLE")
	require.Contains(t, text, "users")
}
