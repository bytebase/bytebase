package spanner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/googlesql/googlesqltest"
)

// TestQuerySpanTypeForSelect guards BYT-9627: a SELECT must produce a query
// span with Type == base.Select. Spanner is in EngineSupportQueryNewACL, where
// the access check rejects a QueryTypeUnknown span as "disallowed query type",
// so a missing Type makes every Spanner SELECT fail in normal (non-admin) mode.
func TestQuerySpanTypeForSelect(t *testing.T) {
	getter, lister := googlesqltest.BuildMockDatabaseMetadataGetter(storepb.Engine_SPANNER, []*storepb.DatabaseSchemaMetadata{{Name: "db"}})
	result, err := GetQuerySpan(
		context.Background(),
		base.GetQuerySpanContext{
			GetDatabaseMetadataFunc: getter,
			ListDatabaseNamesFunc:   lister,
		},
		base.Statement{Text: "SELECT 1"},
		"db",
		"",
		false,
	)
	require.NoError(t, err)
	require.Equal(t, base.Select, result.Type,
		"a plain SELECT must have query span Type=Select; otherwise the new-ACL access check rejects it as \"disallowed query type\"")
}
