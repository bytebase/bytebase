package v1

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// mcpClassification is one v1 RPC's MCP annotations, as the compiled
// descriptors carry them.
type mcpClassification struct {
	procedure  string
	class      v1pb.MCPMethodClass
	reason     v1pb.MCPDenialReason
	permission string
	audit      bool
	// request is the method's input descriptor, which the redaction lint
	// below reads to see what a denial would write into an audit row.
	request protoreflect.MessageDescriptor
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
				reason, ok := proto.GetExtension(md.Options(), v1pb.E_McpDenialReason).(v1pb.MCPDenialReason)
				require.True(t, ok, "method %s carries a malformed mcp_denial_reason", md.FullName())
				permission, ok := proto.GetExtension(md.Options(), v1pb.E_Permission).(string)
				require.True(t, ok, "method %s carries a malformed permission", md.FullName())
				audit, ok := proto.GetExtension(md.Options(), v1pb.E_Audit).(bool)
				require.True(t, ok, "method %s carries a malformed audit annotation", md.FullName())
				rows = append(rows, mcpClassification{
					procedure:  fmt.Sprintf("/%s/%s", sd.FullName(), md.Name()),
					class:      class,
					reason:     reason,
					permission: permission,
					audit:      audit,
					request:    md.Input(),
				})
			}
		}
		return true
	})
	slices.SortFunc(rows, func(a, b mcpClassification) int { return strings.Compare(a.procedure, b.procedure) })
	return rows
}

// forbiddenProceduresFromDescriptors narrows the classification to the
// FORBIDDEN set, whose membership several assertions below pin. The gate
// enforces every class, not this one — its own doc owns that claim.
func forbiddenProceduresFromDescriptors(t *testing.T) map[string]v1pb.MCPDenialReason {
	t.Helper()
	found := map[string]v1pb.MCPDenialReason{}
	for _, row := range mcpClassificationsFromDescriptors(t) {
		if row.class == v1pb.MCPMethodClass_FORBIDDEN {
			found[row.procedure] = row.reason
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
	wantReason := map[string]v1pb.MCPDenialReason{
		v1connect.AuthServiceLoginProcedure:                              v1pb.MCPDenialReason_MINTS_CREDENTIAL,
		v1connect.AuthServiceSignupProcedure:                             v1pb.MCPDenialReason_MINTS_CREDENTIAL,
		v1connect.AuthServiceExchangeTokenProcedure:                      v1pb.MCPDenialReason_MINTS_CREDENTIAL,
		v1connect.AuthServiceRefreshProcedure:                            v1pb.MCPDenialReason_MINTS_CREDENTIAL,
		v1connect.AuthServiceSwitchWorkspaceProcedure:                    v1pb.MCPDenialReason_MINTS_CREDENTIAL,
		v1connect.AuthServiceRequestPasswordResetProcedure:               v1pb.MCPDenialReason_RESETS_CREDENTIAL,
		v1connect.AuthServiceResetPasswordProcedure:                      v1pb.MCPDenialReason_RESETS_CREDENTIAL,
		v1connect.AuthServiceSendEmailLoginCodeProcedure:                 v1pb.MCPDenialReason_RESETS_CREDENTIAL,
		v1connect.UserServiceUpdateUserProcedure:                         v1pb.MCPDenialReason_TAKES_OVER_ACCOUNT,
		v1connect.AuthServiceLogoutProcedure:                             v1pb.MCPDenialReason_ENDS_SESSION,
		v1connect.WorkspaceServiceLeaveWorkspaceProcedure:                v1pb.MCPDenialReason_ENDS_MEMBERSHIP,
		v1connect.WorkspaceServiceDeleteWorkspaceProcedure:               v1pb.MCPDenialReason_ENDS_MEMBERSHIP,
		v1connect.ServiceAccountServiceCreateServiceAccountProcedure:     v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.ServiceAccountServiceUpdateServiceAccountProcedure:     v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.WorkspaceServiceRotateDirectorySyncTokenProcedure:      v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.UserServiceCreateUserProcedure:                         v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.UserServiceUpdateEmailProcedure:                        v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.IdentityProviderServiceCreateIdentityProviderProcedure: v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.IdentityProviderServiceUpdateIdentityProviderProcedure: v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.IdentityProviderServiceTestIdentityProviderProcedure:   v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.WorkloadIdentityServiceCreateWorkloadIdentityProcedure: v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.WorkloadIdentityServiceUpdateWorkloadIdentityProcedure: v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.SettingServiceTestEmailSettingProcedure:                v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.SettingServiceUpdateSettingProcedure:                   v1pb.MCPDenialReason_REWRITES_SESSION_BOUNDARY,
		v1connect.InstanceServiceUpdateDataSourceProcedure:               v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS,
		v1connect.IssueServiceApproveIssueProcedure:                      v1pb.MCPDenialReason_DRIVES_THE_APPROVAL_DECISION,
		v1connect.IssueServiceRejectIssueProcedure:                       v1pb.MCPDenialReason_DRIVES_THE_APPROVAL_DECISION,
		v1connect.IssueServiceRetryIssueApprovalProcedure:                v1pb.MCPDenialReason_DRIVES_THE_APPROVAL_DECISION,
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

// The serving table this lint holds the classification against is
// mcpServingClasses, the one the gate evaluates — not a copy. 1b-1 declared it
// here as a specification with a note that the gate PR had to lift it into the
// code, because two copies would let this lint stay green while the runtime
// rules drifted away from it. This is that lift.

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
// method says why it is refused and has wording to say it with, and the reason
// it gives is one that belongs to its class; a served method claims none,
// because a reason without its class denies nothing and would read, to anyone
// grepping for it, as a method that is not served.
//
// One enum carries both refused classes, so the pairing is the check. Two
// enums made a wrong-class reason a parse-level impossibility on paper and, in
// practice, only added the states where a method carried both or carried the
// kind that contradicted its class — states this clause used to spend four of
// its six branches policing. What is left is the one thing that was ever real.
func checkReasonsMatchTheClass(rows []mcpClassification, wording map[v1pb.MCPDenialReason]mcpDenialWording) []string {
	var violations []string
	for _, row := range rows {
		refused := row.class == v1pb.MCPMethodClass_FORBIDDEN || row.class == v1pb.MCPMethodClass_EXCLUDED
		hasReason := row.reason != v1pb.MCPDenialReason_MCP_DENIAL_REASON_UNSPECIFIED
		switch {
		case refused && !hasReason:
			violations = append(violations, fmt.Sprintf("%s: is %v and records no mcp_denial_reason", row.procedure, row.class))
		case refused:
			recorded, ok := wording[row.reason]
			switch {
			case !ok:
				violations = append(violations, fmt.Sprintf("%s: is refused for %v, which has no sentence to say so", row.procedure, row.reason))
			case recorded.class != row.class:
				violations = append(violations, fmt.Sprintf("%s: is %v but records %v, which is a %v reason",
					row.procedure, row.class, row.reason, recorded.class))
			default:
			}
		case hasReason:
			violations = append(violations, fmt.Sprintf("%s: records %v, a denial reason, but is not refused", row.procedure, row.reason))
		default:
		}
	}
	slices.Sort(violations)
	return violations
}

// mcpDeniedClasses is the other half of the same decision: the classes no mode
// serves. Held against mcpServingClasses (mcp_gate.go) below.
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
	serving map[storepb.WorkspaceProfileSetting_MCPCapability][]v1pb.MCPMethodClass,
	denied []v1pb.MCPMethodClass,
	classes protoreflect.EnumDescriptor,
	modes protoreflect.EnumDescriptor,
) []string {
	var violations []string
	servedBy := map[v1pb.MCPMethodClass][]storepb.WorkspaceProfileSetting_MCPCapability{}
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
		mode := storepb.WorkspaceProfileSetting_MCPCapability(value.Number())
		if mode == storepb.WorkspaceProfileSetting_MCP_CAPABILITY_UNSPECIFIED {
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
	require.Empty(t, checkReasonsMatchTheClass(mcpClassificationsFromDescriptors(t), mcpDenialReasons))
}

// mcpEnums reads the two enums the serving decision spans out of the compiled
// descriptors, so the lint below tracks the protos rather than a Go copy of
// them. The ceiling comes from the STORE enum because that is the one a
// setting row holds and the gate reads back.
func mcpEnums(t *testing.T) (classes, modes protoreflect.EnumDescriptor) {
	t.Helper()
	return v1pb.MCPMethodClass(0).Descriptor(), storepb.WorkspaceProfileSetting_MCPCapability(0).Descriptor()
}

func TestLintEveryClassHasAServingDecision(t *testing.T) {
	classes, modes := mcpEnums(t)
	require.Empty(t, checkEveryClassHasAServingDecision(
		mcpClassificationsFromDescriptors(t), mcpServingClasses, mcpDeniedClasses, classes, modes))
}

// TestLintRefusedClassesMatchTheServingTable is what lets a second package
// decide what to advertise without carrying a copy of the serving rules. The
// MCP OpenAPI index hides an endpoint no ceiling serves, and it asks
// auth.MCPClassIsRefused, which cannot see this table. This holds the two
// against each other over the whole class vocabulary, so a mode that starts
// serving a class cannot leave discovery hiding it — or the reverse, which is
// the one that would offer an agent work it can never do.
func TestLintRefusedClassesMatchTheServingTable(t *testing.T) {
	served := map[v1pb.MCPMethodClass]bool{}
	for _, classes := range mcpServingClasses {
		for _, class := range classes {
			served[class] = true
		}
	}
	values := v1pb.MCPMethodClass(0).Descriptor().Values()
	for i := range values.Len() {
		class := v1pb.MCPMethodClass(values.Get(i).Number())
		require.Equal(t, !served[class], auth.MCPClassIsRefused(class),
			"%v: the serving table and auth.MCPClassIsRefused disagree about whether any ceiling serves it", class)
	}
}

// TestMCPCapabilityEnumsAgree is what lets the lint above read one enum and the
// gate read the other. The settings API writes the v1 ceiling, the setting row
// stores the store ceiling, and the conversion between them is by number
// (setting_service_converter.go). If the two ever disagree — a mode added to
// one, a number reused — a customer could set a ceiling the gate would read as
// something else entirely, and every serving decision above would be about the
// wrong vocabulary.
func TestMCPCapabilityEnumsAgree(t *testing.T) {
	v1Values := v1pb.WorkspaceProfileSetting_MCPCapability(0).Descriptor().Values()
	storeValues := storepb.WorkspaceProfileSetting_MCPCapability(0).Descriptor().Values()

	byNumber := func(values protoreflect.EnumValueDescriptors) map[int32]string {
		out := map[int32]string{}
		for i := range values.Len() {
			value := values.Get(i)
			out[int32(value.Number())] = string(value.Name())
		}
		return out
	}
	require.Equal(t, byNumber(v1Values), byNumber(storeValues),
		"the API ceiling enum and the stored ceiling enum must name the same numbers; "+
			"they convert by number and the gate reads the stored one")
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

	t.Run("a refused method with no reason fails", func(t *testing.T) {
		broken := []mcpClassification{
			{procedure: "/p", class: v1pb.MCPMethodClass_FORBIDDEN},
			{procedure: "/q", class: v1pb.MCPMethodClass_EXCLUDED},
		}
		require.Equal(t, []string{
			"/p: is FORBIDDEN and records no mcp_denial_reason",
			"/q: is EXCLUDED and records no mcp_denial_reason",
		}, checkReasonsMatchTheClass(broken, mcpDenialReasons))
	})

	t.Run("a reason with no wording fails", func(t *testing.T) {
		broken := []mcpClassification{{
			procedure: "/p",
			class:     v1pb.MCPMethodClass_FORBIDDEN,
			reason:    v1pb.MCPDenialReason(9999),
		}}
		require.Equal(t, []string{"/p: is refused for 9999, which has no sentence to say so"},
			checkReasonsMatchTheClass(broken, mcpDenialReasons))
	})

	t.Run("a reason belonging to the other refused class fails", func(t *testing.T) {
		// The one real mistake the merged enum leaves representable, and the
		// reason each row of the wording table carries its class. Both
		// directions, because a FORBIDDEN method explaining itself as a scope
		// decision and an EXCLUDED one claiming a boundary-breaking mechanism
		// mislead an operator in opposite directions.
		broken := []mcpClassification{
			{procedure: "/p", class: v1pb.MCPMethodClass_FORBIDDEN, reason: v1pb.MCPDenialReason_ADMINISTERS_THE_WORKSPACE},
			{procedure: "/q", class: v1pb.MCPMethodClass_EXCLUDED, reason: v1pb.MCPDenialReason_MINTS_CREDENTIAL},
		}
		require.Equal(t, []string{
			"/p: is FORBIDDEN but records ADMINISTERS_THE_WORKSPACE, which is a EXCLUDED reason",
			"/q: is EXCLUDED but records MINTS_CREDENTIAL, which is a FORBIDDEN reason",
		}, checkReasonsMatchTheClass(broken, mcpDenialReasons))
	})

	t.Run("a served method carrying a denial reason fails", func(t *testing.T) {
		broken := []mcpClassification{
			{procedure: "/p", class: v1pb.MCPMethodClass_READ, reason: v1pb.MCPDenialReason_MINTS_CREDENTIAL},
			{procedure: "/q", class: v1pb.MCPMethodClass_WRITE, reason: v1pb.MCPDenialReason_ADMINISTERS_THE_WORKSPACE},
		}
		require.Equal(t, []string{
			"/p: records MINTS_CREDENTIAL, a denial reason, but is not refused",
			"/q: records ADMINISTERS_THE_WORKSPACE, a denial reason, but is not refused",
		}, checkReasonsMatchTheClass(broken, mcpDenialReasons))
	})

	// The serving-decision clause is over the vocabulary, so its mutations are
	// vocabulary mutations: a class that two decisions claim, a class no decision
	// claims, a ceiling mode nobody wrote a row for, and a row reaching a class
	// the tables skipped.
	classes, modes := mcpEnums(t)

	t.Run("a class both served and denied fails once, not once per method", func(t *testing.T) {
		widened := map[storepb.WorkspaceProfileSetting_MCPCapability][]v1pb.MCPMethodClass{
			storepb.WorkspaceProfileSetting_DISABLED:  {},
			storepb.WorkspaceProfileSetting_READ_ONLY: {v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_EXCLUDED},
			storepb.WorkspaceProfileSetting_READ_WRITE: {
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
		}, checkEveryClassHasAServingDecision(rows, map[storepb.WorkspaceProfileSetting_MCPCapability][]v1pb.MCPMethodClass{
			storepb.WorkspaceProfileSetting_DISABLED:   {},
			storepb.WorkspaceProfileSetting_READ_ONLY:  {v1pb.MCPMethodClass_READ},
			storepb.WorkspaceProfileSetting_READ_WRITE: {v1pb.MCPMethodClass_READ},
		}, mcpDeniedClasses, classes, modes))
	})

	t.Run("a ceiling mode with no serving row fails", func(t *testing.T) {
		require.Equal(t, []string{"ceiling mode DISABLED does not say which classes it serves"},
			checkEveryClassHasAServingDecision(nil, map[storepb.WorkspaceProfileSetting_MCPCapability][]v1pb.MCPMethodClass{
				storepb.WorkspaceProfileSetting_READ_ONLY:  {v1pb.MCPMethodClass_READ},
				storepb.WorkspaceProfileSetting_READ_WRITE: {v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_WRITE},
			}, mcpDeniedClasses, classes, modes))
	})

	t.Run("a row carrying an undecided class fails", func(t *testing.T) {
		rows := []mcpClassification{{procedure: "/p", class: v1pb.MCPMethodClass(9999)}}
		require.Equal(t, []string{"/p: carries 9999, which has no serving decision"},
			checkEveryClassHasAServingDecision(rows, mcpServingClasses, mcpDeniedClasses, classes, modes))
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
		if row.reason == v1pb.MCPDenialReason_RETURNS_A_STORED_SECRET {
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
	b.WriteString("Rendered from the `bytebase.v1.mcp_method_class` and `mcp_denial_reason`\n")
	b.WriteString("annotations on the v1 RPCs. The annotations are the source of\n")
	b.WriteString("truth; this file is a reviewable view of them and nothing reads it at runtime.\n\n")
	b.WriteString("Regenerate with:\n\n```\n" + mcpInventoryRegenerate + "\n```\n\n")
	b.WriteString("Every class is enforced by the MCP gate. READ and WRITE are the serving classes a\n")
	b.WriteString("workspace's MCP capability ceiling selects between; EXCLUDED and FORBIDDEN are\n")
	b.WriteString("served by no ceiling.\n\n")

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
		{v1pb.MCPMethodClass_EXCLUDED, "served by no ceiling this phase ships"},
		{v1pb.MCPMethodClass_FORBIDDEN, "never served, whatever the ceiling"},
		{v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED, "unclassified — CI rejects this, and the gate refuses it"},
	} {
		fmt.Fprintf(&b, "| %v | %d | %s |\n", line.class, counts[line.class], line.meaning)
	}
	fmt.Fprintf(&b, "| **total** | **%d** | |\n\n", len(rows))

	b.WriteString("| Method | Class | Reason | Permission |\n|---|---|---|---|\n")
	for _, row := range rows {
		reason := ""
		if row.reason != v1pb.MCPDenialReason_MCP_DENIAL_REASON_UNSPECIFIED {
			reason = row.reason.String()
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

// --- the gate ---

// mcpGateStore stands in for the workspace's stored ceiling. A READ_ONLY
// ceiling cannot be reached end to end yet — mcpConnectionAllowed still refuses
// the connection until 1b-3 lands the SQL clamp — so the only way to exercise
// the read-only half of the rule today is here.
type mcpGateStore struct {
	ceiling storepb.WorkspaceProfileSetting_MCPCapability
	err     error
}

func (s mcpGateStore) GetMCPCapabilityUncached(context.Context, string) (storepb.WorkspaceProfileSetting_MCPCapability, error) {
	return s.ceiling, s.err
}

func readWriteCeiling() mcpGateStore {
	return mcpGateStore{ceiling: storepb.WorkspaceProfileSetting_READ_WRITE}
}

func readOnlyCeiling() mcpGateStore {
	return mcpGateStore{ceiling: storepb.WorkspaceProfileSetting_READ_ONLY}
}

type mcpGateResult struct {
	dispatched  bool
	auditMarked bool
	err         error
}

// invokeMCPGate runs one request through the gate alone. auditMarked is what
// the audit interceptor reads when the request comes back out; standing in for
// it here keeps the class rules provable without a database, and
// TestMCPGateDenialIsAuditedWithoutAnAuditAnnotation proves the real
// interceptor honors the mark.
func invokeMCPGate(t *testing.T, stores mcpCeilingReader, authCtx *common.AuthContext, procedure string, req connect.AnyRequest) mcpGateResult {
	t.Helper()
	out := mcpGateResult{}
	next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		out.dispatched = true
		return connect.NewResponse(&v1pb.User{}), nil
	}
	ctx := context.Background()
	if authCtx != nil {
		ctx = context.WithValue(ctx, common.AuthContextKey, authCtx)
	}
	ctx = context.WithValue(ctx, common.WorkspaceIDContextKey, auditTestWorkspace)
	ctx = common.WithSetMCPPolicyDenied(ctx, func() { out.auditMarked = true })
	_, out.err = NewInternalMCPGateInterceptor(stores).WrapUnary(next)(ctx,
		&specRequest{AnyRequest: req, procedure: procedure})
	return out
}

func classContext(class v1pb.MCPMethodClass) *common.AuthContext {
	return &common.AuthContext{MCPMethodClass: class}
}

// TestMCPGateRefusesTheDeniedClasses walks every method the annotations refuse
// outright and proves the gate refuses it under the most permissive ceiling
// there is. FORBIDDEN was already enforced; EXCLUDED is what this PR turns from
// a recorded classification into a denial, and it is the larger population by
// far — 93 methods the console and the public API serve today.
func TestMCPGateRefusesTheDeniedClasses(t *testing.T) {
	for _, row := range mcpClassificationsFromDescriptors(t) {
		if row.class != v1pb.MCPMethodClass_FORBIDDEN && row.class != v1pb.MCPMethodClass_EXCLUDED {
			continue
		}
		wantReason := mcpDenialReasons[row.reason].sentence
		t.Run(row.procedure, func(t *testing.T) {
			got := invokeMCPGate(t, readWriteCeiling(), &common.AuthContext{
				MCPMethodClass:  row.class,
				MCPDenialReason: row.reason,
			}, row.procedure, connect.NewRequest(&v1pb.GetUserRequest{}))

			require.Error(t, got.err, "a %v method must never reach its handler", row.class)
			require.False(t, got.dispatched, "the denial must land before dispatch, so no handler side effect can")
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(got.err))
			require.Contains(t, got.err.Error(), row.procedure, "the message must name the method the agent called")
			require.NotEmpty(t, wantReason, "every refused method's reason must have wording")
			require.Contains(t, got.err.Error(), wantReason, "the message must name why, so the agent can act on it")
			require.True(t, got.auditMarked, "every denial is an audited outcome")
		})
	}
}

// TestMCPGateServesTheAdmittedClasses is the other half: under a read-write
// ceiling both serving classes reach their handler, and the gate grants
// nothing on its own — ACL runs after it, unchanged.
func TestMCPGateServesTheAdmittedClasses(t *testing.T) {
	for _, class := range []v1pb.MCPMethodClass{v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_WRITE} {
		t.Run(class.String(), func(t *testing.T) {
			got := invokeMCPGate(t, readWriteCeiling(), classContext(class),
				v1connect.DatabaseServiceGetDatabaseProcedure, connect.NewRequest(&v1pb.GetDatabaseRequest{}))
			require.NoError(t, got.err)
			require.True(t, got.dispatched)
			require.False(t, got.auditMarked)
		})
	}
}

// TestMCPGateUnderAReadOnlyCeiling is the rule this PR exists for: the same
// session that reaches a WRITE method under read-write is refused it under
// read-only, while its reads keep working. It is unit-level because READ_ONLY
// cannot connect until 1b-3: TestMCPConnectionAllowed and the ws-readonly row
// of TestMCPCeilingStoredValueFailsClosed (backend/api/mcp) are the pins that
// the cutover has not moved.
func TestMCPGateUnderAReadOnlyCeiling(t *testing.T) {
	t.Run("a READ method is served", func(t *testing.T) {
		got := invokeMCPGate(t, readOnlyCeiling(), classContext(v1pb.MCPMethodClass_READ),
			v1connect.DatabaseServiceGetDatabaseProcedure, connect.NewRequest(&v1pb.GetDatabaseRequest{}))
		require.NoError(t, got.err)
		require.True(t, got.dispatched)
	})

	t.Run("a WRITE method is refused, naming the ceiling and the way out", func(t *testing.T) {
		// A WRITE method with no request-shape rule of its own, so the refusal
		// can only be the ceiling's.
		got := invokeMCPGate(t, readOnlyCeiling(), classContext(v1pb.MCPMethodClass_WRITE),
			v1connect.SheetServiceCreateSheetProcedure, connect.NewRequest(&v1pb.CreateSheetRequest{}))
		require.Error(t, got.err)
		require.False(t, got.dispatched)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(got.err))
		require.Contains(t, got.err.Error(), "READ_ONLY", "the denial must name the ceiling in force")
		require.Contains(t, got.err.Error(), "raise the MCP ceiling", "the denial must name the way out")
		require.True(t, got.auditMarked)
	})
}

// TestMCPGateFailsClosedOnTheCeiling covers the two states where the workspace
// has a ceiling the gate cannot act on. Both refuse: a policy that cannot be
// read is not a policy that permits.
func TestMCPGateFailsClosedOnTheCeiling(t *testing.T) {
	t.Run("a failed read refuses, as an outage rather than a verdict", func(t *testing.T) {
		got := invokeMCPGate(t, mcpGateStore{err: errors.New("db unreachable")},
			classContext(v1pb.MCPMethodClass_READ),
			v1connect.DatabaseServiceGetDatabaseProcedure, connect.NewRequest(&v1pb.GetDatabaseRequest{}))
		require.Error(t, got.err)
		require.False(t, got.dispatched, "an unknown policy never permits")
		require.Equal(t, connect.CodeUnavailable, connect.CodeOf(got.err),
			"a database blip is not an authorization decision about the caller, and a client must be able to retry it")
		require.NotContains(t, got.err.Error(), "db unreachable",
			"the agent gets the outcome, not the workspace's storage state")
		require.False(t, got.auditMarked,
			"the denial rows an operator filters on must be denials; an outage is not one")
	})

	t.Run("a stored value this build cannot interpret is a policy refusal", func(t *testing.T) {
		// The opposite half of the same failure. No retry fixes a mistyped
		// ceiling, so promising one would be a lie, and an admin has to act —
		// which makes it a denial, and an audited one. The /mcp connection
		// gate splits the same two the same way.
		got := invokeMCPGate(t, mcpGateStore{err: errors.Wrap(store.ErrMCPCapabilityUnreadable, "READ_WRTIE")},
			classContext(v1pb.MCPMethodClass_READ),
			v1connect.DatabaseServiceGetDatabaseProcedure, connect.NewRequest(&v1pb.GetDatabaseRequest{}))
		require.Error(t, got.err)
		require.False(t, got.dispatched)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(got.err))
		require.NotContains(t, got.err.Error(), "READ_WRTIE",
			"the agent gets the outcome, not the workspace's storage state")
		require.True(t, got.auditMarked)
	})

	t.Run("a DISABLED ceiling is an ordinary denial", func(t *testing.T) {
		// DISABLED refuses the connection, so this arm is not reachable end to
		// end — but the serving table carries an explicit empty list for it
		// precisely so that a ceiling which decided to serve nothing reads as a
		// denial rather than as the unknown-ceiling error, and that distinction
		// is only real if something exercises it.
		got := invokeMCPGate(t, mcpGateStore{ceiling: storepb.WorkspaceProfileSetting_DISABLED},
			classContext(v1pb.MCPMethodClass_READ),
			v1connect.DatabaseServiceGetDatabaseProcedure, connect.NewRequest(&v1pb.GetDatabaseRequest{}))
		require.Error(t, got.err)
		require.False(t, got.dispatched)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(got.err))
		require.Contains(t, got.err.Error(), "serves no method",
			"an empty serving list is a mode that serves nothing, not a mode nobody decided about")
		require.NotContains(t, got.err.Error(), "not one this build serves")
		require.True(t, got.auditMarked)
	})

	t.Run("a stored ceiling this build does not serve refuses", func(t *testing.T) {
		// The reserved number 2, which was METADATA_ONLY before the tier was
		// dropped. It survives protojson as an unknown enum number, so unlike
		// an unknown NAME it reaches the gate intact — and no mode serves it.
		got := invokeMCPGate(t, mcpGateStore{ceiling: storepb.WorkspaceProfileSetting_MCPCapability(2)},
			classContext(v1pb.MCPMethodClass_READ),
			v1connect.DatabaseServiceGetDatabaseProcedure, connect.NewRequest(&v1pb.GetDatabaseRequest{}))
		require.Error(t, got.err)
		require.False(t, got.dispatched)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(got.err))
		require.True(t, got.auditMarked)
	})

	t.Run("no workspace on the request refuses", func(t *testing.T) {
		// The internal auth interceptor puts one on every request it admits, so
		// this is the reordered-chain case: a bug in this process rather than a
		// verdict about the caller or a passing outage.
		out := mcpGateResult{}
		next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			out.dispatched = true
			return connect.NewResponse(&v1pb.User{}), nil
		}
		ctx := context.WithValue(context.Background(), common.AuthContextKey, classContext(v1pb.MCPMethodClass_READ))
		_, err := NewInternalMCPGateInterceptor(readWriteCeiling()).WrapUnary(next)(ctx, &specRequest{
			AnyRequest: connect.NewRequest(&v1pb.GetDatabaseRequest{}),
			procedure:  v1connect.DatabaseServiceGetDatabaseProcedure,
		})
		require.Error(t, err)
		require.False(t, out.dispatched)
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	})
}

// TestMCPGateFailsClosedOnAnUnclassifiedMethod pins the arm CI is supposed to
// make unreachable. A new RPC arrives unannotated, which is UNSPECIFIED; the
// lint fails the build on one, and if a build ever ships past that, the method
// is refused rather than served.
func TestMCPGateFailsClosedOnAnUnclassifiedMethod(t *testing.T) {
	got := invokeMCPGate(t, readWriteCeiling(), classContext(v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED),
		"/bytebase.v1.NewService/NewMethod", connect.NewRequest(&v1pb.GetUserRequest{}))
	require.Error(t, got.err)
	require.False(t, got.dispatched)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(got.err))
	require.Contains(t, got.err.Error(), "carries no MCP classification")
	require.True(t, got.auditMarked)
}

// TestMCPGateFallsBackToGenericWording pins that a refusal whose reason this
// build does not recognize still refuses. The class denies; the reason table
// only supplies a better sentence. This is the rolling-upgrade case, on both
// refused classes.
func TestMCPGateFallsBackToGenericWording(t *testing.T) {
	t.Run("forbidden", func(t *testing.T) {
		got := invokeMCPGate(t, readWriteCeiling(), &common.AuthContext{
			MCPMethodClass:  v1pb.MCPMethodClass_FORBIDDEN,
			MCPDenialReason: v1pb.MCPDenialReason(9999),
		}, v1connect.AuthServiceLoginProcedure, connect.NewRequest(&v1pb.LoginRequest{}))
		require.Error(t, got.err)
		require.False(t, got.dispatched)
		require.Contains(t, got.err.Error(), reasonForbiddenClass)
	})

	t.Run("excluded", func(t *testing.T) {
		got := invokeMCPGate(t, readWriteCeiling(), &common.AuthContext{
			MCPMethodClass:  v1pb.MCPMethodClass_EXCLUDED,
			MCPDenialReason: v1pb.MCPDenialReason(9999),
		}, v1connect.UserServiceListUsersProcedure, connect.NewRequest(&v1pb.ListUsersRequest{}))
		require.Error(t, got.err)
		require.False(t, got.dispatched)
		require.Contains(t, got.err.Error(), reasonExcludedClass)
	})
}

// TestMCPGateFailsClosedWithoutAnAuthContext pins that the gate does not guess.
func TestMCPGateFailsClosedWithoutAnAuthContext(t *testing.T) {
	got := invokeMCPGate(t, readWriteCeiling(), nil,
		v1connect.AuthServiceLoginProcedure, connect.NewRequest(&v1pb.LoginRequest{}))
	require.Error(t, got.err, "without a resolved classification the interceptor must not guess")
	require.False(t, got.dispatched)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(got.err))
}

// TestMCPGateRefusesGrantIssues pins the one refusal a per-method class cannot
// express. CreateIssue is WRITE for the database-change issue an agent exists
// to compose; the grant types complete on creation and hand out access with no
// human step. UpdateIssue is the same method under another name whenever
// allow_missing is set.
func TestMCPGateRefusesGrantIssues(t *testing.T) {
	write := classContext(v1pb.MCPMethodClass_WRITE)

	t.Run("an ordinary change issue is served", func(t *testing.T) {
		got := invokeMCPGate(t, readWriteCeiling(), write, v1connect.IssueServiceCreateIssueProcedure,
			connect.NewRequest(&v1pb.CreateIssueRequest{
				Parent: "projects/p",
				Issue:  &v1pb.Issue{Type: v1pb.Issue_DATABASE_CHANGE, Plan: "projects/p/plans/1"},
			}))
		require.NoError(t, got.err)
		require.True(t, got.dispatched)
	})

	for _, issueType := range []v1pb.Issue_Type{
		v1pb.Issue_ROLE_GRANT,
		v1pb.Issue_ACCESS_GRANT,
	} {
		t.Run("CreateIssue refuses "+issueType.String(), func(t *testing.T) {
			got := invokeMCPGate(t, readWriteCeiling(), write, v1connect.IssueServiceCreateIssueProcedure,
				connect.NewRequest(&v1pb.CreateIssueRequest{
					Parent: "projects/p",
					Issue:  &v1pb.Issue{Type: issueType},
				}))
			require.Error(t, got.err)
			require.False(t, got.dispatched)
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(got.err))
			require.Contains(t, got.err.Error(), issueType.String())
			require.True(t, got.auditMarked)
		})
	}

	t.Run("an issue type this build does not know is refused", func(t *testing.T) {
		// The allow-list's whole point: a type added after this build shipped
		// is refused rather than served, without anyone remembering to come
		// back here.
		got := invokeMCPGate(t, readWriteCeiling(), write, v1connect.IssueServiceCreateIssueProcedure,
			connect.NewRequest(&v1pb.CreateIssueRequest{
				Parent: "projects/p",
				Issue:  &v1pb.Issue{Type: v1pb.Issue_Type(9999)},
			}))
		require.Error(t, got.err)
		require.False(t, got.dispatched)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(got.err))
	})

	t.Run("an unset type is left to the handler", func(t *testing.T) {
		// Nothing can be created from an unspecified type — buildIssueMessage
		// rejects it as an invalid argument — so refusing it here would protect
		// nothing while telling the caller a mechanism that does not apply to
		// its call. UpdateIssue below is where that costs something real.
		got := invokeMCPGate(t, readWriteCeiling(), write, v1connect.IssueServiceCreateIssueProcedure,
			connect.NewRequest(&v1pb.CreateIssueRequest{
				Parent: "projects/p",
				Issue:  &v1pb.Issue{Type: v1pb.Issue_TYPE_UNSPECIFIED},
			}))
		require.NoError(t, got.err)
		require.True(t, got.dispatched)
	})

	// UpdateIssue reaches the gate unrefused whatever its body says, including
	// allow_missing with a grant type. The gate cannot tell an upsert that
	// creates from one that edits — "the issue does not exist" is not a field
	// of the request — and an AIP upsert sends the complete resource, so a
	// caller PATCHing an existing ROLE_GRANT issue carries that type in a body
	// the handler ignores. rejectMCPOriginatedGrantIssue guards the creation
	// where the creation happens; TestMCPGateRefusesGrantIssues (backend/tests)
	// is the end-to-end proof that the bypass stays closed.
	for name, request := range map[string]*v1pb.UpdateIssueRequest{
		"allow_missing with a grant type": {
			Issue:        &v1pb.Issue{Name: "projects/p/issues/1", Type: v1pb.Issue_ROLE_GRANT},
			AllowMissing: true,
		},
		"allow_missing with no type": {
			Issue:        &v1pb.Issue{Name: "projects/p/issues/1", Title: "retitled"},
			AllowMissing: true,
		},
		"an ordinary edit": {
			Issue: &v1pb.Issue{Name: "projects/p/issues/1", Type: v1pb.Issue_ROLE_GRANT},
		},
	} {
		t.Run("UpdateIssue is served: "+name, func(t *testing.T) {
			got := invokeMCPGate(t, readWriteCeiling(), write, v1connect.IssueServiceUpdateIssueProcedure,
				connect.NewRequest(request))
			require.NoError(t, got.err)
			require.True(t, got.dispatched)
		})
	}

	t.Run("a request the table cannot read refuses", func(t *testing.T) {
		// A wiring bug — the table keyed by a procedure whose request type is
		// something else — fails closed rather than waving the call through.
		got := invokeMCPGate(t, readWriteCeiling(), write, v1connect.IssueServiceCreateIssueProcedure,
			connect.NewRequest(&v1pb.GetUserRequest{}))
		require.Error(t, got.err)
		require.False(t, got.dispatched)
	})
}

// TestMCPGateDenialIsAuditedWithoutAnAuditAnnotation is the typed
// policy-denial record, end to end through the real audit interceptor and a
// real store. Every method below is refused and carries no audit annotation, so
// its denial produced nothing at all before the gate started marking its own
// refusals: needAudit reads the annotation and nothing else.
//
// The first four are the FORBIDDEN half of that population, which was the whole
// of it while the gate refused FORBIDDEN alone. Enforcing EXCLUDED raises it to
// 47, and TestWebhook is the fifth case here because it is the member the wider
// population added whose request carries a credential.
//
// Each request below carries a secret or an unbounded body, which is why the
// rows are checked for what they wrote as well as that they wrote. The gate
// refuses before dispatch, so nothing in them was ever used — recording one
// verbatim would turn a silent denial into a worse one.
func TestMCPGateDenialIsAuditedWithoutAnAuditAnnotation(t *testing.T) {
	st := newAuditLiveStore(t)
	auditIn := NewAuditInterceptor(st, "test-secret", &config.Profile{})
	gate := NewInternalMCPGateInterceptor(readWriteCeiling())

	invoke := func(t *testing.T, correlationID, procedure string, req connect.AnyRequest) {
		t.Helper()
		authCtx := &common.AuthContext{
			// No audit annotation: the point of the test.
			Audit:           false,
			AuthMethod:      common.AuthMethodCustom,
			MCPMethodClass:  v1pb.MCPMethodClass_FORBIDDEN,
			MCPDenialReason: v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS,
			DelegatedGrant:  &common.DelegatedGrant{CorrelationID: correlationID, Scope: "mcp:read-write"},
		}
		handlerReached := false
		handler := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			handlerReached = true
			return connect.NewResponse(&v1pb.User{}), nil
		}
		// The internal chain's order: audit outside, gate inside.
		chain := auditIn.WrapUnary(gate.WrapUnary(handler))
		_, err := chain(newAuditTestContext(authCtx), &specRequest{AnyRequest: req, procedure: procedure})
		require.Error(t, err)
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
		require.False(t, handlerReached)
	}

	assertOneDeniedRow := func(t *testing.T, correlationID, procedure string) *storepb.AuditLog {
		t.Helper()
		rows := findRowsByCorrelation(t, st, correlationID)
		require.Len(t, rows, 1, "a policy denial must be recorded even where the method asks for no audit row")
		row := rows[0].Payload
		require.Equal(t, procedure, row.Method)
		require.Equal(t, int32(connect.CodePermissionDenied), row.GetStatus().GetCode())
		require.Equal(t, "workspaces/"+auditTestWorkspace, row.Parent)
		require.Equal(t, "mcp:read-write", row.GetMcpDelegation().GetScope(),
			"the row must carry the MCP provenance an operator filters on")
		return row
	}

	t.Run("Refresh", func(t *testing.T) {
		invoke(t, "corr-refresh", v1connect.AuthServiceRefreshProcedure, connect.NewRequest(&v1pb.RefreshRequest{}))
		assertOneDeniedRow(t, "corr-refresh", v1connect.AuthServiceRefreshProcedure)
	})

	t.Run("SwitchWorkspace", func(t *testing.T) {
		// All three MFA proofs, not just the one: each is a separate masking
		// statement, and a test that populates one field cannot notice the
		// other two being dropped.
		otp, recovery, temp := "otp-123456", "recovery-abcdef", "mfa-temp-token"
		invoke(t, "corr-switch", v1connect.AuthServiceSwitchWorkspaceProcedure,
			connect.NewRequest(&v1pb.SwitchWorkspaceRequest{
				Workspace:    "workspaces/other",
				OtpCode:      &otp,
				RecoveryCode: &recovery,
				MfaTempToken: &temp,
			}))
		row := assertOneDeniedRow(t, "corr-switch", v1connect.AuthServiceSwitchWorkspaceProcedure)
		for _, proof := range []string{otp, recovery, temp} {
			require.NotContains(t, row.Request, proof, "an MFA proof must not be transcribed into the row")
		}
	})

	// One subtest per oneof arm: each arm masks a different credential, and
	// only the arm the request carries runs.
	idpRequest := func() *v1pb.TestIdentityProviderRequest {
		return &v1pb.TestIdentityProviderRequest{
			IdentityProvider: &v1pb.IdentityProvider{
				Config: &v1pb.IdentityProviderConfig{
					Config: &v1pb.IdentityProviderConfig_OidcConfig{
						OidcConfig: &v1pb.OIDCIdentityProviderConfig{ClientSecret: "idp-client-secret"},
					},
				},
			},
		}
	}
	oidc, oauth2, ldap := idpRequest(), idpRequest(), idpRequest()
	oidc.Context = &v1pb.TestIdentityProviderRequest_OidcContext{
		OidcContext: &v1pb.OIDCIdentityProviderTestRequestContext{Code: "oidc-authorization-code"},
	}
	oauth2.Context = &v1pb.TestIdentityProviderRequest_Oauth2Context{
		Oauth2Context: &v1pb.OAuth2IdentityProviderTestRequestContext{Code: "oauth2-authorization-code"},
	}
	ldap.Context = &v1pb.TestIdentityProviderRequest_LdapContext{
		LdapContext: &v1pb.LDAPIdentityProviderTestRequestContext{Username: "svc", Password: "directory-password"},
	}
	for name, arm := range map[string]struct {
		request *v1pb.TestIdentityProviderRequest
		secret  string
	}{
		"oidc":   {oidc, "oidc-authorization-code"},
		"oauth2": {oauth2, "oauth2-authorization-code"},
		"ldap":   {ldap, "directory-password"},
	} {
		t.Run("TestIdentityProvider/"+name, func(t *testing.T) {
			correlationID := "corr-idp-" + name
			invoke(t, correlationID, v1connect.IdentityProviderServiceTestIdentityProviderProcedure,
				connect.NewRequest(arm.request))
			row := assertOneDeniedRow(t, correlationID, v1connect.IdentityProviderServiceTestIdentityProviderProcedure)
			require.NotContains(t, row.Request, "idp-client-secret")
			require.NotContains(t, row.Request, arm.secret)
		})
	}

	t.Run("TestIdentityProvider/empty context arm", func(t *testing.T) {
		// A oneof wrapper set with a nil payload. It cannot arrive over JSON,
		// but the audit path must not panic on a message shape the type system
		// permits — a panic here costs the row the gate exists to write.
		invoke(t, "corr-idp-nil", v1connect.IdentityProviderServiceTestIdentityProviderProcedure,
			connect.NewRequest(&v1pb.TestIdentityProviderRequest{
				Context: &v1pb.TestIdentityProviderRequest_LdapContext{},
			}))
		assertOneDeniedRow(t, "corr-idp-nil", v1connect.IdentityProviderServiceTestIdentityProviderProcedure)
	})

	t.Run("TestWebhook", func(t *testing.T) {
		// EXCLUDED rather than FORBIDDEN, and the member the wider refused
		// population adds whose request carries a credential: an incoming
		// webhook URL posts into the customer's Slack as Bytebase, which is
		// why the audit path already masks it on AddWebhook.
		invoke(t, "corr-webhook", v1connect.ProjectServiceTestWebhookProcedure,
			connect.NewRequest(&v1pb.TestWebhookRequest{
				Project: "projects/p",
				Webhook: &v1pb.Webhook{Url: "https://hooks.example.com/T000/B000/secret-token"},
			}))
		row := assertOneDeniedRow(t, "corr-webhook", v1connect.ProjectServiceTestWebhookProcedure)
		require.NotContains(t, row.Request, "secret-token",
			"the webhook URL is a bearer credential and must not be transcribed into the row")
	})

	aiChatBody := "the whole conversation the caller sent"
	t.Run("AIService/Chat", func(t *testing.T) {
		// Not a secret but an unbounded body, and the audit row stores it
		// whole — the size cap applies to the stdout logger only. A denial
		// needs the fact of the call, not the transcript.
		invoke(t, "corr-ai", v1connect.AIServiceChatProcedure,
			connect.NewRequest(&v1pb.AIChatRequest{
				Messages: []*v1pb.AIChatMessage{{Content: &aiChatBody}},
			}))
		row := assertOneDeniedRow(t, "corr-ai", v1connect.AIServiceChatProcedure)
		require.NotContains(t, row.Request, aiChatBody)
	})

	t.Run("TestEmailSetting", func(t *testing.T) {
		invoke(t, "corr-email", v1connect.SettingServiceTestEmailSettingProcedure,
			connect.NewRequest(&v1pb.TestEmailSettingRequest{
				Parent: "workspaces/" + auditTestWorkspace,
				To:     "someone@example.com",
				EmailSetting: &v1pb.EmailSetting{
					Config: &v1pb.EmailSetting_Smtp{
						Smtp: &v1pb.EmailSetting_SMTPConfig{Host: "smtp.attacker.example", Password: "relay-password"},
					},
				},
			}))
		row := assertOneDeniedRow(t, "corr-email", v1connect.SettingServiceTestEmailSettingProcedure)
		require.NotContains(t, row.Request, "relay-password")
		require.Contains(t, row.Request, "smtp.attacker.example",
			"the host the agent named is the point of the row and stays readable")
	})
}

// mcpDenialRequestReview is one recorded decision: which fields of a request
// were looked at, and what audit.go does with them.
type mcpDenialRequestReview struct {
	fields   []string
	decision string
}

// mcpDenialRequestsUnderReview is the population the redaction sweep has to
// consider: every method carrying NO audit annotation whose request holds
// anything beyond the AIP vocabulary below. Before the gate, none of these
// produced an audit row at all; now a denial writes one, so whatever the
// request holds is what lands in the audit_log table.
//
// It covers READ and WRITE as well as the refused classes, because the gate can
// refuse those too — an uninterpretable stored ceiling refuses whatever the
// method's class is, and that refusal is a policy denial like any other.
//
// The decision is keyed on (method, field), not on the field name alone. A name
// exempted globally is an exemption for every future method that happens to
// reuse it, which is the same silent admission the exercise is trying to close:
// `schema` looks inert next to `name` and `parent`, and on DiffSchema it is the
// caller's entire DDL body.
//
// This exists because the first sweep was done by reading, and reading missed
// ListInstanceDatabase — whose request carries an entire Instance, and with it
// every data-source credential the product stores. A list nobody can add to
// silently is the only version of that sweep worth trusting.
var mcpDenialRequestsUnderReview = map[string]mcpDenialRequestReview{
	// Masked. Each carries a credential that a sibling audited method already
	// masks, and the gate refuses before dispatch, so it was never even used.
	v1connect.AIServiceChatProcedure: {
		[]string{"messages", "tool_definitions"},
		"redactAIChatRequest drops the conversation body, which is unbounded and stored untruncated",
	},
	v1connect.AuthServiceSwitchWorkspaceProcedure: {
		[]string{"mfa_temp_token", "otp_code", "recovery_code", "web", "workspace"},
		"redactSwitchWorkspaceRequest masks the three MFA proofs; the workspace it targeted and the cookie flag stay",
	},
	v1connect.IdentityProviderServiceTestIdentityProviderProcedure: {
		[]string{"identity_provider", "ldap_context", "oauth2_context", "oidc_context"},
		"redactTestIdentityProviderRequest masks the provider secret and the test credential",
	},
	v1connect.InstanceServiceListInstanceDatabaseProcedure: {
		[]string{"instance"},
		"redactInstance masks every data-source credential; pinned by TestAuditRedactsEveryInputOnlyDataSourceField",
	},
	v1connect.ProjectServiceTestWebhookProcedure: {
		[]string{"project", "webhook"},
		"redactWebhook masks the URL, which is a bearer credential; the project stays",
	},
	v1connect.SettingServiceTestEmailSettingProcedure: {
		[]string{"email_setting", "to"},
		"redactEmailSetting masks the SMTP password; the address the agent chose to mail stays, which is the point of the row",
	},

	// Recorded as sent. None of it is a credential, and what the row keeps is
	// what the agent actually attempted: the schema it proposed, the CEL it
	// wrote, the resources it named, the flags it set.
	v1connect.CelServiceBatchParseProcedure: {
		[]string{"expressions"}, "recorded: CEL the caller wrote",
	},
	v1connect.CelServiceBatchDeparseProcedure: {
		[]string{"expressions"}, "recorded: CEL the caller wrote",
	},
	v1connect.DatabaseServiceDiffMetadataProcedure: {
		[]string{"target_metadata"},
		"recorded: the caller's own proposed schema, which is the subject of the call",
	},
	v1connect.DatabaseServiceDiffSchemaProcedure: {
		[]string{"changelog", "schema"},
		"recorded: the caller's own proposed schema, or the changelog it named",
	},
	v1connect.DatabaseServiceGetSchemaStringProcedure: {
		[]string{"metadata", "object", "schema", "type"},
		"recorded: the caller's own proposed schema",
	},
	v1connect.DatabaseServiceGetDatabaseMetadataProcedure: {
		[]string{"limit"}, "recorded: a row limit",
	},
	v1connect.DatabaseServiceGetDatabaseSDLSchemaProcedure: {
		[]string{"format"}, "recorded: an output format",
	},
	v1connect.InstanceServiceBatchSyncInstancesProcedure: {
		[]string{"requests"},
		"recorded: a batch of instance names and a full-sync flag, nothing more",
	},
	v1connect.InstanceServiceSyncInstanceProcedure: {
		[]string{"enable_full_sync"}, "recorded: a sync flag",
	},
	v1connect.InstanceRoleServiceListInstanceRolesProcedure: {
		[]string{"refresh"}, "recorded: a refresh flag",
	},
	v1connect.OrgPolicyServiceListPoliciesProcedure: {
		[]string{"policy_type"}, "recorded: which policy kind was asked for",
	},
	v1connect.ReleaseServiceCheckReleaseProcedure: {
		[]string{"custom_rules", "release", "targets", "vcs_user"},
		"recorded: the caller's own release files, the same way an audited Query records its statement",
	},
	v1connect.SavedQueryServiceUpdateSavedQueryStarProcedure: {
		[]string{"starred"}, "recorded: a star flag",
	},
	v1connect.SubscriptionServiceVerifyCheckoutSessionProcedure: {
		[]string{"session_id"},
		"recorded: an opaque checkout id the caller supplied; reaching the payment provider with it needs Bytebase's own key",
	},
	v1connect.DatabaseServiceBatchGetDatabasesProcedure: {
		[]string{"names"}, "recorded: resource names",
	},
	v1connect.DatabaseServiceBatchSyncDatabasesProcedure: {
		[]string{"names"}, "recorded: resource names",
	},
	v1connect.GroupServiceBatchGetGroupsProcedure: {
		[]string{"names"}, "recorded: resource names",
	},
	v1connect.ProjectServiceBatchGetProjectsProcedure: {
		[]string{"names"}, "recorded: resource names",
	},
	v1connect.UserServiceBatchGetUsersProcedure: {
		[]string{"names"}, "recorded: resource names",
	},

	v1connect.IssueServiceListIssuesProcedure: {
		[]string{"query"}, "recorded: the caller's own search text",
	},
	v1connect.IssueServiceSearchIssuesProcedure: {
		[]string{"query"}, "recorded: the caller's own search text",
	},
	v1connect.AuthServiceGetAuthenticationRestrictionProcedure: {
		[]string{"workspace"}, "recorded: a workspace resource name",
	},
	v1connect.IssueServiceGetIssueProcedure: {
		[]string{"force"}, "recorded: a refresh flag",
	},
	v1connect.SheetServiceGetSheetProcedure: {
		[]string{"raw"}, "recorded: a flag asking for untruncated content",
	},
}

// mcpAIPRequestFields are the standard method-signature fields whose meaning is
// fixed by AIP rather than by whoever writes the next RPC: a resource name, a
// parent, the paging and filtering vocabulary, the update controls. They are
// exempt globally because their semantics cannot drift into carrying a payload
// — `page_token` is an opaque cursor on every method that has one, by
// definition.
//
// Nothing else is exempt by name, and the near misses are the reason. `schema`
// looks like vocabulary and is DiffSchema's entire DDL body; `query` looks like
// `filter` and is free text the caller wrote. A field means whatever its own
// RPC decided, so it is reviewed per method above.
var mcpAIPRequestFields = map[string]bool{
	"name": true, "parent": true, "filter": true, "order_by": true,
	"page_size": true, "page_token": true, "show_deleted": true, "view": true,
	"update_mask": true, "etag": true, "allow_missing": true, "validate_only": true,
	// The IAM-policy resource, standard on Get/SetIamPolicy.
	"resource": true,
}

// mcpRequestFieldNeedsReview reports whether a request field could carry more
// into an audit row than the AIP vocabulary does: a nested message, whose whole
// subtree travels, or any scalar the RPC named for itself.
func mcpRequestFieldNeedsReview(field protoreflect.FieldDescriptor) bool {
	if field.Kind() == protoreflect.MessageKind || field.Kind() == protoreflect.GroupKind {
		return true
	}
	return !mcpAIPRequestFields[string(field.Name())]
}

// TestLintDenialRequestsAreReviewedForRedaction fails when a method, or a field
// on one, joins the population above without anyone deciding what its denial
// row may carry.
//
// It cannot check that a redactor exists — getRequestString's type switch is
// not readable from here — so it checks the thing that failed twice: that
// somebody looked, at each field rather than at the method as a whole. A new
// unaudited RPC, or a new field on one already reviewed, breaks the build; the
// fix is a redactor plus a row here, or a row here saying why none is needed.
func TestLintDenialRequestsAreReviewedForRedaction(t *testing.T) {
	needsReview := map[string][]string{}
	for _, row := range mcpClassificationsFromDescriptors(t) {
		if row.audit {
			continue
		}
		fields := row.request.Fields()
		for i := range fields.Len() {
			if field := fields.Get(i); mcpRequestFieldNeedsReview(field) {
				needsReview[row.procedure] = append(needsReview[row.procedure], string(field.Name()))
			}
		}
		slices.Sort(needsReview[row.procedure])
	}

	var undecided []string
	for procedure, fields := range needsReview {
		reviewed, ok := mcpDenialRequestsUnderReview[procedure]
		if !ok {
			undecided = append(undecided, fmt.Sprintf("%s holds %v, and nobody has decided about it", procedure, fields))
			continue
		}
		if !slices.Equal(fields, reviewed.fields) {
			undecided = append(undecided, fmt.Sprintf(
				"%s holds %v; the recorded decision covers %v", procedure, fields, reviewed.fields))
		}
	}
	slices.Sort(undecided)
	require.Empty(t, undecided,
		"these methods are unaudited and the gate can refuse them, so a denial writes their request into an "+
			"audit row: decide what may be recorded and record it in mcpDenialRequestsUnderReview")

	var stale []string
	for procedure := range mcpDenialRequestsUnderReview {
		if _, ok := needsReview[procedure]; !ok {
			stale = append(stale, procedure)
		}
	}
	slices.Sort(stale)
	require.Empty(t, stale, "these no longer need a redaction decision; drop them so the list keeps meaning something")
}

// TestRejectMCPOriginatedGrantIssue pins the guard that sits at the creation
// rather than at the gate, and in particular pins the half that makes putting
// MCP policy inside a service handler safe: a human in the console is a request
// with no delegated grant, and must be untouched by it.
//
// The gate cannot take this decision. UpdateIssue creates only when the issue is
// missing, which is not a field of the request, and an AIP upsert sends the
// complete resource — so a caller PATCHing an existing ROLE_GRANT issue carries
// that type in a body the handler ignores. Refusing on the request shape there
// would refuse an ordinary edit for a mechanism that is not running.
func TestRejectMCPOriginatedGrantIssue(t *testing.T) {
	mcpSession := context.WithValue(context.Background(), common.AuthContextKey,
		&common.AuthContext{DelegatedGrant: &common.DelegatedGrant{CorrelationID: "corr-issue"}})
	console := context.WithValue(context.Background(), common.AuthContextKey, &common.AuthContext{})

	for name, tt := range map[string]struct {
		ctx       context.Context
		issueType v1pb.Issue_Type
		refused   bool
	}{
		"an MCP session may not create a role grant":    {mcpSession, v1pb.Issue_ROLE_GRANT, true},
		"an MCP session may not create an access grant": {mcpSession, v1pb.Issue_ACCESS_GRANT, true},
		"an MCP session may not create an unknown type": {mcpSession, v1pb.Issue_Type(9999), true},
		"an MCP session composes database changes":      {mcpSession, v1pb.Issue_DATABASE_CHANGE, false},
		"the console creates a role grant":              {console, v1pb.Issue_ROLE_GRANT, false},
		"a request with no auth context at all":         {context.Background(), v1pb.Issue_ROLE_GRANT, false},
	} {
		t.Run(name, func(t *testing.T) {
			err := rejectMCPOriginatedGrantIssue(tt.ctx, tt.issueType)
			if !tt.refused {
				require.NoError(t, err, "this guard binds MCP sessions and nothing else")
				return
			}
			require.Error(t, err)
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
			require.Contains(t, err.Error(), tt.issueType.String(), "the denial must name the type it refused")
		})
	}
}
