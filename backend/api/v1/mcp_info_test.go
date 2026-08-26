package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
)

// TestMCPInfoResolvesMethodsLive shows that the served-method list follows the
// descriptors rather than any copy of them: the same resolver, run over two
// descriptor sets differing only in one method's mcp_method_class annotation,
// answers differently.
//
// This is the property the API exists for. A stored or hand-kept list would
// pass every assertion below on the day it was written and drift silently
// afterwards, and the first sign would be an admin choosing a mode from a page
// that described the build before this one.
func TestMCPInfoResolvesMethodsLive(t *testing.T) {
	asRead := mcpServedMethods(descriptorsClassifying(t, v1pb.MCPMethodClass_READ))
	require.Len(t, asRead, 1)
	require.Equal(t, "/bytebase.v1.LiveResolutionService/Probe", asRead[0].Method)
	require.Equal(t, v1pb.MCPMethodClass_READ, asRead[0].Class)
	require.Equal(t, "bb.probes.get", asRead[0].Permission)

	asWrite := mcpServedMethods(descriptorsClassifying(t, v1pb.MCPMethodClass_WRITE))
	require.Len(t, asWrite, 1)
	require.Equal(t, v1pb.MCPMethodClass_WRITE, asWrite[0].Class,
		"the response must move when the annotation moves")

	// A refused class leaves the method out entirely: the response answers what
	// can be done, so a refused entry would read as an offer.
	for _, class := range []v1pb.MCPMethodClass{
		v1pb.MCPMethodClass_FORBIDDEN,
		v1pb.MCPMethodClass_EXCLUDED,
		v1pb.MCPMethodClass_MCP_METHOD_CLASS_UNSPECIFIED,
	} {
		require.Empty(t, mcpServedMethods(descriptorsClassifying(t, class)),
			"%v is served by no ceiling", class)
	}
}

// descriptorsClassifying builds a descriptor set holding one bytebase.v1
// service with one method carrying the given classification. The file is
// compiled the same way the real ones are, so the resolver reads a real
// annotation off a real descriptor, not a stand-in for one.
func descriptorsClassifying(t *testing.T, class v1pb.MCPMethodClass) *protoregistry.Files {
	t.Helper()

	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, v1pb.E_McpMethodClass, class)
	proto.SetExtension(options, v1pb.E_Permission, "bb.probes.get")
	if class == v1pb.MCPMethodClass_FORBIDDEN || class == v1pb.MCPMethodClass_EXCLUDED {
		// The build's own lint requires a reason on a refused method; keep the
		// synthetic file consistent with that rule.
		proto.SetExtension(options, v1pb.E_McpDenialReason, v1pb.MCPDenialReason_ADMINISTERS_THE_WORKSPACE)
	}

	probe := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/live_resolution.proto"),
		Package: proto.String("bytebase.v1"),
		Syntax:  proto.String("proto3"),
		Dependency: []string{
			"v1/annotation.proto",
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("ProbeRequest")},
			{Name: proto.String("ProbeResponse")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("LiveResolutionService"),
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name:       proto.String("Probe"),
				InputType:  proto.String(".bytebase.v1.ProbeRequest"),
				OutputType: proto.String(".bytebase.v1.ProbeResponse"),
				Options:    options,
			}},
		}},
	}

	set := &descriptorpb.FileDescriptorSet{File: append(fileAndDependencies(t, "v1/annotation.proto"), probe)}
	files, err := protodesc.NewFiles(set)
	require.NoError(t, err)
	return files
}

// fileAndDependencies returns the given compiled file and everything it imports,
// dependencies first, as descriptor protos.
func fileAndDependencies(t *testing.T, path string) []*descriptorpb.FileDescriptorProto {
	t.Helper()
	seen := map[string]bool{}
	var out []*descriptorpb.FileDescriptorProto
	var walk func(fd protoreflect.FileDescriptor)
	walk = func(fd protoreflect.FileDescriptor) {
		if seen[fd.Path()] {
			return
		}
		seen[fd.Path()] = true
		imports := fd.Imports()
		for i := range imports.Len() {
			walk(imports.Get(i).FileDescriptor)
		}
		out = append(out, protodesc.ToFileDescriptorProto(fd))
	}
	fd, err := protoregistry.GlobalFiles.FindFileByPath(path)
	require.NoError(t, err)
	walk(fd)
	return out
}

// TestMCPInfoServedMethodsMatchTheServingTable holds the real response against
// the gate's own serving table: every method it lists must be one some ceiling
// serves, and every method the annotations classify as served must be listed.
func TestMCPInfoServedMethodsMatchTheServingTable(t *testing.T) {
	methods := mcpServedMethods(protoregistry.GlobalFiles)
	require.NotEmpty(t, methods)

	listed := map[string]v1pb.MCPMethodClass{}
	for _, m := range methods {
		require.False(t, auth.MCPClassIsRefused(m.Class),
			"%s is %v, which no ceiling serves, so it must not be advertised", m.Method, m.Class)
		listed[m.Method] = m.Class
	}

	byMethod := map[string]*v1pb.MCPMethod{}
	for _, m := range methods {
		byMethod[m.Method] = m
	}
	for _, row := range mcpClassificationsFromDescriptors(t) {
		if auth.MCPClassIsRefused(row.class) {
			require.NotContains(t, listed, row.procedure)
			continue
		}
		require.Equal(t, row.class, listed[row.procedure],
			"%s is classified %v and must be listed as such", row.procedure, row.class)
		// The permission is the part an agent acts on to predict a denial, so
		// it is checked on every real method, not only on a synthetic one.
		require.Equal(t, row.permission, byMethod[row.procedure].Permission,
			"%s must report the permission its descriptor declares", row.procedure)
	}

	// The order is the response's contract: protoregistry's RangeFiles order is
	// unspecified, so without the sort the list would flap between requests.
	for range 4 {
		next := mcpServedMethods(protoregistry.GlobalFiles)
		require.Len(t, next, len(methods))
		for i := range methods {
			require.Equal(t, methods[i].Method, next[i].Method, "method order must be stable")
		}
	}
}

// TestMCPInfoModesMatchTheServingTable pins that the modes the response
// describes are the gate's, not a second copy of them.
func TestMCPInfoModesMatchTheServingTable(t *testing.T) {
	modes := mcpCapabilityModes()
	require.Len(t, modes, len(mcpServingClasses))

	got := map[storepb.MCPSetting_Capability][]v1pb.MCPMethodClass{}
	for _, mode := range modes {
		capability := convertToStoreMCPCapability(mode.Capability)
		_, known := mcpServingClasses[capability]
		require.True(t, known, "%v is not a ceiling the gate decides about", mode.Capability)
		got[capability] = mode.ServedClasses
	}

	// Spelled out rather than compared against the table, so flipping what a
	// mode serves fails here instead of agreeing with itself.
	require.Empty(t, got[storepb.MCPSetting_DISABLED])
	require.Equal(t, []v1pb.MCPMethodClass{v1pb.MCPMethodClass_READ}, got[storepb.MCPSetting_READ_ONLY])
	require.Equal(t,
		[]v1pb.MCPMethodClass{v1pb.MCPMethodClass_READ, v1pb.MCPMethodClass_WRITE},
		got[storepb.MCPSetting_READ_WRITE])

	// And not the gate's own slice: a response that aliases the serving table
	// lets anything downstream write into the ceiling.
	for _, mode := range modes {
		served := mcpServingClasses[convertToStoreMCPCapability(mode.Capability)]
		if len(served) > 0 && len(mode.ServedClasses) > 0 {
			require.NotSame(t, &served[0], &mode.ServedClasses[0],
				"%v hands out the table the gate evaluates", mode.Capability)
		}
	}

	// The order is the response's contract: Go randomizes map iteration, so
	// without the sort every request would return a different one.
	for range 8 {
		next := mcpCapabilityModes()
		require.Len(t, next, len(modes))
		for i := range modes {
			require.Equal(t, modes[i].Capability, next[i].Capability, "mode order must be stable")
		}
	}
}

// TestMCPInfoReadOnlyDepthMirrorsTheDrivers holds the reported depth against
// the two things that decide it.
//
// The classifier half is asked, not mirrored: HasQueryValidator is the registry
// the clamp itself consults, so an engine gaining or losing a validator moves
// here on the same commit and this test follows it.
//
// The session half cannot be asked — a driver reports it by acting on a flag
// when a connection opens, and none exists here — so readOnlyDriverSession
// mirrors the drivers by hand. What this pins is that mcpReadOnlyDepth reports
// that table faithfully, NOT that the table matches the drivers: a fourth
// driver that starts honoring the flag leaves both unchanged and this green.
// Nothing in a unit test can catch that; the safe direction is that an
// unmirrored driver understates its own depth rather than overstating it.
func TestMCPInfoReadOnlyDepthMirrorsTheDrivers(t *testing.T) {
	// readOnlyDriverSession's table, restated so an unmirrored edit to it fails.
	sessionDepth := map[storepb.Engine]bool{
		storepb.Engine_POSTGRES:    true,
		storepb.Engine_COCKROACHDB: true,
	}

	values := storepb.Engine(0).Descriptor().Values()
	for i := range values.Len() {
		engine := storepb.Engine(values.Get(i).Number())
		if engine == storepb.Engine_ENGINE_UNSPECIFIED {
			continue
		}
		got := mcpReadOnlyDepth(engine)
		switch {
		case !parserbase.HasQueryValidator(engine):
			require.Equal(t, v1pb.MCPEngineEnforcement_UNSUPPORTED, got,
				"%v has no read-only classifier, so a read-only session is refused every statement", engine)
		case sessionDepth[engine]:
			require.Equal(t, v1pb.MCPEngineEnforcement_STATEMENT_AND_SESSION, got, "%v", engine)
		default:
			require.Equal(t, v1pb.MCPEngineEnforcement_STATEMENT, got, "%v", engine)
		}
	}

	require.NotEmpty(t, mcpEngineNote(storepb.Engine_REDSHIFT),
		"redshift's floor is only honest with the datashare exception said out loud")
}

// TestMCPInfoMaskingMirrorsTheMaskers holds the three masking states against
// the two functions the query path branches on.
//
// It does not pin the branch ORDER, and cannot: it also pins that the two
// predicates are disjoint, which is what makes the order unobservable. That
// disjointness is the real invariant — an engine in both sets would make
// document-first the only correct order, and the assertion below is what would
// catch one appearing.
//
// The third state is the one worth having: a document engine masks, so it is
// not "no masking", but the document masker's interface carries neither the
// user nor their exemptions, so ignoring exemptions changes nothing there
// (BOT-95). Collapsing it into either neighbour would overclaim.
func TestMCPInfoMaskingMirrorsTheMaskers(t *testing.T) {
	values := storepb.Engine(0).Descriptor().Values()
	var column, document, none int
	for i := range values.Len() {
		engine := storepb.Engine(values.Get(i).Number())
		if engine == storepb.Engine_ENGINE_UNSPECIFIED {
			continue
		}
		got := mcpMaskingSupport(engine)
		switch {
		case getDocumentMasker(engine) != nil:
			require.Equal(t, v1pb.MCPEngineEnforcement_DOCUMENT, got, "%v", engine)
			require.False(t, common.EngineSupportMasking(engine),
				"%v routes to the document masker, so the column masker is never reached", engine)
			document++
		case common.EngineSupportMasking(engine):
			require.Equal(t, v1pb.MCPEngineEnforcement_COLUMN, got, "%v", engine)
			column++
		default:
			require.Equal(t, v1pb.MCPEngineEnforcement_NONE, got, "%v", engine)
			none++
		}
	}
	require.Equal(t, 3, document, "cosmosdb, mongodb, elasticsearch")
	require.NotZero(t, column)
	require.NotZero(t, none)
}

// TestMCPInfoCoversEveryEngine pins that the response leaves no engine out: an
// admin choosing read-only must not have to guess about the one engine the
// table skipped.
func TestMCPInfoCoversEveryEngine(t *testing.T) {
	engines := mcpEngineEnforcement()
	values := storepb.Engine(0).Descriptor().Values()
	require.Len(t, engines, values.Len()-1, "every engine but the unspecified zero value")

	seen := map[v1pb.Engine]bool{}
	for _, e := range engines {
		require.NotEqual(t, v1pb.Engine_ENGINE_UNSPECIFIED, e.Engine,
			"an engine the v1 converter does not know would report as unspecified")
		require.False(t, seen[e.Engine], "%v listed twice", e.Engine)
		seen[e.Engine] = true
	}
}

// TestGetMCPInfoHandler drives the RPC itself, which the helper tests above
// cannot: they each hand mcpServedMethods a descriptor source, so nothing they
// assert pins that the handler passes the real registry rather than a cached or
// stored set — the one property this API exists to guarantee. They also leave
// every error branch unexercised.
func TestGetMCPInfoHandler(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	require.NoError(t, migrator.MigrateSchema(ctx, container.GetDB()))

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort())
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	service := NewWorkspaceService(stores, nil, &config.Profile{}, licenseService, nil)

	const workspaceID = "mcp-info-handler"
	_, err = stores.CreateWorkspace(ctx, &store.WorkspaceMessage{
		ResourceID: workspaceID,
		Payload:    &storepb.WorkspacePayload{Title: "MCP info handler"},
	}, "admin@example.com")
	require.NoError(t, err)

	get := func(ctx context.Context) (*v1pb.MCPInfo, error) {
		resp, err := service.GetMCPInfo(ctx, connect.NewRequest(&v1pb.GetMCPInfoRequest{}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}
	workspaceCtx := func() context.Context {
		return context.WithValue(ctx, common.WorkspaceIDContextKey, workspaceID)
	}
	setCeiling := func(t *testing.T, value string) {
		t.Helper()
		_, err := container.GetDB().ExecContext(ctx, `
			INSERT INTO setting (name, workspace, value) VALUES ('MCP', $1, $2)
			ON CONFLICT (workspace, name) DO UPDATE SET value = EXCLUDED.value`,
			workspaceID, value)
		require.NoError(t, err)
	}

	t.Run("a workspace that never configured MCP", func(t *testing.T) {
		info, err := get(workspaceCtx())
		require.NoError(t, err)
		require.Equal(t, "workspaces/"+workspaceID, info.Workspace)
		require.Equal(t, v1pb.MCPSetting_READ_WRITE, info.Capability)
		require.False(t, info.IgnoreMaskingExemptions)
		require.False(t, info.PolicyUnreadable,
			"a workspace that never configured MCP is readable, not broken")

		// The list is the live registry's, not a fixture: every entry must be a
		// method this build actually compiled, and the count must match what
		// the descriptors classify as served right now.
		require.Equal(t, len(mcpServedMethods(protoregistry.GlobalFiles)), len(info.Methods))
		byMethod := map[string]*v1pb.MCPMethod{}
		for _, m := range info.Methods {
			byMethod[m.Method] = m
		}
		query := byMethod["/bytebase.v1.SQLService/Query"]
		require.NotNil(t, query, "the handler must resolve from the compiled descriptors")
		require.Equal(t, "bytebase.v1.SQLService.Query", query.OperationId)
		require.NotContains(t, byMethod, "/bytebase.v1.AuthService/Login",
			"a FORBIDDEN method must never be advertised as reachable")

		require.Len(t, info.Modes, 3)
		require.NotEmpty(t, info.Engines)
	})

	t.Run("the toggle reaches the response", func(t *testing.T) {
		setCeiling(t, `{"capability":"READ_ONLY","ignoreMaskingExemptions":true}`)
		info, err := get(workspaceCtx())
		require.NoError(t, err)
		require.Equal(t, v1pb.MCPSetting_READ_ONLY, info.Capability)
		require.True(t, info.IgnoreMaskingExemptions,
			"every masking state is written in terms of this, so withholding it leaves the table unusable")
	})

	t.Run("a disabled workspace is still answered", func(t *testing.T) {
		setCeiling(t, `{"capability":"DISABLED"}`)
		info, err := get(workspaceCtx())
		require.NoError(t, err)
		require.Equal(t, v1pb.MCPSetting_DISABLED, info.Capability)
		require.NotEmpty(t, info.Modes,
			"an admin with MCP off is exactly who needs to see what the other modes contain")
	})

	// The two subtests below are BOT-106. Both ceilings refuse every MCP
	// connection and both used to refuse this call whole, which took the mode
	// comparison away from the admin repairing the row and left the consent
	// page with no policy to disclose.
	t.Run("a ceiling this build cannot read is described, not refused", func(t *testing.T) {
		setCeiling(t, `{"capability":"READ_ONLYY"}`)
		info, err := get(workspaceCtx())
		require.NoError(t, err, "none of the mode contents come from the stored row")
		require.True(t, info.PolicyUnreadable)
		require.Equal(t, v1pb.MCPSetting_CAPABILITY_UNSPECIFIED, info.Capability,
			"a ceiling nobody can resolve must not arrive as a mode, least of all the permissive one")
		require.False(t, info.IgnoreMaskingExemptions,
			"no MCP request runs under this ceiling, so the toggle carries neither the row's value nor a decision")
		require.Len(t, info.Modes, 3, "the comparison this page exists for")
		require.Equal(t, len(mcpServedMethods(protoregistry.GlobalFiles)), len(info.Methods))
		require.NotEmpty(t, info.Engines)
	})

	// Codex, #21254: the subtest above covers only a mistyped enum NAME, which
	// unmarshals away to unset. A wrong-TYPED value fails the whole unmarshal
	// instead, and the store wraps both in ErrMCPCapabilityUnreadable — so this
	// field is wider than MCPSetting.capability_unreadable, which never
	// describes these rows because GetSetting refuses them outright
	// (backend/tests/mcp_capability_setting_test.go). Untested, that difference
	// is invisible until someone reads the field as if the two were the same.
	//
	// The last two rows are why the field is policy_unreadable rather than
	// capability_unreadable: the capability is readable and a sibling field is
	// not, and one failed unmarshal takes the whole row with it.
	t.Run("a row that does not unmarshal is described the same way", func(t *testing.T) {
		for _, row := range []string{
			`{"capability":{}}`,
			`{"capability":true}`,
			`{"capability":1.5}`,
			`{"capability":[]}`,
			`{"capability":"READ_ONLY","ignoreMaskingExemptions":[]}`,
			`{"capability":"READ_ONLY","ignoreMaskingExemptions":"yes"}`,
		} {
			token := row
			setCeiling(t, row)
			info, err := get(workspaceCtx())
			require.NoError(t, err, "%s: the mode contents do not come from the stored row", token)
			require.True(t, info.PolicyUnreadable,
				"%s: no ceiling can be resolved from this row", token)
			require.Equal(t, v1pb.MCPSetting_CAPABILITY_UNSPECIFIED, info.Capability, token)
			require.Len(t, info.Modes, 3, "%s: the comparison this page exists for", token)
			require.Equal(t, len(mcpServedMethods(protoregistry.GlobalFiles)), len(info.Methods), token)
			require.NotEmpty(t, info.Engines, token)
		}
	})

	t.Run("a ceiling no mode serves is described by the value nobody serves", func(t *testing.T) {
		// The reserved 2, or a value a newer release wrote. It parses, so the
		// number is the answer and needs no field of its own: modes has no row
		// for it, which is what tells a client this ceiling serves nothing.
		setCeiling(t, `{"capability":2}`)
		info, err := get(workspaceCtx())
		require.NoError(t, err)
		require.False(t, info.PolicyUnreadable, "it parsed; nothing failed to read it")
		require.Equal(t, v1pb.MCPSetting_Capability(2), info.Capability)
		// Len first: the loop below passes vacuously on an empty table, and an
		// empty table is what turns every consent page into this same card.
		require.Len(t, info.Modes, 3, "the comparison this page exists for")
		for _, mode := range info.Modes {
			require.NotEqual(t, v1pb.MCPSetting_Capability(2), mode.Capability)
		}
		require.NotEmpty(t, info.Engines)
	})

	t.Run("the gate's resolution wins over a second read", func(t *testing.T) {
		// The stored ceiling still says DISABLED below; the gate admitted this
		// request under READ_ONLY. Answering from the store would report a
		// ceiling the request was not admitted under.
		setCeiling(t, `{"capability":"DISABLED"}`)
		stamped := withMCPSettings(workspaceCtx(), store.MCPSettings{
			Capability:              storepb.MCPSetting_READ_ONLY,
			IgnoreMaskingExemptions: true,
		})
		info, err := get(stamped)
		require.NoError(t, err)
		require.Equal(t, v1pb.MCPSetting_READ_ONLY, info.Capability)
		require.True(t, info.IgnoreMaskingExemptions)
	})

	t.Run("no workspace on the request", func(t *testing.T) {
		_, err := get(ctx)
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	})
}
