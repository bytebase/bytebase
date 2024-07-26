package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestGetResourceFromRequest(t *testing.T) {
	tests := []struct {
		request any
		method  string
		want    *common.Resource
	}{
		{
			request: &v1pb.RemoveWebhookRequest{
				Webhook: &v1pb.Webhook{
					Name: "projects/aaa/webhooks/bbb",
				},
			},
			method: "/bytebase.v1.ProjectService/RemoveWebhook",
			want:   &common.Resource{Name: "projects/aaa/webhooks/bbb"},
		},
	}

	for _, tt := range tests {
		got := getResourceFromRequest(tt.request, tt.method)
		require.Equal(t, tt.want, got, tt.method)
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "DatabaseGroup",
			want:  "database_group",
		},
		{
			input: "User",
			want:  "user",
		},
		{
			input: "Instance",
			want:  "instance",
		},
	}

	for _, tt := range tests {
		got := toSnakeCase(tt.input)
		require.Equal(t, tt.want, got, tt.input)
	}
}
