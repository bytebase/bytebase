package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/api/v1alpha"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestConvertToApprovalNode(t *testing.T) {
	tests := []struct {
		node *storepb.ApprovalNode
		want *v1pb.ApprovalNode
	}{
		{
			node: &storepb.ApprovalNode{
				Type: storepb.ApprovalNode_ANY_IN_GROUP,
				Role: common.FormatRole(base.WorkspaceDBA.String()),
			},
			want: &v1pb.ApprovalNode{
				Type: v1pb.ApprovalNode_ANY_IN_GROUP,
				Role: common.FormatRole(base.WorkspaceDBA.String()),
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
