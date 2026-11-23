package redis

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestQueryValidatorRegistration(t *testing.T) {
	require.True(t, base.HasQueryValidator(storepb.Engine_REDIS))

	tests := []struct {
		statement string
		readOnly  bool
		allQuery  bool
		wantErr   bool
	}{
		{statement: "GET hello", readOnly: true, allQuery: true},
		{statement: "MEMORY USAGE hello", readOnly: true, allQuery: true},
		{statement: "JSON.GET doc $.name", readOnly: true, allQuery: true},
		{statement: "SET hello 1", readOnly: false, allQuery: false},
		{statement: "SELECT 3\nKEYS *", readOnly: false, allQuery: false},
		{statement: `GET "unclosed`, wantErr: true},
	}

	for _, tc := range tests {
		gotReadOnly, gotAllQuery, err := base.ValidateSQLForEditor(storepb.Engine_REDIS, tc.statement)
		if tc.wantErr {
			require.Error(t, err, tc.statement)
			continue
		}
		require.NoError(t, err, tc.statement)
		require.Equal(t, tc.readOnly, gotReadOnly, tc.statement)
		require.Equal(t, tc.allQuery, gotAllQuery, tc.statement)
	}
}
