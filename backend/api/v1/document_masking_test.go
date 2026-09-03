package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestMaskersAreDisjoint holds the two masking registries apart across every
// engine the build knows.
//
// The query path checks the document masker first and reaches the column masker
// only in the else arm of queryRetry's masking block (sql_service.go), so an
// engine in both sets would take the document path and never be column-masked
// there. Nothing else asserts this; the branch order alone decides it.
func TestMaskersAreDisjoint(t *testing.T) {
	values := storepb.Engine(0).Descriptor().Values()
	document := 0
	for i := range values.Len() {
		engine := storepb.Engine(values.Get(i).Number())
		if getDocumentMasker(engine) == nil {
			continue
		}
		document++
		require.False(t, common.EngineSupportMasking(engine),
			"%v routes to the document masker, so the column masker is never reached", engine)
	}
	require.Equal(t, 3, document, "cosmosdb, mongodb, elasticsearch")
}
