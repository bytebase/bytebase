package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

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

// TODO(p0ny): update tests for isUserReviewer
