package omnicatalog

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
)

// TestSnapshotObjectFindsSequences covers the report key the TiDB catalog uses
// for a degraded sequence, so a changed sequence is refused rather than diffed
// as a stand-in.
func TestSnapshotObjectFindsSequences(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:      "public",
		Sequences: []*metadatapb.SequenceMetadata{{Name: "s"}},
	}}}
	object := snapshotObject(meta, "sequence:s")
	require.NotNil(t, object)
	require.Equal(t, "s", object.(*metadatapb.SequenceMetadata).GetName())
}
