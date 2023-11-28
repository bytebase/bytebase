package model

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestBuildTablesMetadata(t *testing.T) {
	testCases := []struct {
		input       *storepb.TableMetadata
		wantNames   []string
		wantColumns []*storepb.ColumnMetadata
	}{
		{
			input: &storepb.TableMetadata{
				Name: "orders",
				Columns: []*storepb.ColumnMetadata{
					{
						Name: "id",
					},
				},
				Partitions: []*storepb.TablePartitionMetadata{
					{
						Name: "orders_0_100",
						Subpartitions: []*storepb.TablePartitionMetadata{
							{
								Name: "orders_0_50",
							},
							{
								Name: "orders_50_100",
							},
						},
					},
					{
						Name: "orders_100_200",
						Subpartitions: []*storepb.TablePartitionMetadata{
							{
								Name: "orders_100_150",
							},
							{
								Name: "orders_150_200",
							},
						},
					},
				},
			},
			wantNames: []string{"orders", "orders_0_100", "orders_0_50", "orders_50_100", "orders_100_200", "orders_100_150", "orders_150_200"},
			wantColumns: []*storepb.ColumnMetadata{
				{
					Name: "id",
				},
			},
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		tables, names := buildTablesMetadata(tc.input)

		// The length of the tables should be the same as the length of the names.
		a.Equal(len(tables), len(names))

		// The names should be the same as the expected names.
		a.Equal(sort.StringSlice(names), sort.StringSlice(tc.wantNames))

		// Each table should have the same columns as the input.
		for _, table := range tables {
			a.Equal(len(table.GetColumns()), len(tc.wantColumns))
			for _, column := range tc.wantColumns {
				a.NotNil(table.GetColumn(column.Name))
			}
		}
	}
}
