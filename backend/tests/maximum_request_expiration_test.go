package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// setMaximumRequestExpiration sets the workspace-level maximum request expiration cap.
// Passing nil clears the cap.
func (ctl *controller) setMaximumRequestExpiration(ctx context.Context, d *durationpb.Duration) error {
	_, err := ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_WORKSPACE_PROFILE.String(),
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceProfile{
					WorkspaceProfile: &v1pb.WorkspaceProfileSetting{
						MaximumRequestExpiration: d,
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"value.workspace_profile.maximum_request_expiration"},
		},
	}))
	return err
}

// TestRoleGrantMaximumExpiration verifies that creating a ROLE_GRANT issue is
// validated server-side against the workspace maximum request expiration cap.
func TestRoleGrantMaximumExpiration(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Cap request expiration to 7 days.
	a.NoError(ctl.setMaximumRequestExpiration(ctx, durationpb.New(7*24*time.Hour)))

	newRoleGrantIssue := func(condition string) *connect.Request[v1pb.CreateIssueRequest] {
		return connect.NewRequest(&v1pb.CreateIssueRequest{
			Parent: ctl.project.Name,
			Issue: &v1pb.Issue{
				Title: "Request projectDeveloper role",
				Type:  v1pb.Issue_ROLE_GRANT,
				RoleGrant: &v1pb.RoleGrant{
					Role:      "roles/projectDeveloper",
					User:      ctl.principalName,
					Condition: &expr.Expr{Expression: condition},
				},
			},
		})
	}

	// Within the cap (1 day): allowed.
	withinCap := fmt.Sprintf(`request.time < timestamp("%s")`, time.Now().Add(24*time.Hour).Format(time.RFC3339))
	_, err = ctl.issueServiceClient.CreateIssue(ctx, newRoleGrantIssue(withinCap))
	a.NoError(err, "expiration within the cap should be allowed")

	// Exceeding the cap (30 days): rejected.
	overCap := fmt.Sprintf(`request.time < timestamp("%s")`, time.Now().Add(30*24*time.Hour).Format(time.RFC3339))
	_, err = ctl.issueServiceClient.CreateIssue(ctx, newRoleGrantIssue(overCap))
	a.Error(err, "expiration exceeding the cap should be rejected")
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// Missing expiration while a cap is configured: rejected (the grant would
	// otherwise be treated as unbounded, bypassing the cap).
	_, err = ctl.issueServiceClient.CreateIssue(ctx, newRoleGrantIssue(""))
	a.Error(err, "missing expiration should be rejected when a cap is configured")
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// Clearing the cap restores the unbounded behavior.
	a.NoError(ctl.setMaximumRequestExpiration(ctx, nil))
	_, err = ctl.issueServiceClient.CreateIssue(ctx, newRoleGrantIssue(overCap))
	a.NoError(err, "expiration should be unbounded when no cap is configured")
}

// TestAccessGrantMaximumExpiration verifies that creating an access grant is
// validated server-side against the workspace maximum request expiration cap.
func TestAccessGrantMaximumExpiration(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create an instance and database to target.
	instanceDir := t.TempDir()
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("inst"),
		Instance: &v1pb.Instance{
			Title:       "Test Instance",
			Engine:      v1pb.Engine_SQLITE,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{
				Type: v1pb.DataSourceType_ADMIN,
				Host: instanceDir,
				Id:   "admin",
			}},
		},
	}))
	a.NoError(err)

	dbName := generateRandomString("db")
	a.NoError(ctl.createDatabase(ctx, ctl.project, instanceResp.Msg, nil, dbName, ""))
	target := fmt.Sprintf("%s/databases/%s", instanceResp.Msg.Name, dbName)

	// Cap role expiration to 7 days.
	a.NoError(ctl.setMaximumRequestExpiration(ctx, durationpb.New(7*24*time.Hour)))

	newAccessGrant := func(exp any) *connect.Request[v1pb.CreateAccessGrantRequest] {
		ag := &v1pb.AccessGrant{
			Creator: ctl.principalName,
			Targets: []string{target},
			Query:   "SELECT 1",
			Reason:  "testing",
		}
		switch e := exp.(type) {
		case *durationpb.Duration:
			ag.Expiration = &v1pb.AccessGrant_Ttl{Ttl: e}
		case *timestamppb.Timestamp:
			ag.Expiration = &v1pb.AccessGrant_ExpireTime{ExpireTime: e}
		default:
			// No expiration set.
		}
		return connect.NewRequest(&v1pb.CreateAccessGrantRequest{
			Parent:      ctl.project.Name,
			AccessGrant: ag,
		})
	}

	// ttl within the cap (1 day): allowed.
	_, err = ctl.accessGrantServiceClient.CreateAccessGrant(ctx, newAccessGrant(durationpb.New(24*time.Hour)))
	a.NoError(err, "ttl within the cap should be allowed")

	// ttl exceeding the cap (30 days): rejected.
	_, err = ctl.accessGrantServiceClient.CreateAccessGrant(ctx, newAccessGrant(durationpb.New(30*24*time.Hour)))
	a.Error(err, "ttl exceeding the cap should be rejected")
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// expire_time exceeding the cap (30 days): rejected.
	_, err = ctl.accessGrantServiceClient.CreateAccessGrant(ctx, newAccessGrant(timestamppb.New(time.Now().Add(30*24*time.Hour))))
	a.Error(err, "expire_time exceeding the cap should be rejected")
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	// Clearing the cap restores the unbounded behavior.
	a.NoError(ctl.setMaximumRequestExpiration(ctx, nil))
	_, err = ctl.accessGrantServiceClient.CreateAccessGrant(ctx, newAccessGrant(durationpb.New(30*24*time.Hour)))
	a.NoError(err, "ttl should be unbounded when no cap is configured")
}
