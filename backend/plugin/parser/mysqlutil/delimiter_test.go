package mysqlutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDealWithDelimiter(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      string
	}{
		{
			name:      "passthrough without delimiter directive",
			statement: "SELECT 1; SELECT 'DELIMITER ;;';",
			want:      "SELECT 1; SELECT 'DELIMITER ;;';",
		},
		{
			name:      "custom delimiter",
			statement: "DELIMITER ;;\nCREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND;;\nDELIMITER ;\nCALL p();",
			want: strings.Repeat(" ", len("DELIMITER ;;")) +
				"\nCREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND; \n" +
				strings.Repeat(" ", len("DELIMITER ;")) + "\nCALL p();",
		},
		{
			name:      "delimiter in comments is ignored",
			statement: "-- DELIMITER ;;\n/* DELIMITER $$ */\nSELECT 1;",
			want:      "-- DELIMITER ;;\n/* DELIMITER $$ */\nSELECT 1;",
		},
		{
			name:      "multicharacter delimiter",
			statement: "DELIMITER //\nSELECT 1//\nDELIMITER ;\nSELECT 2;",
			want: strings.Repeat(" ", len("DELIMITER //")) +
				"\nSELECT 1; \n" +
				strings.Repeat(" ", len("DELIMITER ;")) + "\nSELECT 2;",
		},
		{
			name:      "delimiter identifier in multi statement SQL is not a directive",
			statement: "SELECT 0;\nSELECT delimiter FROM t;\nSELECT 2;",
			want:      "SELECT 0;\nSELECT delimiter FROM t;\nSELECT 2;",
		},
		{
			name:      "delimiter identifier at line start is not a directive",
			statement: "CREATE TABLE t (\n  delimiter INT,\n  c INT\n);",
			want:      "CREATE TABLE t (\n  delimiter INT,\n  c INT\n);",
		},
		{
			name:      "delimiter identifier inside procedure body is preserved",
			statement: "DELIMITER ;;\nCREATE PROCEDURE p()\nBEGIN\n  SELECT delimiter FROM t;\nEND;;\nDELIMITER ;\nCALL p();",
			want: strings.Repeat(" ", len("DELIMITER ;;")) +
				"\nCREATE PROCEDURE p()\nBEGIN\n  SELECT delimiter FROM t;\nEND; \n" +
				strings.Repeat(" ", len("DELIMITER ;")) + "\nCALL p();",
		},
		{
			name:      "delimiter directives preserve line count",
			statement: "DELIMITER //\nSELECT 1//\nDELIMITER ;\nSELECT bad;",
			want: strings.Repeat(" ", len("DELIMITER //")) +
				"\nSELECT 1; \n" +
				strings.Repeat(" ", len("DELIMITER ;")) + "\nSELECT bad;",
		},
		{
			name:      "delimiter directive inside multiline single quoted string is preserved",
			statement: "SELECT 'before\nDELIMITER //\nafter';\nSELECT 1;",
			want:      "SELECT 'before\nDELIMITER //\nafter';\nSELECT 1;",
		},
		{
			name:      "delimiter directive inside multiline double quoted string is preserved",
			statement: "SELECT \"before\nDELIMITER //\nafter\";\nSELECT 1;",
			want:      "SELECT \"before\nDELIMITER //\nafter\";\nSELECT 1;",
		},
		{
			name:      "active delimiter inside multiline block comment is preserved",
			statement: "DELIMITER //\n/* comment starts\ncontains // delimiter text\nends */\nSELECT 1//\nDELIMITER ;",
			want: strings.Repeat(" ", len("DELIMITER //")) +
				"\n/* comment starts\ncontains // delimiter text\nends */\nSELECT 1; \n" +
				strings.Repeat(" ", len("DELIMITER ;")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := DealWithDelimiter(test.statement)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestDealWithDelimiterPreservesByteOffsets(t *testing.T) {
	statement := "DELIMITER //\nSELECT 1//\nDELIMITER ;\nSELECT bad;"
	want := strings.Repeat(" ", len("DELIMITER //")) + "\n" +
		"SELECT 1; \n" +
		strings.Repeat(" ", len("DELIMITER ;")) + "\n" +
		"SELECT bad;"

	got, err := DealWithDelimiter(statement)
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.Len(t, got, len(statement))
	require.Equal(t, strings.Index(statement, "SELECT bad"), strings.Index(got, "SELECT bad"))
}
