package v1

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// The SQL clamp is the third point that holds an MCP session to the workspace
// ceiling, and the only one that reads what the request carries rather than
// which method it is.
//
// SQLService/Query is classified READ, so the method gate serves it under a
// read-only ceiling. But what Query does is decided by its statement: on every
// engine in EngineSupportQueryNewACL the handler classifies each statement and
// authorizes DML with bb.sql.dml and DDL with bb.sql.ddl, against the caller's
// own RBAC. A read-only session held by someone who may write would therefore
// write. The clamp closes that: under a READ_ONLY ceiling every statement in
// the request must classify as a read, or the whole request is refused.
//
// What it claims is classifier-enforced read-only, reinforced by a read-only
// driver session where the engine has one. It is not a proof that nothing can
// write: a statement that reads structurally can still call a side-effecting
// function, and a classifier can be wrong about a grammar it does not fully
// model. Those are the conformance lane's business, not this gate's.

// mcpCeilingContextKey carries the ceiling the MCP gate resolved for this
// request. The gate reads the workspace's ceiling once per request; stamping it
// here is what lets the clamp hold the same request against the same value,
// rather than reading the setting a second time and possibly disagreeing with
// the gate that already admitted the call.
//
// It travels inward — gate to handler — so it is a plain context value, not the
// callback shape SetMCPPolicyDenied needs to reach the audit interceptor
// wrapping the gate from outside.
type mcpCeilingContextKey struct{}

func withMCPCeiling(ctx context.Context, ceiling storepb.WorkspaceProfileSetting_MCPCapability) context.Context {
	return context.WithValue(ctx, mcpCeilingContextKey{}, ceiling)
}

func mcpCeilingFromContext(ctx context.Context) (storepb.WorkspaceProfileSetting_MCPCapability, bool) {
	ceiling, ok := ctx.Value(mcpCeilingContextKey{}).(storepb.WorkspaceProfileSetting_MCPCapability)
	return ceiling, ok
}

// mcpReadOnlyClampApplies reports whether this request must be held to
// read-only statements.
//
// It keys on the presence of the delegated grant, never on any of its values:
// a legacy or scope-omitting session leaves every grant field empty, so no
// value is a sound proxy for MCP origin. Public-chain requests carry no grant
// at all and are untouched.
//
// An MCP request that carries no ceiling means the gate did not run, which can
// only happen if the internal chain was reordered. The clamp refuses rather
// than guess: an unresolved ceiling is not a read-write one.
func mcpReadOnlyClampApplies(ctx context.Context) (bool, error) {
	authCtx, ok := common.GetAuthContextFromContext(ctx)
	if !ok || authCtx.DelegatedGrant == nil {
		return false, nil
	}
	ceiling, ok := mcpCeilingFromContext(ctx)
	if !ok {
		return false, connect.NewError(connect.CodeInternal, errors.New(
			"this MCP request cannot be checked against the workspace MCP capability ceiling, so it fails closed"))
	}
	return ceiling == storepb.WorkspaceProfileSetting_READ_ONLY, nil
}

// refuseNonReadOnlyStatement is the fail-closed classifier the clamp owns.
//
// It exists because base.ValidateSQLForEditor answers "read-only" for an engine
// with no registered validator: that default suits a caller who only routes or
// formats with the verdict, and admits every write for a caller who must refuse
// one. Nothing here calls it bare.
//
// A request is refused when the engine has no classifier, when a statement will
// not parse, or when any statement classifies as anything but a read — so a
// batch is served only if every statement in it reads.
func refuseNonReadOnlyStatement(engine storepb.Engine, statement string) error {
	if !parserbase.HasQueryValidator(engine) {
		return refuseClampedStatement(fmt.Sprintf(
			"Bytebase has no read-only classifier for %v, so no statement on this engine can be shown to be a read", engine))
	}
	units := mcpClampUnits(engine, statement)
	for i, unit := range units {
		readOnly, _, err := parserbase.ValidateSQLForEditor(engine, unit)
		if err != nil {
			return refuseClampedStatement(fmt.Sprintf(
				"%s could not be parsed, so it cannot be shown to be a read: %v", describeClampUnit(i, len(units)), err))
		}
		if !readOnly {
			return refuseClampedStatement(fmt.Sprintf("%s is not a read", describeClampUnit(i, len(units))))
		}
	}
	return nil
}

// mcpClampUnits returns the statements the driver will run, so the clamp
// classifies exactly the text that would execute — including the whole-request
// fallback queryRetryStopOnError itself takes when an engine has no splitter
// (Redis) or the split fails. Classifying a different unit than the executor
// runs is how a batch gets past a per-statement rule.
func mcpClampUnits(engine storepb.Engine, statement string) []string {
	statements, err := parserbase.SplitMultiSQL(engine, statement)
	if err != nil {
		return []string{statement}
	}
	var units []string
	for _, s := range statements {
		if s.Empty {
			continue
		}
		units = append(units, s.Text)
	}
	if len(units) == 0 {
		return []string{statement}
	}
	return units
}

// describeClampUnit names the offending statement the way a denial should read
// for a request holding one statement and for a batch alike.
func describeClampUnit(index, total int) string {
	if total <= 1 {
		return "the statement"
	}
	return fmt.Sprintf("statement %d of %d", index+1, total)
}

// refuseClampedStatement wraps a reason in the denial the gate set the shape
// for: what refused, why, and the two ways out.
func refuseClampedStatement(reason string) error {
	return connect.NewError(connect.CodePermissionDenied, errors.Errorf(
		"this workspace's MCP capability ceiling is READ_ONLY, so an MCP session may only run statements Bytebase can show are reads: %s. "+
			"Ask a workspace admin to raise the MCP ceiling in the workspace settings, "+
			"or run this statement signed in to the Bytebase console instead", reason))
}
