package gonative

import (
	"reflect"
	"slices"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/pkg/errors"
)

// Engine is the pure-Go implementation of queryengine.QueryEngine. Construct
// it with Default() and treat the value as opaque.
type Engine struct{}

// Compile-time check that *Engine satisfies the interface.
var _ queryengine.QueryEngine = (*Engine)(nil)

// Default returns a new Engine.
func Default() *Engine {
	return &Engine{}
}

// supportedFeatures enumerates the gateway-known feature names the engine
// can serve. The string is comma-separated per the gateway's query-plan
// protocol.
//
//	Aggregate              — VALUE-form aggregate queries
//	NonValueAggregate      — aliased-form aggregates
//	MultipleAggregates     — multi-aggregate in a single SELECT
//	DistinctValue          — SELECT DISTINCT VALUE c.field FROM c
//	Distinct               — SELECT DISTINCT c.field FROM c — object form
//	OrderBy                — single-key ORDER BY
//	Top                    — SELECT TOP N FROM c
//	OffsetAndLimit         — OFFSET N LIMIT M
//	GroupBy                — SELECT … FROM c GROUP BY c.field
const supportedFeatures = "Aggregate,NonValueAggregate,MultipleAggregates,Distinct,DistinctValue,OrderBy,Top,OffsetAndLimit,GroupBy"

// SupportedFeatures implements queryengine.QueryEngine.
func (*Engine) SupportedFeatures() string {
	return supportedFeatures
}

// Names of the planQueryInfo fields a dispatch branch claims. A plan field the
// chosen pipeline does not claim fails the query — see requireHandled.
const (
	fieldRewrittenQuery              = "RewrittenQuery"
	fieldHasSelectValue              = "HasSelectValue"
	fieldAggregates                  = "Aggregates"
	fieldGroupByAliases              = "GroupByAliases"
	fieldGroupByAliasToAggregateType = "GroupByAliasToAggregateType"
	fieldDistinctType                = "DistinctType"
	fieldTop                         = "Top"
	fieldOffset                      = "Offset"
	fieldLimit                       = "Limit"
	fieldOrderBy                     = "OrderBy"
	fieldOrderByExpressions          = "OrderByExpressions"
	fieldGroupByExpressions          = "GroupByExpressions"
)

// CreateQueryPipeline implements queryengine.QueryEngine. It parses the plan,
// picks the pipeline for the plan's shape and rejects any plan field that
// pipeline cannot honour.
func (*Engine) CreateQueryPipeline(query string, plan string, pkranges string) (queryengine.QueryPipeline, error) {
	p, err := parsePlan([]byte(plan))
	if err != nil {
		return nil, err
	}
	rangeIDs, err := parsePKRangeIDs([]byte(pkranges))
	if err != nil {
		return nil, err
	}
	qi := p.QueryInfo
	switch {
	case len(qi.OrderBy) > 0:
		// ORDER BY owns every operator that applies to an ordered stream:
		// DISTINCT dedups it, TOP and OFFSET/LIMIT cut it.
		if err := requireHandled(qi, fieldRewrittenQuery, fieldHasSelectValue,
			fieldOrderBy, fieldOrderByExpressions,
			fieldDistinctType, fieldTop, fieldOffset, fieldLimit); err != nil {
			return nil, err
		}
		return newOrderByPipeline(p, rangeIDs)
	case qi.DistinctType != "":
		if err := requireHandled(qi, fieldRewrittenQuery, fieldHasSelectValue,
			fieldDistinctType, fieldTop, fieldOffset, fieldLimit); err != nil {
			return nil, err
		}
		return newDistinctPipeline(p, rangeIDs, query)
	case len(qi.Aggregates) > 0:
		if err := requireHandled(qi, fieldRewrittenQuery, fieldHasSelectValue,
			fieldAggregates, fieldOffset, fieldLimit); err != nil {
			return nil, err
		}
		return newValuePipeline(p, rangeIDs)
	case len(qi.GroupByExpressions) > 0:
		// GROUP BY plans carry groupByExpressions; the aliased-aggregate
		// (single-group) path has groupByAliasToAggregateType populated but
		// groupByExpressions empty. The check order matters.
		if err := requireHandled(qi, fieldRewrittenQuery, fieldHasSelectValue,
			fieldGroupByExpressions, fieldGroupByAliases, fieldGroupByAliasToAggregateType); err != nil {
			return nil, err
		}
		return newGroupPipeline(p, rangeIDs)
	case len(qi.GroupByAliasToAggregateType) > 0:
		if err := requireHandled(qi, fieldRewrittenQuery, fieldHasSelectValue,
			fieldGroupByAliases, fieldGroupByAliasToAggregateType, fieldOffset, fieldLimit); err != nil {
			return nil, err
		}
		return newAliasedPipeline(p, rangeIDs)
	case qi.Top != nil || qi.Offset != nil || qi.Limit != nil:
		if err := requireHandled(qi, fieldRewrittenQuery, fieldHasSelectValue,
			fieldTop, fieldOffset, fieldLimit); err != nil {
			return nil, err
		}
		return newSkipTakePipeline(p, rangeIDs, query)
	default:
		return nil, errors.Wrap(ErrUnsupportedPlanFeature, "no pipeline for this plan shape")
	}
}

// requireHandled rejects a plan whose queryInfo sets a field the chosen
// pipeline does not claim. It walks planQueryInfo by reflection rather than
// checking a fixed list, so a field added to the struct later is rejected
// until a pipeline claims it: an operator the pipeline cannot apply must fail
// the query rather than vanish from the answer.
func requireHandled(qi planQueryInfo, handled ...string) error {
	v := reflect.ValueOf(qi)
	t := v.Type()
	for i := range t.NumField() {
		name := t.Field(i).Name
		if !planFieldSet(v.Field(i)) || slices.Contains(handled, name) {
			continue
		}
		return errors.Wrapf(ErrUnsupportedPlanFeature, "pipeline cannot honour queryInfo.%s", name)
	}
	return nil
}

// planFieldSet reports whether a queryInfo field carries an instruction. A
// plan that spells a field out as empty — `"aggregates": []` — says nothing,
// and reflect.Value.IsZero would call that slice set because it is non-nil.
// A pointer field is set whenever it is non-nil: `"top": 0` means TOP 0.
func planFieldSet(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map, reflect.String:
		return v.Len() > 0
	case reflect.Pointer:
		return !v.IsNil()
	default:
		return !v.IsZero()
	}
}

// CreateReadManyPipeline implements queryengine.QueryEngine.
func (*Engine) CreateReadManyPipeline(_ []queryengine.ItemIdentity, _ string, _ string, _ uint8, _ []string) (queryengine.QueryPipeline, error) {
	return nil, ErrUnsupportedPlanFeature
}
