package redis

import (
	"github.com/bytebase/omni/redis/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.Engine{ValidateQuery: analysis.ValidateQuery}, storepb.Engine_REDIS)
}
