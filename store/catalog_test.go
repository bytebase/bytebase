package store

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	"github.com/stretchr/testify/assert"
)

func TestConvertIndexList(t *testing.T) {
	type testData struct {
		origin []*api.Index
		want   []*catalog.Index
	}

	tests := []testData{
		{
			origin: []*api.Index{
				{
					Name:       "idx_1",
					Position:   1,
					Expression: "id",
				},
				{
					Name:       "idx_1",
					Position:   2,
					Expression: "name",
				},
				{
					Name:       "idx_1",
					Position:   3,
					Expression: "user",
				},
				{
					Name:       "idx_2",
					Position:   1,
					Expression: "name",
				},
			},
			want: []*catalog.Index{
				{
					Name:           "idx_1",
					ExpressionList: []string{"id", "name", "user"},
				},
				{
					Name:           "idx_2",
					ExpressionList: []string{"name"},
				},
			},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.want, convertIndexList(test.origin))
	}
}
