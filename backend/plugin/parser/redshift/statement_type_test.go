package redshift

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetStatementTypes(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []storepb.StatementType
	}{
		{
			name:      "truncate table",
			statement: "TRUNCATE TABLE t1;",
			want: []storepb.StatementType{
				storepb.StatementType_TRUNCATE,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmts, err := base.ParseStatements(storepb.Engine_REDSHIFT, tt.statement)
			require.NoError(t, err)

			got, err := GetStatementTypes(base.ExtractASTs(stmts))
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
