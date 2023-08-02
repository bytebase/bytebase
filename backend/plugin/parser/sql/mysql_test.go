package parser

import (
	"testing"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"
	"github.com/stretchr/testify/require"
)

func TestMySQLParser(t *testing.T) {
	tests := []struct {
		statement    string
		errorMessage string
	}{
		{
			statement:    "aaa",
			errorMessage: "Syntax error at line 1:0 \nrelated text: aaa",
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
		_, err := ParseMySQL(test.statement)
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
		trees, err := ParseMySQL(test.statement)
		require.NoError(t, err)
		err = MySQLValidateForEditor(trees[0].Tree)
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

func TestSplitMySQLStatements(t *testing.T) {
	tests := []struct {
		statement string
		expected  []string
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
			expected: []string{
				"SELECT * FROM t1 WHERE c1 = 1;",
				" SELECT * FROM t2;",
			},
		},
		{
			statement: `CREATE PROCEDURE my_procedure (IN id INT, OUT name VARCHAR(255))
			BEGIN
			  SELECT name INTO name FROM users WHERE id = id;
			END; SELECT * FROM t2;`,
			expected: []string{
				`CREATE PROCEDURE my_procedure (IN id INT, OUT name VARCHAR(255))
			BEGIN
			  SELECT name INTO name FROM users WHERE id = id;
			END;`,
				" SELECT * FROM t2;",
			},
		},
		{
			statement: `CREATE PROCEDURE my_procedure (IN id INT, OUT name VARCHAR(255))
			BEGIN
				SELECT IF(id = 1, 'one', 'other') INTO name FROM users;
			END; SELECT REPEAT('123', a) FROM t2;`,
			expected: []string{
				`CREATE PROCEDURE my_procedure (IN id INT, OUT name VARCHAR(255))
			BEGIN
				SELECT IF(id = 1, 'one', 'other') INTO name FROM users;
			END;`,
				" SELECT REPEAT('123', a) FROM t2;",
			},
		},
	}

	for _, test := range tests {
		lexer := parser.NewMySQLLexer(antlr.NewInputStream(test.statement))
		stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
		list, err := splitMySQLStatement(stream)
		require.NoError(t, err)
		require.Equal(t, len(test.expected), len(list))
		for i, statement := range list {
			require.Equal(t, test.expected[i], statement.Text)
		}
	}
}

func TestExtractMySQLChangedResources(t *testing.T) {
	tests := []struct {
		statement string
		expected  []SchemaResource
	}{
		{
			statement: "CREATE TABLE t1 (c1 INT);",
			expected: []SchemaResource{
				{
					Database: "db",
					Table:    "t1",
				},
			},
		},
		{
			statement: "DROP TABLE t1;",
			expected: []SchemaResource{
				{
					Database: "db",
					Table:    "t1",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD COLUMN c1 INT;",
			expected: []SchemaResource{
				{
					Database: "db",
					Table:    "t1",
				},
			},
		},
		{
			statement: "RENAME TABLE t1 TO t2;",
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
		resources, err := extractMySQLChangedResources("db", test.statement)
		require.NoError(t, err)
		require.Equal(t, test.expected, resources, test.statement)
	}
}
