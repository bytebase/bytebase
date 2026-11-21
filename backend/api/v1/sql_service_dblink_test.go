package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/permission"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
)

const wantAuditMark = "the audit interceptor stamps a refusal WARNING only when the request is marked"

func TestRemoteColumnRefusal(t *testing.T) {
	const connected = "oracle-dblink"
	local := parserbase.ColumnResource{Database: "ALLOWED_S", Table: "ALLOWED_T"}
	sameInstance := parserbase.ColumnResource{Instance: connected, Server: "REMOTE", Database: "SECRET_SCHEMA", Table: "SECRET_T"}
	otherInstance := parserbase.ColumnResource{Instance: "oracle-remote", Server: "REMOTE3", Database: "APP", Table: "T"}
	unresolved := parserbase.ColumnResource{Server: "REMOTE2", Database: "SECRET_SCHEMA", Table: "SECRET_T"}
	unresolvedUnqualified := parserbase.ColumnResource{Server: "REMOTE2", Table: "SECRET_T"}
	unresolvedLocalName := parserbase.ColumnResource{Server: "REMOTE2", Database: "ALLOWED_S", Table: "ALLOWED_T"}

	tests := []struct {
		name    string
		columns []parserbase.ColumnResource
		want    string
	}{
		{"local only", []parserbase.ColumnResource{local}, ""},
		{"link resolved to the connected instance", []parserbase.ColumnResource{local, sameInstance}, ""},
		{"link resolved to another instance", []parserbase.ColumnResource{local, otherInstance}, `table APP.T reached through database link "REMOTE3" is on instance "oracle-remote"`},
		{"unresolved link", []parserbase.ColumnResource{local, unresolved}, `table SECRET_SCHEMA.SECRET_T reached through database link "REMOTE2" cannot be authorized: Bytebase could not establish which database it reaches. A link is authorized when the connect string Bytebase synced for it names the host, port and service of a data source of exactly one instance`},
		{"unresolved link, unqualified table", []parserbase.ColumnResource{unresolvedUnqualified}, `table SECRET_T reached through database link "REMOTE2"`},
		{"unresolved link naming a local database", []parserbase.ColumnResource{unresolvedLocalName}, `table ALLOWED_S.ALLOWED_T reached through database link "REMOTE2"`},
	}
	for _, tc := range tests {
		columns := make(parserbase.SourceColumnSet, len(tc.columns))
		for _, column := range tc.columns {
			columns[column] = true
		}
		marked := false
		ctx := withSetPermissionDenied(context.Background(), func() { marked = true })
		got := remoteColumnRefusal(ctx, columns, connected, permission.SQLSelect)
		if tc.want == "" {
			require.Nil(t, got, tc.name)
			require.False(t, marked, tc.name)
			continue
		}
		require.NotNil(t, got, tc.name)
		require.Contains(t, got.Error(), tc.want, tc.name)
		require.Nil(t, got.resources, "%s: the refusal must offer no resource to request access to", tc.name)
		require.Equal(t, permission.SQLSelect, got.permission, tc.name)
		require.True(t, marked, "%s: %s", tc.name, wantAuditMark)
	}
}

func TestLinkedTargetProjectRefusal(t *testing.T) {
	column := parserbase.ColumnResource{Instance: "oracle-dblink", Server: "REMOTE", Database: "SECRET_SCHEMA", Table: "SECRET_T"}
	tests := []struct {
		name   string
		target *store.DatabaseMessage
		want   string
	}{
		{"target in the request's project", &store.DatabaseMessage{ProjectID: "default"}, ""},
		{"target Bytebase does not track", nil, `SECRET_SCHEMA.SECRET_T reached through database link "REMOTE" is in a database Bytebase does not track`},
		{"target in another project", &store.DatabaseMessage{ProjectID: "other"}, `is in project "other"`},
	}
	for _, tc := range tests {
		marked := false
		ctx := withSetPermissionDenied(context.Background(), func() { marked = true })
		got := linkedTargetProjectRefusal(ctx, column, tc.target, "default", permission.SQLSelect)
		if tc.want == "" {
			require.Nil(t, got, tc.name)
			require.False(t, marked, tc.name)
			continue
		}
		require.NotNil(t, got, tc.name)
		require.Contains(t, got.Error(), tc.want, tc.name)
		require.Nil(t, got.resources, "%s: the refusal must offer no resource to request access to", tc.name)
		require.Equal(t, permission.SQLSelect, got.permission, tc.name)
		require.True(t, marked, "%s: %s", tc.name, wantAuditMark)
	}
}

func TestPrivateLinkRefusal(t *testing.T) {
	linked := []parserbase.ColumnResource{{Instance: "oracle-dblink", Server: "REMOTE", Database: "SECRET_SCHEMA", Table: "SECRET_T"}}
	marked := false
	ctx := withSetPermissionDenied(context.Background(), func() { marked = true })

	require.NoError(t, privateLinkRefusal(ctx, linked, false, nil))
	require.False(t, marked, "no refusal, no mark")

	// An outage is not a verdict about the caller's permission.
	failed := privateLinkRefusal(ctx, linked, false, errors.New("ORA-00942"))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(failed))
	require.Contains(t, failed.Error(), "ORA-00942")
	require.Contains(t, failed.Error(), `reached through database link "REMOTE" is not executed`)
	require.False(t, marked, "a failed check must stay unmarked")

	owns := privateLinkRefusal(ctx, linked, true, nil)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(owns))
	require.Contains(t, owns.Error(), `table SECRET_SCHEMA.SECRET_T reached through database link "REMOTE" cannot be authorized: the executing account owns private database links`)
	require.True(t, marked, wantAuditMark)
}

func TestLinkedColumns(t *testing.T) {
	local := parserbase.ColumnResource{Database: "ALLOWED_S", Table: "ALLOWED_T"}
	linkedB := parserbase.ColumnResource{Instance: "oracle-dblink", Server: "REMOTE", Database: "SECRET_SCHEMA", Table: "B"}
	linkedA := parserbase.ColumnResource{Instance: "oracle-dblink", Server: "REMOTE", Database: "SECRET_SCHEMA", Table: "A"}
	unresolved := parserbase.ColumnResource{Server: "REMOTE2", Database: "SECRET_SCHEMA", Table: "SECRET_T"}
	spans := []*parserbase.QuerySpan{
		nil,
		{SourceColumns: parserbase.SourceColumnSet{local: true, linkedB: true}},
		{SourceColumns: parserbase.SourceColumnSet{linkedA: true, unresolved: true}},
	}
	require.Equal(t, []parserbase.ColumnResource{linkedA, linkedB}, linkedColumns(spans))
	require.Empty(t, linkedColumns(nil))
	require.Empty(t, linkedColumns([]*parserbase.QuerySpan{{SourceColumns: parserbase.SourceColumnSet{local: true, unresolved: true}}}))
}

// privateLinkDriver is the shape a driver over omni's contract has: a
// private-link probe that asks with a session of its own.
type privateLinkDriver struct {
	db.Driver
	owns bool
	err  error
}

func (d *privateLinkDriver) OwnsPrivateDatabaseLink(context.Context) (bool, error) {
	return d.owns, d.err
}

// plainDriver is a driver of the same shape without the private-link probe.
type plainDriver struct {
	db.Driver
}

// TestLinkedReadReachesTheStatementAfterTheCutover is the trap this cutover had
// to disarm: the guard used to take the caller's *sql.Conn and refuse every
// linked read when there was none, which a driver over omni's contract cannot
// provide. The guard now asks the driver, so a linked read whose account owns no
// private link reaches the statement.
func TestLinkedReadReachesTheStatementAfterTheCutover(t *testing.T) {
	instance := &store.InstanceMessage{ResourceID: "oracle-dblink", Metadata: &storepb.Instance{Engine: storepb.Engine_ORACLE}}
	linked := []*parserbase.QuerySpan{{SourceColumns: parserbase.SourceColumnSet{
		{Instance: "oracle-dblink", Server: "REMOTE", Database: "SECRET_SCHEMA", Table: "SECRET_T"}: true,
	}}}
	nonAdmin := db.QueryContext{DataSourceType: storepb.DataSourceType_READ_ONLY}

	guard := func(driver db.Driver, queryContext db.QueryContext, spans []*parserbase.QuerySpan) (error, bool) {
		marked := false
		ctx := withSetPermissionDenied(context.Background(), func() { marked = true })
		return refuseLinkedReadsShadowedByPrivateLinks(ctx, driver, instance, queryContext, spans), marked
	}

	got, marked := guard(&privateLinkDriver{}, nonAdmin, linked)
	require.NoError(t, got, "the account owns no private link, so the linked read reaches the statement")
	require.False(t, marked, "an allowed read must not be marked")

	got, marked = guard(&privateLinkDriver{owns: true}, nonAdmin, linked)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(got))
	require.True(t, marked)

	got, _ = guard(&privateLinkDriver{err: errors.New("ORA-00942")}, nonAdmin, linked)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(got))

	got, _ = guard(&privateLinkDriver{owns: true}, db.QueryContext{DataSourceType: storepb.DataSourceType_ADMIN}, linked)
	require.NoError(t, got, "the admin data source is the account the sync recorded")

	got, _ = guard(&privateLinkDriver{owns: true}, nonAdmin, nil)
	require.NoError(t, got, "a read that reaches no linked database is not the guard's to refuse")

	got, _ = guard(&plainDriver{}, nonAdmin, linked)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(got))
	require.Contains(t, got.Error(), "cannot report the executing account's private database links")
}
