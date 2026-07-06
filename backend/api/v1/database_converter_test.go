package v1

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func srid(v uint32) *uint32 { return &v }

// TestConvertColumnMetadataSRIDInvisibleRoundTrip closes gap (a) at the unit level: the
// v1 converters (convertStoreColumnMetadata store->v1, convertV1ColumnMetadata v1->store)
// must carry the spatial SRID (presence + value, including the valid explicit SRID 0) and
// the INVISIBLE flag in BOTH directions. Before the fix these converters silently dropped
// the fields, so any metadata crossing the v1 API boundary (Schema Editor, DiffMetadata)
// lost them.
func TestConvertColumnMetadataSRIDInvisibleRoundTrip(t *testing.T) {
	testCases := []struct {
		name string
		in   *storepb.ColumnMetadata
	}{
		{
			name: "spatial_srid_4326",
			in:   &storepb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false, Srid: srid(4326)},
		},
		{
			// Explicit SRID 0 is a valid spatial reference system, distinct from "no SRID".
			// Presence (not a zero sentinel) must survive the round-trip.
			name: "explicit_srid_zero",
			in:   &storepb.ColumnMetadata{Name: "g", Type: "geometry", Nullable: false, Srid: srid(0)},
		},
		{
			// Custom SRSs may exceed int32 range; the value must not be squeezed.
			name: "srid_above_int32",
			in:   &storepb.ColumnMetadata{Name: "g", Type: "geometry", Nullable: false, Srid: srid(3000000000)},
		},
		{
			name: "invisible_column",
			in:   &storepb.ColumnMetadata{Name: "secret", Type: "int", Nullable: false, IsInvisible: true},
		},
		{
			// No SRID declared: presence must stay unset (nil), not collapse to SRID 0.
			name: "no_srid_visible",
			in:   &storepb.ColumnMetadata{Name: "c", Type: "int", Nullable: true},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v1Col := convertStoreColumnMetadata(tc.in)
			// store -> v1 preserves the fields.
			if tc.in.Srid == nil {
				require.Nil(t, v1Col.Srid, "no-SRID column must stay unset after store->v1")
			} else {
				require.NotNil(t, v1Col.Srid)
				require.Equal(t, *tc.in.Srid, *v1Col.Srid)
			}
			require.Equal(t, tc.in.IsInvisible, v1Col.IsInvisible)

			// v1 -> store round-trips back to the original.
			out := convertV1ColumnMetadata(v1Col)
			if tc.in.Srid == nil {
				require.Nil(t, out.Srid, "no-SRID column must stay unset after v1->store")
			} else {
				require.NotNil(t, out.Srid)
				require.Equal(t, *tc.in.Srid, *out.Srid)
			}
			require.Equal(t, tc.in.IsInvisible, out.IsInvisible)
		})
	}
}

// TestDiffMetadataPreservesSRIDInvisible exercises the exact conversion + diff pipeline the
// DiffMetadata gRPC handler runs (convertV1DatabaseMetadata -> DiffMigration). A geometry
// column carrying SRID 4326 and an INVISIBLE column are unchanged between source and
// target while an UNRELATED attribute (the table comment on another column) is edited: the
// SRID/INVISIBLE fields must not be stripped by the v1->store conversion, so the diff shows
// only the intended change and never a spurious spatial/visibility MODIFY.
func TestDiffMetadataPreservesSRIDInvisible(t *testing.T) {
	mkV1 := func(noteComment string) *v1pb.DatabaseMetadata {
		return &v1pb.DatabaseMetadata{
			Schemas: []*v1pb.SchemaMetadata{{
				Name: "",
				Tables: []*v1pb.TableMetadata{{
					Name: "geo",
					Columns: []*v1pb.ColumnMetadata{
						{Name: "id", Type: "int", Nullable: false},
						{Name: "pt", Type: "point", Nullable: false, Srid: srid(4326)},
						{Name: "secret", Type: "int", Nullable: false, IsInvisible: true},
						{Name: "note", Type: "int", Nullable: true, Comment: noteComment},
					},
				}},
			}},
		}
	}

	// The v1->store conversion the handler performs must retain SRID + INVISIBLE.
	storeSource := convertV1DatabaseMetadata(mkV1("old"))
	cols := storeSource.Schemas[0].Tables[0].Columns
	require.NotNil(t, cols[1].Srid)
	require.Equal(t, uint32(4326), *cols[1].Srid)
	require.True(t, cols[2].IsInvisible)

	storeTarget := convertV1DatabaseMetadata(mkV1("new"))

	source := model.NewDatabaseMetadata(storeSource, nil, nil, storepb.Engine_MYSQL, true)
	target := model.NewDatabaseMetadata(storeTarget, nil, nil, storepb.Engine_MYSQL, true)
	diff, err := schema.DiffMigration(storepb.Engine_MYSQL, source, target)
	require.NoError(t, err)

	// The only change is the unrelated comment edit; SRID/INVISIBLE stay put, so the diff
	// must not touch the spatial SRID or column visibility.
	require.NotContains(t, diff, "SRID", "unrelated edit must not produce a spatial SRID change")
	require.NotContains(t, diff, "INVISIBLE", "unrelated edit must not produce a visibility change")

	// Sanity: an identical source/target yields no diff at all — the SRID/INVISIBLE columns
	// are stable under the v1<->store conversion (no phantom churn).
	same := model.NewDatabaseMetadata(convertV1DatabaseMetadata(mkV1("same")), nil, nil, storepb.Engine_MYSQL, true)
	sameDiff, err := schema.DiffMigration(storepb.Engine_MYSQL, same,
		model.NewDatabaseMetadata(convertV1DatabaseMetadata(mkV1("same")), nil, nil, storepb.Engine_MYSQL, true))
	require.NoError(t, err)
	require.Empty(t, strings.TrimSpace(sameDiff), "identical metadata must self-diff empty")
}
