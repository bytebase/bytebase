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
		Sequences: []*metadatapb.SequenceMetadata{{Name: "s", Start: "1", LastValue: "42"}},
	}}}
	object := snapshotObject(meta, "sequence:s")
	sequence, ok := object.(*metadatapb.SequenceMetadata)
	require.True(t, ok)
	require.Equal(t, "s", sequence.GetName())
	require.Equal(t, "1", sequence.GetStart())
	require.Empty(t, sequence.GetLastValue())
}
