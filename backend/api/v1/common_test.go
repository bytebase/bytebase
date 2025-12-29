package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestIsValidResourceID(t *testing.T) {
	tests := []struct {
		resourceID string
		want       bool
	}{
		{
			resourceID: "hello123",
			want:       true,
		},
		{
			resourceID: "hello-123",
			want:       true,
		},
		{
			resourceID: "你好",
			want:       false,
		},
		{
			resourceID: "123abc",
			want:       false,
		},
		{
			resourceID: "a1234567890123456789012345678901234567890123456789012345678901234567890",
			want:       false,
		},
	}

	for _, test := range tests {
		got := isValidResourceID(test.resourceID)
		require.Equal(t, test.want, got, test.resourceID)
	}
}

func TestParseFilter(t *testing.T) {
	testCases := []struct {
		input string
		want  []Expression
		err   error
	}{
		{
			input: `resource="environments/e1/instances/i2"`,
			want: []Expression{
				{
					Key:      "resource",
					Operator: ComparatorTypeEqual,
					Value:    "environments/e1/instances/i2",
				},
			},
		},
		{
			input: `project = "p1" && start_time>="2020-01-01T00:00:00Z" && start_time<2020-01-02T00:00:00Z`,
			want: []Expression{
				{
					Key:      "project",
					Operator: ComparatorTypeEqual,
					Value:    "p1",
				},
				{
					Key:      "start_time",
					Operator: ComparatorTypeGreaterEqual,
					Value:    "2020-01-01T00:00:00Z",
				},
				{
					Key:      "start_time",
					Operator: ComparatorTypeLess,
					Value:    "2020-01-02T00:00:00Z",
				},
			},
		},
	}

	for _, test := range testCases {
		got, err := ParseFilter(test.input)
		if test.err != nil {
			require.EqualError(t, err, test.err.Error())
		} else {
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		}
	}
}

func TestIsValidateOnlyRequest(t *testing.T) {
	tests := []struct {
		name    string
		request any
		want    bool
	}{
		{
			name:    "nil request",
			request: nil,
			want:    false,
		},
		{
			name:    "non-proto request",
			request: "not a proto",
			want:    false,
		},
		{
			name: "request without validate_only field",
			request: &v1pb.GetInstanceRequest{
				Name: "instances/test",
			},
			want: false,
		},
		{
			name: "CreateInstanceRequest with validate_only=false",
			request: &v1pb.CreateInstanceRequest{
				InstanceId:   "test",
				Instance:     &v1pb.Instance{},
				ValidateOnly: false,
			},
			want: false,
		},
		{
			name: "CreateInstanceRequest with validate_only=true",
			request: &v1pb.CreateInstanceRequest{
				InstanceId:   "test",
				Instance:     &v1pb.Instance{},
				ValidateOnly: true,
			},
			want: true,
		},
		{
			name: "UpdateSettingRequest with validate_only=true",
			request: &v1pb.UpdateSettingRequest{
				Setting:      &v1pb.Setting{},
				ValidateOnly: true,
			},
			want: true,
		},
		{
			name: "AddDataSourceRequest with validate_only=true",
			request: &v1pb.AddDataSourceRequest{
				Name:         "instances/test",
				DataSource:   &v1pb.DataSource{},
				ValidateOnly: true,
			},
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := isValidateOnlyRequest(test.request)
			require.Equal(t, test.want, got)
		})
	}
}
