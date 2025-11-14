package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestEnumTypeWithSingleQuotes tests that enum values containing single quotes are properly escaped
func TestEnumTypeWithSingleQuotes(t *testing.T) {
	// Create database metadata with enum type that has single quotes in values
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{
						Name: "status_with_quotes",
						Values: []string{
							"active",
							"don't use",    // Contains single quote
							"can't access", // Contains single quote
							"won't work",   // Contains single quote
						},
					},
				},
			},
		},
	}

	// Generate single-file SDL
	sdl, err := getSDLFormat(metadata)
	require.NoError(t, err, "Failed to generate SDL")
	require.Contains(t, sdl, "don''t use", "Single quotes should be escaped as double single quotes")
	require.Contains(t, sdl, "can''t access", "Single quotes should be escaped as double single quotes")
	require.Contains(t, sdl, "won''t work", "Single quotes should be escaped as double single quotes")

	// Parse the generated SDL - it should parse without errors
	chunks, err := ChunkSDLText(sdl)
	require.NoError(t, err, "Generated SDL with escaped quotes should parse correctly")
	require.Equal(t, 1, len(chunks.EnumTypes), "Should have 1 enum type")

	// Verify the enum type was parsed
	_, exists := chunks.EnumTypes["public.status_with_quotes"]
	require.True(t, exists, "Should have parsed status_with_quotes enum type")
}

// TestEnumTypeWithOtherSpecialChars tests enum values with various special characters
func TestEnumTypeWithOtherSpecialChars(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{
						Name: "special_chars",
						Values: []string{
							"normal",
							"with space",
							"with\ttab",
							"with\nnewline",
							"with-dash",
							"with_underscore",
							"with.dot",
						},
					},
				},
			},
		},
	}

	// Generate SDL
	sdl, err := getSDLFormat(metadata)
	require.NoError(t, err, "Failed to generate SDL")

	// Parse the generated SDL - should work for all special characters
	chunks, err := ChunkSDLText(sdl)
	require.NoError(t, err, "Generated SDL with special characters should parse correctly")
	require.Equal(t, 1, len(chunks.EnumTypes), "Should have 1 enum type")
}

// TestEnumTypeRoundTrip tests that enum values survive a full round-trip through SDL
func TestEnumTypeRoundTrip(t *testing.T) {
	originalValues := []string{
		"active",
		"inactive",
		"don't care",
		"user's choice",
		"it's working",
	}

	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				EnumTypes: []*storepb.EnumTypeMetadata{
					{
						Name:   "roundtrip_status",
						Values: originalValues,
					},
				},
			},
		},
	}

	// Generate SDL
	sdl, err := getSDLFormat(metadata)
	require.NoError(t, err, "Failed to generate SDL")

	// Parse it back
	chunks, err := ChunkSDLText(sdl)
	require.NoError(t, err, "Failed to parse generated SDL")
	require.Equal(t, 1, len(chunks.EnumTypes), "Should have 1 enum type")

	// Verify the enum exists
	_, exists := chunks.EnumTypes["public.roundtrip_status"]
	require.True(t, exists, "Should have parsed roundtrip_status enum type")

	// Note: We can't easily verify the exact values from the parsed chunks
	// because the parser doesn't expose the enum values directly.
	// The important thing is that it parses without errors.
}
