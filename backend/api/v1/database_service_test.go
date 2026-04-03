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
			input:    `name.contains("Employee")`,
			wantSQL:  `(LOWER(db.name) LIKE $1)`,
			wantArgs: []any{"%employee%"},
		},
		{
			input:    `table.contains("user")`,
			wantSQL:  "(EXISTS (\n\t\t\t\t\t\tSELECT 1\n\t\t\t\t\t\tFROM json_array_elements(ds.metadata->'schemas') AS s,\n\t\t\t\t\t\t \t json_array_elements(s->'tables') AS t\n\t\t\t\t\t\tWHERE t->>'name' LIKE $1))",
			wantArgs: []any{"%user%"},
		},
		{
			input: `name.matches("Employee")`,
			error: connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected function matches")),
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
			wantSQL:  `(((db.metadata->'labels'->>'region' = $1 OR db.metadata->'labels'->>'tenant' = $2) AND db.project != $3 AND db.project != 'default'))`,
			wantArgs: []any{"asia", "bytebase", common.DefaultProjectID("test-workspace")},
		},
		{
			input:    `labels.region in ["asia", "europe"] && labels.tenant == "bytebase"`,
			wantSQL:  `((db.metadata->'labels'->>'region' = ANY($1) AND db.metadata->'labels'->>'tenant' = $2))`,
			wantArgs: []any{[]any{"asia", "europe"}, "bytebase"},
		},
	}

	for _, tc := range testCases {
		filterQ, err := store.GetListDatabaseFilter("test-workspace", tc.input)
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

func TestGetDatabaseMetadataFilter(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		wantSchema   *string
		wantTable    *string
		wantWildcard bool
		errContains  string
	}{
		{
			name:         "table contains",
			input:        `table.contains("user")`,
			wantTable:    ptrValue("user"),
			wantWildcard: true,
		},
		{
			name:         "schema and table contains",
			input:        `schema == "public" && table.contains("user")`,
			wantSchema:   ptrValue("public"),
			wantTable:    ptrValue("user"),
			wantWildcard: true,
		},
		{
			name:        "table matches unsupported",
			input:       `table.matches("user")`,
			errContains: "unexpected function matches",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := getDatabaseMetadataFilter(tc.input)
			if tc.errContains != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errContains)
				return
			}

			require.NoError(t, err)
			if tc.wantSchema != nil {
				require.NotNil(t, filter.schema)
				require.Equal(t, *tc.wantSchema, *filter.schema)
			}
			if tc.wantTable != nil {
				require.NotNil(t, filter.table)
				require.Equal(t, *tc.wantTable, filter.table.name)
				require.Equal(t, tc.wantWildcard, filter.table.wildcard)
			}
		})
	}
}

func ptrValue[T any](v T) *T {
	return &v
}
