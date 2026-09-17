package pg

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/standard"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_POSTGRES, validateQueryANTLR)
	// Redshift has its own implementation in the redshift package
	base.RegisterQueryValidator(storepb.Engine_COCKROACHDB, validateQueryANTLR)
	base.RegisterExplainFunc(storepb.Engine_POSTGRES, ExplainStatement)
	base.RegisterExplainFunc(storepb.Engine_COCKROACHDB, standard.ExplainStatement)
}
