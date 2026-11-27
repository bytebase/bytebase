package snowflake

import (
	"github.com/bytebase/omni/snowflake/analysis"
	omniast "github.com/bytebase/omni/snowflake/ast"
	sfcompletion "github.com/bytebase/omni/snowflake/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, sfcompletion.Engine)
	base.Register(engine, storepb.Engine_SNOWFLAKE)
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (omniast.Node, bool) {
	node, ok := base.ExtractNode[omniast.Node](a)
	return node, ok && node != nil
}
