package schemasync

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestEqualDatabaseMetadataFast(t *testing.T) {
	tests := []struct {
		x    *storepb.DatabaseSchemaMetadata
		y    *storepb.DatabaseSchemaMetadata
		want bool
	}{
		{
			x:    &storepb.DatabaseSchemaMetadata{},
			y:    &storepb.DatabaseSchemaMetadata{},
			want: true,
		},
		{
			x:    nil,
			y:    &storepb.DatabaseSchemaMetadata{},
			want: true,
		},
		{
			x: &storepb.DatabaseSchemaMetadata{
				Name: "hello",
			},
			y: &storepb.DatabaseSchemaMetadata{
				Name: "world",
			},
			want: false,
		},
		{
			x: &storepb.DatabaseSchemaMetadata{
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
			y: &storepb.DatabaseSchemaMetadata{
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
			x: &storepb.DatabaseSchemaMetadata{
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
			y: &storepb.DatabaseSchemaMetadata{
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
			x: &storepb.DatabaseSchemaMetadata{
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
			y: &storepb.DatabaseSchemaMetadata{
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
		got := common.EqualDatabaseSchemaMetadataFast(test.x, test.y)
		assert.Equal(t, test.want, got)
	}
}
