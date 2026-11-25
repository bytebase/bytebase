package v1

import (
	"context"
	"database/sql"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
)

func instanceOn(engine storepb.Engine) *store.InstanceMessage {
	return &store.InstanceMessage{
		ResourceID: "inst",
		Metadata:   &storepb.Instance{Engine: engine},
	}
}

func unresolvedSpan() *parserbase.QuerySpan {
	return &parserbase.QuerySpan{
		Type: parserbase.Select,
		UnresolvedColumnsError: &parserbase.UnresolvedColumnsError{
			Relations: []parserbase.ColumnResource{
				{Database: "db", Schema: "public", Table: "t"},
			},
		},
	}
}

func TestMaskingBlockedByUnresolvedColumns(t *testing.T) {
	testCases := []struct {
		name     string
		span     *parserbase.QuerySpan
		instance *store.InstanceMessage
		want     bool
	}{
		{
			name:     "unresolved columns on a masking engine blocks",
			span:     unresolvedSpan(),
			instance: instanceOn(storepb.Engine_POSTGRES),
			want:     true,
		},
		{
			// CockroachDB shares the PostgreSQL extractor but does not support masking.
			name:     "engine that never masks does not block",
			span:     unresolvedSpan(),
			instance: instanceOn(storepb.Engine_COCKROACHDB),
			want:     false,
		},
		{
			name:     "resolved span does not block",
			span:     &parserbase.QuerySpan{Type: parserbase.Select},
			instance: instanceOn(storepb.Engine_POSTGRES),
			want:     false,
		},
		{
			name:     "nil span does not block",
			span:     nil,
			instance: instanceOn(storepb.Engine_POSTGRES),
			want:     false,
		},
		{
			name:     "nil instance does not block",
			span:     unresolvedSpan(),
			instance: nil,
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, maskingBlockedByUnresolvedColumns(tc.span, tc.instance))
		})
	}
}

// These engines must enforce the signal once their extractors produce it.
func TestMaskingEnginesAgreeWithSignalProducers(t *testing.T) {
	for _, engine := range []storepb.Engine{
		storepb.Engine_MYSQL,
		storepb.Engine_ORACLE,
		storepb.Engine_MSSQL,
		storepb.Engine_REDSHIFT,
	} {
		require.True(t, maskingBlockedByUnresolvedColumns(unresolvedSpan(), instanceOn(engine)),
			"%s masks, so it must act on the signal once its extractor sets one", engine)
	}
}

func TestColumnsMisalignedWithSpan(t *testing.T) {
	spanOf := func(n int) *parserbase.QuerySpan {
		return &parserbase.QuerySpan{Type: parserbase.Select, Results: make([]parserbase.QuerySpanResult, n)}
	}
	resultOf := func(columns, rows int) *v1pb.QueryResult {
		r := &v1pb.QueryResult{ColumnNames: make([]string, columns)}
		for range rows {
			r.Rows = append(r.Rows, &v1pb.QueryRow{Values: make([]*v1pb.RowValue, columns)})
		}
		return r
	}
	testCases := []struct {
		name   string
		span   *parserbase.QuerySpan
		result *v1pb.QueryResult
		want   bool
	}{
		{name: "same column count", span: spanOf(3), result: resultOf(3, 2), want: false},
		{name: "more columns than the span resolved", span: spanOf(4), result: resultOf(5, 2), want: true},
		{name: "fewer columns than the span resolved", span: spanOf(3), result: resultOf(2, 2), want: true},
		{name: "no rows", span: spanOf(3), result: resultOf(0, 0), want: false},
		{name: "DML affected rows", span: &parserbase.QuerySpan{Type: parserbase.DML}, result: resultOf(1, 1), want: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, columnsMisalignedWithSpan(tc.span, tc.result))
		})
	}
}

const queryMaskingWorkspace = "query-masking"

func newQueryMaskingFixture(t *testing.T) (context.Context, *sql.DB, *store.Store, *store.InstanceMessage) {
	t.Helper()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, queryMaskingWorkspace)
	db, stores, _ := testcontainer.NewMetadataDBWithCache(t, true)
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ($1)`, queryMaskingWorkspace)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', $1, 'A')`, queryMaskingWorkspace)
	require.NoError(t, err)
	instance, err := stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "test", Workspace: queryMaskingWorkspace,
		Metadata: &storepb.Instance{Engine: storepb.Engine_POSTGRES, Activation: true, DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}}},
	})
	require.NoError(t, err)
	_, err = stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID: "test", DatabaseName: "db", ProjectID: "project-a", Metadata: &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	stored, err := stores.UpsertDBSchema(ctx, "test", "db", &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{{
				Name: "records", Columns: []*metadatapb.ColumnMetadata{{Name: "contents", Type: "text"}},
			}},
		}},
	}, nil, time.Now())
	require.NoError(t, err)
	require.True(t, stored)
	return ctx, db, stores, instance
}

func queryMaskingSpans() (parserbase.ColumnResource, []*parserbase.QuerySpan) {
	column := parserbase.ColumnResource{Instance: "test", Database: "db", Schema: "public", Table: "records", Column: "contents"}
	columns := parserbase.SourceColumnSet{column: true}
	return column, []*parserbase.QuerySpan{{
		Results:       []parserbase.QuerySpanResult{{Name: "contents", SourceColumns: columns, IsPlainField: true}},
		SourceColumns: columns, PredicateColumns: columns,
	}}
}

func queryMaskingResult(value string) *v1pb.QueryResult {
	return &v1pb.QueryResult{
		ColumnNames: []string{"contents"},
		Rows:        []*v1pb.QueryRow{{Values: []*v1pb.RowValue{{Kind: &v1pb.RowValue_StringValue{StringValue: value}}}}},
	}
}

func putQueryMaskingRule(ctx context.Context, t *testing.T, stores *store.Store, expression, semanticType string) {
	t.Helper()
	payload, err := protojson.Marshal(&storepb.MaskingRulePolicy{Rules: []*storepb.MaskingRulePolicy_MaskingRule{{
		Id: "rule", Condition: &expr.Expr{Expression: expression}, SemanticType: semanticType,
	}}})
	require.NoError(t, err)
	_, err = stores.CreatePolicy(ctx, &store.PolicyMessage{
		Workspace: queryMaskingWorkspace, Resource: common.FormatWorkspace(queryMaskingWorkspace), ResourceType: storepb.Policy_WORKSPACE,
		Type: storepb.Policy_MASKING_RULE, Enforce: true, Payload: string(payload),
	})
	require.NoError(t, err)
}

func TestQueryMaskingPrincipalGroups(t *testing.T) {
	t.Parallel()
	const (
		workspace = queryMaskingWorkspace
		other     = "other-workspace"
		email     = "caller@example.com"
		secret    = "example-value"
	)
	ctx, db, stores, instance := newQueryMaskingFixture(t)
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ($1)`, other)
	require.NoError(t, err)

	createGroup := func(ws, id, groupEmail, member string) {
		t.Helper()
		_, err := stores.CreateGroup(ctx, &store.GroupMessage{
			ID: id, Workspace: ws, Email: groupEmail, Title: id,
			Payload: &storepb.GroupPayload{Members: []*storepb.GroupMember{{Role: storepb.GroupMember_MEMBER, Member: member}}},
		})
		require.NoError(t, err)
	}
	setRule := func(expression string) {
		t.Helper()
		putQueryMaskingRule(ctx, t, stores, expression, masker.DefaultSemanticTypeID)
	}
	column, spans := queryMaskingSpans()
	checkMasker := func(m *queryResultMasker, wantMasked bool) {
		t.Helper()
		predicates, err := m.extractSensitivePredicateColumns(ctx, spans, instance)
		require.NoError(t, err)
		result := queryMaskingResult(secret)
		require.NoError(t, m.maskResults(ctx, spans, []*v1pb.QueryResult{result}, instance))
		errorResults := []*v1pb.QueryResult{{Error: secret}}
		require.NoError(t, m.maskResults(ctx, spans, errorResults, instance))
		if wantMasked {
			require.Equal(t, [][]parserbase.ColumnResource{{column}}, predicates)
			require.Equal(t, "******", result.Rows[0].Values[0].GetStringValue())
			require.NotContains(t, errorResults[0].Error, secret)
		} else {
			require.Empty(t, predicates[0])
			require.Equal(t, secret, result.Rows[0].Values[0].GetStringValue())
			require.Equal(t, secret, errorResults[0].Error)
		}
	}
	check := func(principalType storepb.PrincipalType, wantMasked bool) *queryResultMasker {
		t.Helper()
		m, err := newQueryResultMasker(ctx, stores, instance, &store.UserMessage{Type: principalType, Email: email})
		require.NoError(t, err)
		checkMasker(m, wantMasked)
		return m
	}

	setRule(`principal.groups.exists(group, group in ["groups/analysts@example.com", "groups/automation", "groups/nested-audience"])`)
	createGroup(other, "other-analysts", "analysts@example.com", "users/"+email)
	createGroup(workspace, "nested", "", "users/"+email)
	createGroup(workspace, "nested-audience", "", "groups/nested")
	createGroup(workspace, "automation", "", "serviceAccounts/"+email)
	check(storepb.PrincipalType_END_USER, false)
	check(storepb.PrincipalType_SERVICE_ACCOUNT, true)
	check(storepb.PrincipalType_WORKLOAD_IDENTITY, false)
	createGroup(workspace, "workloads", "", "workloadIdentities/"+email)
	setRule(`"groups/workloads" in principal.groups`)
	check(storepb.PrincipalType_END_USER, false)
	check(storepb.PrincipalType_SERVICE_ACCOUNT, false)
	check(storepb.PrincipalType_WORKLOAD_IDENTITY, true)

	// CreateGroup does not invalidate this snapshot; a new query must nonetheless see the new membership.
	snapshot, err := stores.GetUserGroupsSnapshot(ctx, workspace, "users/"+email)
	require.NoError(t, err)
	require.Equal(t, []string{"groups/nested"}, snapshot)
	createGroup(workspace, "analysts", "analysts@example.com", "users/"+email)
	snapshot, err = stores.GetUserGroupsSnapshot(ctx, workspace, "users/"+email)
	require.NoError(t, err)
	require.Equal(t, []string{"groups/nested"}, snapshot)
	setRule(`"groups/analysts@example.com" in principal.groups`)
	m := check(storepb.PrincipalType_END_USER, true)

	_, err = db.ExecContext(ctx, `ALTER TABLE user_group RENAME TO unavailable_user_group`)
	require.NoError(t, err)
	checkMasker(m, true)
	principal := &store.UserMessage{Type: storepb.PrincipalType_END_USER, Email: email}
	for _, expression := range []string{
		`"groups/analysts@example.com" in principal.groups`,
		`!("groups/analysts@example.com" in principal.groups)`,
	} {
		setRule(expression)
		m, err := newQueryResultMasker(ctx, stores, instance, principal)
		require.ErrorContains(t, err, "failed to resolve groups for masking")
		require.Nil(t, m)
	}
	setRule(`resource.column_name == "principal.groups" || resource.column_name == "contents"`)
	check(storepb.PrincipalType_END_USER, true)
}

func TestQueryMaskingSemanticTypeSnapshot(t *testing.T) {
	t.Parallel()
	const secret = "example-value"
	const matchingColumn = `resource.column_name == "contents"`
	principal := &store.UserMessage{Type: storepb.PrincipalType_END_USER, Email: "caller@example.com"}
	newType := func(id, replacement string) *storepb.SemanticTypeSetting_SemanticType {
		return &storepb.SemanticTypeSetting_SemanticType{Id: id, Title: id, Algorithm: &storepb.Algorithm{
			Mask: &storepb.Algorithm_FullMask_{FullMask: &storepb.Algorithm_FullMask{Substitution: replacement}},
		}}
	}
	setTypes := func(t *testing.T, ctx context.Context, stores *store.Store, types ...*storepb.SemanticTypeSetting_SemanticType) {
		t.Helper()
		_, err := stores.UpsertSetting(ctx, &store.SettingMessage{
			Name: storepb.SettingName_SEMANTIC_TYPES, Workspace: queryMaskingWorkspace,
			Value: &storepb.SemanticTypeSetting{Types: types},
		})
		require.NoError(t, err)
	}
	assertMask := func(t *testing.T, ctx context.Context, m *queryResultMasker, instance *store.InstanceMessage, want string) {
		t.Helper()
		column, spans := queryMaskingSpans()
		sensitive, err := m.extractSensitivePredicateColumns(ctx, spans, instance)
		require.NoError(t, err)
		row := queryMaskingResult(secret)
		require.NoError(t, m.maskResults(ctx, spans, []*v1pb.QueryResult{row}, instance))
		dbError := &v1pb.QueryResult{Error: secret}
		require.NoError(t, m.maskResults(ctx, spans, []*v1pb.QueryResult{dbError}, instance))
		assert.Equal(t, want, row.Rows[0].Values[0].GetStringValue())
		assert.Equal(t, [][]parserbase.ColumnResource{{column}}, sensitive)
		assert.NotContains(t, dbError.Error, secret)
	}
	t.Run("rule changes while query is executing", func(t *testing.T) {
		t.Parallel()
		ctx, _, stores, instance := newQueryMaskingFixture(t)
		setTypes(t, ctx, stores, newType("mask-a", "mask from A"))
		putQueryMaskingRule(ctx, t, stores, matchingColumn, "mask-a")
		inFlight, err := newQueryResultMasker(ctx, stores, instance, principal)
		require.NoError(t, err)

		setTypes(t, ctx, stores, newType("mask-a", "mask from A"), newType("mask-b", "mask from B"))
		putQueryMaskingRule(ctx, t, stores, matchingColumn, "mask-b")
		setTypes(t, ctx, stores, newType("mask-b", "mask from B"))
		assertMask(t, ctx, inFlight, instance, "mask from A")
		fresh, err := newQueryResultMasker(ctx, stores, instance, principal)
		require.NoError(t, err)
		assertMask(t, ctx, fresh, instance, "mask from B")
	})
	t.Run("catalog references type added since query start", func(t *testing.T) {
		t.Parallel()
		ctx, _, stores, instance := newQueryMaskingFixture(t)
		inFlight, err := newQueryResultMasker(ctx, stores, instance, principal)
		require.NoError(t, err)

		setTypes(t, ctx, stores, newType("mask-b", "mask from B"))
		require.NoError(t, stores.UpdateDBSchema(ctx, "test", "db", &store.UpdateDBSchemaMessage{Config: &storepb.DatabaseConfig{
			Schemas: []*storepb.SchemaCatalog{{
				Name: "public",
				Tables: []*storepb.TableCatalog{{
					Name: "records", Columns: []*storepb.ColumnCatalog{{Name: "contents", SemanticType: "mask-b"}},
				}},
			}},
		}}))
		assertMask(t, ctx, inFlight, instance, "******")
		fresh, err := newQueryResultMasker(ctx, stores, instance, principal)
		require.NoError(t, err)
		assertMask(t, ctx, fresh, instance, "mask from B")
	})
}
