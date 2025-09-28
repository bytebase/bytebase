package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestFunctionRewriteOperations tests the ANTLR TokenStreamRewriter functionality
// for standalone CREATE FUNCTION operations including add, modify, delete
func TestFunctionRewriteOperations(t *testing.T) {
	testCases := []struct {
		name                  string
		originalSDL           string
		currentFunctions      []*storepb.FunctionMetadata
		previousFunctions     []*storepb.FunctionMetadata
		expectedFunctionCount int
		description           string
	}{
		{
			name: "Add new standalone function",
			originalSDL: `CREATE TABLE users (
    id INTEGER NOT NULL,
    email VARCHAR(255) NOT NULL
);`,
			currentFunctions: []*storepb.FunctionMetadata{
				{
					Name: "get_user_count",
					Definition: `CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
	RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql`,
				},
			},
			previousFunctions:     []*storepb.FunctionMetadata{},
			expectedFunctionCount: 1,
			description:           "Should add new standalone function to chunks",
		},
		{
			name: "Add function with complex signature",
			originalSDL: `CREATE TABLE orders (
    id INTEGER NOT NULL,
    amount DECIMAL(10,2) NOT NULL
);`,
			currentFunctions: []*storepb.FunctionMetadata{
				{
					Name: "calculate_total",
					Definition: `CREATE OR REPLACE FUNCTION calculate_total(
	p_start_date DATE,
	p_end_date DATE,
	p_discount_rate DECIMAL DEFAULT 0.0
)
RETURNS DECIMAL(10,2) AS $$
BEGIN
	RETURN (
		SELECT COALESCE(SUM(amount * (1 - p_discount_rate)), 0)
		FROM orders
		WHERE created_at::DATE BETWEEN p_start_date AND p_end_date
	);
END;
$$ LANGUAGE plpgsql`,
				},
			},
			previousFunctions:     []*storepb.FunctionMetadata{},
			expectedFunctionCount: 1,
			description:           "Should add new function with complex signature to chunks",
		},
		{
			name: "Drop standalone function",
			originalSDL: `CREATE TABLE users (
    id INTEGER NOT NULL,
    email VARCHAR(255) NOT NULL
);

CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
	RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql;`,
			currentFunctions: []*storepb.FunctionMetadata{},
			previousFunctions: []*storepb.FunctionMetadata{
				{
					Name: "get_user_count",
					Definition: `CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
	RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql`,
				},
			},
			expectedFunctionCount: 0,
			description:           "Should remove standalone function from chunks",
		},
		{
			name: "Modify function (change definition)",
			originalSDL: `CREATE TABLE users (
    id INTEGER NOT NULL,
    email VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true
);

CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
	RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql;`,
			currentFunctions: []*storepb.FunctionMetadata{
				{
					Name: "get_user_count",
					Definition: `CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
	RETURN (SELECT COUNT(*) FROM users WHERE active = true);
END;
$$ LANGUAGE plpgsql`,
				},
			},
			previousFunctions: []*storepb.FunctionMetadata{
				{
					Name: "get_user_count",
					Definition: `CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
	RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql`,
				},
			},
			expectedFunctionCount: 1,
			description:           "Should modify existing function when definition changes",
		},
		{
			name: "Multiple function operations",
			originalSDL: `CREATE TABLE orders (
    id INTEGER NOT NULL,
    customer_id INTEGER,
    total DECIMAL(10,2),
    status VARCHAR(20)
);

CREATE OR REPLACE FUNCTION get_order_count()
RETURNS INTEGER AS $$
BEGIN
	RETURN (SELECT COUNT(*) FROM orders);
END;
$$ LANGUAGE plpgsql;`,
			currentFunctions: []*storepb.FunctionMetadata{
				{
					Name: "get_total_amount",
					Definition: `CREATE OR REPLACE FUNCTION get_total_amount()
RETURNS DECIMAL(10,2) AS $$
BEGIN
	RETURN (SELECT SUM(total) FROM orders);
END;
$$ LANGUAGE plpgsql`,
				},
				{
					Name: "get_active_orders",
					Definition: `CREATE OR REPLACE FUNCTION get_active_orders()
RETURNS INTEGER AS $$
BEGIN
	RETURN (SELECT COUNT(*) FROM orders WHERE status = 'active');
END;
$$ LANGUAGE plpgsql`,
				},
			},
			previousFunctions: []*storepb.FunctionMetadata{
				{
					Name: "get_order_count",
					Definition: `CREATE OR REPLACE FUNCTION get_order_count()
RETURNS INTEGER AS $$
BEGIN
	RETURN (SELECT COUNT(*) FROM orders);
END;
$$ LANGUAGE plpgsql`,
				},
			},
			expectedFunctionCount: 2,
			description:           "Should handle multiple function operations (drop old, add new)",
		},
		{
			name: "Schema-qualified function names",
			originalSDL: `CREATE SCHEMA test_schema;
CREATE TABLE test_schema.products (
    id INTEGER NOT NULL,
    name VARCHAR(255)
);`,
			currentFunctions: []*storepb.FunctionMetadata{
				{
					Name: "get_product_count",
					Definition: `CREATE OR REPLACE FUNCTION test_schema.get_product_count()
RETURNS INTEGER AS $$
BEGIN
	RETURN (SELECT COUNT(*) FROM test_schema.products);
END;
$$ LANGUAGE plpgsql`,
				},
			},
			previousFunctions:     []*storepb.FunctionMetadata{},
			expectedFunctionCount: 1,
			description:           "Should handle schema-qualified function names",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the original SDL to create chunks
			chunks, err := ChunkSDLText(tc.originalSDL)
			require.NoError(t, err, "Failed to parse original SDL")

			// Create function maps for simulation
			currentFunctions := make(map[string]*storepb.FunctionMetadata)
			previousFunctions := make(map[string]*storepb.FunctionMetadata)

			for _, fn := range tc.currentFunctions {
				key := formatFunctionKey("public", fn)
				currentFunctions[key] = fn
			}

			for _, fn := range tc.previousFunctions {
				key := formatFunctionKey("public", fn)
				previousFunctions[key] = fn
			}

			// Apply function changes directly to test the core logic
			err = applyFunctionChangesInternal(chunks, currentFunctions, previousFunctions)
			require.NoError(t, err, "Failed to apply function changes")

			// Verify the expected number of functions
			assert.Len(t, chunks.Functions, tc.expectedFunctionCount,
				"Expected %d functions, got %d", tc.expectedFunctionCount, len(chunks.Functions))

			// Verify each current function is present
			for key, fn := range currentFunctions {
				chunk, exists := chunks.Functions[key]
				assert.True(t, exists, "Expected function %s not found in chunks", key)
				if exists {
					assert.NotNil(t, chunk.ASTNode, "Function chunk should have AST node")
					// Verify we can get text from the chunk using the proper extraction function
					if functionAST, ok := chunk.ASTNode.(*parser.CreatefunctionstmtContext); ok {
						text := extractFunctionTextFromAST(functionAST)
						assert.NotEmpty(t, text, "Function chunk should have non-empty text")
						assert.Contains(t, text, "CREATE", "Function text should contain CREATE")
						assert.Contains(t, text, "FUNCTION", "Function text should contain FUNCTION")
						assert.Contains(t, text, fn.Name, "Function text should contain function name")
					}
				}
			}

			// Verify previous functions that shouldn't exist are removed
			for key := range previousFunctions {
				if _, stillExists := currentFunctions[key]; !stillExists {
					_, exists := chunks.Functions[key]
					assert.False(t, exists, "Function %s should have been removed from chunks", key)
				}
			}

			t.Logf("Test case '%s' passed: %s", tc.name, tc.description)
		})
	}
}

// applyFunctionChangesInternal is a test helper that directly applies function changes to chunks
func applyFunctionChangesInternal(chunks *schema.SDLChunks, currentFunctions, previousFunctions map[string]*storepb.FunctionMetadata) error {
	if chunks == nil {
		return nil
	}

	// Process function additions: create new function chunks
	for functionKey, currentFunction := range currentFunctions {
		if _, exists := previousFunctions[functionKey]; !exists {
			// New function - create a chunk for it
			err := createFunctionChunk(chunks, currentFunction, functionKey)
			if err != nil {
				return err
			}
		}
	}

	// Process function modifications: update existing chunks
	for functionKey, currentFunction := range currentFunctions {
		if previousFunction, exists := previousFunctions[functionKey]; exists {
			// Function exists in both - check if it needs modification
			err := updateFunctionChunkIfNeeded(chunks, currentFunction, previousFunction, functionKey)
			if err != nil {
				return err
			}
		}
	}

	// Process function deletions: remove dropped function chunks
	for functionKey := range previousFunctions {
		if _, exists := currentFunctions[functionKey]; !exists {
			// Function was dropped - remove it from chunks
			deleteFunctionChunk(chunks, functionKey)
		}
	}

	return nil
}

// TestGenerateCreateFunctionSDL tests the SDL generation for CREATE FUNCTION statements
func TestGenerateCreateFunctionSDL(t *testing.T) {
	testCases := []struct {
		name        string
		function    *storepb.FunctionMetadata
		expectedSDL string
	}{
		{
			name: "Simple function",
			function: &storepb.FunctionMetadata{
				Name: "get_user_count",
				Definition: `CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
	RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql`,
			},
			expectedSDL: `CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INTEGER AS $$
BEGIN
	RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql`,
		},
		{
			name: "Function with parameters",
			function: &storepb.FunctionMetadata{
				Name: "calculate_total",
				Definition: `CREATE OR REPLACE FUNCTION calculate_total(
	p_start_date DATE,
	p_end_date DATE,
	p_discount_rate DECIMAL DEFAULT 0.0
)
RETURNS DECIMAL(10,2) AS $$
BEGIN
	RETURN (
		SELECT COALESCE(SUM(amount * (1 - p_discount_rate)), 0)
		FROM orders
		WHERE created_at::DATE BETWEEN p_start_date AND p_end_date
	);
END;
$$ LANGUAGE plpgsql`,
			},
			expectedSDL: `CREATE OR REPLACE FUNCTION calculate_total(
	p_start_date DATE,
	p_end_date DATE,
	p_discount_rate DECIMAL DEFAULT 0.0
)
RETURNS DECIMAL(10,2) AS $$
BEGIN
	RETURN (
		SELECT COALESCE(SUM(amount * (1 - p_discount_rate)), 0)
		FROM orders
		WHERE created_at::DATE BETWEEN p_start_date AND p_end_date
	);
END;
$$ LANGUAGE plpgsql`,
		},
		{
			name: "Function with trailing semicolon",
			function: &storepb.FunctionMetadata{
				Name: "simple_func",
				Definition: `CREATE OR REPLACE FUNCTION simple_func()
RETURNS TEXT AS $$
BEGIN
	RETURN 'hello';
END;
$$ LANGUAGE plpgsql;`,
			},
			expectedSDL: `CREATE OR REPLACE FUNCTION simple_func()
RETURNS TEXT AS $$
BEGIN
	RETURN 'hello';
END;
$$ LANGUAGE plpgsql`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := generateCreateFunctionSDL(tc.function)
			assert.Equal(t, tc.expectedSDL, result)
		})
	}
}

// TestFunctionDefinitionsEqual tests the function comparison logic
func TestFunctionDefinitionsEqual(t *testing.T) {
	testCases := []struct {
		name      string
		function1 *storepb.FunctionMetadata
		function2 *storepb.FunctionMetadata
		expected  bool
	}{
		{
			name: "Identical functions",
			function1: &storepb.FunctionMetadata{
				Name: "test_func",
				Definition: `CREATE OR REPLACE FUNCTION test_func()
RETURNS INTEGER AS $$
BEGIN
	RETURN 1;
END;
$$ LANGUAGE plpgsql`,
			},
			function2: &storepb.FunctionMetadata{
				Name: "test_func",
				Definition: `CREATE OR REPLACE FUNCTION test_func()
RETURNS INTEGER AS $$
BEGIN
	RETURN 1;
END;
$$ LANGUAGE plpgsql`,
			},
			expected: true,
		},
		{
			name: "Different names",
			function1: &storepb.FunctionMetadata{
				Name: "test_func1",
				Definition: `CREATE OR REPLACE FUNCTION test_func1()
RETURNS INTEGER AS $$
BEGIN
	RETURN 1;
END;
$$ LANGUAGE plpgsql`,
			},
			function2: &storepb.FunctionMetadata{
				Name: "test_func2",
				Definition: `CREATE OR REPLACE FUNCTION test_func2()
RETURNS INTEGER AS $$
BEGIN
	RETURN 1;
END;
$$ LANGUAGE plpgsql`,
			},
			expected: false,
		},
		{
			name: "Different definitions",
			function1: &storepb.FunctionMetadata{
				Name: "test_func",
				Definition: `CREATE OR REPLACE FUNCTION test_func()
RETURNS INTEGER AS $$
BEGIN
	RETURN 1;
END;
$$ LANGUAGE plpgsql`,
			},
			function2: &storepb.FunctionMetadata{
				Name: "test_func",
				Definition: `CREATE OR REPLACE FUNCTION test_func()
RETURNS INTEGER AS $$
BEGIN
	RETURN 2;
END;
$$ LANGUAGE plpgsql`,
			},
			expected: false,
		},
		{
			name: "Same definition with different trailing semicolons",
			function1: &storepb.FunctionMetadata{
				Name: "test_func",
				Definition: `CREATE OR REPLACE FUNCTION test_func()
RETURNS INTEGER AS $$
BEGIN
	RETURN 1;
END;
$$ LANGUAGE plpgsql`,
			},
			function2: &storepb.FunctionMetadata{
				Name: "test_func",
				Definition: `CREATE OR REPLACE FUNCTION test_func()
RETURNS INTEGER AS $$
BEGIN
	RETURN 1;
END;
$$ LANGUAGE plpgsql;`,
			},
			expected: true,
		},
		{
			name: "Same definition with different whitespace",
			function1: &storepb.FunctionMetadata{
				Name: "test_func",
				Definition: `CREATE OR REPLACE FUNCTION test_func()
RETURNS INTEGER AS $$
BEGIN
	RETURN 1;
END;
$$ LANGUAGE plpgsql`,
			},
			function2: &storepb.FunctionMetadata{
				Name: "test_func",
				Definition: `  CREATE OR REPLACE FUNCTION test_func()
RETURNS INTEGER AS $$
BEGIN
	RETURN 1;
END;
$$ LANGUAGE plpgsql  `,
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := functionDefinitionsEqual(tc.function1, tc.function2)
			assert.Equal(t, tc.expected, result)
		})
	}
}
