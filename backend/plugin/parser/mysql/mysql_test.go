package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExtractDelimiter(t *testing.T) {
	tests := []struct {
		stmt    string
		want    string
		wantErr bool
	}{
		{
			stmt:    "DELIMITER ;;",
			want:    ";;",
			wantErr: false,
		},
		{
			stmt:    "DELIMITER //",
			want:    "//",
			wantErr: false,
		},
		{
			stmt:    "DELIMITER $$",
			want:    "$$",
			wantErr: false,
		},
		{
			stmt:    "DELIMITER    @@   ",
			want:    "@@",
			wantErr: false,
		},
		{
			stmt:    "DELIMITER    @@//",
			want:    "@@//",
			wantErr: false,
		},
		{
			stmt:    "DELIMITER    @@//",
			want:    "@@//",
			wantErr: false,
		},
		// DELIMITER cannot contain a backslash character
		{
			stmt:    "DELIMITER    \\",
			wantErr: true,
		},
	}
	a := require.New(t)
	for _, test := range tests {
		got, err := ExtractDelimiter(test.stmt)
		if test.wantErr {
			a.Error(err)
		} else {
			a.NoError(err)
			a.Equal(test.want, got)
		}
	}
}

func TestMySQLParser(t *testing.T) {
	tests := []struct {
		statement    string
		errorMessage string
		total        int
	}{
		{
			statement: "INSERT INTO tbl (id, name, gender, height) VALUES(1, 'Alice', B'0', B'01111111');",
			total:     1,
		},
		{
			statement:    "aaa",
			errorMessage: "Syntax error at line 1:1 \nrelated text: aaa",
		},
		{
			statement:    "select 1;\n   selec 5;\nselect 6;",
			errorMessage: "Syntax error at line 2:4 \nrelated text: \n   selec",
		},
		{
			statement: "select * from t;\n -- comments",
			total:     1,
		},
		{
			statement: "SELECT count(t.a) as TID from t1 as t;",
			total:     1,
		},
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;   		",
			total:     2,
		},
		{
			statement: "CREATE TABLE t1 (c1 INT);",
			total:     1,
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
			total: 4,
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
			total: 2,
		},
	}

	for i, test := range tests {
		list, err := ParseMySQL(test.statement)
		if test.errorMessage == "" {
			require.NoError(t, err, i)
			require.Equal(t, test.total, len(list), i)
		} else {
			require.EqualError(t, err, test.errorMessage)
		}
	}
}

func TestParseMySQLStatements(t *testing.T) {
	statement := "SELECT 1; SELECT 2;"

	statements, err := base.ParseStatements(storepb.Engine_MYSQL, statement)
	require.NoError(t, err)

	require.Len(t, statements, 2)

	// Check first statement
	require.Equal(t, "SELECT 1;", statements[0].Text)
	require.False(t, statements[0].Empty)
	require.NotNil(t, statements[0].AST)
	require.NotNil(t, statements[0].Start)

	// Check second statement
	require.Contains(t, statements[1].Text, "SELECT 2")
	require.False(t, statements[1].Empty)
	require.NotNil(t, statements[1].AST)
}
