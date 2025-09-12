package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestTableConstraintsSDLDiff tests all table constraint types (PK, UK, FK, Check)
func TestTableConstraintsSDLDiff(t *testing.T) {
	testCases := []struct {
		name        string
		currentSDL  string
		previousSDL string
		validate    func(t *testing.T, diff *schema.MetadataDiff)
	}{
		{
			name: "Add primary key constraint",
			currentSDL: `CREATE TABLE users (
				id INTEGER,
				name VARCHAR(255),
				CONSTRAINT pk_users PRIMARY KEY (id)
			);`,
			previousSDL: `CREATE TABLE users (
				id INTEGER,
				name VARCHAR(255)
			);`,
			validate: func(t *testing.T, diff *schema.MetadataDiff) {
				assert.Len(t, diff.TableChanges, 1)
				tableDiff := diff.TableChanges[0]
				assert.Equal(t, schema.MetadataDiffActionAlter, tableDiff.Action)
				assert.Len(t, tableDiff.IndexChanges, 1)
				indexChange := tableDiff.IndexChanges[0]
				assert.Equal(t, schema.MetadataDiffActionCreate, indexChange.Action)
				// AST node should be present for created primary key
				assert.NotNil(t, indexChange.NewASTNode)
				assert.Nil(t, indexChange.OldASTNode)
			},
		},
		{
			name: "Drop foreign key constraint",
			currentSDL: `CREATE TABLE orders (
				id INTEGER PRIMARY KEY,
				customer_id INTEGER
			);`,
			previousSDL: `CREATE TABLE orders (
				id INTEGER PRIMARY KEY,
				customer_id INTEGER,
				CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id) REFERENCES customers(id)
			);`,
			validate: func(t *testing.T, diff *schema.MetadataDiff) {
				assert.Len(t, diff.TableChanges, 1)
				tableDiff := diff.TableChanges[0]
				assert.Equal(t, schema.MetadataDiffActionAlter, tableDiff.Action)
				assert.Len(t, tableDiff.ForeignKeyChanges, 1)
				fkChange := tableDiff.ForeignKeyChanges[0]
				assert.Equal(t, schema.MetadataDiffActionDrop, fkChange.Action)
				// AST node should be present for dropped FK
				assert.NotNil(t, fkChange.OldASTNode)
				assert.Nil(t, fkChange.NewASTNode)
			},
		},
		{
			name: "Modify check constraint",
			currentSDL: `CREATE TABLE products (
				id INTEGER PRIMARY KEY,
				price DECIMAL(10,2),
				CONSTRAINT chk_price CHECK (price >= 0)
			);`,
			previousSDL: `CREATE TABLE products (
				id INTEGER PRIMARY KEY,
				price DECIMAL(10,2),
				CONSTRAINT chk_price CHECK (price > 0)
			);`,
			validate: func(t *testing.T, diff *schema.MetadataDiff) {
				assert.Len(t, diff.TableChanges, 1)
				tableDiff := diff.TableChanges[0]
				assert.Equal(t, schema.MetadataDiffActionAlter, tableDiff.Action)
				// Should drop old and create new for modifications
				assert.Len(t, tableDiff.CheckConstraintChanges, 2)

				// Find drop and create actions
				var dropChange, createChange *schema.CheckConstraintDiff
				for _, change := range tableDiff.CheckConstraintChanges {
					switch change.Action {
					case schema.MetadataDiffActionDrop:
						dropChange = change
					case schema.MetadataDiffActionCreate:
						createChange = change
					default:
						// Other actions not expected in this test
					}
				}

				assert.NotNil(t, dropChange)
				assert.NotNil(t, createChange)
				// AST nodes should be present for check constraint changes
				assert.NotNil(t, dropChange.OldASTNode)
				assert.Nil(t, dropChange.NewASTNode)
				assert.NotNil(t, createChange.NewASTNode)
				assert.Nil(t, createChange.OldASTNode)
			},
		},
		{
			name: "Multiple constraint types in one table",
			currentSDL: `CREATE TABLE orders (
				id INTEGER,
				customer_id INTEGER,
				order_date DATE,
				amount DECIMAL(10,2),
				status VARCHAR(20),
				CONSTRAINT pk_orders PRIMARY KEY (id),
				CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id) REFERENCES customers(id),
				CONSTRAINT uk_orders_date_customer UNIQUE (order_date, customer_id),
				CONSTRAINT chk_positive_amount CHECK (amount > 0),
				CONSTRAINT chk_valid_status CHECK (status IN ('pending', 'completed', 'cancelled'))
			);`,
			previousSDL: `CREATE TABLE orders (
				id INTEGER,
				customer_id INTEGER,
				order_date DATE,
				amount DECIMAL(10,2),
				status VARCHAR(20)
			);`,
			validate: func(t *testing.T, diff *schema.MetadataDiff) {
				assert.Len(t, diff.TableChanges, 1)
				tableDiff := diff.TableChanges[0]
				assert.Equal(t, schema.MetadataDiffActionAlter, tableDiff.Action)

				// Should have 2 index changes (PK and UNIQUE)
				assert.Len(t, tableDiff.IndexChanges, 2)

				// Should have 1 foreign key change
				assert.Len(t, tableDiff.ForeignKeyChanges, 1)
				fkChange := tableDiff.ForeignKeyChanges[0]
				assert.Equal(t, schema.MetadataDiffActionCreate, fkChange.Action)
				// AST node should be present for created FK
				assert.NotNil(t, fkChange.NewASTNode)
				assert.Nil(t, fkChange.OldASTNode)

				// Should have 2 check constraint changes
				assert.Len(t, tableDiff.CheckConstraintChanges, 2)
				for _, checkChange := range tableDiff.CheckConstraintChanges {
					assert.Equal(t, schema.MetadataDiffActionCreate, checkChange.Action)
					// AST node should be present for created check constraints
					assert.NotNil(t, checkChange.NewASTNode)
					assert.Nil(t, checkChange.OldASTNode)
				}
			},
		},
		{
			name: "Complex foreign key with schema qualification",
			currentSDL: `CREATE TABLE orders (
				id INTEGER PRIMARY KEY,
				customer_id INTEGER,
				CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id) REFERENCES public.customers(id)
			);`,
			previousSDL: `CREATE TABLE orders (
				id INTEGER PRIMARY KEY,
				customer_id INTEGER
			);`,
			validate: func(t *testing.T, diff *schema.MetadataDiff) {
				assert.Len(t, diff.TableChanges, 1)
				tableDiff := diff.TableChanges[0]
				assert.Len(t, tableDiff.ForeignKeyChanges, 1)
				fkChange := tableDiff.ForeignKeyChanges[0]
				assert.Equal(t, schema.MetadataDiffActionCreate, fkChange.Action)
				// AST node should be present for created FK with schema qualification
				assert.NotNil(t, fkChange.NewASTNode)
				assert.Nil(t, fkChange.OldASTNode)
			},
		},
		{
			name: "Composite unique constraint",
			currentSDL: `CREATE TABLE user_sessions (
				user_id INTEGER,
				session_token VARCHAR(255),
				created_at TIMESTAMP,
				CONSTRAINT uk_user_sessions UNIQUE (user_id, session_token)
			);`,
			previousSDL: `CREATE TABLE user_sessions (
				user_id INTEGER,
				session_token VARCHAR(255),
				created_at TIMESTAMP
			);`,
			validate: func(t *testing.T, diff *schema.MetadataDiff) {
				assert.Len(t, diff.TableChanges, 1)
				tableDiff := diff.TableChanges[0]
				assert.Len(t, tableDiff.IndexChanges, 1)
				indexChange := tableDiff.IndexChanges[0]
				assert.Equal(t, schema.MetadataDiffActionCreate, indexChange.Action)
				// AST node should be present for created unique constraint
				assert.NotNil(t, indexChange.NewASTNode)
				assert.Nil(t, indexChange.OldASTNode)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tc.currentSDL, tc.previousSDL, nil, nil)
			require.NoError(t, err, "GetSDLDiff should not return error")
			tc.validate(t, diff)
		})
	}
}
