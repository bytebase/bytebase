package mysql

import (
	"context"
	"fmt"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type MockCatalogService struct{}

const (
	MockOldIndexName = "old_index"
	MockOldUKName    = "old_uk"
)

var (
	MockIndexColumnList = []string{"id", "name"}
)

func (c *MockCatalogService) FindIndex(ctx context.Context, find *catalog.IndexFind) (*catalog.Index, error) {
	switch find.IndexName {
	case MockOldIndexName:
		return &catalog.Index{
			Name:              MockOldIndexName,
			ColumnExpressions: MockIndexColumnList,
		}, nil
	case MockOldUKName:
		return &catalog.Index{
			Unique:            true,
			Name:              MockOldIndexName,
			ColumnExpressions: MockIndexColumnList,
		}, nil
	}
	return nil, fmt.Errorf("cannot find index for %v", find)
}

func runSchemaReviewRuleTests(
	t *testing.T,
	tests []test,
	adv advisor.Advisor,
	rule *api.SchemaReviewRule,
	catalog catalog.Service,
) {
	logger, _ := zap.NewDevelopmentConfig().Build()
	ctx := advisor.Context{
		Logger:    logger,
		Charset:   "",
		Collation: "",
		Rule:      rule,
		Catalog:   catalog,
	}
	for _, tc := range tests {
		adviceList, err := adv.Check(ctx, tc.statement)
		require.NoError(t, err)
		assert.Equal(t, tc.want, adviceList)
	}
}
