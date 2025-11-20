package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestExcludeConstraintWithRegularIndexes tests that regular CREATE INDEX statements
// are properly executed alongside EXCLUDE constraints
func TestExcludeConstraintWithRegularIndexes(t *testing.T) {
	// Initial state: empty database
	previousSDL := ""

	// Current SDL: Table with EXCLUDE constraint and regular indexes
	currentSDL := `
-- Table with EXCLUDE constraint for employee shift assignments
CREATE TABLE "public"."employee_shifts" (
    "shift_id" bigserial,
    "employee_id" integer NOT NULL,
    "department_id" integer NOT NULL,
    "shift_start" timestamp(6) with time zone NOT NULL,
    "shift_end" timestamp(6) with time zone NOT NULL,
    "shift_type" character varying(50) DEFAULT 'regular',
    "created_at" timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "pk_employee_shifts" PRIMARY KEY (shift_id),
    CONSTRAINT "chk_employee_shifts_time" CHECK (shift_end > shift_start),
    -- EXCLUDE constraint: same employee cannot have overlapping shifts
    CONSTRAINT "excl_employee_no_overlap" EXCLUDE USING gist (
        employee_id WITH =,
        tstzrange(shift_start, shift_end) WITH &&
    )
);

CREATE INDEX idx_employee_shifts_employee ON "public"."employee_shifts" (employee_id);
CREATE INDEX idx_employee_shifts_department ON "public"."employee_shifts" (department_id);
CREATE INDEX idx_employee_shifts_time ON "public"."employee_shifts" USING gist (tstzrange(shift_start, shift_end));
CREATE INDEX idx_employee_shifts_type ON "public"."employee_shifts" (shift_type);
`

	previousDBMetadata := &storepb.DatabaseSchemaMetadata{
		Name:    "testdb",
		Schemas: []*storepb.SchemaMetadata{},
	}

	currentDBMetadata := &storepb.DatabaseSchemaMetadata{
		Name:    "testdb",
		Schemas: []*storepb.SchemaMetadata{},
	}

	previousSchema := model.NewDatabaseMetadata(previousDBMetadata, nil, nil, storepb.Engine_POSTGRES, false)
	currentSchema := model.NewDatabaseMetadata(currentDBMetadata, nil, nil, storepb.Engine_POSTGRES, false)

	diff, err := GetSDLDiff(currentSDL, previousSDL, currentSchema, previousSchema)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Debug: Check what's in the diff
	t.Logf("Number of table changes: %d", len(diff.TableChanges))
	for i, tc := range diff.TableChanges {
		t.Logf("Table %d: %s.%s - IndexChanges: %d", i, tc.SchemaName, tc.TableName, len(tc.IndexChanges))
		for j, ic := range tc.IndexChanges {
			t.Logf("  Index change %d: Action=%v, HasNewAST=%v, HasOldAST=%v",
				j, ic.Action, ic.NewASTNode != nil, ic.OldASTNode != nil)
		}
	}

	// Generate migration DDL
	ddl, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated DDL:\n%s", ddl)

	// Verify the migration includes:
	// 1. CREATE TABLE statement
	require.Contains(t, ddl, "CREATE TABLE", "Should create the table")
	require.Contains(t, ddl, "employee_shifts", "Should create employee_shifts table")

	// 2. EXCLUDE constraint in the CREATE TABLE
	require.Contains(t, ddl, "EXCLUDE USING gist", "Should include EXCLUDE constraint")
	require.Contains(t, ddl, "excl_employee_no_overlap", "Should include constraint name")

	// 3. Regular CREATE INDEX statements (THIS IS THE BUG - these might be missing)
	require.Contains(t, ddl, "CREATE INDEX idx_employee_shifts_employee", "Should create idx_employee_shifts_employee index")
	require.Contains(t, ddl, "CREATE INDEX idx_employee_shifts_department", "Should create idx_employee_shifts_department index")
	require.Contains(t, ddl, "CREATE INDEX idx_employee_shifts_time", "Should create idx_employee_shifts_time index")
	require.Contains(t, ddl, "CREATE INDEX idx_employee_shifts_type", "Should create idx_employee_shifts_type index")

	// 4. Should NOT create an index for the EXCLUDE constraint (it's auto-created)
	// The EXCLUDE constraint will create an index, but we shouldn't output a separate CREATE INDEX for it
}

// TestExcludeConstraintIndexNotDuplicated tests that EXCLUDE constraint indexes
// are not output as separate CREATE INDEX statements
func TestExcludeConstraintIndexNotDuplicated(t *testing.T) {
	previousSDL := ""

	currentSDL := `
CREATE TABLE "public"."meetings" (
    "meeting_id" serial PRIMARY KEY,
    "room_id" integer NOT NULL,
    "meeting_start" timestamp NOT NULL,
    "meeting_end" timestamp NOT NULL,
    CONSTRAINT "excl_room_no_overlap" EXCLUDE USING gist (
        room_id WITH =,
        tsrange(meeting_start, meeting_end, '[)') WITH &&
    )
);
`

	// Simulate database state where EXCLUDE constraint has created an index
	currentDBMetadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "meetings",
						Columns: []*storepb.ColumnMetadata{
							{Name: "meeting_id", Type: "integer"},
							{Name: "room_id", Type: "integer"},
							{Name: "meeting_start", Type: "timestamp without time zone"},
							{Name: "meeting_end", Type: "timestamp without time zone"},
						},
						ExcludeConstraints: []*storepb.ExcludeConstraintMetadata{
							{
								Name:       "excl_room_no_overlap",
								Expression: "EXCLUDE USING gist (room_id WITH =, tsrange(meeting_start, meeting_end, '[)') WITH &&)",
							},
						},
						Indexes: []*storepb.IndexMetadata{
							{
								Name: "excl_room_no_overlap",
								Expressions: []string{
									"room_id",
									"tsrange(meeting_start, meeting_end, '[)')",
								},
								Type:         "gist",
								Unique:       false,
								Primary:      false,
								IsConstraint: true, // This index is created by EXCLUDE constraint
							},
						},
					},
				},
			},
		},
	}

	previousDBMetadata := &storepb.DatabaseSchemaMetadata{
		Name:    "testdb",
		Schemas: []*storepb.SchemaMetadata{},
	}

	previousSchema := model.NewDatabaseMetadata(previousDBMetadata, nil, nil, storepb.Engine_POSTGRES, false)
	currentSchema := model.NewDatabaseMetadata(currentDBMetadata, nil, nil, storepb.Engine_POSTGRES, false)

	diff, err := GetSDLDiff(currentSDL, previousSDL, currentSchema, previousSchema)
	require.NoError(t, err)

	ddl, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated DDL:\n%s", ddl)

	// Should include EXCLUDE constraint
	require.Contains(t, ddl, "EXCLUDE USING gist", "Should include EXCLUDE constraint")

	// Should NOT include a separate CREATE INDEX for the constraint index
	// The index is automatically created by PostgreSQL when creating the EXCLUDE constraint
	require.NotContains(t, ddl, "CREATE INDEX excl_room_no_overlap", "Should NOT create separate index for EXCLUDE constraint")
}

// TestMaterializedViewWithRegularIndexes tests that regular CREATE INDEX statements
// are properly executed alongside materialized view creation
func TestMaterializedViewWithRegularIndexes(t *testing.T) {
	// Initial state: empty database
	previousSDL := ""

	// Current SDL: Materialized view with regular indexes
	currentSDL := `
CREATE TABLE "public"."orders" (
    "order_id" serial PRIMARY KEY,
    "customer_id" integer NOT NULL,
    "order_date" date NOT NULL,
    "total_amount" numeric(10,2) NOT NULL
);

CREATE MATERIALIZED VIEW "public"."order_summary" AS
SELECT
    customer_id,
    DATE_TRUNC('month', order_date) as order_month,
    COUNT(*) as order_count,
    SUM(total_amount) as total_sales
FROM "public"."orders"
GROUP BY customer_id, DATE_TRUNC('month', order_date);

CREATE INDEX idx_order_summary_customer ON "public"."order_summary" (customer_id);
CREATE INDEX idx_order_summary_month ON "public"."order_summary" (order_month);
CREATE UNIQUE INDEX idx_order_summary_unique ON "public"."order_summary" (customer_id, order_month);
`

	previousDBMetadata := &storepb.DatabaseSchemaMetadata{
		Name:    "testdb",
		Schemas: []*storepb.SchemaMetadata{},
	}

	currentDBMetadata := &storepb.DatabaseSchemaMetadata{
		Name:    "testdb",
		Schemas: []*storepb.SchemaMetadata{},
	}

	previousSchema := model.NewDatabaseMetadata(previousDBMetadata, nil, nil, storepb.Engine_POSTGRES, false)
	currentSchema := model.NewDatabaseMetadata(currentDBMetadata, nil, nil, storepb.Engine_POSTGRES, false)

	diff, err := GetSDLDiff(currentSDL, previousSDL, currentSchema, previousSchema)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Debug: Check what's in the diff
	t.Logf("Number of materialized view changes: %d", len(diff.MaterializedViewChanges))
	for i, mvc := range diff.MaterializedViewChanges {
		t.Logf("MV %d: %s.%s - Action=%v, IndexChanges: %d", i, mvc.SchemaName, mvc.MaterializedViewName, mvc.Action, len(mvc.IndexChanges))
		for j, ic := range mvc.IndexChanges {
			t.Logf("  Index change %d: Action=%v, HasNewAST=%v, HasOldAST=%v",
				j, ic.Action, ic.NewASTNode != nil, ic.OldASTNode != nil)
		}
	}

	// Generate migration DDL
	ddl, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated DDL:\n%s", ddl)

	// Verify the migration includes:
	// 1. CREATE TABLE statement
	require.Contains(t, ddl, "CREATE TABLE", "Should create the orders table")
	require.Contains(t, ddl, "orders", "Should create orders table")

	// 2. CREATE MATERIALIZED VIEW statement
	require.Contains(t, ddl, "CREATE MATERIALIZED VIEW", "Should create materialized view")
	require.Contains(t, ddl, "order_summary", "Should create order_summary materialized view")

	// 3. Regular CREATE INDEX statements (THIS IS THE BUG - these might be missing)
	require.Contains(t, ddl, "CREATE INDEX idx_order_summary_customer", "Should create idx_order_summary_customer index")
	require.Contains(t, ddl, "CREATE INDEX idx_order_summary_month", "Should create idx_order_summary_month index")
	require.Contains(t, ddl, "CREATE UNIQUE INDEX idx_order_summary_unique", "Should create idx_order_summary_unique unique index")
}

// TestIndexCreatedBeforeForeignKey tests that indexes are created before foreign keys that might reference them
// This is important because FK performance benefits from indexes on referenced columns
func TestIndexCreatedBeforeForeignKey(t *testing.T) {
	// Initial state: empty database
	previousSDL := ""

	// Current SDL: Two tables where orders.customer_id references customers.id
	// customers.id has an index that should be created before the FK
	currentSDL := `
CREATE TABLE "public"."customers" (
    "id" serial PRIMARY KEY,
    "email" varchar(255) NOT NULL,
    "name" varchar(100) NOT NULL
);

CREATE INDEX idx_customers_id ON "public"."customers" (id);
CREATE INDEX idx_customers_email ON "public"."customers" (email);

CREATE TABLE "public"."orders" (
    "id" serial PRIMARY KEY,
    "customer_id" integer NOT NULL,
    "order_date" date NOT NULL,
    "total_amount" numeric(10,2) NOT NULL,
    CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id) REFERENCES "public"."customers" (id)
);

CREATE INDEX idx_orders_customer ON "public"."orders" (customer_id);
`

	previousDBMetadata := &storepb.DatabaseSchemaMetadata{
		Name:    "testdb",
		Schemas: []*storepb.SchemaMetadata{},
	}

	currentDBMetadata := &storepb.DatabaseSchemaMetadata{
		Name:    "testdb",
		Schemas: []*storepb.SchemaMetadata{},
	}

	previousSchema := model.NewDatabaseMetadata(previousDBMetadata, nil, nil, storepb.Engine_POSTGRES, false)
	currentSchema := model.NewDatabaseMetadata(currentDBMetadata, nil, nil, storepb.Engine_POSTGRES, false)

	diff, err := GetSDLDiff(currentSDL, previousSDL, currentSchema, previousSchema)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Generate migration DDL
	ddl, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated DDL:\n%s", ddl)

	// Verify the migration includes all necessary statements
	require.Contains(t, ddl, "CREATE TABLE", "Should create tables")
	require.Contains(t, ddl, "customers", "Should create customers table")
	require.Contains(t, ddl, "orders", "Should create orders table")

	// Verify indexes are created
	require.Contains(t, ddl, "CREATE INDEX idx_customers_id", "Should create idx_customers_id index")
	require.Contains(t, ddl, "CREATE INDEX idx_customers_email", "Should create idx_customers_email index")
	require.Contains(t, ddl, "CREATE INDEX idx_orders_customer", "Should create idx_orders_customer index")

	// Verify FK is created
	require.Contains(t, ddl, "fk_orders_customer", "Should create foreign key")

	// CRITICAL: Verify that indexes on customers table are created BEFORE the foreign key
	// This is important for FK performance
	idxCustomersPos := strings.Index(ddl, "CREATE INDEX idx_customers_id")
	fkPos := strings.Index(ddl, "fk_orders_customer")

	require.NotEqual(t, -1, idxCustomersPos, "idx_customers_id index should exist")
	require.NotEqual(t, -1, fkPos, "fk_orders_customer should exist")
	require.Less(t, idxCustomersPos, fkPos,
		"Index on referenced column (customers.id) should be created BEFORE the foreign key that references it")
}
