package parser

import (
	"testing"

	_ "embed"

	"github.com/stretchr/testify/require"
)

//go:embed test1.sql
var test1 []byte

//go:embed test2.sql
var test2 []byte

//go:embed test3.sql
var test3 []byte

//go:embed test4.sql
var test4 []byte

func TestMySQLParser(t *testing.T) {
	tests := []struct {
		statement    string
		errorMessage string
	}{
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
				        SELECT column_name INTO output FROM table_name WHERE condition;
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
		_, err := ParseMySQL(test.statement, "", "")
		if test.errorMessage == "" {
			require.NoError(t, err, i)
		} else {
			require.EqualError(t, err, test.errorMessage)
		}
	}
}
