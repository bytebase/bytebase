package v1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// TestSelectBestAccessGrantPicksHighestCapability covers the operation-
// agnostic ranking introduced when the accessOperation enum was removed —
// callers (Query, Export) gate on the capability they care about, so
// preCheckAccess just returns the most capable matching grant.
func TestSelectBestAccessGrantPicksHighestCapability(t *testing.T) {
	grantOf := func(unmask, export bool) *store.AccessGrantMessage {
		return &store.AccessGrantMessage{
			Payload: &storepb.AccessGrantPayload{Unmask: unmask, Export: export},
		}
	}

	t.Run("returns nil for empty input", func(t *testing.T) {
		require.Nil(t, selectBestAccessGrant(nil))
		require.Nil(t, selectBestAccessGrant([]*store.AccessGrantMessage{}))
	})

	t.Run("skips nil-payload grants and returns nil when none remain", func(t *testing.T) {
		got := selectBestAccessGrant([]*store.AccessGrantMessage{
			{Payload: nil},
			{Payload: nil},
		})
		require.Nil(t, got)
	})

	t.Run("returns the only valid grant when others have nil payloads", func(t *testing.T) {
		valid := grantOf(true, false)
		got := selectBestAccessGrant([]*store.AccessGrantMessage{
			{Payload: nil},
			valid,
			{Payload: nil},
		})
		require.Same(t, valid, got)
	})

	t.Run("dual-capability grant outranks single-capability grants", func(t *testing.T) {
		dual := grantOf(true, true)
		got := selectBestAccessGrant([]*store.AccessGrantMessage{
			grantOf(true, false),
			grantOf(false, true),
			dual,
		})
		require.Same(t, dual, got)
	})

	t.Run("single-capability tie resolves to slice order", func(t *testing.T) {
		// Both A{unmask:true} and B{export:true} score 1. With strict ">",
		// the first one in the slice wins. This is the documented
		// behavior — callers gate on the specific capability they need.
		a := grantOf(true, false)
		b := grantOf(false, true)
		got := selectBestAccessGrant([]*store.AccessGrantMessage{a, b})
		require.Same(t, a, got, "first single-capability grant in slice order should win on tie")
	})

	t.Run("returns a no-capability grant when nothing better matches", func(t *testing.T) {
		// A grant with neither Unmask nor Export scores 0 but is still
		// returned (it confers ACL bypass for Query).
		zero := grantOf(false, false)
		got := selectBestAccessGrant([]*store.AccessGrantMessage{zero})
		require.Same(t, zero, got)
	})
}

// TestBuildExportQueryContextPropagatesSkipMasking pins the bug fix from PR
// #20487: when an export is authorized by a JIT grant with unmask=true,
// SkipMasking must reach db.QueryContext so the driver doesn't mask rows at
// query time. The earlier code only consulted skipMasking around the
// post-execution MaskResults pass — by then drivers like postgres with
// query-time masking rewrites had already returned masked rows.
func TestBuildExportQueryContextPropagatesSkipMasking(t *testing.T) {
	restriction := &store.EffectiveQueryDataPolicy{
		MaximumResultRows: 1000,
		MaximumResultSize: 1 << 20,
	}

	t.Run("grant with unmask=true sets SkipMasking", func(t *testing.T) {
		qc := buildExportQueryContext(restriction, "alice@example.com", nil, true)
		require.True(t, qc.SkipMasking, "SkipMasking must propagate so driver-level masking is bypassed")
	})

	t.Run("grant with unmask=false keeps SkipMasking false", func(t *testing.T) {
		qc := buildExportQueryContext(restriction, "alice@example.com", nil, false)
		require.False(t, qc.SkipMasking, "SkipMasking must stay false so masking still applies")
	})
}

// TestBuildExportQueryContextPropagatesOtherFields guards against accidental
// drops of the surrounding fields if buildExportQueryContext is edited.
func TestBuildExportQueryContextPropagatesOtherFields(t *testing.T) {
	schema := "public"
	restriction := &store.EffectiveQueryDataPolicy{
		MaximumResultRows:        500,
		MaximumResultSize:        2 << 20,
		MaxQueryTimeoutInSeconds: 30,
	}

	qc := buildExportQueryContext(restriction, "alice@example.com", &schema, false)

	require.Equal(t, 500, qc.Limit)
	require.Equal(t, "alice@example.com", qc.OperatorEmail)
	require.Equal(t, int64(2<<20), qc.MaximumSQLResultSize)
	require.Equal(t, "public", qc.Schema)
	require.NotNil(t, qc.Timeout)
	require.Equal(t, int64(30), qc.Timeout.Seconds)
}

// TestBuildExportQueryContextOmitsTimeoutWhenZero verifies that an unset
// MaxQueryTimeoutInSeconds doesn't leak a zero-Duration timeout into the
// query context (which the driver layer treats differently from "no timeout").
func TestBuildExportQueryContextOmitsTimeoutWhenZero(t *testing.T) {
	restriction := &store.EffectiveQueryDataPolicy{
		MaximumResultRows: 100,
		MaximumResultSize: 1 << 20,
	}

	qc := buildExportQueryContext(restriction, "alice@example.com", nil, false)
	require.Nil(t, qc.Timeout)
	require.Equal(t, "", qc.Schema)
}

func TestResolveDataSourceIDUsesAdminForNonReadOnlyAutomaticQueryWhenAllowed(t *testing.T) {
	instance := &store.InstanceMessage{
		Metadata: &storepb.Instance{
			Engine: storepb.Engine_MYSQL,
			DataSources: []*storepb.DataSource{
				{Id: "admin", Type: storepb.DataSourceType_ADMIN},
				{Id: "readonly", Type: storepb.DataSourceType_READ_ONLY},
			},
		},
	}

	got, err := resolveDataSourceID(context.Background(), instance, "", "INSERT INTO books VALUES (1, 'Bytebase');", true)
	require.NoError(t, err)
	require.Equal(t, "admin", got)
}

func TestResolveDataSourceIDKeepsReadOnlyForReadOnlyAutomaticQueryWhenAllowed(t *testing.T) {
	instance := &store.InstanceMessage{
		Metadata: &storepb.Instance{
			Engine: storepb.Engine_MYSQL,
			DataSources: []*storepb.DataSource{
				{Id: "admin", Type: storepb.DataSourceType_ADMIN},
				{Id: "readonly", Type: storepb.DataSourceType_READ_ONLY},
			},
		},
	}

	got, err := resolveDataSourceID(context.Background(), instance, "", "SELECT * FROM books;", true)
	require.NoError(t, err)
	require.Equal(t, "readonly", got)
}

func TestResolveDataSourceIDKeepsReadOnlyForNonReadOnlyAutomaticQueryWhenAdminDisallowed(t *testing.T) {
	instance := &store.InstanceMessage{
		Metadata: &storepb.Instance{
			Engine: storepb.Engine_MYSQL,
			DataSources: []*storepb.DataSource{
				{Id: "admin", Type: storepb.DataSourceType_ADMIN},
				{Id: "readonly", Type: storepb.DataSourceType_READ_ONLY},
			},
		},
	}

	got, err := resolveDataSourceID(context.Background(), instance, "", "INSERT INTO books VALUES (1, 'Bytebase');", false)
	require.NoError(t, err)
	require.Equal(t, "readonly", got)
}

func TestResolveDataSourceIDUsesAdminForDocumentEngineAutomaticWriteQueryWhenAllowed(t *testing.T) {
	tests := []struct {
		name      string
		engine    storepb.Engine
		statement string
	}{
		{
			name:      "MongoDB DML",
			engine:    storepb.Engine_MONGODB,
			statement: `db.users.insertOne({name: "Bytebase"})`,
		},
		{
			name:      "MongoDB DDL",
			engine:    storepb.Engine_MONGODB,
			statement: `db.createCollection("users")`,
		},
		{
			name:      "Elasticsearch DML",
			engine:    storepb.Engine_ELASTICSEARCH,
			statement: "POST /users/_doc\n{\"name\":\"Bytebase\"}",
		},
		{
			name:      "Elasticsearch DDL",
			engine:    storepb.Engine_ELASTICSEARCH,
			statement: "PUT /users",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			instance := &store.InstanceMessage{
				Metadata: &storepb.Instance{
					Engine: tc.engine,
					DataSources: []*storepb.DataSource{
						{Id: "admin", Type: storepb.DataSourceType_ADMIN},
						{Id: "readonly", Type: storepb.DataSourceType_READ_ONLY},
					},
				},
			}

			got, err := resolveDataSourceID(context.Background(), instance, "", tc.statement, true)
			require.NoError(t, err)
			require.Equal(t, "admin", got)
		})
	}
}

func TestResolveDataSourceIDKeepsReadOnlyForDocumentEngineAutomaticReadQueryWhenAllowed(t *testing.T) {
	tests := []struct {
		name      string
		engine    storepb.Engine
		statement string
	}{
		{
			name:      "MongoDB read",
			engine:    storepb.Engine_MONGODB,
			statement: `db.users.find({})`,
		},
		{
			name:      "Elasticsearch read",
			engine:    storepb.Engine_ELASTICSEARCH,
			statement: "GET /users/_search",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			instance := &store.InstanceMessage{
				Metadata: &storepb.Instance{
					Engine: tc.engine,
					DataSources: []*storepb.DataSource{
						{Id: "admin", Type: storepb.DataSourceType_ADMIN},
						{Id: "readonly", Type: storepb.DataSourceType_READ_ONLY},
					},
				},
			}

			got, err := resolveDataSourceID(context.Background(), instance, "", tc.statement, true)
			require.NoError(t, err)
			require.Equal(t, "readonly", got)
		})
	}
}
