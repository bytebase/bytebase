package tests

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestCollisionMetricsLicenseSeats(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	actuator, err := ctl.actuatorServiceClient.GetActuatorInfo(ctx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	a.NoError(err)
	base := actuator.Msg.UserCountInIam
	a.Greater(base, int32(0), "workspace seat count precondition")

	projectOnly, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Title:    "project-only",
			Email:    "project-only@bytebase.com",
			Password: "1024bytebase",
		},
	}))
	a.NoError(err)

	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: ctl.project.Name,
	}))
	a.NoError(err)
	member := fmt.Sprintf("user:%s", projectOnly.Msg.Email)
	policyResp.Msg.Bindings = append(policyResp.Msg.Bindings, &v1pb.Binding{
		Role:    "roles/projectDeveloper",
		Members: []string{member},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: ctl.project.Name,
		Policy:   policyResp.Msg,
	}))
	a.NoError(err)

	readback, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: ctl.project.Name,
	}))
	a.NoError(err)
	found := false
	for _, binding := range readback.Msg.Bindings {
		if binding.Role == "roles/projectDeveloper" {
			found = found || slices.Contains(binding.Members, member)
		}
	}
	a.True(found, "project IAM collision precondition")

	actuator, err = ctl.actuatorServiceClient.GetActuatorInfo(ctx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	a.NoError(err)
	a.Equal(base, actuator.Msg.UserCountInIam, "project IAM member must not occupy a workspace seat")

	resp, err := ctl.client.Get(ctl.rootURL + "/metrics")
	a.NoError(err)
	defer resp.Body.Close()
	a.Equal(http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	a.NoError(err)
	a.Regexp(regexp.MustCompile(fmt.Sprintf(`(?m)^bytebase_license_seats_used %d$`, base)), string(body))
}
