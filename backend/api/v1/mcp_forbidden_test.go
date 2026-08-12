package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// forbiddenProceduresFromDescriptors reads the FORBIDDEN class straight off
// the compiled descriptors — the same place getAuthContext reads it — so the
// assertions below are against the annotations themselves, not against a Go
// copy of them that could agree with itself while the protos say otherwise.
func forbiddenProceduresFromDescriptors(t *testing.T) map[string]v1pb.MCPForbiddenReason {
	t.Helper()
	found := map[string]v1pb.MCPForbiddenReason{}
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Package() != "bytebase.v1" {
			return true
		}
		services := fd.Services()
		for i := range services.Len() {
			sd := services.Get(i)
			methods := sd.Methods()
			for j := range methods.Len() {
				md := methods.Get(j)
				class, ok := proto.GetExtension(md.Options(), v1pb.E_McpMethodClass).(v1pb.MCPMethodClass)
				require.True(t, ok, "method %s carries a malformed mcp_method_class", md.FullName())
				reason, ok := proto.GetExtension(md.Options(), v1pb.E_McpForbiddenReason).(v1pb.MCPForbiddenReason)
				require.True(t, ok, "method %s carries a malformed mcp_forbidden_reason", md.FullName())
				if class == v1pb.MCPMethodClass_FORBIDDEN {
					found[fmt.Sprintf("/%s/%s", sd.FullName(), md.Name())] = reason
				} else {
					// A reason without the class denies nothing and would read,
					// to anyone grepping for it, as a method that is refused.
					require.Equal(t, v1pb.MCPForbiddenReason_MCP_FORBIDDEN_REASON_UNSPECIFIED, reason,
						"method %s records why it is forbidden but is not classified FORBIDDEN", md.FullName())
				}
			}
		}
		return true
	})
	return found
}

// TestForbiddenClassMembership pins which RPCs are annotated FORBIDDEN.
// Membership is a security decision, so adding or removing one has to be
// deliberate: this list is the second signature on that decision.
func TestForbiddenClassMembership(t *testing.T) {
	want := []string{
		v1connect.AuthServiceLoginProcedure,
		v1connect.AuthServiceSignupProcedure,
		v1connect.AuthServiceExchangeTokenProcedure,
		v1connect.AuthServiceRefreshProcedure,
		v1connect.AuthServiceLogoutProcedure,
		v1connect.AuthServiceRequestPasswordResetProcedure,
		v1connect.AuthServiceResetPasswordProcedure,
		v1connect.AuthServiceSendEmailLoginCodeProcedure,
		v1connect.AuthServiceSwitchWorkspaceProcedure,
		v1connect.UserServiceUpdateUserProcedure,
		v1connect.WorkspaceServiceLeaveWorkspaceProcedure,
		v1connect.WorkspaceServiceDeleteWorkspaceProcedure,
		v1connect.ServiceAccountServiceCreateServiceAccountProcedure,
		v1connect.ServiceAccountServiceUpdateServiceAccountProcedure,
		v1connect.WorkspaceServiceRotateDirectorySyncTokenProcedure,
		v1connect.UserServiceCreateUserProcedure,
		v1connect.IdentityProviderServiceCreateIdentityProviderProcedure,
		v1connect.IdentityProviderServiceUpdateIdentityProviderProcedure,
		v1connect.IdentityProviderServiceTestIdentityProviderProcedure,
		v1connect.WorkloadIdentityServiceCreateWorkloadIdentityProcedure,
		v1connect.WorkloadIdentityServiceUpdateWorkloadIdentityProcedure,
		v1connect.SettingServiceTestEmailSettingProcedure,
		v1connect.UserServiceUpdateEmailProcedure,
		v1connect.SettingServiceUpdateSettingProcedure,
	}
	got := forbiddenProceduresFromDescriptors(t)

	for _, procedure := range want {
		require.Contains(t, got, procedure, "%s must be annotated mcp_method_class = FORBIDDEN", procedure)
	}
	require.Len(t, got, len(want),
		"an RPC was annotated FORBIDDEN without being added here; membership must be an explicit decision")

	// Which mechanism each method is refused for, read off the annotations
	// rather than off a Go copy of them. The reason has to describe what the
	// method does, not merely be present: Logout destroys a session rather than
	// issuing a credential, and a denial that says otherwise teaches the next
	// reader something false. These are near-synonyms in English and are not
	// near-synonyms in what they tell an operator, so every method is pinned.
	wantReason := map[string]v1pb.MCPForbiddenReason{
		v1connect.AuthServiceLoginProcedure:                              v1pb.MCPForbiddenReason_MINTS_CREDENTIAL,
		v1connect.AuthServiceSignupProcedure:                             v1pb.MCPForbiddenReason_MINTS_CREDENTIAL,
		v1connect.AuthServiceExchangeTokenProcedure:                      v1pb.MCPForbiddenReason_MINTS_CREDENTIAL,
		v1connect.AuthServiceRefreshProcedure:                            v1pb.MCPForbiddenReason_MINTS_CREDENTIAL,
		v1connect.AuthServiceSwitchWorkspaceProcedure:                    v1pb.MCPForbiddenReason_MINTS_CREDENTIAL,
		v1connect.AuthServiceRequestPasswordResetProcedure:               v1pb.MCPForbiddenReason_RESETS_CREDENTIAL,
		v1connect.AuthServiceResetPasswordProcedure:                      v1pb.MCPForbiddenReason_RESETS_CREDENTIAL,
		v1connect.AuthServiceSendEmailLoginCodeProcedure:                 v1pb.MCPForbiddenReason_RESETS_CREDENTIAL,
		v1connect.UserServiceUpdateUserProcedure:                         v1pb.MCPForbiddenReason_TAKES_OVER_ACCOUNT,
		v1connect.AuthServiceLogoutProcedure:                             v1pb.MCPForbiddenReason_ENDS_SESSION,
		v1connect.WorkspaceServiceLeaveWorkspaceProcedure:                v1pb.MCPForbiddenReason_ENDS_MEMBERSHIP,
		v1connect.WorkspaceServiceDeleteWorkspaceProcedure:               v1pb.MCPForbiddenReason_ENDS_MEMBERSHIP,
		v1connect.ServiceAccountServiceCreateServiceAccountProcedure:     v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.ServiceAccountServiceUpdateServiceAccountProcedure:     v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.WorkspaceServiceRotateDirectorySyncTokenProcedure:      v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.UserServiceCreateUserProcedure:                         v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.UserServiceUpdateEmailProcedure:                        v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.IdentityProviderServiceCreateIdentityProviderProcedure: v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.IdentityProviderServiceUpdateIdentityProviderProcedure: v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.IdentityProviderServiceTestIdentityProviderProcedure:   v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.WorkloadIdentityServiceCreateWorkloadIdentityProcedure: v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.WorkloadIdentityServiceUpdateWorkloadIdentityProcedure: v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.SettingServiceTestEmailSettingProcedure:                v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.SettingServiceUpdateSettingProcedure:                   v1pb.MCPForbiddenReason_REWRITES_SESSION_BOUNDARY,
	}
	require.Equal(t, wantReason, got,
		"every FORBIDDEN method records the mechanism it is refused for, and this is the second signature on it")

	// Wording exists for every mechanism in use. A missing sentence is a UX
	// regression, not a security one — the class annotation is what denies —
	// so it is asserted separately from membership, and it is now one check per
	// mechanism rather than one per method.
	for procedure, reason := range got {
		require.Contains(t, mcpForbiddenReasons, reason,
			"%s is refused for %v, which has no sentence to say so", procedure, reason)
	}
}

// TestForbiddenClassLeavesReadsAlone pins the other half of the second batch's
// ruling, which is the half a later change could quietly widen: the reads of
// the services whose writes were just forbidden stay UNCLASSIFIED, and so stay
// discoverable and callable.
//
// They carry no credential. Every identity-provider read goes through the one
// conversion that blanks the client secret and the LDAP bind password
// (idp_service.go, three "SECURITY: We do not expose" lines), service-account
// reads never populate ServiceKey (only create and the rotation branch do), and
// workload-identity reads return issuer, audience and subject pattern —
// configuration, not a secret. ListIdentityProviders is additionally
// allow_without_credential, the endpoint the login page calls unauthenticated,
// so forbidding it would cost an agent legitimate work while protecting nothing
// an anonymous client could not already read.
func TestForbiddenClassLeavesReadsAlone(t *testing.T) {
	got := forbiddenProceduresFromDescriptors(t)
	for _, procedure := range []string{
		v1connect.IdentityProviderServiceGetIdentityProviderProcedure,
		v1connect.IdentityProviderServiceListIdentityProvidersProcedure,
		v1connect.WorkloadIdentityServiceGetWorkloadIdentityProcedure,
		v1connect.WorkloadIdentityServiceListWorkloadIdentitiesProcedure,
		v1connect.ServiceAccountServiceGetServiceAccountProcedure,
		v1connect.ServiceAccountServiceListServiceAccountsProcedure,
	} {
		require.NotContains(t, got, procedure,
			"%s is a read that carries no credential; forbidding it is scope this batch did not take", procedure)
	}
}

// TestInternalMCPForbiddenInterceptor pins what the interceptor does with a
// classification once the auth interceptor has resolved one.
func TestInternalMCPForbiddenInterceptor(t *testing.T) {
	interceptor := NewInternalMCPForbiddenInterceptor()

	invoke := func(ctx context.Context, procedure string) (bool, error) {
		dispatched := false
		next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			dispatched = true
			return connect.NewResponse(&v1pb.User{}), nil
		}
		req := &specRequest{
			AnyRequest: connect.NewRequest(&v1pb.GetUserRequest{}),
			procedure:  procedure,
		}
		_, err := interceptor.WrapUnary(next)(ctx, req)
		return dispatched, err
	}
	withClass := func(class v1pb.MCPMethodClass, reason v1pb.MCPForbiddenReason) context.Context {
		return context.WithValue(context.Background(), common.AuthContextKey,
			&common.AuthContext{MCPMethodClass: class, MCPForbiddenReason: reason})
	}

	for procedure, reason := range forbiddenProceduresFromDescriptors(t) {
		t.Run(procedure, func(t *testing.T) {
			dispatched, err := invoke(withClass(v1pb.MCPMethodClass_FORBIDDEN, reason), procedure)
			require.Error(t, err, "a FORBIDDEN method must never reach its handler")
			require.False(t, dispatched, "the denial must happen before dispatch, so no handler side effect can land")
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
			require.Contains(t, err.Error(), procedure, "the message must name the method the agent called")
			require.Contains(t, err.Error(), mcpForbiddenReasons[reason],
				"the message must name why, so the agent can act on it")
		})
	}

	// A method classified FORBIDDEN whose reason this build does not recognize
	// is still refused — the class is what denies, and the wording degrades to
	// the generic sentence rather than the denial degrading to nothing. This is
	// the rolling-upgrade case: an old replica meeting a reason value added
	// after it was built.
	t.Run("unknown reason still denies", func(t *testing.T) {
		dispatched, err := invoke(withClass(v1pb.MCPMethodClass_FORBIDDEN, v1pb.MCPForbiddenReason(9999)),
			v1connect.AuthServiceLoginProcedure)
		require.Error(t, err)
		require.False(t, dispatched)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
		require.Contains(t, err.Error(), reasonForbiddenClass)
	})

	// Only FORBIDDEN is enforced in this phase: an unclassified method is
	// served exactly as before, and the serving classes 1b-2 will select
	// between are not gated here yet.
	for name, class := range map[string]v1pb.MCPMethodClass{
		"unclassified": v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED,
		"read":         v1pb.MCPMethodClass_READ,
		"write":        v1pb.MCPMethodClass_WRITE,
	} {
		t.Run("dispatched: "+name, func(t *testing.T) {
			dispatched, err := invoke(
				withClass(class, v1pb.MCPForbiddenReason_MCP_FORBIDDEN_REASON_UNSPECIFIED),
				v1connect.UserServiceGetUserProcedure)
			require.NoError(t, err)
			require.True(t, dispatched)
		})
	}

	t.Run("no auth context fails closed", func(t *testing.T) {
		dispatched, err := invoke(context.Background(), v1connect.AuthServiceLoginProcedure)
		require.Error(t, err, "without a resolved classification the interceptor must not guess")
		require.False(t, dispatched)
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	})
}
