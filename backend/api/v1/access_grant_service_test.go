package v1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
