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
	omniSpan, err := es.GetQuerySpan(stmt.Text)
	if err != nil {
		return nil, err
	}
	if omniSpan == nil {
		return &base.QuerySpan{Type: base.QueryTypeUnknown}, nil
	}

	span := &base.QuerySpan{
		Type: convertQueryType(omniSpan.Type),
	}

	if omniSpan.ElasticsearchAnalysis != nil {
		span.ElasticsearchAnalysis = convertAnalysis(omniSpan.ElasticsearchAnalysis)
	}

	if omniSpan.PredicatePaths != nil {
		span.PredicatePaths = make(map[string]*base.PathAST, len(omniSpan.PredicatePaths))
		for key, p := range omniSpan.PredicatePaths {
			span.PredicatePaths[key] = convertPathAST(p)
		}
	}

	return span, nil
}

// convertPathAST converts an omni PathAST to a bytebase base.PathAST.
func convertPathAST(p *es.PathAST) *base.PathAST {
	if p == nil || p.Root == nil {
		return nil
	}
	root := base.NewItemSelector(p.Root.Name)
	current := base.SelectorNode(root)
	omniCurrent := p.Root.Next
	for omniCurrent != nil {
		next := base.NewItemSelector(omniCurrent.Name)
		current.SetNext(next)
		current = next
		omniCurrent = omniCurrent.Next
	}
	return base.NewPathAST(root)
}

// convertAnalysis converts an omni analysis.RequestAnalysis to a bytebase base.ElasticsearchAnalysis.
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
			result.BlockedFeatures[i] = convertBlockedFeature(bf)
		}
	}

	return result
}

// convertMaskableAPI maps an omni analysis.MaskableAPI to a bytebase base.ElasticsearchMaskableAPI.
func convertMaskableAPI(api analysis.MaskableAPI) base.ElasticsearchMaskableAPI {
	switch api {
	case analysis.APIMaskSearch:
		return base.ElasticsearchAPIMaskSearch
	case analysis.APIMaskGetDoc:
		return base.ElasticsearchAPIMaskGetDoc
	case analysis.APIMaskGetSource:
		return base.ElasticsearchAPIMaskGetSource
	case analysis.APIMaskMGet:
		return base.ElasticsearchAPIMaskMGet
	case analysis.APIMaskExplain:
		return base.ElasticsearchAPIMaskExplain
	case analysis.APIBlocked:
		return base.ElasticsearchAPIBlocked
	default:
		return base.ElasticsearchAPIUnsupported
	}
}

// convertBlockedFeature maps an omni analysis.BlockedFeature to a bytebase base.ElasticsearchBlockedFeature.
func convertBlockedFeature(bf analysis.BlockedFeature) base.ElasticsearchBlockedFeature {
	switch bf {
	case analysis.BlockedFeatureAggs:
		return base.ElasticsearchBlockedFeatureAggs
	case analysis.BlockedFeatureSuggest:
		return base.ElasticsearchBlockedFeatureSuggest
	case analysis.BlockedFeatureScriptFields:
		return base.ElasticsearchBlockedFeatureScriptFields
	case analysis.BlockedFeatureRuntimeMappings:
		return base.ElasticsearchBlockedFeatureRuntimeMappings
	case analysis.BlockedFeatureStoredFields:
		return base.ElasticsearchBlockedFeatureStoredFields
	case analysis.BlockedFeatureDocvalueFields:
		return base.ElasticsearchBlockedFeatureDocvalueFields
	default:
		return base.ElasticsearchBlockedFeatureAggs // fallback
	}
}
