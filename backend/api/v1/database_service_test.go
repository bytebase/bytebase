package v1

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestListDatabaseFilter(t *testing.T) {
	testCases := []struct {
		input    string
		wantSQL  string
		wantArgs []any
		error    *connect.Error
	}{
		{
			input:    `environment == "environments/test"`,
			wantSQL:  `(COALESCE(db.environment, instance.environment) = $1)`,
			wantArgs: []any{"test"},
		},
		{
			input: `environment == "test"`,
			error: connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid environment filter %q", "test")),
		},
		{
			input:    `project == "projects/sample"`,
			wantSQL:  `(db.project = $1)`,
			wantArgs: []any{"sample"},
		},
		{
			input:    `name.matches("Employee")`,
			wantSQL:  `(LOWER(db.name) LIKE $1)`,
			wantArgs: []any{"%employee%"},
		},
		{
			input:    `engine in ["MYSQL", "POSTGRES"]`,
			wantSQL:  `(instance.metadata->>'engine' = ANY($1))`,
			wantArgs: []any{[]any{v1pb.Engine_MYSQL.String(), v1pb.Engine_POSTGRES.String()}},
		},
		{
			input:    `labels.region == "asia" && labels.tenant == "bytebase"`,
			wantSQL:  `((db.metadata->'labels'->>'region' = $1 AND db.metadata->'labels'->>'tenant' = $2))`,
			wantArgs: []any{"asia", "bytebase"},
		},
		{
			input:    `(labels.region == "asia" || labels.tenant == "bytebase") && exclude_unassigned == true`,
			wantSQL:  `(((db.metadata->'labels'->>'region' = $1 OR db.metadata->'labels'->>'tenant' = $2) AND db.project != $3))`,
			wantArgs: []any{"asia", "bytebase", common.DefaultProjectID},
		},
		{
			input:    `labels.region in ["asia", "europe"] && labels.tenant == "bytebase"`,
			wantSQL:  `((db.metadata->'labels'->>'region' = ANY($1) AND db.metadata->'labels'->>'tenant' = $2))`,
			wantArgs: []any{[]any{"asia", "europe"}, "bytebase"},
		},
	}

	for _, tc := range testCases {
		filterQ, err := store.GetListDatabaseFilter(tc.input)
		if tc.error != nil {
			require.Error(t, err)
			require.Equal(t, tc.error.Message(), err.Error())
		} else {
			require.NoError(t, err)
			sql, args, err := filterQ.ToSQL()
			require.NoError(t, err)
			require.Equal(t, tc.wantSQL, sql)
			require.Equal(t, tc.wantArgs, args)
		}
	}
}
