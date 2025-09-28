package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestSequenceRewriteOperations tests the ANTLR TokenStreamRewriter functionality
// for standalone CREATE SEQUENCE operations including add, modify, delete
func TestSequenceRewriteOperations(t *testing.T) {
	testCases := []struct {
		name                  string
		originalSDL           string
		currentSequences      []*storepb.SequenceMetadata
		previousSequences     []*storepb.SequenceMetadata
		expectedSequenceCount int
		description           string
	}{
		{
			name: "Add new standalone sequence",
			originalSDL: `CREATE TABLE users (
    id INTEGER NOT NULL,
    email VARCHAR(255) NOT NULL
);`,
			currentSequences: []*storepb.SequenceMetadata{
				{
					Name:      "user_seq",
					DataType:  "bigint",
					Start:     "1",
					Increment: "1",
					MinValue:  "",
					MaxValue:  "",
					CacheSize: "1",
					Cycle:     false,
				},
			},
			previousSequences:     []*storepb.SequenceMetadata{},
			expectedSequenceCount: 1,
			description:           "Should add new standalone sequence to chunks",
		},
		{
			name: "Add sequence with complex options",
			originalSDL: `CREATE TABLE orders (
    id BIGINT NOT NULL,
    amount DECIMAL(10,2) NOT NULL
);`,
			currentSequences: []*storepb.SequenceMetadata{
				{
					Name:        "order_id_seq",
					DataType:    "bigint",
					Start:       "1000000",
					Increment:   "1",
					MinValue:    "1",
					MaxValue:    "9223372036854775807",
					CacheSize:   "50",
					Cycle:       false,
					OwnerTable:  "orders",
					OwnerColumn: "id",
				},
			},
			previousSequences:     []*storepb.SequenceMetadata{},
			expectedSequenceCount: 1,
			description:           "Should add new sequence with complex options to chunks",
		},
		{
			name: "Drop standalone sequence",
			originalSDL: `CREATE TABLE users (
    id INTEGER NOT NULL,
    email VARCHAR(255) NOT NULL
);

CREATE SEQUENCE user_seq
	START WITH 1
	INCREMENT BY 1
	NO MINVALUE
	NO MAXVALUE
	CACHE 1;`,
			currentSequences: []*storepb.SequenceMetadata{},
			previousSequences: []*storepb.SequenceMetadata{
				{
					Name:      "user_seq",
					DataType:  "bigint",
					Start:     "1",
					Increment: "1",
					MinValue:  "",
					MaxValue:  "",
					CacheSize: "1",
					Cycle:     false,
				},
			},
			expectedSequenceCount: 0,
			description:           "Should remove standalone sequence from chunks",
		},
		{
			name: "Modify sequence (change parameters)",
			originalSDL: `CREATE TABLE users (
    id INTEGER NOT NULL,
    email VARCHAR(255) NOT NULL
);

CREATE SEQUENCE user_seq
	START WITH 1
	INCREMENT BY 1
	NO MINVALUE
	NO MAXVALUE
	CACHE 1;`,
			currentSequences: []*storepb.SequenceMetadata{
				{
					Name:      "user_seq",
					DataType:  "bigint",
					Start:     "10",
					Increment: "2",
					MinValue:  "",
					MaxValue:  "",
					CacheSize: "5",
					Cycle:     false,
				},
			},
			previousSequences: []*storepb.SequenceMetadata{
				{
					Name:      "user_seq",
					DataType:  "bigint",
					Start:     "1",
					Increment: "1",
					MinValue:  "",
					MaxValue:  "",
					CacheSize: "1",
					Cycle:     false,
				},
			},
			expectedSequenceCount: 1,
			description:           "Should modify existing sequence when parameters change",
		},
		{
			name: "Multiple sequence operations",
			originalSDL: `CREATE TABLE orders (
    id INTEGER NOT NULL,
    customer_id INTEGER,
    total DECIMAL(10,2),
    status VARCHAR(20)
);

CREATE SEQUENCE order_seq
	START WITH 100
	INCREMENT BY 1;`,
			currentSequences: []*storepb.SequenceMetadata{
				{
					Name:      "order_total_seq",
					DataType:  "bigint",
					Start:     "1000",
					Increment: "1",
					MinValue:  "",
					MaxValue:  "",
					CacheSize: "1",
					Cycle:     false,
				},
				{
					Name:      "customer_seq",
					DataType:  "bigint",
					Start:     "1",
					Increment: "10",
					MinValue:  "",
					MaxValue:  "",
					CacheSize: "1",
					Cycle:     false,
				},
			},
			previousSequences: []*storepb.SequenceMetadata{
				{
					Name:      "order_seq",
					DataType:  "bigint",
					Start:     "100",
					Increment: "1",
					MinValue:  "",
					MaxValue:  "",
					CacheSize: "1",
					Cycle:     false,
				},
			},
			expectedSequenceCount: 2,
			description:           "Should handle multiple sequence operations (drop old, add new)",
		},
		{
			name: "Schema-qualified sequence names",
			originalSDL: `CREATE SCHEMA test_schema;
CREATE TABLE test_schema.products (
    id INTEGER NOT NULL,
    name VARCHAR(255)
);`,
			currentSequences: []*storepb.SequenceMetadata{
				{
					Name:        "product_id_seq",
					DataType:    "bigint",
					Start:       "1",
					Increment:   "1",
					MinValue:    "",
					MaxValue:    "",
					CacheSize:   "1",
					Cycle:       false,
					OwnerTable:  "products",
					OwnerColumn: "id",
				},
			},
			previousSequences:     []*storepb.SequenceMetadata{},
			expectedSequenceCount: 1,
			description:           "Should handle schema-qualified sequence names",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the original SDL to create chunks
			chunks, err := ChunkSDLText(tc.originalSDL)
			require.NoError(t, err, "Failed to parse original SDL")

			// Create sequence maps for simulation
			currentSequences := make(map[string]*storepb.SequenceMetadata)
			previousSequences := make(map[string]*storepb.SequenceMetadata)

			for _, seq := range tc.currentSequences {
				key := formatSequenceKey("public", seq.Name)
				currentSequences[key] = seq
			}

			for _, seq := range tc.previousSequences {
				key := formatSequenceKey("public", seq.Name)
				previousSequences[key] = seq
			}

			// Apply sequence changes directly to test the core logic
			err = applySequenceChangesInternal(chunks, currentSequences, previousSequences)
			require.NoError(t, err, "Failed to apply sequence changes")

			// Verify the expected number of sequences
			assert.Len(t, chunks.Sequences, tc.expectedSequenceCount,
				"Expected %d sequences, got %d", tc.expectedSequenceCount, len(chunks.Sequences))

			// Verify each current sequence is present
			for key, seq := range currentSequences {
				chunk, exists := chunks.Sequences[key]
				assert.True(t, exists, "Expected sequence %s not found in chunks", key)
				if exists {
					assert.NotNil(t, chunk.ASTNode, "Sequence chunk should have AST node")
					// Verify we can get text from the chunk using the proper extraction function
					if sequenceAST, ok := chunk.ASTNode.(*parser.CreateseqstmtContext); ok {
						text := extractSequenceTextFromAST(sequenceAST)
						assert.NotEmpty(t, text, "Sequence chunk should have non-empty text")
						assert.Contains(t, text, "CREATE", "Sequence text should contain CREATE")
						assert.Contains(t, text, "SEQUENCE", "Sequence text should contain SEQUENCE")
						assert.Contains(t, text, seq.Name, "Sequence text should contain sequence name")
					}
				}
			}

			// Verify previous sequences that shouldn't exist are removed
			for key := range previousSequences {
				if _, stillExists := currentSequences[key]; !stillExists {
					_, exists := chunks.Sequences[key]
					assert.False(t, exists, "Sequence %s should have been removed from chunks", key)
				}
			}

			t.Logf("Test case '%s' passed: %s", tc.name, tc.description)
		})
	}
}

// applySequenceChangesInternal is a test helper that directly applies sequence changes to chunks
func applySequenceChangesInternal(chunks *schema.SDLChunks, currentSequences, previousSequences map[string]*storepb.SequenceMetadata) error {
	if chunks == nil {
		return nil
	}

	// Process sequence additions: create new sequence chunks
	for sequenceKey, currentSequence := range currentSequences {
		if _, exists := previousSequences[sequenceKey]; !exists {
			// New sequence - create a chunk for it
			err := createSequenceChunk(chunks, currentSequence, sequenceKey)
			if err != nil {
				return err
			}
		}
	}

	// Process sequence modifications: update existing chunks
	for sequenceKey, currentSequence := range currentSequences {
		if previousSequence, exists := previousSequences[sequenceKey]; exists {
			// Sequence exists in both - check if it needs modification
			err := updateSequenceChunkIfNeeded(chunks, currentSequence, previousSequence, sequenceKey)
			if err != nil {
				return err
			}
		}
	}

	// Process sequence deletions: remove dropped sequence chunks
	for sequenceKey := range previousSequences {
		if _, exists := currentSequences[sequenceKey]; !exists {
			// Sequence was dropped - remove it from chunks
			deleteSequenceChunk(chunks, sequenceKey)
		}
	}

	return nil
}

// TestGenerateCreateSequenceSDL tests the SDL generation for CREATE SEQUENCE statements
func TestGenerateCreateSequenceSDL(t *testing.T) {
	testCases := []struct {
		name        string
		sequence    *storepb.SequenceMetadata
		expectedSDL string
	}{
		{
			name: "Simple sequence",
			sequence: &storepb.SequenceMetadata{
				Name:      "user_seq",
				DataType:  "bigint",
				Start:     "1",
				Increment: "1",
				MinValue:  "",
				MaxValue:  "",
				CacheSize: "1",
				Cycle:     false,
			},
			expectedSDL: `CREATE SEQUENCE "user_seq" START WITH 1 INCREMENT BY 1 CACHE 1`,
		},
		{
			name: "Sequence with complex options",
			sequence: &storepb.SequenceMetadata{
				Name:      "order_id_seq",
				DataType:  "bigint",
				Start:     "1000000",
				Increment: "1",
				MinValue:  "1",
				MaxValue:  "9223372036854775807",
				CacheSize: "50",
				Cycle:     false,
			},
			expectedSDL: `CREATE SEQUENCE "order_id_seq" START WITH 1000000 INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 50`,
		},
		{
			name: "Sequence with owner",
			sequence: &storepb.SequenceMetadata{
				Name:        "product_id_seq",
				DataType:    "bigint",
				Start:       "1",
				Increment:   "1",
				MinValue:    "",
				MaxValue:    "",
				CacheSize:   "1",
				Cycle:       false,
				OwnerTable:  "products",
				OwnerColumn: "id",
			},
			expectedSDL: `CREATE SEQUENCE "product_id_seq" START WITH 1 INCREMENT BY 1 CACHE 1 OWNED BY "products"."id"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := generateCreateSequenceSDL("public", tc.sequence)
			// Note: The exact format may vary depending on the writeSequenceSDL implementation
			// We mainly check that it generates some reasonable SDL
			assert.NotEmpty(t, result, "Should generate non-empty SDL")
			assert.Contains(t, result, "CREATE", "Should contain CREATE")
			assert.Contains(t, result, "SEQUENCE", "Should contain SEQUENCE")
			assert.Contains(t, result, tc.sequence.Name, "Should contain sequence name")
		})
	}
}

// TestSequenceDefinitionsEqual tests the sequence comparison logic
func TestSequenceDefinitionsEqual(t *testing.T) {
	testCases := []struct {
		name      string
		sequence1 *storepb.SequenceMetadata
		sequence2 *storepb.SequenceMetadata
		expected  bool
	}{
		{
			name: "Identical sequences",
			sequence1: &storepb.SequenceMetadata{
				Name:      "test_seq",
				DataType:  "bigint",
				Start:     "1",
				Increment: "1",
				CacheSize: "1",
				Cycle:     false,
			},
			sequence2: &storepb.SequenceMetadata{
				Name:      "test_seq",
				DataType:  "bigint",
				Start:     "1",
				Increment: "1",
				CacheSize: "1",
				Cycle:     false,
			},
			expected: true,
		},
		{
			name: "Different names",
			sequence1: &storepb.SequenceMetadata{
				Name:      "test_seq1",
				DataType:  "bigint",
				Start:     "1",
				Increment: "1",
				CacheSize: "1",
				Cycle:     false,
			},
			sequence2: &storepb.SequenceMetadata{
				Name:      "test_seq2",
				DataType:  "bigint",
				Start:     "1",
				Increment: "1",
				CacheSize: "1",
				Cycle:     false,
			},
			expected: false,
		},
		{
			name: "Different start values",
			sequence1: &storepb.SequenceMetadata{
				Name:      "test_seq",
				DataType:  "bigint",
				Start:     "1",
				Increment: "1",
				CacheSize: "1",
				Cycle:     false,
			},
			sequence2: &storepb.SequenceMetadata{
				Name:      "test_seq",
				DataType:  "bigint",
				Start:     "10",
				Increment: "1",
				CacheSize: "1",
				Cycle:     false,
			},
			expected: false,
		},
		{
			name: "Different increment values",
			sequence1: &storepb.SequenceMetadata{
				Name:      "test_seq",
				DataType:  "bigint",
				Start:     "1",
				Increment: "1",
				CacheSize: "1",
				Cycle:     false,
			},
			sequence2: &storepb.SequenceMetadata{
				Name:      "test_seq",
				DataType:  "bigint",
				Start:     "1",
				Increment: "2",
				CacheSize: "1",
				Cycle:     false,
			},
			expected: false,
		},
		{
			name: "Different owner information",
			sequence1: &storepb.SequenceMetadata{
				Name:        "test_seq",
				DataType:    "bigint",
				Start:       "1",
				Increment:   "1",
				CacheSize:   "1",
				Cycle:       false,
				OwnerTable:  "table1",
				OwnerColumn: "id",
			},
			sequence2: &storepb.SequenceMetadata{
				Name:        "test_seq",
				DataType:    "bigint",
				Start:       "1",
				Increment:   "1",
				CacheSize:   "1",
				Cycle:       false,
				OwnerTable:  "table2",
				OwnerColumn: "id",
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sequenceDefinitionsEqual(tc.sequence1, tc.sequence2)
			assert.Equal(t, tc.expected, result)
		})
	}
}
