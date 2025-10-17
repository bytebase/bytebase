package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractReturnQueryStatements(t *testing.T) {
	tests := []struct {
		name    string
		asBody  string
		want    []string
		wantErr bool
	}{
		{
			name: "single RETURN QUERY",
			asBody: `BEGIN
  RETURN QUERY SELECT id, name FROM users WHERE active = true;
END`,
			want: []string{
				"SELECT id, name FROM users WHERE active = true",
			},
		},
		{
			name: "multiple RETURN QUERY",
			asBody: `BEGIN
  RETURN QUERY SELECT id, name FROM users WHERE active = true;
  RETURN QUERY SELECT id, name FROM users WHERE admin = true;
END`,
			want: []string{
				"SELECT id, name FROM users WHERE active = true",
				"SELECT id, name FROM users WHERE admin = true",
			},
		},
		{
			name: "RETURN QUERY with complex SELECT",
			asBody: `BEGIN
  RETURN QUERY
    SELECT u.id, u.name, p.phone
    FROM users u
    JOIN profiles p ON u.id = p.user_id
    WHERE u.active = true;
END`,
			want: []string{
				"SELECT u.id, u.name, p.phone\n    FROM users u\n    JOIN profiles p ON u.id = p.user_id\n    WHERE u.active = true",
			},
		},
		{
			name: "RETURN QUERY in IF block",
			asBody: `BEGIN
  IF some_condition THEN
    RETURN QUERY SELECT id, name FROM users WHERE active = true;
  ELSE
    RETURN QUERY SELECT id, name FROM users WHERE admin = true;
  END IF;
END`,
			want: []string{
				"SELECT id, name FROM users WHERE active = true",
				"SELECT id, name FROM users WHERE admin = true",
			},
		},
		{
			name: "RETURN without QUERY (should be ignored)",
			asBody: `BEGIN
  RETURN 1;
END`,
			want: []string{},
		},
		{
			name: "RETURN NEXT (should be ignored)",
			asBody: `BEGIN
  FOR r IN SELECT id, name FROM users LOOP
    RETURN NEXT r;
  END LOOP;
END`,
			want: []string{},
		},
		{
			name: "mixed RETURN QUERY and RETURN NEXT",
			asBody: `BEGIN
  RETURN QUERY SELECT id, name FROM users WHERE active = true;
  FOR r IN SELECT id, name FROM admins LOOP
    RETURN NEXT r;
  END LOOP;
END`,
			want: []string{
				"SELECT id, name FROM users WHERE active = true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractReturnQueryStatements(tt.asBody)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "extracted SQL mismatch")
		})
	}
}
