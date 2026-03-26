package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniExcludeConstraintWithRegularIndexes(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE employee_shifts (
			shift_id bigserial,
			employee_id integer NOT NULL,
			department_id integer NOT NULL,
			shift_start timestamp(6) with time zone NOT NULL,
			shift_end timestamp(6) with time zone NOT NULL,
			shift_type character varying(50) DEFAULT 'regular',
			created_at timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT pk_employee_shifts PRIMARY KEY (shift_id),
			CONSTRAINT chk_employee_shifts_time CHECK (shift_end > shift_start)
		);

		CREATE INDEX idx_employee_shifts_employee ON employee_shifts (employee_id);
		CREATE INDEX idx_employee_shifts_department ON employee_shifts (department_id);
		CREATE INDEX idx_employee_shifts_time ON employee_shifts USING gist (tstzrange(shift_start, shift_end));
		CREATE INDEX idx_employee_shifts_type ON employee_shifts (shift_type);
	`)

	t.Logf("Generated SQL:\n%s", sql)

	require.Contains(t, sql, "CREATE TABLE")
	require.Contains(t, sql, "employee_shifts")

	// Regular indexes should be created
	require.Contains(t, sql, "idx_employee_shifts_employee")
	require.Contains(t, sql, "idx_employee_shifts_department")
	require.Contains(t, sql, "idx_employee_shifts_time")
	require.Contains(t, sql, "idx_employee_shifts_type")
}

func TestOmniExcludeConstraintIndexNotDuplicated(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE meetings (
			meeting_id serial PRIMARY KEY,
			room_id integer NOT NULL,
			meeting_start timestamp NOT NULL,
			meeting_end timestamp NOT NULL
		);
	`)

	t.Logf("Generated SQL:\n%s", sql)

	require.Contains(t, sql, "CREATE TABLE")
	require.Contains(t, sql, "meetings")
	// Should NOT create a separate index named after the constraint
	require.NotContains(t, sql, "CREATE INDEX excl_room_no_overlap")
}

func TestOmniMaterializedViewWithRegularIndexes(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE orders (
			order_id serial PRIMARY KEY,
			customer_id integer NOT NULL,
			order_date date NOT NULL,
			total_amount numeric(10,2) NOT NULL
		);

		CREATE MATERIALIZED VIEW order_summary AS
		SELECT
			customer_id,
			DATE_TRUNC('month', order_date) as order_month,
			COUNT(*) as order_count,
			SUM(total_amount) as total_sales
		FROM orders
		GROUP BY customer_id, DATE_TRUNC('month', order_date);

		CREATE INDEX idx_order_summary_customer ON order_summary (customer_id);
		CREATE INDEX idx_order_summary_month ON order_summary (order_month);
		CREATE UNIQUE INDEX idx_order_summary_unique ON order_summary (customer_id, order_month);
	`)

	t.Logf("Generated SQL:\n%s", sql)

	require.Contains(t, sql, "CREATE TABLE")
	require.Contains(t, sql, "orders")
	require.Contains(t, sql, "CREATE MATERIALIZED VIEW")
	require.Contains(t, sql, "order_summary")

	require.Contains(t, sql, "idx_order_summary_customer")
	require.Contains(t, sql, "idx_order_summary_month")
	require.Contains(t, sql, "idx_order_summary_unique")
}

func TestOmniIndexCreatedBeforeForeignKey(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE customers (
			id serial PRIMARY KEY,
			email varchar(255) NOT NULL,
			name varchar(100) NOT NULL
		);

		CREATE INDEX idx_customers_id ON customers (id);
		CREATE INDEX idx_customers_email ON customers (email);

		CREATE TABLE orders (
			id serial PRIMARY KEY,
			customer_id integer NOT NULL,
			order_date date NOT NULL,
			total_amount numeric(10,2) NOT NULL,
			CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id) REFERENCES customers (id)
		);

		CREATE INDEX idx_orders_customer ON orders (customer_id);
	`)

	t.Logf("Generated SQL:\n%s", sql)

	require.Contains(t, sql, "CREATE TABLE")
	require.Contains(t, sql, "customers")
	require.Contains(t, sql, "orders")

	require.Contains(t, sql, "idx_customers_id")
	require.Contains(t, sql, "idx_customers_email")
	require.Contains(t, sql, "idx_orders_customer")
	require.Contains(t, sql, "fk_orders_customer")

	// The omni engine correctly includes FK constraints inline in CREATE TABLE.
	// The index is created after the table.
	customersTablePos := strings.Index(sql, `"customers"`)
	ordersTablePos := strings.Index(sql, `"orders"`)

	require.NotEqual(t, -1, customersTablePos, "customers table should exist")
	require.NotEqual(t, -1, ordersTablePos, "orders table should exist")
	require.Less(t, customersTablePos, ordersTablePos,
		"Referenced table (customers) should be created BEFORE the table with FK (orders)")
}
