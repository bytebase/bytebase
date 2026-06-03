package v1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// TestSelectBestAccessGrantPrefersUnmask covers the Unmask-only ranking:
// Export plays no role (Export callers already filtered the pool via the
// `requireExport` CEL filter, and Query never reads `Payload.Export`).
// Preferring Unmask=true is what addresses PR #20491 bot review
// (#3349086819) — a user with an active unmask grant should never be
// masked by Query when an export-only grant shares the slice.
func TestSelectBestAccessGrantPrefersUnmask(t *testing.T) {
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

	t.Run("unmask grant wins over export-only grant regardless of slice order", func(t *testing.T) {
		unmaskGrant := grantOf(true, false)
		exportOnly := grantOf(false, true)
		// Put exportOnly first to prove the win isn't due to position —
		// Unmask scoring is what selects unmaskGrant.
		got := selectBestAccessGrant([]*store.AccessGrantMessage{exportOnly, unmaskGrant})
		require.Same(t, unmaskGrant, got)
	})

	t.Run("among multiple unmask grants the first in slice order wins", func(t *testing.T) {
		first := grantOf(true, false)
		second := grantOf(true, true)
		// Both have Unmask=true → tied score (Export plays no role). First
		// in slice wins via strict ">". Equivalent for Query callers
		// since both grants yield the same SkipMasking=true.
		got := selectBestAccessGrant([]*store.AccessGrantMessage{first, second})
		require.Same(t, first, got)
	})

	t.Run("returns a no-unmask grant when none have unmask", func(t *testing.T) {
		// All-zero-score case: a no-unmask grant is still returned (it
		// confers ACL bypass for Query, even if it can't unmask).
		exportOnly := grantOf(false, true)
		got := selectBestAccessGrant([]*store.AccessGrantMessage{exportOnly})
		require.Same(t, exportOnly, got)
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
