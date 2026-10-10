package v1

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// SQLService/Query is classified READ, but on the EngineSupportQueryNewACL
// engines it authorizes DML and DDL per statement against the caller's own
// RBAC — so its class depends on its argument, and a read-only session held by
// someone who may write would write. The clamp is the point that reads the
// argument.
//
// Ceiling: classifier-enforced, not proven. A structurally-reading statement
// can still have an effect, and a classifier can be wrong about a grammar it
// does not fully model; closing that is the conformance lane's work.

// mcpReadOnlyClampApplies reports whether this request must be held to
// read-only statements.
//
// Keyed on the grant's presence, never a field value: a legacy or
// scope-omitting session leaves every field empty, so no value marks MCP
// origin. An MCP request with no stamped ceiling means the gate never ran, and
// an unresolved ceiling is not a read-write one.
func mcpReadOnlyClampApplies(ctx context.Context) (bool, error) {
	authCtx, ok := common.GetAuthContextFromContext(ctx)
	if !ok || authCtx.DelegatedGrant == nil {
		return false, nil
	}
	settings, ok := mcpSettingsFromContext(ctx)
	if !ok {
		return false, connect.NewError(connect.CodeInternal, errors.New(
			"this MCP request cannot be checked against the workspace's MCP access policy, so it is refused"))
	}
	return settings.Capability == storepb.MCPSetting_READ_ONLY, nil
}

// refuseNonReadOnlyStatement classifies a request fail-closed. Nothing here
// calls base.ValidateSQLForEditor bare: it answers "read-only" for an engine
// with no registered validator, which suits a caller that only routes on the
// verdict and admits every write for one that must refuse a write.
//
// A statement that returns no data is refused as well as one that is not a
// read. Every unit of a request runs on one connection, and Postgres applies a
// changed default_transaction_read_only to the next transaction, so
// "SET default_transaction_read_only = off" followed by a classifier-admitted
// read disarms the depth layer and writes (verified, PG 17). The same bool
// carries the other rebinding statements — Trino USE and SET ROLE — which
// repoint the connection so a later read resolves somewhere the caller never
// named. Redis SELECT rebinds the same way but its classifier refuses it as
// not a read, so it never reaches this tier.
//
// Not closed: a structurally-reading statement can still call a function that
// rewrites the same setting (set_config, BOT-88).
func refuseNonReadOnlyStatement(engine storepb.Engine, statement string) error {
	if !parserbase.HasQueryValidator(engine) {
		return refuseClampedStatement(
			"Bytebase cannot check statements on this database engine, so none can be verified as a read",
			clampNextStep)
	}
	units := mcpClampUnits(engine, statement)
	for i, unit := range units {
		readOnly, allQuery, err := parserbase.ValidateSQLForEditor(engine, unit)
		if err != nil {
			// A parse failure is usually a syntax error, which neither way out
			// below fixes; a valid statement the parser does not model needs them.
			return refuseClampedStatement(fmt.Sprintf(
				"Bytebase could not parse %s (%v), so it cannot verify it is a read", describeClampUnit(i, len(units)), err),
				"Check the statement's syntax for this database engine. If it is valid, "+lowerFirst(clampNextStep))
		}
		if !readOnly {
			return refuseClampedStatement(fmt.Sprintf("%s is not a read", describeClampUnit(i, len(units))), clampNextStep)
		}
		if !allQuery {
			return refuseClampedStatement(fmt.Sprintf(
				"%s returns no data, and a statement like that can change the session the rest of the request runs on, "+
					"switching off its read-only protection, or run the statement it measures",
				describeClampUnit(i, len(units))), clampNextStep)
		}
	}
	return nil
}

// mcpClampUnits splits a request the way queryRetryStopOnError does, including
// its whole-request fallback for an engine with no splitter (Redis) or a failed
// split, so the clamp classifies the text that would execute.
//
// A splitter can also succeed and still return the request whole: ClickHouse
// and Hive break on newlines rather than terminators (BOT-86). Their validator
// refuses a unit carrying a terminator it did not end with, which is what keeps
// that from being read on its leading statement.
//
// MSSQL differs in the safe direction: split for analysis, sent to the driver
// whole to keep variable scope, so the clamp is stricter than the executor.
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
// for: what refused, why, and what to do instead.
func refuseClampedStatement(reason, nextStep string) error {
	return connect.NewError(connect.CodePermissionDenied, errors.Errorf(
		"This workspace's MCP access policy is Read-only, so an MCP session may only run statements Bytebase "+
			"can verify are reads: %s. %s", reason, nextStep))
}

// clampNextStep is the way out when the statement itself is sound: a policy
// that runs it, or a person who may.
const clampNextStep = "To run it, ask a workspace admin to switch the policy to Read-write under " +
	auth.MCPAccessPolicyLocation + ", or, if your role allows it, run it in the Bytebase console."

// lowerFirst lowercases a sentence's first letter so it can follow a comma.
func lowerFirst(sentence string) string {
	if sentence == "" {
		return ""
	}
	return strings.ToLower(sentence[:1]) + sentence[1:]
}
