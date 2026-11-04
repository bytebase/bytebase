package gonative

import "errors"

// ErrUnsupportedPlanFeature reports that the gateway's query plan needs a
// feature this engine does not implement, so the caller should surface the
// gateway's own error rather than the engine's. Upstream's queryengine package
// defines no such sentinel, so it lives with the engine that returns it.
var ErrUnsupportedPlanFeature = errors.New("cosmosdb/queryengine: unsupported plan feature")
