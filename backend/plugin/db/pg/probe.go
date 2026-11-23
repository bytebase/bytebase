package pg

import (
	"context"
	"fmt"

	omndriver "github.com/bytebase/omni/driver"
	pgdriver "github.com/bytebase/omni/pg/driver"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// CountAffectedRows returns the planner's estimate of the rows the statement
// modifies. A search path that put a schema before the statement is applied for
// the EXPLAIN in a transaction that is rolled back.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.impl.CountAffectedRows(ctx, statement)
}

// Probe runs a live review rule's probe on one session of the driver's.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return d.ProbeOmni(ctx, pCtx, statement, d.probeOptions())
}

// probeOnSession runs the transaction, the tenant-mode prelude, the
// pre-executions, the statement and the rollback on one session, so the owner
// the prelude switches to is the one the statement plans as.
func (d *Driver) probeOnSession(ctx context.Context, session omndriver.Session, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return util.ProbeOmniSession(ctx, session, pCtx, statement, d.probeOptions())
}

func (d *Driver) probeOptions() util.OmniProbeOptions {
	var prepareTx func(context.Context, omndriver.Session) ([]string, error)
	if d.connectionCtx.TenantMode {
		prepareTx = func(ctx context.Context, session omndriver.Session) ([]string, error) {
			owner, err := probeDatabaseOwner(ctx, session)
			if err != nil {
				return nil, err
			}
			// SET SESSION ROLE, as the executor's tenant-mode prelude does, so the
			// statement plans as the owner the migration will run as.
			return []string{fmt.Sprintf("SET SESSION ROLE '%s';", owner)}, nil
		}
	}
	return util.OmniProbeOptions{
		BeginStatement:           "BEGIN;",
		RollbackStatement:        "ROLLBACK;",
		PrepareTx:                prepareTx,
		CheckPreExecutionResults: true,
		RequireResult:            true,
		FormatCell:               probeCell,
	}
}

// probeDatabaseOwner reads the current database's owner role on the probe's own
// session.
func probeDatabaseOwner(ctx context.Context, session omndriver.Session) (string, error) {
	const query = `
		SELECT
			u.rolname
		FROM
			pg_roles AS u JOIN pg_database AS d ON (d.datdba = u.oid)
		WHERE
			d.datname = current_database();
		`
	results, err := session.Query(ctx, query, omndriver.QueryOptions{})
	if err != nil {
		return "", err
	}
	if len(results) != 1 {
		return "", errors.New("failed to get database owner: no result")
	}
	if results[0].Err != nil {
		return "", results[0].Err
	}
	if len(results[0].Rows) != 1 || len(results[0].Rows[0]) != 1 || results[0].Rows[0][0].Kind != omndriver.String {
		return "", errors.New("failed to get database owner: unexpected result shape")
	}
	return results[0].Rows[0][0].String, nil
}

// probeCell converts one cell for a rule. A binary or temporal cell is text, as
// the *sql.DB probe's column-type scan read it.
func probeCell(value omndriver.Value) any {
	switch value.Kind {
	case omndriver.Null:
		return nil
	case omndriver.Bool:
		return value.Bool
	case omndriver.Int64, omndriver.Int32, omndriver.Uint64, omndriver.Uint32:
		return value.Int64
	case omndriver.Float64, omndriver.Float32:
		return value.Float64
	case omndriver.Bytes:
		return string(value.Bytes)
	case omndriver.Timestamp, omndriver.TimestampTZ:
		return pgdriver.FormatTimestamp(value.Time, value.Kind == omndriver.TimestampTZ)
	default:
		return value.String
	}
}

// TaskRunSession reads the session that carries applicationName with the
// sessions it blocks and the sessions blocking it.
func (d *Driver) TaskRunSession(ctx context.Context, applicationName string) (*v1pb.TaskRunSession, error) {
	activity, err := d.impl.Activity(ctx, applicationName)
	if err != nil {
		return nil, err
	}
	postgres := &v1pb.TaskRunSession_Postgres{}
	if activity.Session != nil {
		postgres.Session = taskRunSession(activity.Session)
	}
	for _, session := range activity.BlockedSessions {
		postgres.BlockedSessions = append(postgres.BlockedSessions, taskRunSession(session))
	}
	for _, session := range activity.BlockingSessions {
		postgres.BlockingSessions = append(postgres.BlockingSessions, taskRunSession(session))
	}
	return &v1pb.TaskRunSession{
		Session: &v1pb.TaskRunSession_Postgres_{Postgres: postgres},
	}, nil
}

func taskRunSession(session *pgdriver.BackendSession) *v1pb.TaskRunSession_Postgres_Session {
	converted := &v1pb.TaskRunSession_Postgres_Session{
		Pid:             session.PID,
		BlockedByPids:   session.BlockedByPIDs,
		Query:           session.Query,
		State:           session.State,
		WaitEventType:   session.WaitEventType,
		WaitEvent:       session.WaitEvent,
		Datname:         session.Database,
		Usename:         session.User,
		ApplicationName: session.ApplicationName,
		ClientAddr:      session.ClientAddr,
		ClientPort:      session.ClientPort,
		BackendStart:    timestamppb.New(session.BackendStart),
	}
	if session.XactStart != nil {
		converted.XactStart = timestamppb.New(*session.XactStart)
	}
	if session.QueryStart != nil {
		converted.QueryStart = timestamppb.New(*session.QueryStart)
	}
	return converted
}
