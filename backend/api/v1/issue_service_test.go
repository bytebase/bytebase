package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestConvertToApprovalNode(t *testing.T) {
	tests := []struct {
		node *storepb.ApprovalNode
		want *v1pb.ApprovalNode
	}{
		{
			node: &storepb.ApprovalNode{
				Type: storepb.ApprovalNode_ANY_IN_GROUP,
				Payload: &storepb.ApprovalNode_GroupValue_{
					GroupValue: storepb.ApprovalNode_WORKSPACE_DBA,
				},
			},
			want: &v1pb.ApprovalNode{
				Type: v1pb.ApprovalNode_ANY_IN_GROUP,
				Payload: &v1pb.ApprovalNode_GroupValue_{
					GroupValue: v1pb.ApprovalNode_WORKSPACE_DBA,
				},
			},
		},
	}

	a := require.New(t)
	for _, test := range tests {
		got := convertToApprovalNode(test.node)
		a.Equal(test.want, got)
	}
}

func TestCanUserApproveStep(t *testing.T) {
	tests := []struct {
		step   *storepb.ApprovalStep
		user   *store.UserMessage
		policy *store.IAMPolicyMessage
		want   bool
	}{
		{
			step: &storepb.ApprovalStep{
				Type: storepb.ApprovalStep_ANY,
				Nodes: []*storepb.ApprovalNode{
					{
						Type: storepb.ApprovalNode_ANY_IN_GROUP,
						Payload: &storepb.ApprovalNode_GroupValue_{
							GroupValue: storepb.ApprovalNode_WORKSPACE_DBA,
						},
					},
				},
			},
			user: &store.UserMessage{
				ID:   1,
				Role: api.Developer,
			},
			policy: &store.IAMPolicyMessage{
				Bindings: []*store.PolicyBinding{
					{
						Role: api.Developer,
						Members: []*store.UserMessage{
							{
								ID:   1,
								Role: api.Developer,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			step: &storepb.ApprovalStep{
				Type: storepb.ApprovalStep_ANY,
				Nodes: []*storepb.ApprovalNode{
					{
						Type: storepb.ApprovalNode_ANY_IN_GROUP,
						Payload: &storepb.ApprovalNode_GroupValue_{
							GroupValue: storepb.ApprovalNode_WORKSPACE_DBA,
						},
					},
				},
			},
			user: &store.UserMessage{
				ID:   1,
				Role: api.DBA,
			},
			policy: &store.IAMPolicyMessage{
				Bindings: []*store.PolicyBinding{
					{
						Role: api.Developer,
						Members: []*store.UserMessage{
							{
								ID:   1,
								Role: api.Developer,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			step: &storepb.ApprovalStep{
				Type: storepb.ApprovalStep_ANY,
				Nodes: []*storepb.ApprovalNode{
					{
						Type: storepb.ApprovalNode_ANY_IN_GROUP,
						Payload: &storepb.ApprovalNode_Role{
							Role: "roles/ProjectDBA",
						},
					},
				},
			},
			user: &store.UserMessage{
				ID:   1,
				Role: api.DBA,
			},
			policy: &store.IAMPolicyMessage{
				Bindings: []*store.PolicyBinding{
					{
						Role: api.Developer,
						Members: []*store.UserMessage{
							{
								ID:   1,
								Role: api.Developer,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			step: &storepb.ApprovalStep{
				Type: storepb.ApprovalStep_ANY,
				Nodes: []*storepb.ApprovalNode{
					{
						Type: storepb.ApprovalNode_ANY_IN_GROUP,
						Payload: &storepb.ApprovalNode_Role{
							Role: "roles/ProjectDBA",
						},
					},
				},
			},
			user: &store.UserMessage{
				ID:   1,
				Role: api.DBA,
			},
			policy: &store.IAMPolicyMessage{
				Bindings: []*store.PolicyBinding{
					{
						Role: api.Developer,
						Members: []*store.UserMessage{
							{
								ID:   1,
								Role: api.Developer,
							},
						},
					},
					{
						Role: "ProjectDBA",
						Members: []*store.UserMessage{
							{
								ID:   1,
								Role: api.Developer,
							},
						},
					},
				},
			},
			want: true,
		},
	}

	a := require.New(t)
	for _, test := range tests {
		got, err := isUserReviewer(test.step, test.user, test.policy)
		a.NoError(err)
		a.Equal(test.want, got)
	}
}
