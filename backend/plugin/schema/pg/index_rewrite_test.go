package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestIndexRewriteOperations tests the ANTLR TokenStreamRewriter functionality
// for standalone CREATE INDEX operations including add, modify, delete
func TestIndexRewriteOperations(t *testing.T) {
	testCases := []struct {
		name               string
		originalSDL        string
		currentIndexes     []*extendedIndexMetadata
		previousIndexes    []*extendedIndexMetadata
		expectedIndexCount int
		description        string
	}{
		{
			name: "Add new standalone index",
			originalSDL: `CREATE TABLE users (
    id INTEGER NOT NULL,
    email VARCHAR(255) NOT NULL
);`,
			currentIndexes: []*extendedIndexMetadata{
				{
					IndexMetadata: &storepb.IndexMetadata{
						Name:        "idx_users_email",
						Expressions: []string{"email"},
						Type:        "btree",
						Unique:      false,
					},
					SchemaName: "public",
					TableName:  "users",
				},
			},
			previousIndexes:    []*extendedIndexMetadata{},
			expectedIndexCount: 1,
			description:        "Should add new standalone index to chunks",
		},
		{
			name: "Add unique index",
			originalSDL: `CREATE TABLE products (
    id INTEGER NOT NULL,
    code VARCHAR(50) NOT NULL
);`,
			currentIndexes: []*extendedIndexMetadata{
				{
					IndexMetadata: &storepb.IndexMetadata{
						Name:        "idx_products_code_unique",
						Expressions: []string{"code"},
						Type:        "btree",
						Unique:      true,
					},
					SchemaName: "public",
					TableName:  "products",
				},
			},
			previousIndexes:    []*extendedIndexMetadata{},
			expectedIndexCount: 1,
			description:        "Should add new unique index to chunks",
		},
		{
			name: "Drop standalone index",
			originalSDL: `CREATE TABLE users (
    id INTEGER NOT NULL,
    email VARCHAR(255) NOT NULL
);

CREATE INDEX idx_users_email ON users(email);`,
			currentIndexes: []*extendedIndexMetadata{},
			previousIndexes: []*extendedIndexMetadata{
				{
					IndexMetadata: &storepb.IndexMetadata{
						Name:        "idx_users_email",
						Expressions: []string{"email"},
						Type:        "btree",
						Unique:      false,
					},
					SchemaName: "public",
					TableName:  "users",
				},
			},
			expectedIndexCount: 0,
			description:        "Should remove standalone index from chunks",
		},
		{
			name: "Modify index (change type)",
			originalSDL: `CREATE TABLE documents (
    id INTEGER NOT NULL,
    content TEXT
);

CREATE INDEX idx_documents_content ON documents(content);`,
			currentIndexes: []*extendedIndexMetadata{
				{
					IndexMetadata: &storepb.IndexMetadata{
						Name:        "idx_documents_content",
						Expressions: []string{"content"},
						Type:        "gin",
						Unique:      false,
					},
					SchemaName: "public",
					TableName:  "documents",
				},
			},
			previousIndexes: []*extendedIndexMetadata{
				{
					IndexMetadata: &storepb.IndexMetadata{
						Name:        "idx_documents_content",
						Expressions: []string{"content"},
						Type:        "btree",
						Unique:      false,
					},
					SchemaName: "public",
					TableName:  "documents",
				},
			},
			expectedIndexCount: 1,
			description:        "Should modify existing index when type changes",
		},
		{
			name: "Multiple index operations",
			originalSDL: `CREATE TABLE orders (
    id INTEGER NOT NULL,
    customer_id INTEGER,
    total DECIMAL(10,2),
    status VARCHAR(20)
);

CREATE INDEX idx_orders_customer ON orders(customer_id);`,
			currentIndexes: []*extendedIndexMetadata{
				{
					IndexMetadata: &storepb.IndexMetadata{
						Name:        "idx_orders_total",
						Expressions: []string{"total"},
						Type:        "btree",
						Unique:      false,
					},
					SchemaName: "public",
					TableName:  "orders",
				},
				{
					IndexMetadata: &storepb.IndexMetadata{
						Name:        "idx_orders_status",
						Expressions: []string{"status"},
						Type:        "btree",
						Unique:      true,
					},
					SchemaName: "public",
					TableName:  "orders",
				},
			},
			previousIndexes: []*extendedIndexMetadata{
				{
					IndexMetadata: &storepb.IndexMetadata{
						Name:        "idx_orders_customer",
						Expressions: []string{"customer_id"},
						Type:        "btree",
						Unique:      false,
					},
					SchemaName: "public",
					TableName:  "orders",
				},
			},
			expectedIndexCount: 2,
			description:        "Should handle multiple index operations (drop old, add new)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the original SDL to create chunks
			chunks, err := ChunkSDLText(tc.originalSDL)
			require.NoError(t, err, "Failed to parse original SDL")

			// Create index maps for simulation
			currentIndexes := make(map[string]*extendedIndexMetadata)
			previousIndexes := make(map[string]*extendedIndexMetadata)

			for _, idx := range tc.currentIndexes {
				key := formatIndexKey(idx.SchemaName, idx.Name)
				currentIndexes[key] = idx
			}

			for _, idx := range tc.previousIndexes {
				key := formatIndexKey(idx.SchemaName, idx.Name)
				previousIndexes[key] = idx
			}

			// Apply index changes directly to test the core logic
			err = applyStandaloneIndexChangesInternal(chunks, currentIndexes, previousIndexes)
			require.NoError(t, err, "Failed to apply index changes")

			// Verify the expected number of indexes
			assert.Len(t, chunks.Indexes, tc.expectedIndexCount,
				"Expected %d indexes, got %d", tc.expectedIndexCount, len(chunks.Indexes))

			// Verify each current index is present
			for key, extIdx := range currentIndexes {
				chunk, exists := chunks.Indexes[key]
				assert.True(t, exists, "Expected index %s not found in chunks", key)
				if exists {
					assert.NotNil(t, chunk.ASTNode, "Index chunk should have AST node")
					// Verify we can get text from the chunk
					text := chunk.GetText()
					assert.NotEmpty(t, text, "Index chunk should have non-empty text")
					assert.Contains(t, text, "CREATE", "Index text should contain CREATE")
					assert.Contains(t, text, "INDEX", "Index text should contain INDEX")
					assert.Contains(t, text, extIdx.Name, "Index text should contain index name")
				}
			}

			// Verify previous indexes that shouldn't exist are removed
			for key := range previousIndexes {
				if _, stillExists := currentIndexes[key]; !stillExists {
					_, exists := chunks.Indexes[key]
					assert.False(t, exists, "Index %s should have been removed from chunks", key)
				}
			}

			t.Logf("Test case '%s' passed: %s", tc.name, tc.description)
		})
	}
}

// applyStandaloneIndexChangesInternal is a test helper that directly applies index changes to chunks
func applyStandaloneIndexChangesInternal(chunks *schema.SDLChunks, currentIndexes, previousIndexes map[string]*extendedIndexMetadata) error {
	if chunks == nil {
		return nil
	}

	// Process index additions: create new index chunks
	for indexKey, currentIndex := range currentIndexes {
		if _, exists := previousIndexes[indexKey]; !exists {
			// New index - create a chunk for it
			err := createIndexChunk(chunks, currentIndex, indexKey)
			if err != nil {
				return err
			}
		}
	}

	// Process index modifications: update existing chunks
	for indexKey, currentIndex := range currentIndexes {
		if previousIndex, exists := previousIndexes[indexKey]; exists {
			// Index exists in both - check if it needs modification
			err := updateIndexChunkIfNeeded(chunks, currentIndex, previousIndex, indexKey)
			if err != nil {
				return err
			}
		}
	}

	// Process index deletions: remove dropped index chunks
	for indexKey := range previousIndexes {
		if _, exists := currentIndexes[indexKey]; !exists {
			// Index was dropped - remove it from chunks
			deleteIndexChunk(chunks, indexKey)
		}
	}

	return nil
}

// TestGenerateCreateIndexSDL tests the SDL generation for CREATE INDEX statements
func TestGenerateCreateIndexSDL(t *testing.T) {
	testCases := []struct {
		name        string
		extIndex    *extendedIndexMetadata
		expectedSDL string
	}{
		{
			name: "Simple btree index",
			extIndex: &extendedIndexMetadata{
				IndexMetadata: &storepb.IndexMetadata{
					Name:        "idx_users_email",
					Expressions: []string{"email"},
					Type:        "btree",
					Unique:      false,
				},
				SchemaName: "public",
				TableName:  "users",
			},
			expectedSDL: `CREATE INDEX "idx_users_email" ON ONLY "public"."users" (email)`,
		},
		{
			name: "Unique index",
			extIndex: &extendedIndexMetadata{
				IndexMetadata: &storepb.IndexMetadata{
					Name:        "idx_products_code",
					Expressions: []string{"code"},
					Type:        "btree",
					Unique:      true,
				},
				SchemaName: "public",
				TableName:  "products",
			},
			expectedSDL: `CREATE UNIQUE INDEX "idx_products_code" ON ONLY "public"."products" (code)`,
		},
		{
			name: "GIN index",
			extIndex: &extendedIndexMetadata{
				IndexMetadata: &storepb.IndexMetadata{
					Name:        "idx_documents_content",
					Expressions: []string{"content"},
					Type:        "gin",
					Unique:      false,
				},
				SchemaName: "public",
				TableName:  "documents",
			},
			expectedSDL: `CREATE INDEX "idx_documents_content" ON ONLY "public"."documents" USING GIN (content)`,
		},
		{
			name: "Multi-column index with DESC",
			extIndex: &extendedIndexMetadata{
				IndexMetadata: &storepb.IndexMetadata{
					Name:        "idx_orders_complex",
					Expressions: []string{"customer_id", "created_at", "total"},
					Type:        "btree",
					Unique:      false,
					Descending:  []bool{false, true, false},
				},
				SchemaName: "public",
				TableName:  "orders",
			},
			expectedSDL: `CREATE INDEX "idx_orders_complex" ON ONLY "public"."orders" (customer_id, created_at DESC, total)`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := generateCreateIndexSDL(tc.extIndex)
			assert.Equal(t, tc.expectedSDL, result)
		})
	}
}

// TestIndexDefinitionsEqual tests the index comparison logic
func TestIndexDefinitionsEqual(t *testing.T) {
	testCases := []struct {
		name     string
		index1   *storepb.IndexMetadata
		index2   *storepb.IndexMetadata
		expected bool
	}{
		{
			name: "Identical indexes",
			index1: &storepb.IndexMetadata{
				Name:        "idx_test",
				Expressions: []string{"col1"},
				Type:        "btree",
				Unique:      false,
			},
			index2: &storepb.IndexMetadata{
				Name:        "idx_test",
				Expressions: []string{"col1"},
				Type:        "btree",
				Unique:      false,
			},
			expected: true,
		},
		{
			name: "Different names",
			index1: &storepb.IndexMetadata{
				Name:        "idx_test1",
				Expressions: []string{"col1"},
				Type:        "btree",
				Unique:      false,
			},
			index2: &storepb.IndexMetadata{
				Name:        "idx_test2",
				Expressions: []string{"col1"},
				Type:        "btree",
				Unique:      false,
			},
			expected: false,
		},
		{
			name: "Different types",
			index1: &storepb.IndexMetadata{
				Name:        "idx_test",
				Expressions: []string{"col1"},
				Type:        "btree",
				Unique:      false,
			},
			index2: &storepb.IndexMetadata{
				Name:        "idx_test",
				Expressions: []string{"col1"},
				Type:        "gin",
				Unique:      false,
			},
			expected: false,
		},
		{
			name: "Different uniqueness",
			index1: &storepb.IndexMetadata{
				Name:        "idx_test",
				Expressions: []string{"col1"},
				Type:        "btree",
				Unique:      false,
			},
			index2: &storepb.IndexMetadata{
				Name:        "idx_test",
				Expressions: []string{"col1"},
				Type:        "btree",
				Unique:      true,
			},
			expected: false,
		},
		{
			name: "Different expressions",
			index1: &storepb.IndexMetadata{
				Name:        "idx_test",
				Expressions: []string{"col1"},
				Type:        "btree",
				Unique:      false,
			},
			index2: &storepb.IndexMetadata{
				Name:        "idx_test",
				Expressions: []string{"col2"},
				Type:        "btree",
				Unique:      false,
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := indexDefinitionsEqual(tc.index1, tc.index2)
			assert.Equal(t, tc.expected, result)
		})
	}
}
