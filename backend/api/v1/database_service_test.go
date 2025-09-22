package v1

import (
	"fmt"
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
		input string
		want  *store.ListResourceFilter
		error *connect.Error
	}{
		{
			input: `environment == "environments/test"`,
			want: &store.ListResourceFilter{
				Where: `(COALESCE(db.environment, instance.environment) = $1)`,
				Args:  []any{"test"},
			},
		},
		{
			input: `environment == "test"`,
			error: connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid environment filter %q", "test")),
		},
		{
			input: `project == "projects/sample"`,
			want: &store.ListResourceFilter{
				Where: "(db.project = $1)",
				Args:  []any{"sample"},
			},
		},
		{
			input: `name.matches("Employee")`,
			want: &store.ListResourceFilter{
				Where: `(LOWER(db.name) LIKE '%employee%')`,
			},
		},
		{
			input: `engine in ["MYSQL", "POSTGRES"]`,
			want: &store.ListResourceFilter{
				Where: fmt.Sprintf(`(instance.metadata->>'engine' IN ('%s','%s'))`, v1pb.Engine_MYSQL.String(), v1pb.Engine_POSTGRES.String()),
				Args:  []any{},
			},
		},
		{
			input: `labels.region == "asia" && labels.tenant == "bytebase"`,
			want: &store.ListResourceFilter{
				Where: `((db.metadata->'labels'->>'region' = 'asia') AND (db.metadata->'labels'->>'tenant' = 'bytebase'))`,
				Args:  []any{},
			},
		},
		{
			input: `(labels.region == "asia" || labels.tenant == "bytebase") && exclude_unassigned == true`,
			want: &store.ListResourceFilter{
				Where: `(((db.metadata->'labels'->>'region' = 'asia') OR (db.metadata->'labels'->>'tenant' = 'bytebase')) AND (db.project != $1))`,
				Args:  []any{common.DefaultProjectID},
			},
		},
		{
			input: `labels.region in ["asia", "europe"] && labels.tenant == "bytebase"`,
			want: &store.ListResourceFilter{
				Where: `((db.metadata->'labels'->>'region' IN ('asia','europe')) AND (db.metadata->'labels'->>'tenant' = 'bytebase'))`,
				Args:  []any{},
			},
		},
	}

	for _, tc := range testCases {
		filter, err := getListDatabaseFilter(tc.input)
		if tc.error != nil {
			require.Error(t, err)
			connectErr := new(connect.Error)
			require.True(t, errors.As(err, &connectErr))
			require.Equal(t, tc.error.Message(), connectErr.Message())
			require.Equal(t, tc.error.Code(), connectErr.Code())
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.want.Where, filter.Where)
		}
	}
}
