package v1

import (
	"bytes"
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/masker"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// mcpSession reports whether this request arrived over MCP.
//
// Keyed on the grant's presence, never a field value: a legacy or
// scope-omitting session leaves every field empty, so no value marks MCP
// origin. A console session carries no delegated grant.
func mcpSession(ctx context.Context) bool {
	authCtx, ok := common.GetAuthContextFromContext(ctx)
	return ok && authCtx.DelegatedGrant != nil
}

// mcpIgnoresMaskingExemptions reports whether this request stops applying the
// caller's own unmasking provisioning.
//
// No stamped settings means the gate never ran. Ignoring the exemptions is the
// safe default there, which is why this answers true where
// mcpReadOnlyClampApplies errors: guessing a ceiling has no safe default.
func mcpIgnoresMaskingExemptions(ctx context.Context) bool {
	if !mcpSession(ctx) {
		return false
	}
	settings, ok := mcpSettingsFromContext(ctx)
	if !ok {
		return true
	}
	return settings.IgnoreMaskingExemptions
}

// maskedWriteRefusal is the reason to refuse SQL carrying a mask sentinel, or
// "" to allow: the agent writes back the placeholder it read, and the real data
// is gone with nothing in the change looking wrong.
//
// It applies to every MCP session, not only where the workspace ignores masking
// exemptions. Masking runs under ordinary policy for any user who holds no
// exemption, so the corruption is reachable in a workspace that never touched
// the toggle — which is most of them.
//
// Only masker.DefaultFullMaskSubstitution is fixed in code: a full mask emits
// it for any value when no substitution is configured, and the range,
// default-range and MD5 maskers emit it on their NULL and bool arms. Everything
// else is undetectable — an MD5 mask is 32 hex characters, indistinguishable
// from a real hash; a range or inner/outer mask embeds real characters from the
// value; and a mask configured with its own substitution emits that admin's
// text — though an all-asterisk substitution reaches the sentinel by arithmetic.
//
// The scan is a substring test over raw statement text, so six asterisks
// written for another reason — an asterisk banner in a comment, a rule of
// stars — are refused too. That is the deliberate direction: a false refusal is
// a message the agent can act on, a missed write-back is data gone silently.
// Narrowing it to string literals means parsing per engine and failing open on
// a parse error; narrowing it to a bounded run of exactly six lets
// "******/2024" and two adjacent placeholders through.
func maskedWriteRefusal(ctx context.Context, statement []byte) string {
	if !mcpSession(ctx) {
		return ""
	}
	if !bytes.Contains(statement, []byte(masker.DefaultFullMaskSubstitution)) {
		return ""
	}
	return fmt.Sprintf(
		"its SQL contains %q, the placeholder Bytebase substitutes for a masked value rather than "+
			"a value anything holds. Writing it back would replace the real data with it, and "+
			"matching on it would match nothing. Remove the literal, whatever the statement does "+
			"with it", masker.DefaultFullMaskSubstitution)
}

// refuseMaskedWriteSheet guards the sheet door. PlanService/CreatePlan carries
// a sheet name and no statement of its own, there is no UpdateSheet, and a
// rollout runs whatever the plan's sheet holds — so propose_database_change and
// a hand-rolled call_api sequence both land here.
func refuseMaskedWriteSheet(ctx context.Context, msg any) string {
	request, ok := msg.(*v1pb.CreateSheetRequest)
	if !ok {
		// The table is keyed by procedure, so the request type is fixed; a
		// mismatch is a wiring bug, and a refusal path meeting one refuses.
		return fmt.Sprintf("its request could not be read as a sheet (%T)", msg)
	}
	return maskedWriteRefusal(ctx, request.GetSheet().GetContent())
}

// refuseMaskedWriteSheetBatch refuses the whole batch on one offending sheet,
// matching the clamp.
func refuseMaskedWriteSheetBatch(ctx context.Context, msg any) string {
	request, ok := msg.(*v1pb.BatchCreateSheetsRequest)
	if !ok {
		return fmt.Sprintf("its request could not be read as a sheet batch (%T)", msg)
	}
	for _, r := range request.GetRequests() {
		if reason := maskedWriteRefusal(ctx, r.GetSheet().GetContent()); reason != "" {
			return reason
		}
	}
	return ""
}

// refuseMaskedWriteQuery guards the execution door. SQLService/Query is READ
// class, but on the EngineSupportQueryNewACL engines it authorizes DML per
// statement, and the read-only clamp only runs under a READ_ONLY ceiling — so
// under READ_WRITE the placeholder goes straight at the table, no sheet needed.
//
// A statement that only reads is refused too: separating the two would mean
// classifying it, which fails open on an engine with no validator.
func refuseMaskedWriteQuery(ctx context.Context, msg any) string {
	request, ok := msg.(*v1pb.QueryRequest)
	if !ok {
		return fmt.Sprintf("its request could not be read as a query (%T)", msg)
	}
	return maskedWriteRefusal(ctx, []byte(request.GetStatement()))
}

// refuseMaskedWriteExport guards the same door under another name. Export takes
// a raw statement, and on MySQL it skips validateQueryRequest (sql_service.go)
// and reaches a driver that ExecContexts anything not all-query — so an export
// request writes.
func refuseMaskedWriteExport(ctx context.Context, msg any) string {
	request, ok := msg.(*v1pb.ExportRequest)
	if !ok {
		return fmt.Sprintf("its request could not be read as an export (%T)", msg)
	}
	return maskedWriteRefusal(ctx, []byte(request.GetStatement()))
}

// refuseMaskedWriteRelease guards the release door. Release.File carries an
// inline statement, CreateRelease turns it into a sheet through
// store.CreateSheets rather than the RPC above, and a plan's
// ChangeDatabaseConfig takes a release in place of a sheet.
//
// UpdateRelease is not guarded because its handler answers Unimplemented, and
// CheckRelease because it runs the advisors and persists nothing.
//
// A file may name a sheet instead of carrying a statement, and that content is
// not read here — CreateRelease checks the reference by hash and never loads it
// either. Sound today only because an MCP session cannot author such a sheet:
// CreateSheet and BatchCreateSheets are guarded, there is no UpdateSheet RPC,
// so sheet content is immutable and every agent-written sheet passed this same
// scan. A referenced sheet holding the placeholder was written by a human, who
// meant it. Closing it properly means checking at the point a statement
// executes rather than at each of the doors that compose one (BOT-98).
func refuseMaskedWriteRelease(ctx context.Context, msg any) string {
	request, ok := msg.(*v1pb.CreateReleaseRequest)
	if !ok {
		return fmt.Sprintf("its request could not be read as a release (%T)", msg)
	}
	for _, file := range request.GetRelease().GetFiles() {
		if reason := maskedWriteRefusal(ctx, file.GetStatement()); reason != "" {
			return reason
		}
	}
	return ""
}

// refuseMaskedWriteSavedQuery guards the saved-query door. A saved query is its
// own table, not a sheet, so none of the doors above sees this statement: the
// content becomes the SQL a SQL Editor tab loads, and a person runs it later.
func refuseMaskedWriteSavedQuery(ctx context.Context, msg any) string {
	request, ok := msg.(*v1pb.CreateSavedQueryRequest)
	if !ok {
		return fmt.Sprintf("its request could not be read as a saved query (%T)", msg)
	}
	return maskedWriteRefusal(ctx, request.GetSavedQuery().GetContent())
}

// refuseMaskedWriteSavedQueryUpdate guards the same door on the update arm:
// content is an update-mask path, so a session holding bb.savedQueries.update
// rewrites a query somebody else authored and already runs.
//
// The mask is not consulted. Scanning only when it names "content" would tie
// this guard to the handler continuing to ignore an unnamed content field. The
// cost is refusing a title-only update that echoes back a stored statement
// holding six asterisks.
func refuseMaskedWriteSavedQueryUpdate(ctx context.Context, msg any) string {
	request, ok := msg.(*v1pb.UpdateSavedQueryRequest)
	if !ok {
		return fmt.Sprintf("its request could not be read as a saved query update (%T)", msg)
	}
	return maskedWriteRefusal(ctx, request.GetSavedQuery().GetContent())
}
