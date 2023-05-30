package schemasync

import (
	"testing"

	"github.com/stretchr/testify/assert"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestEqualDatabaseMetadata(t *testing.T) {
	tests := []struct {
		x    *storepb.DatabaseMetadata
		y    *storepb.DatabaseMetadata
		want bool
	}{
		{
			x:    &storepb.DatabaseMetadata{},
			y:    &storepb.DatabaseMetadata{},
			want: true,
		},
		{
			x:    nil,
			y:    &storepb.DatabaseMetadata{},
			want: false,
		},
		{
			x: &storepb.DatabaseMetadata{
				Name: "hello",
			},
			y: &storepb.DatabaseMetadata{
				Name: "world",
			},
			want: false,
		},
		{
			x: &storepb.DatabaseMetadata{
				Name: "hello",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "students",
							},
						},
					},
				},
			},
			y: &storepb.DatabaseMetadata{
				Name: "hello",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "students",
							},
						},
					},
				},
			},
			want: true,
		},
		{
			x: &storepb.DatabaseMetadata{
				Name: "hello",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "teachers",
							},
						},
					},
				},
			},
			y: &storepb.DatabaseMetadata{
				Name: "hello",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "students",
							},
						},
					},
				},
			},
			want: false,
		},
		{
			x: &storepb.DatabaseMetadata{
				Name: "hello",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name:      "students",
								RowCount:  321,
								DataSize:  321,
								IndexSize: 321,
								DataFree:  321,
							},
						},
					},
				},
			},
			y: &storepb.DatabaseMetadata{
				Name: "hello",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name:      "students",
								RowCount:  123,
								DataSize:  123,
								IndexSize: 123,
								DataFree:  123,
							},
						},
					},
				},
			},
			want: true,
		},
	}
	for _, test := range tests {
		got := equalDatabaseMetadata(test.x, test.y)
		assert.Equal(t, test.want, got)
	}
}
