package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCommentedOutEnumType tests that commented-out enum type definitions are ignored
func TestCommentedOutEnumType(t *testing.T) {
	// This reproduces the issue from the screenshot where commented SQL causes syntax error
	sdl := `-- Enum type: Order status
-- CREATE TYPE "public"."order_status_enum" AS ENUM (
--     'draft',
--     'pending',
--     'processing',
--     'shipped',
--     'delivered',
--     'cancelled',
--     'refunded'
-- );
--
-- COMMENT ON TYPE "public"."order_status_enum" IS 'Order lifecycle status values';
--
-- Enum type: User role
CREATE TYPE "public"."user_role" AS ENUM (
    'admin',
    'user'
);
`

	// Parse the SDL
	chunks, err := ChunkSDLText(sdl)
	require.NoError(t, err, "Should successfully parse SDL with commented-out enum type")
	require.NotNil(t, chunks)

	// Should only have 1 enum type (user_role), not 2
	// The commented-out order_status_enum should be ignored
	require.Equal(t, 1, len(chunks.EnumTypes), "Should only parse non-commented enum types")

	// Verify we got the correct enum type
	_, exists := chunks.EnumTypes["public.user_role"]
	require.True(t, exists, "Should have parsed user_role enum type")

	// Verify the commented-out enum type is NOT parsed
	_, exists = chunks.EnumTypes["public.order_status_enum"]
	require.False(t, exists, "Should NOT parse commented-out order_status_enum")
}

// TestMultipleCommentedOutStatements tests various commented-out SQL patterns
func TestMultipleCommentedOutStatements(t *testing.T) {
	sdl := `-- This is a comment
-- CREATE TYPE "public"."status1" AS ENUM ('a', 'b');

CREATE TYPE "public"."status2" AS ENUM ('x', 'y');

-- CREATE TYPE "public"."status3" AS ENUM (
--     'foo',
--     'bar'
-- );

CREATE TABLE "public"."users" (
    id INTEGER PRIMARY KEY,
    status "public"."status2"
);

-- CREATE TABLE "public"."orders" (
--     id INTEGER PRIMARY KEY
-- );
`

	chunks, err := ChunkSDLText(sdl)
	require.NoError(t, err, "Should parse SDL with mixed commented and uncommented statements")
	require.NotNil(t, chunks)

	// Should only have 1 enum type
	require.Equal(t, 1, len(chunks.EnumTypes), "Should only parse uncommented enum type")

	// Should only have 1 table
	require.Equal(t, 1, len(chunks.Tables), "Should only parse uncommented table")

	// Verify we got the correct objects
	_, exists := chunks.EnumTypes["public.status2"]
	require.True(t, exists, "Should have parsed status2")

	_, exists = chunks.Tables["public.users"]
	require.True(t, exists, "Should have parsed users table")
}

// TestCommentedEnumFromFile simulates reading SDL from a file with commented-out enum
func TestCommentedEnumFromFile(t *testing.T) {
	// Simulate exact content from user's screenshot
	fileContent := `-- Enum type: Order status
-- CREATE TYPE "public"."order_status_enum" AS ENUM (
--     'draft',
--     'pending',
--     'processing',
--     'shipped',
--     'delivered',
--     'cancelled',
--     'refunded'
-- );
--
-- COMMENT ON TYPE "public"."order_status_enum" IS 'Order lifecycle status values';
--
-- Enum type: User role
CREATE TYPE "public"."user_role_enum" AS ENUM (
    'admin',
    'moderator',
    'user',
    'guest'
);

COMMENT ON TYPE "public"."user_role_enum" IS 'User role types';
`

	// Test ChunkSDLText
	chunks, err := ChunkSDLText(fileContent)
	require.NoError(t, err, "Failed to chunk SDL: %v", err)
	require.NotNil(t, chunks)

	t.Logf("Number of enum types parsed: %d", len(chunks.EnumTypes))
	for k := range chunks.EnumTypes {
		t.Logf("  - %s", k)
	}

	// Should only have 1 enum type
	require.Equal(t, 1, len(chunks.EnumTypes), "Should only parse non-commented enum types")

	// Test GetSDLDiff
	previousSDL := ""
	currentSDL := fileContent

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err, "GetSDLDiff should not fail with commented SQL: %v", err)
	require.NotNil(t, diff)

	// Should have 1 enum type creation
	require.Equal(t, 1, len(diff.EnumTypeChanges), "Should have 1 enum type change")
	require.Equal(t, "user_role_enum", diff.EnumTypeChanges[0].EnumTypeName)
}

// TestMultiFileScenarioWithComments simulates reading multiple files
func TestMultiFileScenarioWithComments(t *testing.T) {
	// File 1: types.sql with commented and uncommented enums
	typesFile := `-- Commented out enum
-- CREATE TYPE "public"."old_status" AS ENUM ('a', 'b');

CREATE TYPE "public"."current_status" AS ENUM ('active', 'inactive');
`

	// File 2: tables.sql
	tablesFile := `CREATE TABLE "public"."users" (
    id SERIAL PRIMARY KEY,
    status "public"."current_status" NOT NULL
);
`

	// Concatenate files (simulating what might happen in multi-file mode)
	combinedSDL := typesFile + "\n" + tablesFile

	chunks, err := ChunkSDLText(combinedSDL)
	require.NoError(t, err, "Should parse combined SDL from multiple files")
	require.NotNil(t, chunks)

	// Should only have 1 enum type
	require.Equal(t, 1, len(chunks.EnumTypes))

	// Should have 1 table
	require.Equal(t, 1, len(chunks.Tables))
}

// TestExactContentFromScreenshot tests the exact content from user's screenshot
func TestExactContentFromScreenshot(t *testing.T) {
	// Exact content from screenshot (lines 18188-18202)
	sdl := `-- Enum type: Order status
-- CREATE TYPE "public"."order_status_enum" AS ENUM (
--     'draft',
--     'pending',
--     'processing',
--     'shipped',
--     'delivered',
--     'cancelled',
--     'refunded'
-- );
--
-- COMMENT ON TYPE "public"."order_status_enum" IS 'Order lifecycle status values';
--
-- Enum type: User role
CREATE TYPE "public"."user_role_enum" AS ENUM (
    'admin',
    'user'
);
`

	// This should parse without errors
	chunks, err := ChunkSDLText(sdl)
	if err != nil {
		t.Fatalf("Failed to parse SDL with commented enum: %v\nSDL content:\n%s", err, sdl)
	}
	require.NotNil(t, chunks)

	// Verify only the uncommented enum was parsed
	require.Equal(t, 1, len(chunks.EnumTypes), "Should parse only user_role_enum")

	_, exists := chunks.EnumTypes["public.user_role_enum"]
	require.True(t, exists, "Should have user_role_enum")
}

// TestCommentWithSpecialQuotes tests comments with different quote patterns
func TestCommentWithSpecialQuotes(t *testing.T) {
	sdl := `-- COMMENT ON TYPE "public"."order_status_enum" IS 'Order lifecycle status values';
CREATE TYPE "public"."status" AS ENUM ('active');
`

	chunks, err := ChunkSDLText(sdl)
	require.NoError(t, err, "Should handle commented COMMENT ON TYPE")
	require.NotNil(t, chunks)
	require.Equal(t, 1, len(chunks.EnumTypes))
}
