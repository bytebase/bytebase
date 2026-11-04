package gonative

import (
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
)

// Engine is the pure-Go implementation of queryengine.QueryEngine.
// It is constructed by Default(); callers should not instantiate Engine
// directly because future stages may add required initialization.
type Engine struct {
	// (future stages add fields: supportedFeatures set, cardinality caps, etc.)
}

// Compile-time check that *Engine satisfies the interface.
var _ queryengine.QueryEngine = (*Engine)(nil)

// Default returns a new Engine with Stage-0 capabilities (none).
// Callers should treat the returned value as opaque.
func Default() *Engine {
	return &Engine{}
}

// supportedFeatures enumerates the gateway-known feature names the engine
// can serve. The string is comma-separated per the gateway's query-plan
// protocol; new feature names are appended as each stage lands.
//
//	Aggregate              — VALUE-form aggregate queries (Stage 1)
//	NonValueAggregate      — aliased-form aggregates (Stage 1)
//	MultipleAggregates     — multi-aggregate in a single SELECT (Stage 1)
//	DistinctValue          — SELECT DISTINCT VALUE c.field FROM c (Stage 2)
//	Distinct               — SELECT DISTINCT c.field FROM c — object form (Stage 2)
//	OrderBy                — single-key ORDER BY (Stage 3)
//	Top                    — SELECT TOP N FROM c (Stage 4)
//	OffsetAndLimit         — SELECT … ORDER BY … OFFSET N LIMIT M (Stage 5)
//	GroupBy                — SELECT … FROM c GROUP BY c.field (Stage 6)
const supportedFeatures = "Aggregate,NonValueAggregate,MultipleAggregates,Distinct,DistinctValue,OrderBy,Top,OffsetAndLimit,GroupBy"

// SupportedFeatures implements queryengine.QueryEngine.
func (*Engine) SupportedFeatures() string {
	return supportedFeatures
}

// CreateQueryPipeline implements queryengine.QueryEngine. It parses the plan,
// rejects any feature the engine does not yet support, then dispatches to the
// appropriate pipeline implementation.
func (*Engine) CreateQueryPipeline(query string, plan string, pkranges string) (queryengine.QueryPipeline, error) {
	p, err := parsePlan([]byte(plan))
	if err != nil {
		return nil, err
	}
	if err := rejectUnsupportedPlan(p); err != nil {
		return nil, err
	}
	rangeIDs, err := parsePKRangeIDs([]byte(pkranges))
	if err != nil {
		return nil, err
	}
	switch {
	case len(p.QueryInfo.OrderBy) > 0:
		return newOrderByPipeline(p, rangeIDs)
	case p.QueryInfo.DistinctType == "Unordered":
		pipe, err := newDistinctPipeline(p, rangeIDs)
		if err != nil {
			return nil, err
		}
		pipe.query = query
		return pipe, nil
	case len(p.QueryInfo.Aggregates) > 0:
		pipe, err := newValuePipeline(p, rangeIDs)
		if err != nil {
			return nil, err
		}
		return pipe, nil
	case len(p.QueryInfo.GroupByExpressions) > 0:
		// GROUP BY plans carry groupByExpressions; the aliased-aggregate
		// (single-group) path used by Stage 1 has groupByAliasToAggregateType
		// populated but groupByExpressions empty. The check order matters.
		return newGroupPipeline(p, rangeIDs)
	case len(p.QueryInfo.GroupByAliasToAggregateType) > 0:
		pipe, err := newAliasedPipeline(p, rangeIDs)
		if err != nil {
			return nil, err
		}
		return pipe, nil
	case p.QueryInfo.Top != nil:
		pipe, err := newTopPipeline(p, rangeIDs)
		if err != nil {
			return nil, err
		}
		pipe.query = query
		return pipe, nil
	default:
		return nil, ErrUnsupportedPlanFeature
	}
}

// rejectUnsupportedPlan returns ErrUnsupportedPlanFeature when the plan uses
// any feature beyond the engine's current scope. Later stages will prune
// entries from this guard as their operators come online.
func rejectUnsupportedPlan(p *planDoc) error {
	qi := p.QueryInfo
	// "Ordered" DISTINCT composes with ORDER BY merge; reject until Stage 3
	// wires composition through the merge heap. Stage 3 itself handles plain
	// ORDER BY — see the dispatch above.
	if qi.DistinctType == "Ordered" {
		return ErrUnsupportedPlanFeature
	}
	if qi.DistinctType != "" && qi.DistinctType != "None" && qi.DistinctType != "Unordered" {
		return ErrUnsupportedPlanFeature
	}
	// OFFSET/LIMIT always composes with ORDER BY per Cosmos syntax; when that
	// composition lands the plan has both qi.OrderBy and qi.Offset/qi.Limit,
	// and the dispatch routes to newOrderByPipeline which handles them.
	// Plans that set offset/limit without an ORDER BY fall through to the
	// default case in CreateQueryPipeline and are rejected there.
	if qi.HasNonStreamingOrderBy {
		return ErrUnsupportedPlanFeature
	}
	return nil
}

// CreateReadManyPipeline implements queryengine.QueryEngine.
func (*Engine) CreateReadManyPipeline(_ []queryengine.ItemIdentity, _ string, _ string, _ uint8, _ []string) (queryengine.QueryPipeline, error) {
	return nil, ErrUnsupportedPlanFeature
}
