package v1

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestIsReadOnlyStatementForAccessGrantRejectsDocumentEngineWriteStatements(t *testing.T) {
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
			readOnly, err := isReadOnlyStatementForAccessGrant(context.Background(), tc.engine, tc.statement)
			require.NoError(t, err)
			require.False(t, readOnly)
		})
	}
}

func TestIsReadOnlyStatementForAccessGrantAllowsDocumentEngineReadStatements(t *testing.T) {
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
			readOnly, err := isReadOnlyStatementForAccessGrant(context.Background(), tc.engine, tc.statement)
			require.NoError(t, err)
			require.True(t, readOnly)
		})
	}
}

func TestIsReadOnlyStatementForAccessGrantRejectsDocumentEngineInvalidStatements(t *testing.T) {
	readOnly, err := isReadOnlyStatementForAccessGrant(context.Background(), storepb.Engine_ELASTICSEARCH, `db.users.find({})`)
	require.Error(t, err)
	require.False(t, readOnly)
}

// TestConvertToAccessGrantPropagatesPayloadFields pins the fix that
// `Reason` (alongside the existing Targets / Query / Unmask / Export)
// must be copied from the store payload onto the v1 message — otherwise
// frontend tooltips and audit displays that depend on the user-typed
// reason silently render empty.
func TestConvertToAccessGrantPropagatesPayloadFields(t *testing.T) {
	expire := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	msg := &store.AccessGrantMessage{
		ProjectID:  "proj",
		ID:         "ag1",
		Creator:    "dev@example.com",
		Status:     storepb.AccessGrant_ACTIVE,
		ExpireTime: &expire,
		Payload: &storepb.AccessGrantPayload{
			Targets: []string{"instances/inst/databases/db"},
			Query:   "SELECT * FROM t",
			Unmask:  true,
			Export:  true,
			Reason:  "investigating PR-1234",
		},
	}

	ag := convertToAccessGrant(msg)

	require.Equal(t, []string{"instances/inst/databases/db"}, ag.Targets)
	require.Equal(t, "SELECT * FROM t", ag.Query)
	require.True(t, ag.Unmask)
	require.True(t, ag.Export)
	require.Equal(t, "investigating PR-1234", ag.Reason)
}

// TestConvertToAccessGrantNilPayloadIsSafe guards the `if p := msg.Payload; p != nil`
// branch — a nil payload must not panic and payload-sourced fields stay zero.
func TestConvertToAccessGrantNilPayloadIsSafe(t *testing.T) {
	msg := &store.AccessGrantMessage{
		ProjectID: "proj",
		ID:        "ag1",
		Creator:   "dev@example.com",
		Status:    storepb.AccessGrant_ACTIVE,
		Payload:   nil,
	}

	ag := convertToAccessGrant(msg)

	require.Empty(t, ag.Targets)
	require.Empty(t, ag.Query)
	require.False(t, ag.Unmask)
	require.False(t, ag.Export)
	require.Empty(t, ag.Reason)
}
