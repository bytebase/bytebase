package v1

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
)

// GetMCPInfo reports what MCP does in this workspace.
//
// Everything is resolved when the request is served. The method list comes off
// the compiled descriptors — the same place the gate reads a method's class —
// and the engine answers off the code that enforces them, so this cannot
// describe rules the build does not run. A stored list would drift silently:
// the first sign would be an admin acting on a mode that no longer means what
// the page said.
func (s *WorkspaceService) GetMCPInfo(ctx context.Context, _ *connect.Request[v1pb.GetMCPInfoRequest]) (*connect.Response[v1pb.MCPInfo], error) {
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	if workspaceID == "" {
		return nil, connect.NewError(connect.CodeInternal, errors.New("no workspace on the request"))
	}
	settings, err := s.mcpSettingsForInfo(ctx, workspaceID)
	// Every verdict but an outage is answered, the three refusing ceilings
	// included: the mode contents do not come from the stored row, so a broken
	// one still owes the admin repairing it the comparison, and the consent page
	// a policy to disclose. capability_unreadable carries the refusal (BOT-106).
	var unreadable bool
	switch verdict := auth.ClassifyMCPCeiling(settings.Capability, err); verdict {
	case auth.MCPCeilingServes, auth.MCPCeilingDisabled, auth.MCPCeilingUnserved:
	case auth.MCPCeilingUnreadable:
		unreadable = true
	default:
		// The store error stays in the log. This method is served to MCP
		// sessions and the tool layer renders a connect message into what the
		// model reads, so a driver error text would leave the metadata
		// database's shape in an agent's context.
		//
		// An outage is the one verdict with no ceiling to describe. Answering
		// with an empty one would read as a stored value nobody can resolve,
		// which an admin repairs rather than a retry clearing it.
		slog.Error("failed to read the MCP setting", slog.String("workspace", workspaceID), log.BBError(err))
		return nil, connect.NewError(connect.CodeUnavailable, errors.New(verdict.Refusal()))
	}

	return connect.NewResponse(&v1pb.MCPInfo{
		Workspace:               common.FormatWorkspace(workspaceID),
		Capability:              convertToV1MCPCapability(settings.Capability),
		PolicyUnreadable:        unreadable,
		Modes:                   mcpCapabilityModes(),
		Methods:                 mcpServedMethods(protoregistry.GlobalFiles),
		Engines:                 mcpEngineEnforcement(),
		IgnoreMaskingExemptions: settings.IgnoreMaskingExemptions,
		DataMaskingAvailable:    s.licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_DATA_MASKING) == nil,
	}), nil
}

// mcpSettingsForInfo returns the MCP settings this request is answered from.
//
// On the internal MCP chain the gate already resolved them and stamped them on
// the context; reading again could answer with a ceiling the request was not
// admitted under, which is the rule mcp_gate.go states and the clamp and the
// masking check both follow. Off that chain — the console, an admin opening the
// settings page — nothing is stamped and the store is read.
func (s *WorkspaceService) mcpSettingsForInfo(ctx context.Context, workspaceID string) (store.MCPSettings, error) {
	if settings, ok := mcpSettingsFromContext(ctx); ok {
		return settings, nil
	}
	return s.store.GetMCPSettingsUncached(ctx, workspaceID)
}

// mcpCapabilityModes renders the serving table the gate evaluates, one row per
// ceiling. It reads mcpServingClasses rather than restating it, so a mode that
// starts or stops serving a class cannot leave this stale.
func mcpCapabilityModes() []*v1pb.MCPCapabilityMode {
	modes := make([]*v1pb.MCPCapabilityMode, 0, len(mcpServingClasses))
	for capability, served := range mcpServingClasses {
		modes = append(modes, &v1pb.MCPCapabilityMode{
			Capability: convertToV1MCPCapability(capability),
			// Cloned: the response must not hand out the slice the gate
			// evaluates against on every request.
			ServedClasses: slices.Clone(served),
		})
	}
	slices.SortFunc(modes, func(a, b *v1pb.MCPCapabilityMode) int { return cmp.Compare(a.Capability, b.Capability) })
	return modes
}

// fileRanger is the descriptor set to resolve classifications from.
// *protoregistry.Files satisfies it; a test supplies its own, which is what
// lets "the response follows the descriptors" be shown rather than asserted.
type fileRanger interface {
	RangeFiles(func(protoreflect.FileDescriptor) bool)
}

// mcpServedMethods reads every bytebase.v1 RPC's classification off the
// descriptors and returns the ones some ceiling serves, sorted by procedure.
//
// Methods no ceiling serves are left out: the response answers "what can be
// done here", and an agent handed the refused list would have to be told
// separately that those entries are not offers.
func mcpServedMethods(files fileRanger) []*v1pb.MCPMethod {
	var methods []*v1pb.MCPMethod
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Package() != "bytebase.v1" {
			return true
		}
		services := fd.Services()
		for i := range services.Len() {
			sd := services.Get(i)
			descriptors := sd.Methods()
			for j := range descriptors.Len() {
				md := descriptors.Get(j)
				options, ok := md.Options().(*descriptorpb.MethodOptions)
				if !ok {
					continue
				}
				class, ok := proto.GetExtension(options, v1pb.E_McpMethodClass).(v1pb.MCPMethodClass)
				if !ok {
					// A malformed annotation is not a classification. CI
					// rejects one, and no ceiling serves what it resolves to.
					continue
				}
				if auth.MCPClassIsRefused(class) {
					continue
				}
				method := &v1pb.MCPMethod{
					Method:      "/" + string(sd.FullName()) + "/" + string(md.Name()),
					OperationId: string(sd.FullName()) + "." + string(md.Name()),
					Class:       class,
				}
				if permission, ok := proto.GetExtension(options, v1pb.E_Permission).(string); ok {
					method.Permission = permission
				}
				if authMethod, ok := proto.GetExtension(options, v1pb.E_AuthMethod).(v1pb.AuthMethod); ok {
					method.AuthMethod = authMethod
				}
				methods = append(methods, method)
			}
		}
		return true
	})
	slices.SortFunc(methods, func(a, b *v1pb.MCPMethod) int { return strings.Compare(a.Method, b.Method) })
	return methods
}

// mcpEngineEnforcement answers, per engine, how deep a read-only ceiling goes
// and whether masking reaches anything — the two facts the ceiling alone does
// not show. Every engine the build knows appears, in declaration order.
func mcpEngineEnforcement() []*v1pb.MCPEngineEnforcement {
	values := storepb.Engine(0).Descriptor().Values()
	engines := make([]*v1pb.MCPEngineEnforcement, 0, values.Len())
	for i := range values.Len() {
		engine := storepb.Engine(values.Get(i).Number())
		if engine == storepb.Engine_ENGINE_UNSPECIFIED {
			continue
		}
		engines = append(engines, &v1pb.MCPEngineEnforcement{
			Engine:        convertToEngine(engine),
			ReadOnlyDepth: mcpReadOnlyDepth(engine),
			Masking:       mcpMaskingSupport(engine),
			Note:          mcpEngineNote(engine),
		})
	}
	return engines
}

// mcpReadOnlyDepth reports how much of a read-only ceiling this engine can be
// held to.
//
// The classifier half is asked, not mirrored: parserbase.HasQueryValidator is
// the same registry the clamp consults, so an engine that gains or loses a
// validator moves here on the same commit.
func mcpReadOnlyDepth(engine storepb.Engine) v1pb.MCPEngineEnforcement_ReadOnlyDepth {
	if !parserbase.HasQueryValidator(engine) {
		return v1pb.MCPEngineEnforcement_UNSUPPORTED
	}
	if readOnlyDriverSession(engine) {
		return v1pb.MCPEngineEnforcement_STATEMENT_AND_SESSION
	}
	return v1pb.MCPEngineEnforcement_STATEMENT
}

// readOnlyDriverSession reports whether this engine's driver always turns
// ConnectionContext.ReadOnly into a read-only database session, by setting
// default_transaction_read_only on the connection it opens. Two always do:
// postgres and cockroachdb. Redshift does too, but only outside a datashare
// database, which cannot run a read-only transaction — that is per database,
// off its metadata, and this answer is per engine, so redshift is reported at
// its floor and its note carries the rest (plugin/db/pg, /cockroachdb,
// /redshift).
//
// It mirrors the drivers rather than asking them, because a driver reports this
// by acting on a flag when a connection is opened and no connection exists
// here. A driver that starts honoring the flag without a row here understates
// its own depth, which is the safe way for this to be wrong.
func readOnlyDriverSession(engine storepb.Engine) bool {
	switch engine {
	case storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB:
		return true
	default:
		return false
	}
}

// mcpMaskingSupport reports how Bytebase masks results on this engine, which is
// what decides whether ignoring masking exemptions changes anything.
//
// Document first, because that is the order the query path checks them
// (sql_service.go): an engine with a document masker never reaches the column
// masker. Exemptions are a column-masking mechanism — the document masker's
// interface carries neither the user nor their exemptions — so on a document
// engine the toggle has nothing to suppress (BOT-95).
func mcpMaskingSupport(engine storepb.Engine) v1pb.MCPEngineEnforcement_Masking {
	switch {
	case getDocumentMasker(engine) != nil:
		return v1pb.MCPEngineEnforcement_DOCUMENT
	case common.EngineSupportMasking(engine):
		return v1pb.MCPEngineEnforcement_COLUMN
	default:
		return v1pb.MCPEngineEnforcement_NONE
	}
}

// mcpEngineNote carries the cases where a per-engine answer is the floor rather
// than the whole story.
//
// The console does not render this string: it keys off the engine and reads
// settings.mcp.contents.notes.<engine> so the caveat is translated. Reword here
// and the locale files need the same edit. Adding a note for a new engine
// without its locale entry degrades to a line saying a caveat exists and to
// reload, which is also what an older bundle shows against a newer server — it
// never puts this English in a non-English console. An agent reading
// GetMCPInfo does get this text, which is why it stays prose rather than
// becoming an enum.
func mcpEngineNote(engine storepb.Engine) string {
	if engine == storepb.Engine_REDSHIFT {
		return "Outside a datashare database the driver also opens the session read-only. " +
			"A datashare database cannot run a read-only transaction, so there it is statement classification alone."
	}
	return ""
}
