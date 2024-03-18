package hive

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		input  string
		expect []string
	}{
		{
			input: "SELECT 1;;;;SELECT '2;3;4;';;SELECT 2;;",
			expect: []string{
				"SELECT 1",
				"SELECT '2",
				"3",
				"4",
				"'",
				"SELECT 2",
			},
		},
		{
			input: "\n--- 123456\nSELECT 1;\n--- 6312\nSELECT * FROM pokes;\n---123\n---123\nSELECT 3;",
			expect: []string{
				"\nSELECT 1",
				"\nSELECT * FROM pokes",
				"\nSELECT 3",
			},
		},
		{
			input: " CREATE TABLE pokes ",
			expect: []string{
				" CREATE TABLE pokes ",
			},
		},
		{
			input:  "-- this is a comment\n ---- this is also a comment\r\n",
			expect: []string{},
		},
		{
			input: "\n\t\t\t\tSELECT 1;\n\t\t\t\t\r\nSELECT '2;3;4;';SELECT 2;",
			expect: []string{
				"\n\t\t\t\tSELECT 1",
				"\n\t\t\t\t\r\nSELECT '2",
				"3",
				"4",
				"'",
				"SELECT 2",
			},
		},
		{
			input:  "CREATE TABLE migration_history(id n\t\t\tSTRING,statement STRING,schema\r\n STRING, payload STRING)	",
			expect: []string{"CREATE TABLE migration_history(id n\t\t\tSTRING,statement STRING,schema\r\n STRING, payload STRING)	"},
		},
	}
	a := require.New(t)
	for idx, test := range tests {
		output, err := splitHiveStatements(test.input)
		if err != nil {
			fmt.Printf("failed in the %d test\n", idx)
			t.Fatal(err.Error())
		}
		fmt.Printf("%v\n", output)
		a.Equal(test.expect, output)
	}

}
