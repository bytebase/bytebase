package gonative

import "errors"

// ErrUnsupportedPlanFeature reports that the engine cannot answer the query as
// asked. It comes back two ways, and the difference matters to the caller.
//
// From CreateQueryPipeline, before any row is read: the plan needs a feature
// the engine does not implement, and the driver can still surface the
// gateway's own error for the query.
//
// From a running pipeline, once the rows show what the plan did not: an ORDER
// BY over two values the service orders by its index's own encoding. Nothing
// is left to fall back to, so this one reaches the user as the engine's error
// — which is the point, since the alternative is a wrong order.
//
// Upstream's queryengine package defines no such sentinel, so it lives with
// the engine that returns it.
var ErrUnsupportedPlanFeature = errors.New("cosmosdb/queryengine: unsupported plan feature")
