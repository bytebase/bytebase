package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMySQLParser(t *testing.T) {
	tests := []struct {
		statement    string
		errorMessage string
	}{
		{
			statement:    "aaa",
			errorMessage: "line 1:0 extraneous input 'aaa' expecting {<EOF>, ALTER_SYMBOL, ANALYZE_SYMBOL, BEGIN_SYMBOL, BINLOG_SYMBOL, CACHE_SYMBOL, CALL_SYMBOL, CHANGE_SYMBOL, CHECKSUM_SYMBOL, CHECK_SYMBOL, COMMIT_SYMBOL, CREATE_SYMBOL, DEALLOCATE_SYMBOL, DELETE_SYMBOL, DESC_SYMBOL, DESCRIBE_SYMBOL, DO_SYMBOL, DROP_SYMBOL, EXECUTE_SYMBOL, EXPLAIN_SYMBOL, FLUSH_SYMBOL, FOR_SYMBOL, GET_SYMBOL, GRANT_SYMBOL, HANDLER_SYMBOL, HELP_SYMBOL, IMPORT_SYMBOL, INSERT_SYMBOL, INSTALL_SYMBOL, KILL_SYMBOL, LOAD_SYMBOL, LOCK_SYMBOL, OPTIMIZE_SYMBOL, PREPARE_SYMBOL, PURGE_SYMBOL, RELEASE_SYMBOL, RENAME_SYMBOL, REPAIR_SYMBOL, REPLACE_SYMBOL, RESET_SYMBOL, RESIGNAL_SYMBOL, REVOKE_SYMBOL, ROLLBACK_SYMBOL, SAVEPOINT_SYMBOL, SELECT_SYMBOL, SET_SYMBOL, SHOW_SYMBOL, SHUTDOWN_SYMBOL, SIGNAL_SYMBOL, START_SYMBOL, STOP_SYMBOL, TABLE_SYMBOL, TRUNCATE_SYMBOL, UNINSTALL_SYMBOL, UNLOCK_SYMBOL, UPDATE_SYMBOL, USE_SYMBOL, VALUES_SYMBOL, WITH_SYMBOL, XA_SYMBOL, CLONE_SYMBOL, RESTART_SYMBOL, ';', '('}",
		},
		{
			statement: "select * from t;\n -- comments",
		},
		{
			statement: "SELECT count(t.a) as TID from t1 as t;",
		},
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
		},
		{
			statement: "CREATE TABLE t1 (c1 INT);",
		},
		{
			statement: `
				-- Drop the procedure if it already exists
				DROP PROCEDURE IF EXISTS complex_procedure;
				
				-- Create the procedure
				CREATE PROCEDURE complex_procedure(IN input VARCHAR(255), OUT output VARCHAR(255))
				BEGIN
				    DECLARE var1 VARCHAR(255);
				    DECLARE var2 VARCHAR(255);
				    DECLARE var3 INT;
				    
				    -- Setting initial values
				    SET var1 = 'Hello, ';
				    SET var2 = 'World!';
				    SET var3 = 1;
				    
				    -- If statement
				    IF var3 = 1 THEN
				        -- String concatenation
				        SET output = CONCAT(var1, input, ' and ', var2);
				    ELSE
				        -- Use a SELECT statement to get data from a table
				        SELECT column_name INTO output FROM table_name WHERE condition_expression;
				    END IF;
				END;
				
				-- Call the procedure
				CALL complex_procedure('MySQL', @output_value);
				SELECT @output_value;
			`,
		},
		{
			statement: `CREATE TABLE IF NOT EXISTS test_table (
				id INT PRIMARY KEY,
				name VARCHAR(255),
				description VARCHAR(255)
			);
			
			REPLACE INTO test_table (id, name, description)
			VALUES (1, 'Test', 'This is a test.');
			`,
		},
	}

	for i, test := range tests {
		_, _, err := ParseMySQL(test.statement)
		if test.errorMessage == "" {
			require.NoError(t, err, i)
		} else {
			require.EqualError(t, err, test.errorMessage)
		}
	}
}

func TestMySQLValidateForEditor(t *testing.T) {
	tests := []struct {
		statement string
		validate  bool
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
			validate:  true,
		},
		{
			statement: "CREATE TABLE t1 (c1 INT);",
			validate:  false,
		},
		{
			statement: "UPDATE t1 SET c1 = 1;",
			validate:  false,
		},
		{
			statement: "EXPLAIN SELECT * FROM t1;",
			validate:  true,
		},
		{
			statement: "EXPLAIN FORMAT=JSON DELETE FROM t1;",
			validate:  false,
		},
	}

	for _, test := range tests {
		tree, _, err := ParseMySQL(test.statement)
		require.NoError(t, err)
		err = MySQLValidateForEditor(tree)
		if test.validate {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}

func TestExtractMySQLResourceList(t *testing.T) {
	tests := []struct {
		statement string
		expected  []SchemaResource
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
			expected: []SchemaResource{
				{
					Database: "db",
					Table:    "t1",
				},
				{
					Database: "db",
					Table:    "t2",
				},
			},
		},
		{
			statement: "SELECT * FROM db1.t1 JOIN db2.t2 ON t1.c1 = t2.c1;",
			expected: []SchemaResource{
				{
					Database: "db1",
					Table:    "t1",
				},
				{
					Database: "db2",
					Table:    "t2",
				},
			},
		},
		{
			statement: "SELECT a > (select max(a) from t1) FROM t2;",
			expected: []SchemaResource{
				{
					Database: "db",
					Table:    "t1",
				},
				{
					Database: "db",
					Table:    "t2",
				},
			},
		},
	}

	for _, test := range tests {
		resources, err := extractMySQLResourceList("db", test.statement)
		require.NoError(t, err)
		require.Equal(t, test.expected, resources, test.statement)
	}
}
