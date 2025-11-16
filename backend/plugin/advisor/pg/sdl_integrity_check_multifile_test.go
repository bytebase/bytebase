package pg

import (
	"strings"
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestCheckSDLIntegrity_ValidCrossFileReferences(t *testing.T) {
	files := map[string]string{
		"users.sql": `
			CREATE TABLE public.users (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL,
				email TEXT UNIQUE
			);
		`,
		"orders.sql": `
			CREATE TABLE public.orders (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL,
				total DECIMAL(10,2),
				CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES public.users(id)
			);
		`,
		"order_items.sql": `
			CREATE TABLE public.order_items (
				id SERIAL PRIMARY KEY,
				order_id INTEGER NOT NULL,
				product_name TEXT,
				FOREIGN KEY (order_id) REFERENCES public.orders(id)
			);
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have no errors - all references are valid
	for filePath, advices := range results {
		require.Empty(t, advices, "File %s should have no errors, got: %v", filePath, advices)
	}
}

func TestCheckSDLIntegrity_CrossFileViewDependencies(t *testing.T) {
	files := map[string]string{
		"tables.sql": `
			CREATE TABLE public.products (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL,
				price DECIMAL(10,2)
			);

			CREATE TABLE public.sales (
				id SERIAL PRIMARY KEY,
				product_id INTEGER REFERENCES public.products(id),
				quantity INTEGER,
				sale_date DATE
			);
		`,
		"views.sql": `
			CREATE VIEW public.product_sales AS
			SELECT p.name, s.quantity, s.sale_date
			FROM public.products p
			JOIN public.sales s ON p.id = s.product_id;
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have no errors - view references tables from other file
	for filePath, advices := range results {
		require.Empty(t, advices, "File %s should have no errors, got: %v", filePath, advices)
	}
}

func TestCheckSDLIntegrity_DuplicateTableAcrossFiles(t *testing.T) {
	files := map[string]string{
		"file1.sql": `
			CREATE TABLE public.users (
				id SERIAL PRIMARY KEY,
				name TEXT
			);
		`,
		"file2.sql": `
			CREATE TABLE public.users (
				id INTEGER PRIMARY KEY,
				email TEXT
			);
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have exactly one error total (in either file1.sql or file2.sql)
	totalErrors := len(results["file1.sql"]) + len(results["file2.sql"])
	require.Equal(t, 1, totalErrors, "Should have exactly 1 error total")

	// Find the error (it will be in one of the files)
	var advice *storepb.Advice
	if len(results["file1.sql"]) > 0 {
		advice = results["file1.sql"][0]
	} else {
		advice = results["file2.sql"][0]
	}

	require.Equal(t, storepb.Advice_ERROR, advice.Status)
	require.Equal(t, code.SDLDuplicateTableName.Int32(), advice.Code)
	require.Contains(t, advice.Content, "Table 'public.users' is defined in multiple SDL files")
}

func TestCheckSDLIntegrity_DuplicateIndexAcrossFiles(t *testing.T) {
	files := map[string]string{
		"table1.sql": `
			CREATE TABLE public.employees (
				id SERIAL PRIMARY KEY,
				email TEXT NOT NULL
			);
			CREATE INDEX idx_employee_email ON public.employees(email);
		`,
		"table2.sql": `
			CREATE TABLE public.customers (
				id SERIAL PRIMARY KEY,
				email TEXT NOT NULL
			);
			CREATE INDEX idx_employee_email ON public.customers(email);
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have exactly one error total (in either file)
	totalErrors := len(results["table1.sql"]) + len(results["table2.sql"])
	require.Equal(t, 1, totalErrors, "Should have exactly 1 error total")

	// Find the error
	var advice *storepb.Advice
	if len(results["table1.sql"]) > 0 {
		advice = results["table1.sql"][0]
	} else {
		advice = results["table2.sql"][0]
	}

	require.Equal(t, storepb.Advice_ERROR, advice.Status)
	require.Equal(t, code.SDLDuplicateIndexName.Int32(), advice.Code)
	require.Contains(t, advice.Content, "Index 'public.idx_employee_email' is defined in multiple SDL files")
}

func TestCheckSDLIntegrity_DuplicateConstraintAcrossFiles(t *testing.T) {
	files := map[string]string{
		"orders.sql": `
			CREATE TABLE public.orders (
				id INTEGER NOT NULL,
				email TEXT NOT NULL,
				CONSTRAINT pk_common PRIMARY KEY (id),
				CONSTRAINT uk_email UNIQUE (email)
			);
		`,
		"invoices.sql": `
			CREATE TABLE public.invoices (
				id INTEGER NOT NULL,
				code TEXT NOT NULL,
				CONSTRAINT pk_common PRIMARY KEY (id),
				CONSTRAINT uk_code UNIQUE (code)
			);
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have duplicate PRIMARY KEY constraint error (schema-level uniqueness)
	hasError := false
	for _, advices := range results {
		for _, advice := range advices {
			if advice.Code == code.SDLDuplicateConstraintName.Int32() {
				require.Contains(t, advice.Content, "Constraint 'pk_common' in schema 'public' is defined in multiple SDL files")
				require.Contains(t, advice.Content, "PostgreSQL requires PRIMARY KEY and UNIQUE constraint names to be unique within a schema")
				hasError = true
			}
		}
	}
	require.True(t, hasError, "Should have duplicate constraint error")
}

func TestCheckSDLIntegrity_SameCheckFKNameAcrossTablesAllowed(t *testing.T) {
	files := map[string]string{
		"orders.sql": `
			CREATE TABLE public.users (
				id INTEGER NOT NULL,
				CONSTRAINT pk_users PRIMARY KEY (id)
			);

			CREATE TABLE public.orders (
				id INTEGER NOT NULL,
				user_id INTEGER NOT NULL,
				amount NUMERIC NOT NULL,
				CONSTRAINT pk_orders PRIMARY KEY (id),
				CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id),
				CONSTRAINT chk_positive CHECK (amount > 0)
			);
		`,
		"invoices.sql": `
			CREATE TABLE public.invoices (
				id INTEGER NOT NULL,
				user_id INTEGER NOT NULL,
				total NUMERIC NOT NULL,
				CONSTRAINT pk_invoices PRIMARY KEY (id),
				CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id),
				CONSTRAINT chk_positive CHECK (total > 0)
			);
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have NO errors - CHECK and FK constraints are table-scoped
	for filePath, advices := range results {
		require.Empty(t, advices, "File %s should have no errors, got: %v", filePath, advices)
	}
}

func TestCheckSDLIntegrity_ForeignKeyTableNotFound(t *testing.T) {
	files := map[string]string{
		"orders.sql": `
			CREATE TABLE public.orders (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL,
				FOREIGN KEY (user_id) REFERENCES public.users(id)
			);
		`,
		"products.sql": `
			CREATE TABLE public.products (
				id SERIAL PRIMARY KEY,
				name TEXT
			);
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have FK table not found error in orders.sql
	require.Greater(t, len(results["orders.sql"]), 0, "orders.sql should have errors")

	var foundError bool
	for _, advice := range results["orders.sql"] {
		if advice.Code == code.SDLForeignKeyTableNotFound.Int32() {
			require.Contains(t, advice.Content, "references table 'public.users' which does not exist in any SDL file")
			foundError = true
		}
	}
	require.True(t, foundError, "Should have FK table not found error")
}

func TestCheckSDLIntegrity_ViewReferencesNonExistentTable(t *testing.T) {
	files := map[string]string{
		"products.sql": `
			CREATE TABLE public.products (
				id SERIAL PRIMARY KEY,
				name TEXT
			);
		`,
		"views.sql": `
			CREATE VIEW public.order_summary AS
			SELECT o.id, o.total, p.name
			FROM public.orders o
			JOIN public.products p ON o.product_id = p.id;
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have view dependency error in views.sql
	var foundError bool
	for _, advice := range results["views.sql"] {
		if advice.Code == code.SDLViewDependencyNotFound.Int32() {
			require.Contains(t, advice.Content, "View 'public.order_summary'")
			require.Contains(t, advice.Content, "references table or view 'public.orders' which does not exist")
			foundError = true
		}
	}
	require.True(t, foundError, "Should have view dependency error")
}

func TestCheckSDLIntegrity_ViewReferencesView(t *testing.T) {
	files := map[string]string{
		"tables.sql": `
			CREATE TABLE public.users (
				id INTEGER NOT NULL,
				name TEXT NOT NULL,
				active BOOLEAN NOT NULL DEFAULT true,
				CONSTRAINT pk_users PRIMARY KEY (id)
			);
		`,
		"views.sql": `
			-- Base view
			CREATE VIEW public.active_users AS
			SELECT id, name FROM public.users WHERE active = true;

			-- View referencing another view (should be valid)
			CREATE VIEW public.user_summary AS
			SELECT id, name FROM public.active_users;
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have NO errors - views can reference other views
	for filePath, advices := range results {
		require.Empty(t, advices, "File %s should have no errors, got: %v", filePath, advices)
	}
}

func TestCheckSDLIntegrity_ViewWithCTE(t *testing.T) {
	files := map[string]string{
		"tables.sql": `
			CREATE TABLE public.orders (
				id INTEGER NOT NULL,
				user_id INTEGER NOT NULL,
				total NUMERIC NOT NULL,
				CONSTRAINT pk_orders PRIMARY KEY (id)
			);
		`,
		"views.sql": `
			-- View using CTE
			CREATE VIEW public.order_summary AS
			WITH high_value_orders AS (
				SELECT id, user_id, total
				FROM public.orders
				WHERE total > 1000
			)
			SELECT user_id, COUNT(*) as order_count, SUM(total) as total_amount
			FROM high_value_orders
			GROUP BY user_id;
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have NO errors - CTE 'high_value_orders' is not a real table
	for filePath, advices := range results {
		require.Empty(t, advices, "File %s should have no errors, got: %v", filePath, advices)
	}
}

func TestCheckSDLIntegrity_ViewWithMultipleCTEs(t *testing.T) {
	files := map[string]string{
		"tables.sql": `
			CREATE TABLE public.orders (
				id INTEGER NOT NULL,
				user_id INTEGER NOT NULL,
				product_id INTEGER NOT NULL,
				total NUMERIC NOT NULL,
				CONSTRAINT pk_orders PRIMARY KEY (id)
			);

			CREATE TABLE public.products (
				id INTEGER NOT NULL,
				name TEXT NOT NULL,
				price NUMERIC NOT NULL,
				CONSTRAINT pk_products PRIMARY KEY (id)
			);
		`,
		"views.sql": `
			-- View using multiple CTEs
			CREATE VIEW public.sales_summary AS
			WITH high_value_orders AS (
				SELECT id, user_id, product_id, total
				FROM public.orders
				WHERE total > 1000
			),
			product_stats AS (
				SELECT p.name, COUNT(h.id) as order_count, SUM(h.total) as total_revenue
				FROM high_value_orders h
				JOIN public.products p ON h.product_id = p.id
				GROUP BY p.name
			)
			SELECT * FROM product_stats WHERE order_count > 5;
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have NO errors - both CTEs should be filtered out
	for filePath, advices := range results {
		require.Empty(t, advices, "File %s should have no errors, got: %v", filePath, advices)
	}
}

func TestCheckSDLIntegrity_ViewWithCTEAndMissingTable(t *testing.T) {
	files := map[string]string{
		"views.sql": `
			-- View using CTE but referencing non-existent table
			CREATE VIEW public.order_summary AS
			WITH valid_cte AS (
				SELECT id, total
				FROM public.orders
				WHERE total > 100
			)
			SELECT * FROM valid_cte;
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have error for missing 'orders' table, but NOT for 'valid_cte'
	require.Greater(t, len(results["views.sql"]), 0, "Should have error for missing table")

	var foundOrdersError bool
	var foundCTEError bool
	for _, advice := range results["views.sql"] {
		if advice.Code == code.SDLViewDependencyNotFound.Int32() {
			if strings.Contains(advice.Content, "'public.orders'") {
				foundOrdersError = true
			}
			if strings.Contains(advice.Content, "valid_cte") {
				foundCTEError = true
			}
		}
	}

	require.True(t, foundOrdersError, "Should have error for missing 'orders' table")
	require.False(t, foundCTEError, "Should NOT have error for CTE 'valid_cte'")
}

func TestCheckSDLIntegrity_ViewReferencesSystemTables(t *testing.T) {
	files := map[string]string{
		"tables.sql": `
			CREATE TABLE public.users (
				id SERIAL PRIMARY KEY,
				name TEXT
			);
		`,
		"system_views.sql": `
			-- View referencing pg_* system tables (should not report error)
			CREATE VIEW public.table_sizes AS
			SELECT
				schemaname,
				tablename,
				pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
			FROM pg_tables
			WHERE schemaname = 'public';

			-- View referencing information_schema (should not report error)
			CREATE VIEW public.column_info AS
			SELECT table_name, column_name, data_type
			FROM information_schema.columns
			WHERE table_schema = 'public';

			-- View referencing pg_catalog (should not report error)
			CREATE VIEW public.index_info AS
			SELECT i.relname AS index_name, t.relname AS table_name
			FROM pg_catalog.pg_index idx
			JOIN pg_catalog.pg_class i ON i.oid = idx.indexrelid
			JOIN pg_catalog.pg_class t ON t.oid = idx.indrelid;

			-- Mixed: system tables and user tables (should not report error for system tables)
			CREATE VIEW public.user_stats AS
			SELECT u.name, COUNT(*) as table_count
			FROM public.users u
			CROSS JOIN pg_tables pt
			WHERE pt.schemaname = 'public'
			GROUP BY u.name;
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have no errors - all system table references should be ignored
	for filePath, advices := range results {
		require.Empty(t, advices, "File %s should have no errors, got: %v", filePath, advices)
	}
}

func TestCheckSDLIntegrity_ViewReferencesMixedTablesWithError(t *testing.T) {
	files := map[string]string{
		"tables.sql": `
			CREATE TABLE public.products (
				id SERIAL PRIMARY KEY,
				name TEXT
			);
		`,
		"views.sql": `
			-- This view references both system tables (ok) and non-existent user table (error)
			CREATE VIEW public.order_analysis AS
			SELECT o.id, o.total, p.name, pg_size_pretty(pg_table_size('orders')) as table_size
			FROM public.orders o
			JOIN public.products p ON o.product_id = p.id;
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have error for 'orders' table but not for pg_table_size or pg_size_pretty
	var foundOrdersError bool
	for _, advice := range results["views.sql"] {
		if advice.Code == code.SDLViewDependencyNotFound.Int32() {
			require.Contains(t, advice.Content, "public.orders")
			require.NotContains(t, advice.Content, "pg_")
			foundOrdersError = true
		}
	}
	require.True(t, foundOrdersError, "Should have error for orders table")
}

func TestCheckSDLIntegrity_MultipleErrorsInSameFile(t *testing.T) {
	files := map[string]string{
		"base.sql": `
			CREATE TABLE public.users (
				id SERIAL PRIMARY KEY,
				name TEXT
			);
		`,
		"errors.sql": `
			CREATE TABLE public.users (
				id INTEGER PRIMARY KEY
			);

			CREATE TABLE public.orders (
				id SERIAL PRIMARY KEY,
				user_id BIGINT,
				customer_id INTEGER,
				FOREIGN KEY (user_id) REFERENCES public.users(id),
				FOREIGN KEY (customer_id) REFERENCES public.customers(id)
			);

			CREATE VIEW public.bad_view AS
			SELECT * FROM public.non_existent_table;
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Collect all error codes from both files
	errorCodes := make(map[int32]int)
	for _, advices := range results {
		for _, advice := range advices {
			errorCodes[advice.Code]++
		}
	}

	// Should have: duplicate table, FK table not found, view dependency not found
	require.Greater(t, errorCodes[code.SDLDuplicateTableName.Int32()], 0, "Should have duplicate table")
	require.Greater(t, errorCodes[code.SDLForeignKeyTableNotFound.Int32()], 0, "Should have FK table not found")
	require.Greater(t, errorCodes[code.SDLViewDependencyNotFound.Int32()], 0, "Should have view dependency error")

	// Should have at least 3 total errors
	totalErrors := 0
	for _, advices := range results {
		totalErrors += len(advices)
	}
	require.GreaterOrEqual(t, totalErrors, 3, "Should have at least 3 errors total")
}

func TestCheckSDLIntegrity_EmptyFiles(t *testing.T) {
	files := map[string]string{
		"empty1.sql": "",
		"empty2.sql": "   \n\n  ",
		"valid.sql": `
			CREATE TABLE public.test (id SERIAL PRIMARY KEY);
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Empty files should not cause errors
	for filePath, advices := range results {
		require.Empty(t, advices, "File %s should have no errors", filePath)
	}
}

func TestCheckSDLIntegrity_SyntaxErrorInOneFile(t *testing.T) {
	files := map[string]string{
		"valid.sql": `
			CREATE TABLE public.users (id SERIAL PRIMARY KEY);
		`,
		"invalid.sql": `
			CREATE TABLE public.orders (
				id SERIAL PRIMARY KEY,
				user_id INTEGER
				FOREIGN KEY (user_id) REFERENCES public.users(id)  -- Missing comma
			);
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Syntax error should be reported for invalid.sql
	require.Greater(t, len(results["invalid.sql"]), 0, "invalid.sql should have errors")

	var foundSyntaxError bool
	for _, advice := range results["invalid.sql"] {
		if advice.Code == code.StatementSyntaxError.Int32() {
			foundSyntaxError = true
		}
	}
	require.True(t, foundSyntaxError, "Should have syntax error")

	// valid.sql should have no errors
	require.Empty(t, results["valid.sql"], "valid.sql should have no errors")
}

func TestCheckSDLIntegrity_ComplexMultiFileSchema(t *testing.T) {
	files := map[string]string{
		"01_base.sql": `
			CREATE TABLE public.organizations (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			);

			CREATE TABLE public.users (
				id SERIAL PRIMARY KEY,
				org_id INTEGER NOT NULL,
				email TEXT UNIQUE,
				FOREIGN KEY (org_id) REFERENCES public.organizations(id)
			);
		`,
		"02_business.sql": `
			CREATE TABLE public.projects (
				id SERIAL PRIMARY KEY,
				org_id INTEGER NOT NULL,
				name TEXT NOT NULL,
				owner_id INTEGER,
				FOREIGN KEY (org_id) REFERENCES public.organizations(id),
				FOREIGN KEY (owner_id) REFERENCES public.users(id)
			);

			CREATE TABLE public.tasks (
				id SERIAL PRIMARY KEY,
				project_id INTEGER NOT NULL,
				assignee_id INTEGER,
				title TEXT NOT NULL,
				FOREIGN KEY (project_id) REFERENCES public.projects(id),
				FOREIGN KEY (assignee_id) REFERENCES public.users(id)
			);
		`,
		"03_analytics.sql": `
			CREATE VIEW public.project_stats AS
			SELECT
				p.id as project_id,
				p.name as project_name,
				o.name as org_name,
				COUNT(t.id) as task_count
			FROM public.projects p
			JOIN public.organizations o ON p.org_id = o.id
			LEFT JOIN public.tasks t ON t.project_id = p.id
			GROUP BY p.id, p.name, o.name;

			CREATE VIEW public.user_workload AS
			SELECT
				u.id as user_id,
				u.email,
				COUNT(t.id) as assigned_tasks
			FROM public.users u
			LEFT JOIN public.tasks t ON t.assignee_id = u.id
			GROUP BY u.id, u.email;
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// All files should have no errors
	for filePath, advices := range results {
		require.Empty(t, advices, "File %s should have no errors, got: %v", filePath, advices)
	}
}

func TestCheckSDLIntegrity_ForeignKeyColumnNotFound(t *testing.T) {
	files := map[string]string{
		"users.sql": `
			CREATE TABLE public.users (
				id SERIAL PRIMARY KEY,
				name TEXT
			);
		`,
		"orders.sql": `
			CREATE TABLE public.orders (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL,
				FOREIGN KEY (user_id) REFERENCES public.users(user_uuid)
			);
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have FK column not found error
	var foundError bool
	for _, advice := range results["orders.sql"] {
		if advice.Code == code.SDLForeignKeyColumnNotFound.Int32() {
			require.Contains(t, advice.Content, "references column 'user_uuid'")
			require.Contains(t, advice.Content, "does not exist")
			foundError = true
		}
	}
	require.True(t, foundError, "Should have FK column not found error")
}

func TestCheckSDLIntegrity_DuplicateViewAcrossFiles(t *testing.T) {
	files := map[string]string{
		"tables.sql": `
			CREATE TABLE public.products (id SERIAL PRIMARY KEY);
		`,
		"view1.sql": `
			CREATE VIEW public.product_list AS SELECT * FROM public.products;
		`,
		"view2.sql": `
			CREATE VIEW public.product_list AS SELECT id FROM public.products;
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should have duplicate view error
	var foundError bool
	for _, advices := range results {
		for _, advice := range advices {
			if advice.Code == code.SDLDuplicateTableName.Int32() && advice.Title == "Duplicate view name across files" {
				require.Contains(t, advice.Content, "View 'public.product_list' is defined in multiple SDL files")
				foundError = true
			}
		}
	}
	require.True(t, foundError, "Should have duplicate view error")
}

func TestCheckSDLIntegrity_SingleFile(t *testing.T) {
	files := map[string]string{
		"schema.sql": `
			CREATE TABLE public.users (
				id SERIAL PRIMARY KEY,
				email TEXT UNIQUE
			);

			CREATE TABLE public.orders (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL,
				FOREIGN KEY (user_id) REFERENCES public.users(id)
			);
		`,
	}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)

	// Should work correctly with single file (backward compatibility)
	require.Empty(t, results["schema.sql"], "Single file should have no errors")
}

func TestCheckSDLIntegrity_EmptyInput(t *testing.T) {
	files := map[string]string{}

	results, err := CheckSDLIntegrity(files)
	require.NoError(t, err)
	require.Empty(t, results, "Empty input should return empty results")
}
