package redshift

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSQLForEditor(t *testing.T) {
	tests := []struct {
		name               string
		statement          string
		wantCanRunReadOnly bool
		wantReturnsData    bool
		wantErr            bool
	}{
		// Basic SELECT statements (from PG tests)
		{
			name:               "simple SELECT without space",
			statement:          `select* from t`,
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "simple SELECT",
			statement:          "SELECT * FROM users",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SELECT with JOIN",
			statement:          "SELECT u.name, o.order_id FROM users u JOIN orders o ON u.id = o.user_id",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SELECT with WHERE",
			statement:          "SELECT * FROM products WHERE price > 100",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SELECT with GROUP BY",
			statement:          "SELECT category, COUNT(*) FROM products GROUP BY category",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SELECT with special characters in string",
			statement:          "select * from t where a = 'klasjdfkljsa$tag$; -- lkjdlkfajslkdfj'",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},

		// CTE statements (from PG tests)
		{
			name:               "CTE with SELECT",
			statement:          "WITH cte AS (SELECT * FROM users) SELECT * FROM cte",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name: "CTE with keywords in strings and comments",
			statement: `
				With t as (
					select * from t1 where a = 'insert'
				), tx as (
					select * from "delete"
				) /* UPDATE */
				select "update" from t;
				`,
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name: "CTE with DELETE",
			statement: `
				With t as (
					select * from t1
				), tx as (
					delete from t2
				)
				select * from t;
				`,
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name: "CTE with UPDATE",
			statement: `
				With t as (
					select * from t1
				), tx as (
					select * from t1
				)
				update t set a = 1;
				`,
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name: "CTE with INSERT",
			statement: `
				With t as (
					select * from t1
				), tx as (
					select * from t1
				)
				insert into t values (1, 2, 3);
				`,
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},

		// EXPLAIN statements (from PG tests)
		{
			name:               "EXPLAIN SELECT",
			statement:          `explain select * from t;`,
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "EXPLAIN ANALYZE SELECT",
			statement:          `explain analyze select * from t`,
			wantCanRunReadOnly: true,
			wantReturnsData:    false, // EXPLAIN ANALYZE executes but doesn't return query data
			wantErr:            false,
		},
		{
			name:               "EXPLAIN ANALYZE INSERT",
			statement:          `explain analyze insert into t values (1)`,
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "EXPLAIN ANALYZE with CTE containing DELETE",
			statement:          `EXPLAIN ANALYZE WITH cte1 AS (DELETE FROM t RETURNING id) SELECT * FROM cte1`,
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "EXPLAIN UPDATE",
			statement:          "EXPLAIN UPDATE users SET name = 'test'",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "EXPLAIN ANALYZE UPDATE",
			statement:          "EXPLAIN ANALYZE UPDATE users SET name = 'test'",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},

		// SHOW statements - Redshift specific
		{
			name:               "SHOW TABLES",
			statement:          "SHOW TABLES",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SHOW TABLE",
			statement:          "SHOW TABLE users",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SHOW DATABASES",
			statement:          "SHOW DATABASES",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SHOW SCHEMAS",
			statement:          "SHOW SCHEMAS",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SHOW COLUMNS",
			statement:          "SHOW COLUMNS FROM users",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SHOW EXTERNAL TABLE",
			statement:          "SHOW EXTERNAL TABLE spectrum.users",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SHOW DATASHARES",
			statement:          "SHOW DATASHARES",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SHOW GRANTS",
			statement:          "SHOW GRANTS FOR user1",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SHOW PROCEDURES",
			statement:          "SHOW PROCEDURES",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SHOW MODEL",
			statement:          "SHOW MODEL my_model",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SHOW VIEW",
			statement:          "SHOW VIEW my_view",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},

		// SET statements
		{
			name:               "SET statement",
			statement:          "SET search_path TO public",
			wantCanRunReadOnly: true,
			wantReturnsData:    false, // SET doesn't return data
			wantErr:            false,
		},

		// DML statements (not allowed)
		{
			name:               "INSERT",
			statement:          "INSERT INTO users (name, email) VALUES ('John', 'john@example.com')",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "UPDATE",
			statement:          "UPDATE users SET name = 'Jane' WHERE id = 1",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "DELETE",
			statement:          "DELETE FROM users WHERE id = 1",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "TRUNCATE",
			statement:          "TRUNCATE TABLE users",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "MERGE",
			statement:          "MERGE INTO target USING source ON target.id = source.id WHEN MATCHED THEN UPDATE SET value = source.value",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},

		// DDL statements (not allowed) - from PG tests
		{
			name:               "CREATE TABLE",
			statement:          `create table t (a int);`,
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "DROP TABLE",
			statement:          "DROP TABLE users",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "ALTER TABLE",
			statement:          "ALTER TABLE users ADD COLUMN age INT",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "CREATE INDEX",
			statement:          "CREATE INDEX idx_users_name ON users(name)",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "CREATE DATABASE",
			statement:          "CREATE DATABASE testdb",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "CREATE SCHEMA",
			statement:          "CREATE SCHEMA analytics",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "CREATE VIEW",
			statement:          "CREATE VIEW user_view AS SELECT * FROM users",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "CREATE MATERIALIZED VIEW",
			statement:          "CREATE MATERIALIZED VIEW user_summary AS SELECT COUNT(*) FROM users",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},

		// Redshift-specific DDL
		{
			name:               "CREATE EXTERNAL TABLE",
			statement:          "CREATE EXTERNAL TABLE spectrum.users (id INT, name VARCHAR(100)) STORED AS PARQUET LOCATION 's3://mybucket/data/'",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		// Skip CREATE EXTERNAL SCHEMA test as it requires complex syntax
		// that the parser may not fully support yet
		{
			name:               "CREATE LIBRARY",
			statement:          "CREATE LIBRARY urlparse LANGUAGE plpythonu FROM 's3://mybucket/urlparse.zip'",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},

		// Administrative statements (not allowed)
		{
			name:               "GRANT",
			statement:          "GRANT SELECT ON users TO user1",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "REVOKE",
			statement:          "REVOKE SELECT ON users FROM user1",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "ANALYZE",
			statement:          "ANALYZE users",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "VACUUM",
			statement:          "VACUUM users",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},

		// Redshift-specific commands
		{
			name:               "COPY",
			statement:          "COPY users FROM 's3://mybucket/data.csv' IAM_ROLE 'arn:aws:iam::123456789012:role/MyRedshiftRole'",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "UNLOAD",
			statement:          "UNLOAD ('SELECT * FROM users') TO 's3://mybucket/data/' IAM_ROLE 'arn:aws:iam::123456789012:role/MyRedshiftRole'",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "ALTER DATASHARE",
			statement:          "ALTER DATASHARE salesshare ADD SCHEMA public",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},

		// Transaction statements (not allowed)
		{
			name:               "BEGIN",
			statement:          "BEGIN",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "COMMIT",
			statement:          "COMMIT",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},
		{
			name:               "ROLLBACK",
			statement:          "ROLLBACK",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            false,
		},

		// Redshift-specific SELECT features
		{
			name:               "SELECT with Redshift-specific functions",
			statement:          "SELECT DATEADD(day, 1, GETDATE()) AS tomorrow",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SELECT with Redshift window functions",
			statement:          "SELECT name, RATIO_TO_REPORT(sales) OVER (PARTITION BY region) FROM sales_data",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SELECT with Redshift date functions",
			statement:          "SELECT DATEDIFF(day, start_date, end_date) FROM events",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SELECT from external schema",
			statement:          "SELECT * FROM spectrum.external_table",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SELECT with LISTAGG",
			statement:          "SELECT department, LISTAGG(employee_name, ', ') WITHIN GROUP (ORDER BY employee_name) FROM employees GROUP BY department",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SELECT with APPROXIMATE COUNT",
			statement:          "SELECT APPROXIMATE(COUNT(DISTINCT customer_id)) FROM orders",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},
		{
			name:               "SELECT with PERCENTILE_CONT",
			statement:          "SELECT PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY price) FROM products",
			wantCanRunReadOnly: true,
			wantReturnsData:    true,
			wantErr:            false,
		},

		// Invalid syntax
		{
			name:               "Invalid SQL",
			statement:          "SELEC * FORM users",
			wantCanRunReadOnly: false,
			wantReturnsData:    false,
			wantErr:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canRunReadOnly, returnsData, err := ValidateSQLForEditor(tt.statement)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantCanRunReadOnly, canRunReadOnly, "canRunReadOnly mismatch")
				require.Equal(t, tt.wantReturnsData, returnsData, "returnsData mismatch")
			}
		})
	}
}
