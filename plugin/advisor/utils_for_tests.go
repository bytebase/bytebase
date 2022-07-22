package advisor

import (
	"math/rand"
	"testing"
	"time"

	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	// MockMySQLDatabase is the mock MySQL database for test.
	MockMySQLDatabase = &catalog.Database{
		Name: "test",
		SchemaList: []*catalog.Schema{
			{
				TableList: []*catalog.Table{
					{
						Name: "table",
						ColumnList: []*catalog.Column{
							{Name: "id"},
							{Name: "name"},
						},
						IndexList: []*catalog.Index{
							{
								Name:    "PRIMARY",
								Unique:  true,
								Primary: true,
							},
							{
								Name:   "old_uk",
								Unique: true,
							},
							{
								Name: "old_index",
							},
						},
					},
				},
			},
		},
	}
	// MockPostgreSQLDatabase is the mock PostgreSQL database for test.
	MockPostgreSQLDatabase = &catalog.Database{
		Name: "test",
		SchemaList: []*catalog.Schema{
			{
				Name: "public",
				TableList: []*catalog.Table{
					{
						Name: "table",
						ColumnList: []*catalog.Column{
							{Name: "id"},
							{Name: "name"},
						},
						IndexList: []*catalog.Index{
							{
								Name:    "old_pk",
								Unique:  true,
								Primary: true,
							},
							{
								Name:   "old_uk",
								Unique: true,
							},
							{
								Name: "old_index",
							},
						},
					},
				},
			},
		},
	}
)

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
	database *catalog.Database,
) {
	ctx := Context{
		Charset:   "",
		Collation: "",
		Rule:      rule,
		Database:  database,
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
