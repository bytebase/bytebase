package elasticsearch

import (
	"context"
	"strings"

	lsp "github.com/bytebase/lsp-protocol"
	es "github.com/bytebase/omni/elasticsearch"
	"github.com/bytebase/omni/elasticsearch/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.Engine{
		Split:           SplitMultiSQL,
		StatementRanges: GetStatementRanges,
		Diagnose:        Diagnose,
		QuerySpan:       GetQuerySpan,
	}, storepb.Engine_ELASTICSEARCH)
}

// SplitMultiSQL splits the input into individual ElasticSearch REST API requests.
func SplitMultiSQL(statement string) ([]base.Statement, error) {
	stmts, err := es.SplitMultiSQL(statement)
	if err != nil {
		return nil, err
	}
	if len(stmts) == 0 {
		if len(strings.TrimSpace(statement)) == 0 {
			return nil, nil
		}
		return []base.Statement{{Text: statement}}, nil
	}

	var statements []base.Statement
	for _, s := range stmts {
		statements = append(statements, base.Statement{
			Text:  s.Text,
			Empty: s.Empty,
			Start: &storepb.Position{
				Line:   int32(s.Start.Line),
				Column: int32(s.Start.Column),
			},
			End: &storepb.Position{
				Line:   int32(s.End.Line),
				Column: int32(s.End.Column),
			},
			Range: &storepb.Range{
				Start: int32(s.Range.Start),
				End:   int32(s.Range.End),
			},
		})
	}
	return statements, nil
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	rawRanges, err := es.GetStatementRanges(statement)
	if err != nil {
		return nil, err
	}
	if rawRanges == nil {
		return []base.Range{}, nil
	}

	ranges := make([]base.Range, 0, len(rawRanges))
	for _, r := range rawRanges {
		ranges = append(ranges, base.Range{
			Start: lsp.Position{
				Line:      r.Start.Line,
				Character: r.Start.Character,
			},
			End: lsp.Position{
				Line:      r.End.Line,
				Character: r.End.Character,
			},
		})
	}
	return ranges, nil
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diags, err := es.Diagnose(statement)
	if err != nil {
		return nil, err
	}
	if diags == nil {
		return nil, nil
	}

	var diagnostics []base.Diagnostic
	for _, d := range diags {
		diagnostics = append(diagnostics, base.Diagnostic{
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      uint32(d.Range.Start.Line),
					Character: uint32(d.Range.Start.Column),
				},
				End: lsp.Position{
					Line:      uint32(d.Range.End.Line),
					Character: uint32(d.Range.End.Column),
				},
			},
			Severity: lsp.DiagnosticSeverity(d.Severity),
			Source:   "Syntax check",
			Message:  d.Message,
		})
	}
	return diagnostics, nil
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
		API:             maskableAPIMap[a.API],
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

func convertQueryType(qt es.QueryType) base.QueryType {
	switch qt {
	case es.QueryTypeSelect:
		return base.Select
	case es.QueryTypeExplain:
		return base.Explain
	case es.QueryTypeSelectInfoSchema:
		return base.SelectInfoSchema
	case es.QueryTypeDDL:
		return base.DDL
	case es.QueryTypeDML:
		return base.DML
	default:
		return base.QueryTypeUnknown
	}
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
