package v1

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// GetIamPolicy hands the etag back inside the policy, so the round-trip a
// read-modify-write performs carries it in `policy.etag` and leaves the request
// field empty. Reading only the request field discarded it, and two admins
// editing permissions at once silently lost one edit.
func TestRequestedIamPolicyEtag(t *testing.T) {
	testCases := []struct {
		name        string
		request     *v1pb.SetIamPolicyRequest
		want        string
		wantInvalid bool
	}{
		{
			name:    "round-tripped inside the policy",
			request: &v1pb.SetIamPolicyRequest{Policy: &v1pb.IamPolicy{Etag: "1756000000000"}},
			want:    "1756000000000",
		},
		{
			name:    "set on the request",
			request: &v1pb.SetIamPolicyRequest{Etag: "1756000000000", Policy: &v1pb.IamPolicy{}},
			want:    "1756000000000",
		},
		{
			name:    "both, agreeing",
			request: &v1pb.SetIamPolicyRequest{Etag: "1756000000000", Policy: &v1pb.IamPolicy{Etag: "1756000000000"}},
			want:    "1756000000000",
		},
		{
			name:        "both, disagreeing",
			request:     &v1pb.SetIamPolicyRequest{Etag: "1756000000000", Policy: &v1pb.IamPolicy{Etag: "1756000000001"}},
			wantInvalid: true,
		},
		{
			name:    "neither, so no check is asked for",
			request: &v1pb.SetIamPolicyRequest{Policy: &v1pb.IamPolicy{}},
			want:    "",
		},
		{
			name:    "no policy at all",
			request: &v1pb.SetIamPolicyRequest{},
			want:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := require.New(t)
			etag, err := requestedIamPolicyEtag(tc.request)
			if tc.wantInvalid {
				a.Error(err)
				a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
				return
			}
			a.NoError(err)
			a.Equal(tc.want, etag)
		})
	}
}
