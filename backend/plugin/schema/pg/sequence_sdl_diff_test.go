package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestSequenceSDLDiff(t *testing.T) {
	tests := []struct {
		name                    string
		previousSDL             string
		currentSDL              string
		expectedSequenceChanges int
		expectedActions         []schema.MetadataDiffAction
	}{
		{
			name:        "Create new sequence",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1
					NO MINVALUE
					NO MAXVALUE
					CACHE 1;
			`,
			expectedSequenceChanges: 1,
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
		{
			name: "Drop sequence",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1
					NO MINVALUE
					NO MAXVALUE
					CACHE 1;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			expectedSequenceChanges: 1,
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionDrop},
		},
		{
			name: "Modify sequence (drop and recreate)",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1
					NO MINVALUE
					NO MAXVALUE
					CACHE 1;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE SEQUENCE user_seq
					START WITH 10
					INCREMENT BY 2
					NO MINVALUE
					NO MAXVALUE
					CACHE 5;
			`,
			expectedSequenceChanges: 2, // Drop + Create
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionDrop, schema.MetadataDiffActionCreate},
		},
		{
			name: "No changes to sequence",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1
					NO MINVALUE
					NO MAXVALUE
					CACHE 1;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1
					NO MINVALUE
					NO MAXVALUE
					CACHE 1;
			`,
			expectedSequenceChanges: 0,
			expectedActions:         []schema.MetadataDiffAction{},
		},
		{
			name: "Multiple sequences with different changes",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1;
				
				CREATE SEQUENCE order_seq
					START WITH 100
					INCREMENT BY 1;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1;
				
				CREATE SEQUENCE product_seq
					START WITH 1000
					INCREMENT BY 10;
			`,
			expectedSequenceChanges: 2, // Drop order_seq + Create product_seq
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionCreate, schema.MetadataDiffActionDrop},
		},
		{
			name: "Schema-qualified sequence names",
			previousSDL: `
				CREATE SCHEMA test_schema;
				CREATE TABLE test_schema.products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);
			`,
			currentSDL: `
				CREATE SCHEMA test_schema;
				CREATE TABLE test_schema.products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);
				
				CREATE SEQUENCE test_schema.product_id_seq
					START WITH 1
					INCREMENT BY 1
					OWNED BY test_schema.products.id;
			`,
			expectedSequenceChanges: 1,
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
		{
			name: "Sequence with complex options",
			previousSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					amount DECIMAL(10,2)
				);
			`,
			currentSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					amount DECIMAL(10,2)
				);
				
				CREATE SEQUENCE order_id_seq
					AS BIGINT
					START WITH 1000000
					INCREMENT BY 1
					MINVALUE 1
					MAXVALUE 9223372036854775807
					CACHE 50
					NO CYCLE
					OWNED BY orders.id;
			`,
			expectedSequenceChanges: 1,
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
		{
			name: "SERIAL column changes (implicit sequence handling)",
			previousSDL: `
				CREATE TABLE users (
					name VARCHAR(255) NOT NULL
				);
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			expectedSequenceChanges: 0, // SERIAL sequences are handled at table level, not explicit sequence level
			expectedActions:         []schema.MetadataDiffAction{},
		},
		{
			name: "Add ALTER SEQUENCE OWNED BY to existing sequence",
			previousSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					order_number BIGINT NOT NULL
				);

				CREATE SEQUENCE custom_seq AS bigint START WITH 100 INCREMENT BY 5 MINVALUE 100 MAXVALUE 9223372036854775807 NO CYCLE CACHE 10;
			`,
			currentSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					order_number BIGINT NOT NULL
				);

				CREATE SEQUENCE custom_seq AS bigint START WITH 100 INCREMENT BY 5 MINVALUE 100 MAXVALUE 9223372036854775807 NO CYCLE CACHE 10;

				ALTER SEQUENCE custom_seq OWNED BY orders.order_number;
			`,
			expectedSequenceChanges: 1, // Only ALTER (ownership changed)
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionAlter},
		},
		{
			name: "Modify ALTER SEQUENCE OWNED BY (change owner)",
			previousSDL: `
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
			currentSDL: `
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
			expectedSequenceChanges: 1, // Only ALTER (ownership changed)
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionAlter},
		},
		{
			name: "Remove ALTER SEQUENCE OWNED BY",
			previousSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					order_number BIGINT NOT NULL
				);

				CREATE SEQUENCE custom_seq AS bigint START WITH 100 INCREMENT BY 5 NO CYCLE;

				ALTER SEQUENCE custom_seq OWNED BY orders.order_number;
			`,
			currentSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					order_number BIGINT NOT NULL
				);

				CREATE SEQUENCE custom_seq AS bigint START WITH 100 INCREMENT BY 5 NO CYCLE;
			`,
			expectedSequenceChanges: 1, // Only ALTER (ownership removed)
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionAlter},
		},
		{
			name: "No change when ALTER SEQUENCE OWNED BY is identical",
			previousSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					order_number BIGINT NOT NULL
				);

				CREATE SEQUENCE custom_seq AS bigint START WITH 100 INCREMENT BY 5 NO CYCLE;

				ALTER SEQUENCE custom_seq OWNED BY orders.order_number;
			`,
			currentSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					order_number BIGINT NOT NULL
				);

				CREATE SEQUENCE custom_seq AS bigint START WITH 100 INCREMENT BY 5 NO CYCLE;

				ALTER SEQUENCE custom_seq OWNED BY orders.order_number;
			`,
			expectedSequenceChanges: 0, // No changes
			expectedActions:         []schema.MetadataDiffAction{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			assert.Equal(t, tt.expectedSequenceChanges, len(diff.SequenceChanges),
				"Expected %d sequence changes, got %d", tt.expectedSequenceChanges, len(diff.SequenceChanges))

			// Check that the actions match expectations
			var actualActions []schema.MetadataDiffAction
			for _, seqDiff := range diff.SequenceChanges {
				actualActions = append(actualActions, seqDiff.Action)
			}

			// Handle nil vs empty slice comparison
			if len(tt.expectedActions) == 0 && len(actualActions) == 0 {
				// Both are effectively empty - test passes
				t.Log("Both expected and actual actions are empty")
			} else {
				assert.ElementsMatch(t, tt.expectedActions, actualActions,
					"Expected actions %v, got %v", tt.expectedActions, actualActions)
			}

			// Verify AST nodes are properly set
			for i, seqDiff := range diff.SequenceChanges {
				switch seqDiff.Action {
				case schema.MetadataDiffActionCreate:
					assert.NotNil(t, seqDiff.NewASTNode,
						"Sequence diff %d should have NewASTNode for CREATE action", i)
					assert.Nil(t, seqDiff.OldASTNode,
						"Sequence diff %d should not have OldASTNode for CREATE action", i)
				case schema.MetadataDiffActionDrop:
					assert.NotNil(t, seqDiff.OldASTNode,
						"Sequence diff %d should have OldASTNode for DROP action", i)
					assert.Nil(t, seqDiff.NewASTNode,
						"Sequence diff %d should not have NewASTNode for DROP action", i)
				case schema.MetadataDiffActionAlter:
					// ALTER action can have either NewASTNode (adding/modifying) or OldASTNode (removing)
					// but not both and not neither
					hasNew := seqDiff.NewASTNode != nil
					hasOld := seqDiff.OldASTNode != nil
					assert.True(t, hasNew != hasOld,
						"Sequence diff %d should have exactly one of NewASTNode or OldASTNode for ALTER action", i)
				default:
					t.Errorf("Unexpected action %v for sequence diff %d", seqDiff.Action, i)
				}
			}
		})
	}
}

func TestSequenceIdentifierParsing(t *testing.T) {
	tests := []struct {
		name             string
		identifier       string
		expectedSchema   string
		expectedSequence string
	}{
		{
			name:             "Simple sequence name",
			identifier:       "my_seq",
			expectedSchema:   "public",
			expectedSequence: "my_seq",
		},
		{
			name:             "Schema-qualified sequence name",
			identifier:       "test_schema.my_seq",
			expectedSchema:   "test_schema",
			expectedSequence: "my_seq",
		},
		{
			name:             "Public schema explicit",
			identifier:       "public.user_id_seq",
			expectedSchema:   "public",
			expectedSequence: "user_id_seq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, sequence := parseIdentifier(tt.identifier)
			assert.Equal(t, tt.expectedSchema, schema)
			assert.Equal(t, tt.expectedSequence, sequence)
		})
	}
}
