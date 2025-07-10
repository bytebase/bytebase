package v1

import (
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestIsSecretValid(t *testing.T) {
	testCases := []struct {
		item    *storepb.Secret
		wantErr bool
	}{
		// Disallow empty name.
		{
			item: &storepb.Secret{
				Name:        "",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		// Disallow empty value.
		{
			item: &storepb.Secret{
				Name:        "NAME",
				Value:       "",
				Description: "",
			},
			wantErr: true,
		},
		// Name cannot start with a number.
		{
			item: &storepb.Secret{
				Name:        "1NAME",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		// Name cannot start with the 'BYTEBASE_' prefix.
		{
			item: &storepb.Secret{
				Name:        "BYTEBASE_NAME",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		// Names can only contain alphanumeric characters ([A-Z], [0-9]) or underscores (_). Spaces are not allowed.
		{
			item: &storepb.Secret{
				Name:        "NAME WITH SPACE",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.Secret{
				Name:        "NAME-WITH-DASH",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.Secret{
				Name:        "NAME_WITH_SPECIAL_CHARACTER_©",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.Secret{
				Name:        "NAME_WITH_SPECIAL_CHARACTER_ç",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.Secret{
				Name:        "NAME_WITH_SPECIAL_CHARACTER_Ⅷ",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.Secret{
				Name:        "NAME_WITH_LOWER_CASE_a",
				Value:       "pwd",
				Description: "",
			},
			wantErr: true,
		},
		{
			item: &storepb.Secret{
				Name:        "NORMAL_NAME",
				Value:       "pwd",
				Description: "",
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		err := isSecretValid(tc.item)
		if tc.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestListDatabaseFilter(t *testing.T) {
	testCases := []struct {
		input string
		want  *store.ListResourceFilter
		error *connect.Error
	}{
		{
			input: `environment == "environments/test"`,
			want: &store.ListResourceFilter{
				Where: `(
			COALESCE(
				db.environment,
				instance.environment
			) = $1)`,
				Args: []any{"test"},
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
			input: `label == "region:asia" && label == "tenant:bytebase"`,
			want: &store.ListResourceFilter{
				Where: `((db.metadata->'labels'->>'region' = ANY($1)) AND (db.metadata->'labels'->>'tenant' = ANY($2)))`,
				Args:  []any{"asia", "bytebase"},
			},
		},
		{
			input: `(label == "region:asia" || label == "tenant:bytebase") && exclude_unassigned == true`,
			want: &store.ListResourceFilter{
				Where: `(((db.metadata->'labels'->>'region' = ANY($1)) OR (db.metadata->'labels'->>'tenant' = ANY($2))) AND (db.project != $3))`,
				Args:  []any{"asia", "bytebase", common.DefaultProjectID},
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
