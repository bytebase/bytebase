package advisor

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// BuiltinWalkThroughCheckAdvisor is a no-op advisor for BUILTIN_WALK_THROUGH_CHECK.
// The actual walk-through logic runs in the dedicated block in SQLReviewCheck
// before the rule loop. This advisor exists only so the rule can be registered
// in the advisor registry without causing "unknown advisor" errors.
type BuiltinWalkThroughCheckAdvisor struct{}

func (*BuiltinWalkThroughCheckAdvisor) Check(_ context.Context, _ Context) ([]*storepb.Advice, error) {
	return nil, nil
}

func init() {
	for _, engine := range []storepb.Engine{
		storepb.Engine_MYSQL,
		storepb.Engine_MARIADB,
		storepb.Engine_TIDB,
		storepb.Engine_POSTGRES,
		storepb.Engine_OCEANBASE,
	} {
		Register(engine, storepb.SQLReviewRule_BUILTIN_WALK_THROUGH_CHECK, &BuiltinWalkThroughCheckAdvisor{})
	}
}
