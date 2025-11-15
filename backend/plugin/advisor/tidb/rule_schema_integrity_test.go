package tidb

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestSchemaIntegrity(t *testing.T) {
	advisor.RunSQLReviewRuleTest(t, advisor.SchemaRuleSchemaIntegrity, storepb.Engine_TIDB, true, false /* record */)
}
