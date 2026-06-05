package tests

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestReviewConfigAuditLog(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    "demo@example.com",
		Password: "1024bytebase",
	}))
	a.NoError(err)
	workspace := loginResp.Msg.GetUser().GetWorkspace()
	a.NotEmpty(workspace)

	reviewConfigName := common.FormatReviewConfig(generateRandomString("review"))
	reviewConfig := &v1pb.ReviewConfig{
		Name:    reviewConfigName,
		Title:   "audit review config",
		Enabled: true,
		Rules: []*v1pb.SQLReviewRule{
			{
				Type:  v1pb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT,
				Level: v1pb.SQLReviewRule_ERROR,
			},
		},
	}

	created, err := ctl.reviewConfigServiceClient.CreateReviewConfig(ctx, connect.NewRequest(&v1pb.CreateReviewConfigRequest{
		ReviewConfig: reviewConfig,
	}))
	a.NoError(err)
	a.Equal(reviewConfigName, created.Msg.Name)

	policy, err := ctl.orgPolicyServiceClient.CreatePolicy(ctx, connect.NewRequest(&v1pb.CreatePolicyRequest{
		Parent: "environments/prod",
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_TAG,
			Policy: &v1pb.Policy_TagPolicy{
				TagPolicy: &v1pb.TagPolicy{
					Tags: map[string]string{
						common.ReservedTagReviewConfig: reviewConfigName,
					},
				},
			},
		},
	}))
	a.NoError(err)

	_, err = ctl.reviewConfigServiceClient.UpdateReviewConfig(ctx, connect.NewRequest(&v1pb.UpdateReviewConfigRequest{
		ReviewConfig: &v1pb.ReviewConfig{
			Name:  reviewConfigName,
			Title: "updated audit review config",
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
	}))
	a.NoError(err)

	_, err = ctl.reviewConfigServiceClient.DeleteReviewConfig(ctx, connect.NewRequest(&v1pb.DeleteReviewConfigRequest{
		Name: reviewConfigName,
	}))
	a.NoError(err)

	tagPolicy, err := ctl.orgPolicyServiceClient.GetPolicy(ctx, connect.NewRequest(&v1pb.GetPolicyRequest{
		Name: policy.Msg.Name,
	}))
	a.NoError(err)
	a.NotNil(tagPolicy.Msg.GetTagPolicy())
	a.NotContains(tagPolicy.Msg.GetTagPolicy().Tags, common.ReservedTagReviewConfig)

	for _, method := range []string{
		"/bytebase.v1.ReviewConfigService/CreateReviewConfig",
		"/bytebase.v1.ReviewConfigService/UpdateReviewConfig",
		"/bytebase.v1.ReviewConfigService/DeleteReviewConfig",
	} {
		logs, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
			Parent:  workspace,
			Filter:  `method == "` + method + `"`,
			OrderBy: "create_time desc",
		}))
		a.NoError(err)
		a.NotEmpty(logs.Msg.AuditLogs, "%s must produce an audit entry", method)
		a.Equal(reviewConfigName, logs.Msg.AuditLogs[0].Resource)
	}
}
