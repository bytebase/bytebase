package tsql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSQLForEditor(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		valid     bool
		allQuery  bool
	}{
		{
			name:      "select",
			statement: "SELECT id FROM t",
			valid:     true,
			allQuery:  true,
		},
		{
			name:      "isolation level before a select",
			statement: "SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED\nSELECT id FROM t",
			valid:     true,
		},
		{
			name:      "isolation level with terminators",
			statement: "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE; SELECT id FROM t;",
			valid:     true,
		},
		{
			name:      "every isolation level",
			statement: "SET TRANSACTION ISOLATION LEVEL READ COMMITTED; SET TRANSACTION ISOLATION LEVEL REPEATABLE READ; SET TRANSACTION ISOLATION LEVEL SNAPSHOT; SELECT 1",
			valid:     true,
		},
		{
			name:      "nocount in lower case",
			statement: "set nocount on; select id from t",
			valid:     true,
		},
		{
			name:      "session options alone",
			statement: "SET NOCOUNT OFF; SET TRANSACTION ISOLATION LEVEL READ COMMITTED",
			valid:     true,
		},
		{
			name:      "fmtonly listed after an allowed option",
			statement: "SET NOCOUNT, FMTONLY ON; SELECT id FROM t",
		},
		{
			name:      "an option outside the list",
			statement: "SET XACT_ABORT ON; SELECT id FROM t",
		},
		{
			name:      "statistics xml adds a result set",
			statement: "SET STATISTICS XML ON; SELECT id FROM t",
		},
		{
			name:      "showplan replaces the result",
			statement: "SET SHOWPLAN_XML ON",
		},
		{
			name:      "rowcount",
			statement: "SET ROWCOUNT 10; SELECT id FROM t",
		},
		{
			name:      "identity insert",
			statement: "SET IDENTITY_INSERT t ON",
		},
		{
			name:      "declare",
			statement: "DECLARE @n int = 1; SELECT @n",
		},
		{
			name:      "set a variable",
			statement: "SET @n = 1",
		},
		{
			name:      "isolation level before a write",
			statement: "SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED; DELETE FROM t",
		},
		{
			name:      "isolation level before select into a temp table",
			statement: "SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED; SELECT id INTO #t FROM t",
		},
		{
			name:      "select into a global temp table",
			statement: "SELECT id INTO ##t FROM t",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			valid, allQuery, err := ValidateSQLForEditor(tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.valid, valid)
			require.Equal(t, tc.allQuery, allQuery)
		})
	}
}
