package aiaccess

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// The lint. Every clause is a function over the tables, so the same code runs
// against a deliberately broken copy — each clause has a RED test below that
// mutates one input and asserts the clause fires. A clause with no RED test is
// a clause nobody has proved can fail.

// checkExactlyOnce proves every v1 RPC resolves exactly once: forbidden, or
// explicitly mapped, or derived from a permission annotation. A method matching
// none of the three fails here, which is how a newly added RPC that carries no
// annotation fails CI instead of silently resolving.
//
// It does NOT catch a new RPC that reuses an existing permission annotation —
// that method resolves by derivation and joins whichever sets already hold the
// permission. The inventory golden file is what surfaces those.
func checkExactlyOnce(procedures []string, forbidden map[string]string, explicit map[string]methodEntry) []string {
	var violations []string
	known := make(map[string]bool, len(procedures))
	for _, procedure := range procedures {
		known[procedure] = true
		_, isForbidden := forbidden[procedure]
		entry, isExplicit := explicit[procedure]
		annotated := false
		authContext, err := auth.GetAuthContext(procedure)
		switch {
		case err != nil:
			violations = append(violations, fmt.Sprintf("%s: no auth context: %v", procedure, err))
		case authContext.Permission != "":
			annotated = true
		default:
		}

		if isForbidden && isExplicit {
			violations = append(violations, procedure+": both forbidden and explicitly mapped")
		}
		if !isForbidden && !isExplicit && !annotated {
			violations = append(violations, procedure+": unclassified — no permission annotation and no explicit capability mapping")
		}
		// An entry that requires nothing is admitted by every ceiling, so an
		// unfinished row must fail here rather than ship as a grant.
		if isExplicit && len(entry.capabilities) == 0 {
			violations = append(violations, procedure+": explicit mapping requires no capability")
		}
		// An explicit entry may cover an annotated method only by requiring at
		// least what the ACL requires; it may never launder a method into a
		// cheaper capability than its annotation. This binds only where the
		// annotation binds: the ACL evaluates it for auth_method IAM and skips
		// it otherwise (backend/api/v1/acl.go doIAMPermissionCheck), so on a
		// custom-authorization method the annotation is documentation and an
		// entry may name what the handler really allows instead.
		enforced := annotated && authContext.AuthMethod == common.AuthMethodIAM
		if isExplicit && enforced && !slices.Contains(entry.capabilities, Capability(authContext.Permission)) {
			violations = append(violations, fmt.Sprintf("%s: explicit mapping %v drops its annotated permission %q", procedure, entry.capabilities, authContext.Permission))
		}
	}
	for procedure := range forbidden {
		if !known[procedure] {
			violations = append(violations, procedure+": forbidden entry names a method that does not exist")
		}
	}
	for procedure := range explicit {
		if !known[procedure] {
			violations = append(violations, procedure+": explicit entry names a method that does not exist")
		}
	}
	slices.Sort(violations)
	return violations
}

// checkSetMembersInUniverse proves a set cannot name a capability nothing can
// require — a typo, or a permission that was renamed out from under it.
func checkSetMembersInUniverse(sets []*Set, universe []Capability) []string {
	var violations []string
	for _, set := range sets {
		for _, capability := range set.Capabilities() {
			if !slices.Contains(universe, capability) {
				violations = append(violations, fmt.Sprintf("%s: %q is outside the capability universe", set.Name(), capability))
			}
		}
	}
	slices.Sort(violations)
	return violations
}

// checkCapabilitiesAreReal proves the universe itself is real: every
// non-operation capability is a registered IAM permission, and every operation
// is minted. Without this, a typo in an explicit entry or in the statement list
// defines its own universe entry and validates itself.
func checkCapabilitiesAreReal(universe []Capability) []string {
	var violations []string
	for _, capability := range universe {
		if capability.IsOperation() {
			if !slices.Contains(operations, capability) {
				violations = append(violations, fmt.Sprintf("%q is not a minted operation", capability))
			}
			continue
		}
		if !permission.Exist(string(capability)) {
			violations = append(violations, fmt.Sprintf("%q is not a registered IAM permission", capability))
		}
	}
	slices.Sort(violations)
	return violations
}

// checkForbiddenNotInSets proves the forbidden list is absolute: no set admits a
// forbidden method, whatever it contains.
func checkForbiddenNotInSets(sets []*Set, forbidden map[string]string) []string {
	var violations []string
	for procedure := range forbidden {
		requirement, err := Required(procedure, nil)
		if err != nil {
			violations = append(violations, fmt.Sprintf("%s: %v", procedure, err))
			continue
		}
		if !requirement.Forbidden {
			violations = append(violations, procedure+": resolves to a capability instead of denial")
		}
		for _, set := range sets {
			if set.Allows(requirement) {
				violations = append(violations, fmt.Sprintf("%s: admitted by %s", procedure, set.Name()))
			}
		}
	}
	slices.Sort(violations)
	return violations
}

// checkNoAdminOrApprovalInSets proves decision 7: a mutation is not
// automatically a write, and administration is in no predefined set.
func checkNoAdminOrApprovalInSets(sets []*Set, admin, approval []Capability) []string {
	var violations []string
	for _, set := range sets {
		for _, capability := range set.Capabilities() {
			if slices.Contains(admin, capability) {
				violations = append(violations, fmt.Sprintf("%s: %q is an administration capability", set.Name(), capability))
			}
			if slices.Contains(approval, capability) {
				violations = append(violations, fmt.Sprintf("%s: %q is an approval capability", set.Name(), capability))
			}
		}
	}
	slices.Sort(violations)
	return violations
}

// checkUniversePartitioned proves the classification is total: every capability
// is read-write, administration, or approval, exactly one. A capability
// introducing a new permission therefore fails CI until somebody classifies it,
// rather than landing in no set (a silent denial) or two classes at once.
func checkUniversePartitioned(universe, readWrite, admin, approval []Capability) []string {
	var violations []string
	for _, capability := range universe {
		count := 0
		for _, class := range [][]Capability{readWrite, admin, approval} {
			if slices.Contains(class, capability) {
				count++
			}
		}
		if count != 1 {
			violations = append(violations, fmt.Sprintf("%q is in %d classes, want exactly 1 (read-write, administration, approval)", capability, count))
		}
	}
	for _, class := range [][]Capability{readWrite, admin, approval} {
		for _, capability := range class {
			if !slices.Contains(universe, capability) {
				violations = append(violations, fmt.Sprintf("%q is classified but outside the universe", capability))
			}
		}
	}
	slices.Sort(violations)
	return violations
}

func TestLintEveryMethodResolvesExactlyOnce(t *testing.T) {
	require.Empty(t, checkExactlyOnce(V1Procedures(), forbiddenMethods, explicitMethods))
}

func TestLintSetMembersAreInTheUniverse(t *testing.T) {
	universe, err := Universe()
	require.NoError(t, err)
	require.Empty(t, checkSetMembersInUniverse([]*Set{ReadOnly, ReadWrite}, universe))
}

func TestLintEveryCapabilityIsReal(t *testing.T) {
	universe, err := Universe()
	require.NoError(t, err)
	require.Empty(t, checkCapabilitiesAreReal(universe))
}

func TestLintForbiddenIsInNoSet(t *testing.T) {
	require.Empty(t, checkForbiddenNotInSets([]*Set{ReadOnly, ReadWrite}, forbiddenMethods))
}

func TestLintNoAdminOrApprovalCapabilityInAnySet(t *testing.T) {
	require.Empty(t, checkNoAdminOrApprovalInSets([]*Set{ReadOnly, ReadWrite}, adminCapabilities, approvalCapabilities))
}

func TestLintUniverseIsPartitioned(t *testing.T) {
	universe, err := Universe()
	require.NoError(t, err)
	require.Empty(t, checkUniversePartitioned(universe, ReadWrite.Capabilities(), adminCapabilities, approvalCapabilities))
}

// Read-write and read-only are two independently maintained literals, so this
// containment is a real property rather than a consequence of how they are
// built.
func TestReadOnlyIsContainedInReadWrite(t *testing.T) {
	for _, capability := range ReadOnly.Capabilities() {
		require.True(t, ReadWrite.Contains(capability), "read-write must contain %q", capability)
	}
	require.Greater(t, len(ReadWrite.Capabilities()), len(ReadOnly.Capabilities()))
}

func TestSortedCapabilities(t *testing.T) {
	require.Equal(t,
		[]Capability{"ai.op.workspaces.get", "bb.databases.get", "bb.databases.list"},
		sortedCapabilities([]Capability{"bb.databases.list", "bb.databases.get", "ai.op.workspaces.get", "bb.databases.get"}))
	require.Empty(t, sortedCapabilities(nil))
}

func TestCapabilityIsOperation(t *testing.T) {
	require.True(t, OpRolloutsCreate.IsOperation())
	require.False(t, Capability(permission.DatabasesGet).IsOperation())
	require.False(t, Capability("").IsOperation())
	// The bare prefix is malformed rather than a permission; the well-formed
	// shape check is what rejects it, so IsOperation must route it there.
	require.True(t, Capability(OperationPrefix).IsOperation())
}

// Operation IDs are public API surface, so their shape is pinned: every minted
// ID is used by at least one method, every operation a method requires is
// minted, and all of them read like proto fields.
func TestOperationIDsAreWellFormedAndUsed(t *testing.T) {
	shape := regexp.MustCompile(`^ai\.op\.[a-z][a-zA-Z]*\.[a-z][a-zA-Z]*$`)
	used := map[Capability]bool{}
	for _, procedure := range V1Procedures() {
		requirement, err := Required(procedure, nil)
		require.NoError(t, err)
		for _, capability := range requirement.Capabilities {
			if capability.IsOperation() {
				require.Contains(t, operations, capability, "%s requires unminted operation %q", procedure, capability)
				used[capability] = true
			}
		}
	}
	for _, operation := range operations {
		require.Regexp(t, shape, string(operation))
		require.True(t, used[operation], "operation %q is minted but no method requires it", operation)
	}
}

// The four methods whose annotation is shared with a lower-risk sibling must
// still carry that annotation, which is what keeps the split from lowering the
// bar, and the split must actually separate them.
func TestAnnotationSharingMethodsKeepTheirPermission(t *testing.T) {
	for procedure, want := range map[string][]Capability{
		"/bytebase.v1.SQLService/Export":                {OpDatabasesExport, Capability(permission.DatabasesGet)},
		"/bytebase.v1.DatabaseService/DiffSchema":       {OpDatabasesDiffSchema, Capability(permission.DatabasesGet)},
		"/bytebase.v1.RolloutService/GetTaskRunSession": {OpTaskRunsGetSession, Capability(permission.TaskRunsList)},
		"/bytebase.v1.ActuatorService/SetupSample":      {Capability(permission.InstancesCreate), Capability(permission.ProjectsCreate)},
	} {
		requirement, err := Required(procedure, nil)
		require.NoError(t, err)
		require.Equal(t, want, requirement.Capabilities, "%s", procedure)
		require.False(t, ReadOnly.Allows(requirement), "read-only must deny %s", procedure)
	}

	// The siblings that share each annotation stay admitted — the split has to
	// separate them, not deny both.
	for _, procedure := range []string{
		"/bytebase.v1.SQLService/Query",
		"/bytebase.v1.RolloutService/GetTaskRun",
		"/bytebase.v1.ProjectService/CreateProject",
	} {
		requirement, err := Required(procedure, nil)
		require.NoError(t, err)
		require.True(t, ReadWrite.Allows(requirement), "read-write must admit %s", procedure)
	}
}

// The requirement is resolved per request wherever the ACL resolves it per
// request. ListInstanceDatabase carrying an inline instance connects to a
// caller-supplied host, and the ACL raises it to bb.instances.create; the
// ceiling has to see the same raise or a read-only session keeps the probe.
func TestRequiredFollowsThePerRequestRaise(t *testing.T) {
	const procedure = "/bytebase.v1.InstanceService/ListInstanceDatabase"

	stored, err := Required(procedure, &v1pb.ListInstanceDatabaseRequest{Name: "instances/prod"})
	require.NoError(t, err)
	require.Equal(t, []Capability{Capability(permission.InstancesGet)}, stored.Capabilities)
	require.True(t, ReadOnly.Allows(stored))

	inline, err := Required(procedure, &v1pb.ListInstanceDatabaseRequest{Instance: &v1pb.Instance{}})
	require.NoError(t, err)
	require.Equal(t, []Capability{Capability(permission.InstancesCreate)}, inline.Capabilities)
	require.False(t, ReadOnly.Allows(inline), "read-only must deny the inline-instance probe")
	require.False(t, ReadWrite.Allows(inline), "read-write must deny the inline-instance probe")
}

// The ACL's other per-request raise: allow_missing turns an update into a
// create and demands both permissions, so a ceiling holding only the update
// capability must deny it.
func TestRequiredFollowsTheAllowMissingRaise(t *testing.T) {
	const procedure = "/bytebase.v1.DatabaseGroupService/UpdateDatabaseGroup"

	update, err := Required(procedure, &v1pb.UpdateDatabaseGroupRequest{})
	require.NoError(t, err)
	require.Equal(t, []Capability{Capability(permission.DatabaseGroupsUpdate)}, update.Capabilities)

	create, err := Required(procedure, &v1pb.UpdateDatabaseGroupRequest{AllowMissing: true})
	require.NoError(t, err)
	require.Equal(t, []Capability{Capability(permission.DatabaseGroupsCreate), Capability(permission.DatabaseGroupsUpdate)}, create.Capabilities)

	// Both presets hold the pair, so both admit it; a set holding only the
	// update capability does not, which is the property that matters for the
	// admin-authored sets in 1b-5.
	require.True(t, ReadWrite.Allows(create))
	updateOnly := newSet("UPDATE_ONLY", []Capability{Capability(permission.DatabaseGroupsUpdate)})
	require.True(t, updateOnly.Allows(update))
	require.False(t, updateOnly.Allows(create))
}

func TestForbiddenMethodsAreDeniedWithAReason(t *testing.T) {
	requirement, err := Required("/bytebase.v1.AuthService/Login", nil)
	require.NoError(t, err)
	require.True(t, requirement.Forbidden)
	require.NotEmpty(t, requirement.Reason)
	require.Empty(t, requirement.Capabilities)
	require.False(t, ReadWrite.Allows(requirement))
}

// Every error path returns a denying requirement, so a caller that logs the
// error and proceeds still denies.
func TestRequiredFailsClosed(t *testing.T) {
	requirement, err := Required("/bytebase.v1.NoSuchService/NoSuchMethod", nil)
	require.Error(t, err)
	require.True(t, requirement.Forbidden)
	require.False(t, ReadOnly.Allows(requirement))
	require.False(t, ReadWrite.Allows(requirement))

	// And a requirement that demands nothing is denied even by a set holding
	// the entire universe.
	universe, err := Universe()
	require.NoError(t, err)
	require.False(t, newSet("EVERYTHING", universe).Allows(Requirement{}))
}

// The derivation must read the ACL's own source. If a method's annotation
// changes, its capability changes with it, with no table to update.
func TestDerivationTracksTheAnnotation(t *testing.T) {
	for _, procedure := range V1Procedures() {
		if _, ok := forbiddenMethods[procedure]; ok {
			continue
		}
		if _, ok := explicitMethods[procedure]; ok {
			continue
		}
		authContext, err := auth.GetAuthContext(procedure)
		require.NoError(t, err)
		requirement, err := Required(procedure, nil)
		require.NoError(t, err)
		require.Equal(t, []Capability{Capability(authContext.Permission)}, requirement.Capabilities, "%s", procedure)
		require.NotEmpty(t, requirement.Reason)
	}
}

// RED evidence. Each test breaks one input and asserts the clause fires.

func TestLintCatchesAnUnclassifiedMethod(t *testing.T) {
	// GetWorksheet carries no permission annotation, so dropping its explicit
	// entry leaves it unresolvable — the state a newly added custom-auth RPC
	// starts in.
	const procedure = "/bytebase.v1.WorksheetService/GetWorksheet"
	violations := checkExactlyOnce(V1Procedures(), forbiddenMethods, withoutEntry(procedure))
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], procedure)
	require.Contains(t, violations[0], "unclassified")
}

func TestLintCatchesAnEmptyExplicitMapping(t *testing.T) {
	// The shape an unfinished row takes: a reason, no capabilities. Without the
	// clause it resolves to "requires nothing", which every ceiling admits.
	const procedure = "/bytebase.v1.SQLService/GetQueryHistory"
	broken := withoutEntry(procedure)
	broken[procedure] = methodEntry{reason: "TODO classify"}
	violations := checkExactlyOnce(V1Procedures(), forbiddenMethods, broken)
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "requires no capability")

	requirement, err := Required(procedure, nil)
	require.NoError(t, err)
	require.NotEmpty(t, requirement.Capabilities)
}

func TestLintCatchesAnExplicitMappingThatDropsItsAnnotation(t *testing.T) {
	const procedure = "/bytebase.v1.SQLService/Export"
	broken := withoutEntry(procedure)
	broken[procedure] = methodEntry{capabilities: []Capability{OpDatabasesExport}}
	violations := checkExactlyOnce(V1Procedures(), forbiddenMethods, broken)
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "drops its annotated permission")
}

func TestLintCatchesADoubleClassifiedMethod(t *testing.T) {
	const procedure = "/bytebase.v1.SQLService/Export"
	forbidden := maps.Clone(forbiddenMethods)
	forbidden[procedure] = "pretend this is a credential endpoint"
	violations := checkExactlyOnce(V1Procedures(), forbidden, explicitMethods)
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "both forbidden and explicitly mapped")
}

func TestLintCatchesAStaleProcedureName(t *testing.T) {
	// What a table row becomes after the RPC behind it is renamed.
	forbidden := maps.Clone(forbiddenMethods)
	forbidden["/bytebase.v1.AuthService/LogOut"] = "renamed away"
	violations := checkExactlyOnce(V1Procedures(), forbidden, explicitMethods)
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "forbidden entry names a method that does not exist")

	broken := withoutEntry("/bytebase.v1.WorksheetService/GetWorksheet")
	broken["/bytebase.v1.WorksheetService/GetWorkSheet"] = methodEntry{capabilities: []Capability{OpWorksheetsGet}}
	violations = checkExactlyOnce(V1Procedures(), forbiddenMethods, broken)
	require.Len(t, violations, 2)
	require.Contains(t, violations[0], "explicit entry names a method that does not exist")
	require.Contains(t, violations[1], "unclassified")
}

func TestLintCatchesASetMemberOutsideTheUniverse(t *testing.T) {
	universe, err := Universe()
	require.NoError(t, err)
	broken := newSet("BROKEN", []Capability{"bb.databases.gett"})
	violations := checkSetMembersInUniverse([]*Set{broken}, universe)
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "outside the capability universe")
}

func TestLintCatchesACapabilityThatIsNeitherPermissionNorOperation(t *testing.T) {
	// A typo inside an explicit entry or the statement list would otherwise
	// enter the universe and validate itself against it.
	violations := checkCapabilitiesAreReal([]Capability{"bb.sql.selectt"})
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "not a registered IAM permission")

	violations = checkCapabilitiesAreReal([]Capability{"ai.op.databases.exportt"})
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "not a minted operation")
}

func TestLintCatchesAForbiddenMethodAdmittedByASet(t *testing.T) {
	universe, err := Universe()
	require.NoError(t, err)
	everything := newSet("EVERYTHING", universe)
	require.Empty(t, checkForbiddenNotInSets([]*Set{everything}, forbiddenMethods))

	// A method dropped from the forbidden list is admitted by a set the moment
	// it resolves to a capability instead of denial.
	notForbidden := map[string]string{"/bytebase.v1.SQLService/Query": "pretend this is a credential endpoint"}
	violations := checkForbiddenNotInSets([]*Set{ReadOnly}, notForbidden)
	require.Len(t, violations, 2)
	require.Contains(t, violations[0], "admitted by READ_ONLY")
	require.Contains(t, violations[1], "resolves to a capability instead of denial")
}

func TestLintCatchesAnAdminCapabilityInASet(t *testing.T) {
	broken := newSet("BROKEN", []Capability{Capability(permission.SettingsGet)})
	violations := checkNoAdminOrApprovalInSets([]*Set{broken}, adminCapabilities, approvalCapabilities)
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "administration capability")

	approver := newSet("APPROVER", []Capability{OpIssuesApprove})
	violations = checkNoAdminOrApprovalInSets([]*Set{approver}, adminCapabilities, approvalCapabilities)
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "approval capability")
}

func TestLintCatchesAnUnclassifiedCapability(t *testing.T) {
	universe, err := Universe()
	require.NoError(t, err)
	universe = append(universe, "bb.newFeature.get")
	violations := checkUniversePartitioned(universe, ReadWrite.Capabilities(), adminCapabilities, approvalCapabilities)
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "bb.newFeature.get")
	require.Contains(t, violations[0], "is in 0 classes")
}

func TestLintCatchesADoubleClassifiedCapability(t *testing.T) {
	universe, err := Universe()
	require.NoError(t, err)
	admin := append(slices.Clone(adminCapabilities), OpIssuesApprove)
	violations := checkUniversePartitioned(universe, ReadWrite.Capabilities(), admin, approvalCapabilities)
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "is in 2 classes")
}

func TestLintCatchesAClassifiedCapabilityOutsideTheUniverse(t *testing.T) {
	// What a class entry becomes when the RPC that required it is deleted.
	universe, err := Universe()
	require.NoError(t, err)
	trimmed := slices.DeleteFunc(slices.Clone(universe), func(c Capability) bool {
		return c == Capability(permission.SettingsGet)
	})
	violations := checkUniversePartitioned(trimmed, ReadWrite.Capabilities(), adminCapabilities, approvalCapabilities)
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "classified but outside the universe")
}

func withoutEntry(procedure string) map[string]methodEntry {
	out := maps.Clone(explicitMethods)
	delete(out, procedure)
	return out
}
