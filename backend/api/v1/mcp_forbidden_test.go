package v1

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
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

// mcpClassification is one v1 RPC's MCP annotations, as the compiled
// descriptors carry them.
type mcpClassification struct {
	procedure       string
	class           v1pb.MCPMethodClass
	forbiddenReason v1pb.MCPForbiddenReason
	exclusionReason v1pb.MCPExclusionReason
	permission      string
}

// mcpClassificationsFromDescriptors reads every v1 RPC's classification off the
// compiled descriptors — the same place getAuthContext reads it — so everything
// below is asserted against the annotations themselves, not against a Go copy
// of them that could agree with itself while the protos say otherwise. It only
// reads; the judgments live in the check functions, which take the rows as an
// argument so a deliberately broken copy can be fed to the same code.
func mcpClassificationsFromDescriptors(t *testing.T) []mcpClassification {
	t.Helper()
	var rows []mcpClassification
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
				forbiddenReason, ok := proto.GetExtension(md.Options(), v1pb.E_McpForbiddenReason).(v1pb.MCPForbiddenReason)
				require.True(t, ok, "method %s carries a malformed mcp_forbidden_reason", md.FullName())
				exclusionReason, ok := proto.GetExtension(md.Options(), v1pb.E_McpExclusionReason).(v1pb.MCPExclusionReason)
				require.True(t, ok, "method %s carries a malformed mcp_exclusion_reason", md.FullName())
				permission, ok := proto.GetExtension(md.Options(), v1pb.E_Permission).(string)
				require.True(t, ok, "method %s carries a malformed permission", md.FullName())
				rows = append(rows, mcpClassification{
					procedure:       fmt.Sprintf("/%s/%s", sd.FullName(), md.Name()),
					class:           class,
					forbiddenReason: forbiddenReason,
					exclusionReason: exclusionReason,
					permission:      permission,
				})
			}
		}
		return true
	})
	slices.SortFunc(rows, func(a, b mcpClassification) int { return strings.Compare(a.procedure, b.procedure) })
	return rows
}

// forbiddenProceduresFromDescriptors narrows the classification to the refused
// set, which is the half the interceptor enforces today.
func forbiddenProceduresFromDescriptors(t *testing.T) map[string]v1pb.MCPForbiddenReason {
	t.Helper()
	found := map[string]v1pb.MCPForbiddenReason{}
	for _, row := range mcpClassificationsFromDescriptors(t) {
		if row.class == v1pb.MCPMethodClass_FORBIDDEN {
			found[row.procedure] = row.forbiddenReason
		}
	}
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
		v1connect.InstanceServiceUpdateDataSourceProcedure,
		v1connect.IssueServiceApproveIssueProcedure,
		v1connect.IssueServiceRejectIssueProcedure,
		v1connect.IssueServiceRetryIssueApprovalProcedure,
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
		v1connect.InstanceServiceUpdateDataSourceProcedure:               v1pb.MCPForbiddenReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.IssueServiceApproveIssueProcedure:                      v1pb.MCPForbiddenReason_DRIVES_THE_APPROVAL_DECISION,
		v1connect.IssueServiceRejectIssueProcedure:                       v1pb.MCPForbiddenReason_DRIVES_THE_APPROVAL_DECISION,
		v1connect.IssueServiceRetryIssueApprovalProcedure:                v1pb.MCPForbiddenReason_DRIVES_THE_APPROVAL_DECISION,
	}
	// Wording for every mechanism in use is checkReasonsMatchTheClass's job, over
	// every class rather than only the ones enumerated here.
	require.Equal(t, wantReason, got,
		"every FORBIDDEN method records the mechanism it is refused for, and this is the second signature on it")
}

// The lint. Every clause is a function over the rows rather than a body of
// assertions, so the same code can be run against a deliberately broken copy —
// each clause has a RED test below that mutates one input and asserts the
// clause fires. A clause with no RED test is a clause nobody has proved can
// fail.

// mcpServingModes says which classes each ceiling mode serves. It is the
// specification the gate will implement, written here as data so the lint can
// hold the classification against it; nothing reads it at request time, and the
// interceptor still enforces FORBIDDEN alone. DISABLED serves nothing because
// it refuses the connection.
//
// The gate PR must LIFT this table into the code it evaluates rather than
// declare a second copy. Two copies would let this lint stay green while the
// runtime serving rules drift away from it, which is the one failure mode a
// specification-shaped test cannot catch about itself.
var mcpServingModes = map[v1pb.WorkspaceProfileSetting_MCPCapability][]v1pb.MCPMethodClass{
	v1pb.WorkspaceProfileSetting_DISABLED:   {},
	v1pb.WorkspaceProfileSetting_READ_ONLY:  {v1pb.MCPMethodClass_READ},
	v1pb.WorkspaceProfileSetting_READ_WRITE: {v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_WRITE},
}

// checkEveryMethodIsClassified is the clause that ends the whack-a-mole. A new
// RPC arrives unannotated, which is UNSPECIFIED, which fails the build — so it
// gets classified before it can ship rather than after somebody notices it was
// reachable.
func checkEveryMethodIsClassified(rows []mcpClassification) []string {
	var violations []string
	for _, row := range rows {
		if row.class == v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED {
			violations = append(violations, row.procedure+": carries no mcp_method_class")
		}
	}
	slices.Sort(violations)
	return violations
}

// checkReasonsMatchTheClass proves each row's reason and class agree. A refused
// method says why it is refused and has wording to say it with; an excluded
// method says why it is out; and a served method claims neither, because a
// reason without its class denies nothing and would read, to anyone grepping
// for it, as a method that is not served.
func checkReasonsMatchTheClass(rows []mcpClassification, wording map[v1pb.MCPForbiddenReason]string) []string {
	var violations []string
	for _, row := range rows {
		hasForbiddenReason := row.forbiddenReason != v1pb.MCPForbiddenReason_MCP_FORBIDDEN_REASON_UNSPECIFIED
		hasExclusionReason := row.exclusionReason != v1pb.MCPExclusionReason_MCP_EXCLUSION_REASON_UNSPECIFIED
		switch row.class {
		case v1pb.MCPMethodClass_FORBIDDEN:
			if !hasForbiddenReason {
				violations = append(violations, row.procedure+": is FORBIDDEN and records no mcp_forbidden_reason")
			} else if _, ok := wording[row.forbiddenReason]; !ok {
				violations = append(violations, fmt.Sprintf("%s: is refused for %v, which has no sentence to say so", row.procedure, row.forbiddenReason))
			}
			if hasExclusionReason {
				violations = append(violations, row.procedure+": is FORBIDDEN and also records an exclusion reason")
			}
		case v1pb.MCPMethodClass_EXCLUDED:
			if !hasExclusionReason {
				violations = append(violations, row.procedure+": is EXCLUDED and records no mcp_exclusion_reason")
			}
			if hasForbiddenReason {
				violations = append(violations, row.procedure+": is EXCLUDED and also records a forbidden reason")
			}
		default:
			if hasForbiddenReason {
				violations = append(violations, row.procedure+": records why it is forbidden but is not classified FORBIDDEN")
			}
			if hasExclusionReason {
				violations = append(violations, row.procedure+": records why it is excluded but is not classified EXCLUDED")
			}
		}
	}
	slices.Sort(violations)
	return violations
}

// mcpDeniedClasses is the other half of the same decision: the classes no mode
// serves. Held against mcpServingModes below.
var mcpDeniedClasses = []v1pb.MCPMethodClass{v1pb.MCPMethodClass_FORBIDDEN, v1pb.MCPMethodClass_EXCLUDED}

// checkEveryClassHasAServingDecision proves that "no method is both denied and
// reachable through a serving mode" is true of the whole vocabulary rather than
// of the rows that happen to exist today.
//
// Asserting it row by row would prove nothing: a row carries one class, so
// FORBIDDEN and READ are mutually exclusive by construction, and a check that
// walks 208 rows looking for a contradiction the type system already forbids is
// a check that cannot fail. What can drift is the vocabulary. So this reads the
// two enums out of the compiled descriptors and holds the tables against them:
// every class is served by some mode or denied, never both and never neither,
// and every ceiling mode says what it serves. Add a fifth class to
// MCPMethodClass, or a fourth mode to MCPCapability, and the build fails until
// somebody decides who serves it — which is the decision that would otherwise
// be made by whichever `switch` in the gate happened to have a default arm.
//
// The last loop closes the circle back to the rows. `decided` holds the classes
// some decision claims — served, denied, or, when the tables disagree, both — so
// dropping READ out of the serving table does not merely report READ once. It
// reports every method annotated READ, which is the shape of the damage.
func checkEveryClassHasAServingDecision(
	rows []mcpClassification,
	serving map[v1pb.WorkspaceProfileSetting_MCPCapability][]v1pb.MCPMethodClass,
	denied []v1pb.MCPMethodClass,
	classes protoreflect.EnumDescriptor,
	modes protoreflect.EnumDescriptor,
) []string {
	var violations []string
	servedBy := map[v1pb.MCPMethodClass][]v1pb.WorkspaceProfileSetting_MCPCapability{}
	for mode, served := range serving {
		for _, class := range served {
			servedBy[class] = append(servedBy[class], mode)
		}
	}
	// serving is a map, so without this the mode list in a violation message
	// comes out in a different order on every run.
	for class := range servedBy {
		slices.Sort(servedBy[class])
	}

	inVocabulary := map[v1pb.MCPMethodClass]bool{}
	decided := map[v1pb.MCPMethodClass]bool{}
	values := classes.Values()
	for i := range values.Len() {
		class := v1pb.MCPMethodClass(values.Get(i).Number())
		if class == v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED {
			continue
		}
		inVocabulary[class] = true
		isServed, isDenied := len(servedBy[class]) > 0, slices.Contains(denied, class)
		switch {
		case isServed && isDenied:
			violations = append(violations, fmt.Sprintf("%v is denied and served by %v", class, servedBy[class]))
		case !isServed && !isDenied:
			violations = append(violations, fmt.Sprintf("%v is neither served by a mode nor denied", class))
		default:
		}
		// Having two decisions counts as decided here. The switch above has
		// already said so once; repeating it per method would bury the one
		// actionable line under a hundred rows naming the wrong problem.
		decided[class] = isServed || isDenied
	}
	for _, class := range denied {
		if !inVocabulary[class] {
			violations = append(violations, fmt.Sprintf("%v is denied but is not a class", class))
		}
	}

	modeValues := modes.Values()
	for i := range modeValues.Len() {
		value := modeValues.Get(i)
		mode := v1pb.WorkspaceProfileSetting_MCPCapability(value.Number())
		if mode == v1pb.WorkspaceProfileSetting_MCP_CAPABILITY_UNSPECIFIED {
			continue
		}
		if _, ok := serving[mode]; !ok {
			violations = append(violations, fmt.Sprintf("ceiling mode %v does not say which classes it serves", mode))
		}
	}

	for _, row := range rows {
		if row.class != v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED && !decided[row.class] {
			violations = append(violations, fmt.Sprintf("%s: carries %v, which has no serving decision", row.procedure, row.class))
		}
	}
	slices.Sort(violations)
	return violations
}

func TestLintEveryMethodIsClassified(t *testing.T) {
	require.Empty(t, checkEveryMethodIsClassified(mcpClassificationsFromDescriptors(t)))
}

func TestLintReasonsMatchTheClass(t *testing.T) {
	require.Empty(t, checkReasonsMatchTheClass(mcpClassificationsFromDescriptors(t), mcpForbiddenReasons))
}

// mcpEnums reads the two enums the serving decision spans out of the compiled
// descriptors, so the lint below tracks the protos rather than a Go copy of
// them.
func mcpEnums(t *testing.T) (classes, modes protoreflect.EnumDescriptor) {
	t.Helper()
	return v1pb.MCPMethodClass(0).Descriptor(), v1pb.WorkspaceProfileSetting_MCPCapability(0).Descriptor()
}

func TestLintEveryClassHasAServingDecision(t *testing.T) {
	classes, modes := mcpEnums(t)
	require.Empty(t, checkEveryClassHasAServingDecision(
		mcpClassificationsFromDescriptors(t), mcpServingModes, mcpDeniedClasses, classes, modes))
}

// TestLintClausesFireWhenBroken is the RED half: one mutation per clause,
// asserting the clause catches it. Without these the clauses above are three
// functions that have only ever been shown to return nothing.
func TestLintClausesFireWhenBroken(t *testing.T) {
	t.Run("an unclassified method fails", func(t *testing.T) {
		// The shape a new RPC arrives in: defined, annotated with nothing.
		// TestLintEveryMethodIsClassified owns the claim that the real
		// descriptor set is clean; this owns the claim that the clause fires.
		broken := append(slices.Clone(mcpClassificationsFromDescriptors(t)),
			mcpClassification{procedure: "/bytebase.v1.NewService/NewMethod"})
		require.Contains(t, checkEveryMethodIsClassified(broken),
			"/bytebase.v1.NewService/NewMethod: carries no mcp_method_class")
	})

	t.Run("a reasonless FORBIDDEN fails", func(t *testing.T) {
		broken := []mcpClassification{{procedure: "/p", class: v1pb.MCPMethodClass_FORBIDDEN}}
		require.Equal(t, []string{"/p: is FORBIDDEN and records no mcp_forbidden_reason"},
			checkReasonsMatchTheClass(broken, mcpForbiddenReasons))
	})

	t.Run("a FORBIDDEN reason with no wording fails", func(t *testing.T) {
		broken := []mcpClassification{{
			procedure:       "/p",
			class:           v1pb.MCPMethodClass_FORBIDDEN,
			forbiddenReason: v1pb.MCPForbiddenReason(9999),
		}}
		require.Equal(t, []string{"/p: is refused for 9999, which has no sentence to say so"},
			checkReasonsMatchTheClass(broken, mcpForbiddenReasons))
	})

	t.Run("a reasonless EXCLUDED fails", func(t *testing.T) {
		broken := []mcpClassification{{procedure: "/p", class: v1pb.MCPMethodClass_EXCLUDED}}
		require.Equal(t, []string{"/p: is EXCLUDED and records no mcp_exclusion_reason"},
			checkReasonsMatchTheClass(broken, mcpForbiddenReasons))
	})

	t.Run("a served method carrying a denial reason fails", func(t *testing.T) {
		broken := []mcpClassification{
			{procedure: "/p", class: v1pb.MCPMethodClass_READ, forbiddenReason: v1pb.MCPForbiddenReason_MINTS_CREDENTIAL},
			{procedure: "/q", class: v1pb.MCPMethodClass_WRITE, exclusionReason: v1pb.MCPExclusionReason_ADMINISTERS_THE_WORKSPACE},
			// The mistake two reason enums make possible: both at once. Whichever
			// class the method carries, the other reason is a sentence about it
			// that is not true.
			{
				procedure:       "/r",
				class:           v1pb.MCPMethodClass_FORBIDDEN,
				forbiddenReason: v1pb.MCPForbiddenReason_MINTS_CREDENTIAL,
				exclusionReason: v1pb.MCPExclusionReason_ADMINISTERS_THE_WORKSPACE,
			},
			{
				procedure:       "/s",
				class:           v1pb.MCPMethodClass_EXCLUDED,
				exclusionReason: v1pb.MCPExclusionReason_ADMINISTERS_THE_WORKSPACE,
				forbiddenReason: v1pb.MCPForbiddenReason_MINTS_CREDENTIAL,
			},
		}
		require.Equal(t, []string{
			"/p: records why it is forbidden but is not classified FORBIDDEN",
			"/q: records why it is excluded but is not classified EXCLUDED",
			"/r: is FORBIDDEN and also records an exclusion reason",
			"/s: is EXCLUDED and also records a forbidden reason",
		}, checkReasonsMatchTheClass(broken, mcpForbiddenReasons))
	})

	// The serving-decision clause is over the vocabulary, so its mutations are
	// vocabulary mutations: a class that two decisions claim, a class no decision
	// claims, a ceiling mode nobody wrote a row for, and a row reaching a class
	// the tables skipped.
	classes, modes := mcpEnums(t)

	t.Run("a class both served and denied fails once, not once per method", func(t *testing.T) {
		widened := map[v1pb.WorkspaceProfileSetting_MCPCapability][]v1pb.MCPMethodClass{
			v1pb.WorkspaceProfileSetting_DISABLED:  {},
			v1pb.WorkspaceProfileSetting_READ_ONLY: {v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_EXCLUDED},
			v1pb.WorkspaceProfileSetting_READ_WRITE: {
				v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_WRITE, v1pb.MCPMethodClass_EXCLUDED,
			},
		}
		// The real rows are passed deliberately: 93 of them carry EXCLUDED, and
		// the diagnosis worth reading is the one about the table, not ninety-three
		// about methods whose annotation is fine. Two modes, so the mode list in
		// the message is only stable because servedBy is sorted.
		require.Equal(t, []string{"EXCLUDED is denied and served by [READ_ONLY READ_WRITE]"},
			checkEveryClassHasAServingDecision(mcpClassificationsFromDescriptors(t), widened, mcpDeniedClasses, classes, modes))
	})

	t.Run("a class nobody decided fails, and takes its rows with it", func(t *testing.T) {
		// The damage from dropping a class out of the serving table is every
		// method annotated with it, so the clause reports every method too.
		rows := []mcpClassification{
			{procedure: "/p", class: v1pb.MCPMethodClass_WRITE},
			{procedure: "/q", class: v1pb.MCPMethodClass_READ},
		}
		require.Equal(t, []string{
			"/p: carries WRITE, which has no serving decision",
			"WRITE is neither served by a mode nor denied",
		}, checkEveryClassHasAServingDecision(rows, map[v1pb.WorkspaceProfileSetting_MCPCapability][]v1pb.MCPMethodClass{
			v1pb.WorkspaceProfileSetting_DISABLED:   {},
			v1pb.WorkspaceProfileSetting_READ_ONLY:  {v1pb.MCPMethodClass_READ},
			v1pb.WorkspaceProfileSetting_READ_WRITE: {v1pb.MCPMethodClass_READ},
		}, mcpDeniedClasses, classes, modes))
	})

	t.Run("a ceiling mode with no serving row fails", func(t *testing.T) {
		require.Equal(t, []string{"ceiling mode DISABLED does not say which classes it serves"},
			checkEveryClassHasAServingDecision(nil, map[v1pb.WorkspaceProfileSetting_MCPCapability][]v1pb.MCPMethodClass{
				v1pb.WorkspaceProfileSetting_READ_ONLY:  {v1pb.MCPMethodClass_READ},
				v1pb.WorkspaceProfileSetting_READ_WRITE: {v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_WRITE},
			}, mcpDeniedClasses, classes, modes))
	})

	t.Run("a row carrying an undecided class fails", func(t *testing.T) {
		rows := []mcpClassification{{procedure: "/p", class: v1pb.MCPMethodClass(9999)}}
		require.Equal(t, []string{"/p: carries 9999, which has no serving decision"},
			checkEveryClassHasAServingDecision(rows, mcpServingModes, mcpDeniedClasses, classes, modes))
	})
}

// TestForbiddenClassLeavesReadsAlone pins the other half of the second batch's
// ruling, which is the half a later change could quietly widen: the reads of
// the services whose writes were forbidden are not themselves refused. This PR
// classifies most of them EXCLUDED — identity and credential administration —
// which is a scope decision a future admin-capable ceiling can revisit. What
// they must never become is FORBIDDEN, and that is what this pins.
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

// TestExcludedOnlyForALeak pins the population that is excluded for a defect
// rather than for what the method is. Every row here is an ordinary read that
// belongs in a serving class on its merits, and is out only because its
// response carries a stored secret the product already redacts elsewhere. The
// leak is the thing to fix; when one is fixed its row leaves this test and
// becomes READ, which is a reviewed widening rather than a silent one.
//
// Each entry names the leak and the redaction that closes it:
//
//   - The four project reads share convertToProject
//     (project_service_converter.go:264), which copies the incoming-webhook URL
//     out verbatim. That URL is a bearer credential — whoever holds it posts
//     into the customer's Slack, Feishu, WeCom or Teams as Bytebase — and this
//     repo already treats it as one: redactWebhook masks it in audit rows
//     (audit.go:914) beside the OIDC client secret and the LDAP bind password.
//     ListProjects pages the whole workspace, so one call takes the lot.
//
//   - The two instance reads and ListInstanceRoles share InstanceRole.Attribute
//     (instance_service_converter.go:57), which is the raw SHOW GRANTS text the
//     MySQL driver stores (plugin/db/mysql/role.go:38). On MariaDB that text
//     carries "IDENTIFIED BY PASSWORD '<hash>'", so the response hands over
//     every database account's password hash. The data-source half of the very
//     same converter gets this right and says so: "We don't return the password
//     and SSLs on reads" (instance_service_converter.go:253).
//
//   - GetCurrentUser returns temp_otp_secret and temp_recovery_codes whenever
//     the subject is the caller (user_service.go:742), so a session reading its
//     own profile during an MFA-setup window captures the TOTP seed. The window
//     is opened by the human in the console, not by the agent — UpdateUser,
//     which starts it, is FORBIDDEN — which makes this the narrowest of the
//     three and still a stored secret in a response body.
func TestExcludedOnlyForALeak(t *testing.T) {
	want := []string{
		v1connect.ProjectServiceGetProjectProcedure,
		v1connect.ProjectServiceListProjectsProcedure,
		v1connect.ProjectServiceBatchGetProjectsProcedure,
		v1connect.ProjectServiceSearchProjectsProcedure,
		v1connect.InstanceServiceGetInstanceProcedure,
		v1connect.InstanceServiceListInstancesProcedure,
		v1connect.InstanceRoleServiceListInstanceRolesProcedure,
		v1connect.UserServiceGetCurrentUserProcedure,
	}
	var got []string
	for _, row := range mcpClassificationsFromDescriptors(t) {
		if row.exclusionReason == v1pb.MCPExclusionReason_RETURNS_A_STORED_SECRET {
			got = append(got, row.procedure)
		}
	}
	slices.Sort(want)
	slices.Sort(got)
	require.Equal(t, want, got,
		"this population is excluded for a leak, not for what the method is; adding or removing one is a deliberate decision")
}

// TestDeprecatedSQLServiceAliasesMatchTheirCanonicalMethod pins the three
// query-history RPCs that SQLService still carries as deprecated aliases of
// QueryHistoryService. They have no HTTP binding, but the MCP surface indexes
// every POST path in its generated spec (mcp/openapi_index.go), so call_api
// resolves them by operation ID. An alias classified more permissively than the
// method it forwards to is a straight bypass of that method's classification.
func TestDeprecatedSQLServiceAliasesMatchTheirCanonicalMethod(t *testing.T) {
	class := map[string]v1pb.MCPMethodClass{}
	for _, row := range mcpClassificationsFromDescriptors(t) {
		class[row.procedure] = row.class
	}
	for alias, canonical := range map[string]string{
		v1connect.SQLServiceListQueryHistoriesProcedure:   v1connect.QueryHistoryServiceListQueryHistoriesProcedure,
		v1connect.SQLServiceGetQueryHistoryProcedure:      v1connect.QueryHistoryServiceGetQueryHistoryProcedure,
		v1connect.SQLServiceSearchQueryHistoriesProcedure: v1connect.QueryHistoryServiceSearchQueryHistoriesProcedure,
	} {
		require.Equal(t, class[canonical], class[alias],
			"%s is a deprecated alias of %s and must not be reachable where the canonical method is not", alias, canonical)
	}
}

// mcpInventoryPath holds a rendering of the annotations, committed so a
// reviewer can read the whole classification as one table instead of across
// thirty proto files. It is a rendering and not the source of truth: the
// annotations are, this file is regenerated from them, and nothing reads it at
// runtime.
const mcpInventoryPath = "testdata/mcp_method_classification.md"

const mcpInventoryRegenerate = "MCP_INVENTORY=write go test ./backend/api/v1/ -run TestMCPClassificationInventory"

func renderMCPInventory(rows []mcpClassification) string {
	var b strings.Builder
	b.WriteString("# MCP method classification\n\n")
	b.WriteString("Rendered from the `bytebase.v1.mcp_method_class`, `mcp_forbidden_reason` and\n")
	b.WriteString("`mcp_exclusion_reason` annotations on the v1 RPCs. The annotations are the source of\n")
	b.WriteString("truth; this file is a reviewable view of them and nothing reads it at runtime.\n\n")
	b.WriteString("Regenerate with:\n\n```\n" + mcpInventoryRegenerate + "\n```\n\n")
	b.WriteString("Only FORBIDDEN is enforced today. READ, WRITE and EXCLUDED record where a method\n")
	b.WriteString("belongs; the gate that acts on them is a later change.\n\n")

	counts := map[v1pb.MCPMethodClass]int{}
	for _, row := range rows {
		counts[row.class]++
	}
	b.WriteString("| Class | Methods | Meaning |\n|---|---|---|\n")
	for _, line := range []struct {
		class   v1pb.MCPMethodClass
		meaning string
	}{
		{v1pb.MCPMethodClass_READ, "served to a read-only session and above"},
		{v1pb.MCPMethodClass_WRITE, "served to a read-write session only"},
		{v1pb.MCPMethodClass_EXCLUDED, "served by no mode this phase ships"},
		{v1pb.MCPMethodClass_FORBIDDEN, "never served, enforced today"},
		{v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED, "unclassified — CI rejects this"},
	} {
		fmt.Fprintf(&b, "| %v | %d | %s |\n", line.class, counts[line.class], line.meaning)
	}
	fmt.Fprintf(&b, "| **total** | **%d** | |\n\n", len(rows))

	b.WriteString("| Method | Class | Reason | Permission |\n|---|---|---|---|\n")
	for _, row := range rows {
		reason := ""
		switch row.class {
		case v1pb.MCPMethodClass_FORBIDDEN:
			reason = row.forbiddenReason.String()
		case v1pb.MCPMethodClass_EXCLUDED:
			reason = row.exclusionReason.String()
		default:
		}
		permission := row.permission
		if permission == "" {
			permission = "—"
		}
		if reason == "" {
			reason = "—"
		}
		fmt.Fprintf(&b, "| %s | %v | %s | %s |\n",
			strings.TrimPrefix(row.procedure, "/bytebase.v1."), row.class, reason, permission)
	}
	return b.String()
}

// TestMCPClassificationInventory keeps the committed rendering in step with the
// annotations. It fails on a stale file rather than regenerating silently, so a
// classification change shows up in the diff a reviewer reads.
func TestMCPClassificationInventory(t *testing.T) {
	rendered := renderMCPInventory(mcpClassificationsFromDescriptors(t))
	if os.Getenv("MCP_INVENTORY") == "write" {
		require.NoError(t, os.WriteFile(mcpInventoryPath, []byte(rendered), 0o600))
		return
	}
	committed, err := os.ReadFile(mcpInventoryPath)
	require.NoError(t, err)
	require.Equal(t, string(committed), rendered, "the inventory is stale; regenerate with: %s", mcpInventoryRegenerate)
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

	// Only FORBIDDEN is enforced in this phase, so every other class still
	// dispatches. EXCLUDED is the one that most looks like a denial and is not
	// one yet — it is the largest class, and it covers methods the console and
	// the API serve today — so a later change to the condition above that turned
	// it into a serving-class allowlist would silently deny them. UNSPECIFIED
	// reaches here only in a build that has not been linted; it is served, as
	// before.
	for name, class := range map[string]v1pb.MCPMethodClass{
		"unclassified": v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED,
		"read":         v1pb.MCPMethodClass_READ,
		"write":        v1pb.MCPMethodClass_WRITE,
		"excluded":     v1pb.MCPMethodClass_EXCLUDED,
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
