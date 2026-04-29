package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetListAccessGrantFilter(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		wantSQL     string
		wantArgs    []any
		wantErr     bool
		errContains string
	}{
		{
			name:     "empty filter",
			filter:   "",
			wantSQL:  "",
			wantArgs: nil,
			wantErr:  false,
		},
		{
			name:     "query contains",
			filter:   `query.contains("SELECT * FROM users")`,
			wantSQL:  "(regexp_replace(access_grant.payload->>'query', '\\s+', ' ', 'g') ILIKE $1)",
			wantArgs: []any{"%SELECT * FROM users%"},
			wantErr:  false,
		},
		{
			name:     "query contains normalizes whitespace",
			filter:   `query.contains("SELECT   *   FROM users")`,
			wantSQL:  "(regexp_replace(access_grant.payload->>'query', '\\s+', ' ', 'g') ILIKE $1)",
			wantArgs: []any{"%SELECT * FROM users%"},
			wantErr:  false,
		},
		{
			name:        "matches is unsupported",
			filter:      `query.matches("SELECT")`,
			wantErr:     true,
			errContains: "unsupported function matches",
		},
		// query equality must trim the *same* set of characters on both
		// sides so an "approved" JIT grant whose stored SQL differs from
		// the submitted SQL only by boundary whitespace (most commonly a
		// trailing \n that Monaco's getValue() emits in the request drawer
		// but that the editor's getActiveStatement() doesn't) still
		// matches at run time. Critically, *internal* whitespace must be
		// preserved byte-for-byte: collapsing internal whitespace would
		// let "SELECT * FROM t --\nWHERE tenant=1" compare equal to
		// "SELECT * FROM t -- WHERE tenant=1", which means an approved
		// query gets stripped of its WHERE clause at run time. That's a
		// privilege escalation, not a usability nit. Boundary-only trim
		// fixes the reported customer bug without enabling that class.
		{
			name:     "query equality strips trailing newline",
			filter:   `query == "SELECT 1\n"`,
			wantSQL:  "(btrim(access_grant.payload->>'query', E' \\t\\n\\r\\v\\f') = $1)",
			wantArgs: []any{"SELECT 1"},
			wantErr:  false,
		},
		{
			name:     "query equality strips leading whitespace",
			filter:   `query == "\n\tSELECT 1"`,
			wantSQL:  "(btrim(access_grant.payload->>'query', E' \\t\\n\\r\\v\\f') = $1)",
			wantArgs: []any{"SELECT 1"},
			wantErr:  false,
		},
		{
			name:     "query equality strips trailing CRLF",
			filter:   `query == "SELECT 1\r\n"`,
			wantSQL:  "(btrim(access_grant.payload->>'query', E' \\t\\n\\r\\v\\f') = $1)",
			wantArgs: []any{"SELECT 1"},
			wantErr:  false,
		},
		{
			name:     "query equality preserves internal whitespace byte-for-byte",
			filter:   `query == "SELECT  *\n  FROM   t"`,
			wantSQL:  "(btrim(access_grant.payload->>'query', E' \\t\\n\\r\\v\\f') = $1)",
			wantArgs: []any{"SELECT  *\n  FROM   t"},
			wantErr:  false,
		},
		// Anti-collision regression for the privilege-escalation class
		// described above. The internal \n inside the SQL must survive
		// the normalization so that comment termination is preserved.
		{
			name:     "query equality preserves -- comment newline (no privilege escalation)",
			filter:   `query == "SELECT * FROM t --\nWHERE tenant=1"`,
			wantSQL:  "(btrim(access_grant.payload->>'query', E' \\t\\n\\r\\v\\f') = $1)",
			wantArgs: []any{"SELECT * FROM t --\nWHERE tenant=1"},
			wantErr:  false,
		},
		{
			name:     "query equality preserves whitespace inside string literals",
			filter:   `query == "SELECT 'a  b'"`,
			wantSQL:  "(btrim(access_grant.payload->>'query', E' \\t\\n\\r\\v\\f') = $1)",
			wantArgs: []any{"SELECT 'a  b'"},
			wantErr:  false,
		},
		{
			name:     "query equality preserves single spaces between tokens",
			filter:   `query == "SELECT 1 FROM t"`,
			wantSQL:  "(btrim(access_grant.payload->>'query', E' \\t\\n\\r\\v\\f') = $1)",
			wantArgs: []any{"SELECT 1 FROM t"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := GetListAccessGrantFilter(tt.filter)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.ErrorContains(t, err, tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.filter == "" {
				require.Nil(t, q)
				return
			}

			require.NotNil(t, q)
			sql, args, err := q.ToSQL()
			require.NoError(t, err)
			require.Equal(t, tt.wantSQL, sql)
			require.Equal(t, tt.wantArgs, args)
		})
	}
}
