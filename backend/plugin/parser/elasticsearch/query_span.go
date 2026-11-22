package elasticsearch

import (
	"context"

	es "github.com/bytebase/omni/elasticsearch"
	"github.com/bytebase/omni/elasticsearch/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_ELASTICSEARCH, GetQuerySpan)
}

// GetQuerySpan returns the query span for an ElasticSearch REST API request.
func GetQuerySpan(
	_ context.Context,
	_ base.GetQuerySpanContext,
	stmt base.Statement,
	_, _ string,
	_ bool,
) (*base.QuerySpan, error) {
	rawSpan, err := es.GetQuerySpan(stmt.Text)
	if err != nil {
		return nil, err
	}
	if rawSpan == nil {
		return &base.QuerySpan{Type: base.QueryTypeUnknown}, nil
	}

	span := &base.QuerySpan{
		Type: convertQueryType(rawSpan.Type),
	}

	if rawSpan.ElasticsearchAnalysis != nil {
		span.ElasticsearchAnalysis = convertAnalysis(rawSpan.ElasticsearchAnalysis)
	}

	if rawSpan.PredicatePaths != nil {
		span.PredicatePaths = make(map[string]*base.PathAST, len(rawSpan.PredicatePaths))
		for key, p := range rawSpan.PredicatePaths {
			span.PredicatePaths[key] = convertPathAST(p)
		}
	}

	return span, nil
}

func convertPathAST(p *es.PathAST) *base.PathAST {
	if p == nil || p.Root == nil {
		return nil
	}
	var nodes []base.SelectorNode
	for src := p.Root; src != nil; src = src.Next {
		nodes = append(nodes, base.NewItemSelector(src.Name))
	}
	return base.BuildPathAST(nodes)
}

func convertAnalysis(a *analysis.RequestAnalysis) *base.ElasticsearchAnalysis {
	if a == nil {
		return nil
	}
	result := &base.ElasticsearchAnalysis{
		API:             convertMaskableAPI(a.API),
		Index:           a.Index,
		SourceFields:    a.SourceFields,
		SourceDisabled:  a.SourceDisabled,
		RequestedFields: a.RequestedFields,
		HighlightFields: a.HighlightFields,
		SortFields:      a.SortFields,
		HasInnerHits:    a.HasInnerHits,
		PredicateFields: a.PredicateFields,
	}

	if len(a.BlockedFeatures) > 0 {
		result.BlockedFeatures = make([]base.ElasticsearchBlockedFeature, len(a.BlockedFeatures))
		for i, bf := range a.BlockedFeatures {
			result.BlockedFeatures[i] = blockedFeatureMap[bf]
		}
	}

	return result
}

var maskableAPIMap = map[analysis.MaskableAPI]base.ElasticsearchMaskableAPI{
	analysis.APIMaskSearch:    base.ElasticsearchAPIMaskSearch,
	analysis.APIMaskGetDoc:    base.ElasticsearchAPIMaskGetDoc,
	analysis.APIMaskGetSource: base.ElasticsearchAPIMaskGetSource,
	analysis.APIMaskMGet:      base.ElasticsearchAPIMaskMGet,
	analysis.APIMaskExplain:   base.ElasticsearchAPIMaskExplain,
	analysis.APIBlocked:       base.ElasticsearchAPIBlocked,
}

var blockedFeatureMap = map[analysis.BlockedFeature]base.ElasticsearchBlockedFeature{
	analysis.BlockedFeatureAggs:            base.ElasticsearchBlockedFeatureAggs,
	analysis.BlockedFeatureSuggest:         base.ElasticsearchBlockedFeatureSuggest,
	analysis.BlockedFeatureScriptFields:    base.ElasticsearchBlockedFeatureScriptFields,
	analysis.BlockedFeatureRuntimeMappings: base.ElasticsearchBlockedFeatureRuntimeMappings,
	analysis.BlockedFeatureStoredFields:    base.ElasticsearchBlockedFeatureStoredFields,
	analysis.BlockedFeatureDocvalueFields:  base.ElasticsearchBlockedFeatureDocvalueFields,
}

func convertMaskableAPI(api analysis.MaskableAPI) base.ElasticsearchMaskableAPI {
	return maskableAPIMap[api]
}
