package advisor

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCatalogService is the mock catalog service for test.
type MockCatalogService struct{}

const (
	// MockOldIndexName is the mock old index for test.
	MockOldIndexName = "old_index"
	// MockOldUKName is the mock old unique key for test.
	MockOldUKName = "old_uk"
	// MockOldPKName is the mock old foreign key for test.
	MockOldPKName = "PRIMARY"
)

var (
	// MockIndexColumnList is the mock index column list for test.
	MockIndexColumnList = []string{"id", "name"}
)

// FindIndex implements the catalog interface.
func (*MockCatalogService) FindIndex(ctx context.Context, find *catalog.IndexFind) (*catalog.Index, error) {
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
	case MockOldPKName:
		return &catalog.Index{
			Unique:            true,
			Name:              MockOldPKName,
			ColumnExpressions: MockIndexColumnList,
		}, nil
	}
	return nil, fmt.Errorf("cannot find index for %v", find)
}

// FindTable implements the catalog interface.
func (*MockCatalogService) FindTable(ctx context.Context, find *catalog.TableFind) ([]*catalog.Table, error) {
	return []*catalog.Table{
		{
			Name:         "table",
			DatabaseName: find.DatabaseName,
		},
	}, nil
}

// TestCase is the data struct for test.
type TestCase struct {
	Statement string
	Want      []Advice
}

// RunSchemaReviewRuleTests helps to test the schema review rule.
func RunSchemaReviewRuleTests(
	t *testing.T,
	tests []TestCase,
	adv Advisor,
	rule *SQLReviewRule,
	catalog catalog.Catalog,
) {
	ctx := Context{
		Charset:   "",
		Collation: "",
		Rule:      rule,
		Catalog:   catalog,
	}
	for _, tc := range tests {
		adviceList, err := adv.Check(ctx, tc.Statement)
		require.NoError(t, err)
		assert.Equal(t, tc.Want, adviceList, tc.Statement)
	}
}

// RandomString returns random string with specific length.
func RandomString(length int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")
	rand.Seed(time.Now().UnixNano())

	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
