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
		// query equality must use the same whitespace-normalized comparison as
		// query.contains so that an "approved" JIT grant whose stored SQL
		// differs from the submitted SQL only by invisible whitespace (most
		// commonly a trailing \n that Monaco's getValue() emits in the
		// request drawer but that the editor's getActiveStatement() doesn't)
		// still matches and the grant is honored at run time.
		{
			name:     "query equality strips trailing newline",
			filter:   `query == "SELECT 1\n"`,
			wantSQL:  "(btrim(regexp_replace(access_grant.payload->>'query', '\\s+', ' ', 'g')) = $1)",
			wantArgs: []any{"SELECT 1"},
			wantErr:  false,
		},
		{
			name:     "query equality strips leading whitespace",
			filter:   `query == "\n\tSELECT 1"`,
			wantSQL:  "(btrim(regexp_replace(access_grant.payload->>'query', '\\s+', ' ', 'g')) = $1)",
			wantArgs: []any{"SELECT 1"},
			wantErr:  false,
		},
		{
			name:     "query equality collapses internal whitespace runs",
			filter:   `query == "SELECT  *\n  FROM   t"`,
			wantSQL:  "(btrim(regexp_replace(access_grant.payload->>'query', '\\s+', ' ', 'g')) = $1)",
			wantArgs: []any{"SELECT * FROM t"},
			wantErr:  false,
		},
		{
			name:     "query equality treats CRLF as whitespace",
			filter:   `query == "SELECT 1\r\n"`,
			wantSQL:  "(btrim(regexp_replace(access_grant.payload->>'query', '\\s+', ' ', 'g')) = $1)",
			wantArgs: []any{"SELECT 1"},
			wantErr:  false,
		},
		// Pinned policy trade-off (intentional): the regex collapses
		// whitespace runs inside string literals and comments too. Strict
		// AST-equivalent comparison was rejected for per-engine cost. If
		// this trade-off ever needs to tighten, the right answer is engine-
		// specific lexing, not removing the normalization.
		{
			name:     "query equality also collapses whitespace inside literals (intentional)",
			filter:   `query == "SELECT 'a  b'"`,
			wantSQL:  "(btrim(regexp_replace(access_grant.payload->>'query', '\\s+', ' ', 'g')) = $1)",
			wantArgs: []any{"SELECT 'a b'"},
			wantErr:  false,
		},
		{
			name:     "query equality preserves single spaces between tokens",
			filter:   `query == "SELECT 1 FROM t"`,
			wantSQL:  "(btrim(regexp_replace(access_grant.payload->>'query', '\\s+', ' ', 'g')) = $1)",
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
