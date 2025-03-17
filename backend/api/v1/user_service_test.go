package v1

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestListUserFilter(t *testing.T) {
	testCases := []struct {
		input string
		want  *store.ListResourceFilter
		error error
	}{
		{
			input: `title == "ed"`,
			error: status.Errorf(codes.InvalidArgument, "unsupport variable %q", "title"),
		},
		{
			input: `name == "ed"`,
			want: &store.ListResourceFilter{
				Where: `(principal.name = $1)`,
				Args:  []any{"ed"},
			},
		},
		{
			input: `name.matches("ED")`,
			want: &store.ListResourceFilter{
				Where: `(LOWER(principal.name) LIKE '%ed%')`,
			},
		},
		{
			input: `user_type in ["SERVICE_ACCOUNT", "USER"]`,
			want: &store.ListResourceFilter{
				Where: `(principal.type IN ($1,$2))`,
				Args:  []any{v1pb.UserType_SERVICE_ACCOUNT, v1pb.UserType_USER},
			},
		},
		{
			input: `state == "DELETED"`,
			want: &store.ListResourceFilter{
				Where: `(principal.deleted = $1)`,
				Args:  []any{true},
			},
		},
		{
			input: `project == "projects/sample-project"`,
			want: &store.ListResourceFilter{
				Where: `(TRUE)`,
			},
		},
	}

	for _, tc := range testCases {
		find := &store.FindUserMessage{}
		err := parseListUserFilter(find, tc.input)
		if tc.error != nil {
			require.Error(t, err)
			require.Equal(t, tc.error, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.want.Where, find.Filter.Where)

			if strings.HasPrefix(tc.input, "project ==") {
				require.NotNil(t, find.ProjectID)
			}
		}
	}
}
