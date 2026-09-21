package reviewrun

import (
	"github.com/bytebase/omni/review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// reviewers maps an engine to its omni review entry point. An engine absent
// here has no standard rule review, and a run touching it fails. The map
// fills in as omni ships engine review packages (pg/review, mysql/review,
// ...): each one is a line here and nothing else.
var reviewers = map[storepb.Engine]review.ReviewFunc{}
