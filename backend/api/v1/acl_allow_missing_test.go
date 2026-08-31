package v1

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bytebase/bytebase/backend/common/permission"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// An allow_missing Update creates the resource when it is absent, so it needs
// the create permission too. Two mechanisms supply that check, each failing
// silently:
//
//   - The ACL interceptor covers IAM methods, deriving the permission by
//     rewriting ".update" to ".create". That guess can leave the string
//     unchanged (re-checking what was already checked) or invent one no role
//     holds (denying every caller, as the dead flag on UpdateDatabaseCatalog did).
//   - CUSTOM methods it does not cover at all, and a handler that forgets to
//     check leaves nothing to notice — how UpdateGroup shipped an unchecked
//     create path.
//
// So each allow_missing RPC is listed with what authorizes its create path.
// Gaining or losing the flag fails the build.
var allowMissingCreateChecks = map[string]allowMissingCreateCheck{
	// Interceptor-enforced: the permission the rewrite must produce. The test
	// proves it exists and differs from the declared one.
	"bytebase.v1.InstanceService.UpdateInstance":                 {createPermission: permission.InstancesCreate},
	"bytebase.v1.InstanceService.UpdateDataSource":               {createPermission: permission.InstancesCreate},
	"bytebase.v1.RoleService.UpdateRole":                         {createPermission: permission.RolesCreate},
	"bytebase.v1.IssueService.UpdateIssue":                       {createPermission: permission.IssuesCreate},
	"bytebase.v1.IssueService.UpdateIssueComment":                {createPermission: permission.IssueCommentsCreate},
	"bytebase.v1.ReviewConfigService.UpdateReviewConfig":         {createPermission: permission.ReviewConfigsCreate},
	"bytebase.v1.IdentityProviderService.UpdateIdentityProvider": {createPermission: permission.IdentityProvidersCreate},
	"bytebase.v1.DatabaseGroupService.UpdateDatabaseGroup":       {createPermission: permission.DatabaseGroupsCreate},
	"bytebase.v1.ProjectService.UpdateProject":                   {createPermission: permission.ProjectsCreate},
	"bytebase.v1.ProjectService.UpdateWebhook":                   {createPermission: permission.ProjectsCreate},
	// Nested: the flag rides on the sub-request, which the interceptor reads only
	// because hasAllowMissingEnabled names this type by hand. Proven, not assumed.
	"bytebase.v1.InstanceService.BatchUpdateInstances": {createPermission: permission.InstancesCreate},

	// Handler-enforced: CUSTOM auth, so the note names the call that checks.
	"bytebase.v1.OrgPolicyService.UpdatePolicy": {
		handlerCheck: "org_policy_service.go createPolicyMessage -> checkPolicyPermission(PoliciesCreate, or the MASKING_RULE/MASKING_EXEMPTION variant)",
	},
	"bytebase.v1.PlanService.UpdatePlan": {
		handlerCheck: "plan_service.go UpdatePlan checks permission.PlansCreate before delegating to CreatePlan",
	},
	"bytebase.v1.GroupService.UpdateGroup": {
		handlerCheck: "group_service.go UpdateGroup checks permission.GroupsCreate before delegating to CreateGroup",
	},
	"bytebase.v1.UserService.UpdateUser": {
		handlerCheck: "user_service.go UpdateUser checks permission.UsersCreate before delegating to CreateUser",
	},
	"bytebase.v1.SettingService.UpdateSetting": {
		handlerCheck: "setting_service.go UpdateSetting checks the bb.settings.set family before the existence lookup; a setting has one write permission covering create and update",
	},
}

// allowMissingCreateCheck names how one RPC authorizes its create path. Exactly
// one field is set.
type allowMissingCreateCheck struct {
	createPermission string
	handlerCheck     string
}

// TestAllowMissingCreatePermission fails when an allow_missing create path is
// authorized by nobody, or by a permission that cannot deny or cannot grant.
func TestAllowMissingCreatePermission(t *testing.T) {
	found := allowMissingMethods(t)

	var undeclared []string
	for name, method := range found {
		check, ok := allowMissingCreateChecks[name]
		if !ok {
			shape := "allow_missing"
			if method.nested {
				shape = "a nested allow_missing sub-request, which hasAllowMissingEnabled reads only for the types it names by hand"
			}
			undeclared = append(undeclared, fmt.Sprintf("%s takes %s (auth_method=%s, permission=%q)", name, shape, method.authMethod, method.permission))
			continue
		}

		if check.handlerCheck != "" {
			require.Empty(t, check.createPermission, "%s lists both mechanisms; set exactly one", name)
			require.Equal(t, v1pb.AuthMethod_CUSTOM, method.authMethod,
				"%s is listed as handler-enforced but is %s-authed, so the interceptor covers it — list its derived create permission instead", name, method.authMethod)
			continue
		}

		require.NotEmpty(t, check.createPermission, "%s lists neither mechanism; set exactly one", name)
		require.Equal(t, v1pb.AuthMethod_IAM, method.authMethod,
			"%s is listed as interceptor-enforced but is %s-authed; doIAMPermissionCheck returns true for every non-IAM method, so nothing is verified", name, method.authMethod)
		require.NotEmpty(t, method.permission,
			"%s declares no permission, so the interceptor derives the empty string and denies every caller", name)

		derived := strings.Replace(method.permission, ".update", ".create", 1)
		require.NotEqual(t, method.permission, derived,
			"%s declares %q, which the interceptor's .update -> .create rewrite leaves unchanged, so the secondary check re-checks the permission already checked", name, method.permission)
		require.True(t, permission.Exist(derived),
			"%s derives %q, which is not in permission.yaml; CheckPermission never matches an unknown string, so allow_missing=true denies every caller", name, derived)
		require.Equal(t, check.createPermission, derived,
			"%s derives a different create permission than the one listed here", name)

		requireInterceptorSeesAllowMissing(t, name, method)
	}

	var stale []string
	for name := range allowMissingCreateChecks {
		if _, ok := found[name]; !ok {
			stale = append(stale, name)
		}
	}

	slices.Sort(undeclared)
	slices.Sort(stale)
	require.Empty(t, undeclared, "these RPCs take allow_missing but no one has said what authorizes the create path:\n  %s", strings.Join(undeclared, "\n  "))
	require.Empty(t, stale, "these listed RPCs no longer take allow_missing; drop the entries:\n  %s", strings.Join(stale, "\n  "))
}

type allowMissingMethod struct {
	authMethod v1pb.AuthMethod
	permission string
	// nested: the flag is on a sub-request, the BatchUpdateInstances shape.
	nested bool
	input  protoreflect.FullName
}

// allowMissingMethods returns every bytebase.v1 RPC carrying allow_missing,
// keyed by full method name.
func allowMissingMethods(t *testing.T) map[string]allowMissingMethod {
	t.Helper()
	methods := map[string]allowMissingMethod{}
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Package() != "bytebase.v1" {
			return true
		}
		services := fd.Services()
		for i := range services.Len() {
			sd := services.Get(i)
			descriptors := sd.Methods()
			for j := range descriptors.Len() {
				md := descriptors.Get(j)
				direct, nested := allowMissingShape(md.Input())
				if !direct && !nested {
					continue
				}
				method := allowMissingMethod{nested: !direct, input: md.Input().FullName()}
				if options, ok := md.Options().(*descriptorpb.MethodOptions); ok {
					if p, ok := proto.GetExtension(options, v1pb.E_Permission).(string); ok {
						method.permission = p
					}
					if a, ok := proto.GetExtension(options, v1pb.E_AuthMethod).(v1pb.AuthMethod); ok {
						method.authMethod = a
					}
				}
				methods[string(sd.FullName())+"."+string(md.Name())] = method
			}
		}
		return true
	})
	require.NotEmpty(t, methods, "no allow_missing RPCs found; the descriptor walk is broken, not the API")
	return methods
}

// requireInterceptorSeesAllowMissing sets allow_missing on the real request and
// asserts hasAllowMissingEnabled reads it. This is the point for nested shapes:
// the interceptor reaches a sub-request only via its hand-written type switch,
// so a batch RPC missing from that switch reports false here.
func requireInterceptorSeesAllowMissing(t *testing.T, name string, method allowMissingMethod) {
	t.Helper()

	mt, err := protoregistry.GlobalTypes.FindMessageByName(method.input)
	require.NoError(t, err, "%s: cannot resolve request type %s", name, method.input)
	msg := mt.New()

	if method.nested {
		fd := nestedAllowMissingField(msg.Descriptor())
		require.NotNil(t, fd, "%s: no nested allow_missing field", name)
		sub := msg.Mutable(fd)
		if fd.IsList() {
			sub = sub.List().AppendMutable()
		}
		subMsg := sub.Message()
		subMsg.Set(subMsg.Descriptor().Fields().ByName("allow_missing"), protoreflect.ValueOfBool(true))
	} else {
		msg.Set(msg.Descriptor().Fields().ByName("allow_missing"), protoreflect.ValueOfBool(true))
	}

	require.True(t, hasAllowMissingEnabled(msg.Interface()),
		"%s sets allow_missing but hasAllowMissingEnabled does not see it, so the create permission is never checked; add the request type to its type switch", name)
}

// nestedAllowMissingField returns the sub-request field carrying allow_missing,
// or nil. allowMissingShape defers to it so the two cannot disagree.
func nestedAllowMissingField(md protoreflect.MessageDescriptor) protoreflect.FieldDescriptor {
	fields := md.Fields()
	for i := range fields.Len() {
		fd := fields.Get(i)
		if fd.Kind() != protoreflect.MessageKind || fd.IsMap() {
			continue
		}
		if hasAllowMissingField(fd.Message()) {
			return fd
		}
	}
	return nil
}

// allowMissingShape reports how a request carries allow_missing: directly, or on
// a sub-request one level down. Listing both is what surfaces a new batch RPC
// the interceptor's type switch does not name.
func allowMissingShape(md protoreflect.MessageDescriptor) (direct, nested bool) {
	if hasAllowMissingField(md) {
		return true, false
	}
	return false, nestedAllowMissingField(md) != nil
}

// hasAllowMissingField mirrors hasAllowMissingEnabled's lookup: a bool named
// exactly allow_missing.
func hasAllowMissingField(md protoreflect.MessageDescriptor) bool {
	fd := md.Fields().ByName("allow_missing")
	return fd != nil && fd.Kind() == protoreflect.BoolKind
}
