package mongodb

import (
	"slices"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mongoparser "github.com/bytebase/parser/mongodb"
)

// MaskableAPI categorizes MongoDB collection reads for masking behavior.
type MaskableAPI int

const (
	// MaskableAPIUnsupported means the statement is not relevant for MongoDB masking.
	MaskableAPIUnsupported MaskableAPI = iota
	// MaskableAPIFind means db.<collection>.find(...).
	MaskableAPIFind
	// MaskableAPIFindOne means db.<collection>.findOne(...).
	MaskableAPIFindOne
	// MaskableAPIUnsupportedRead means a read API that is blocked in Milestone 1.
	MaskableAPIUnsupportedRead
	// MaskableAPIAggregate means db.<collection>.aggregate(...) with only shape-preserving stages.
	MaskableAPIAggregate
)

// JoinedCollection records a $lookup or $graphLookup join extracted from an aggregate pipeline.
type JoinedCollection struct {
	// AsField is the output field name added to each document (the "as" argument).
	AsField string
	// Collection is the source collection being joined (the "from" argument).
	Collection string
}

// MaskingAnalysis contains MongoDB statement data needed by masking checks.
type MaskingAnalysis struct {
	API             MaskableAPI
	Operation       string
	Collection      string
	PredicateFields []string
	// UnsupportedStage is the first pipeline stage that prevents masking (e.g. "$group").
	// Only set when API == MaskableAPIUnsupportedRead for aggregate pipelines.
	UnsupportedStage string
	// JoinedCollections holds join info extracted from $lookup and $graphLookup stages.
	JoinedCollections []JoinedCollection
}

// AnalyzeMaskingStatement analyzes a MongoDB shell statement for masking checks.
// It returns nil,nil for statements that are not relevant to masking flow.
func AnalyzeMaskingStatement(statement string) (*MaskingAnalysis, error) {
	raw := parseMongoShellRaw(statement)
	if raw == nil || raw.Tree == nil {
		return nil, nil
	}
	if len(raw.Errors) > 0 {
		return nil, errors.Errorf("failed to parse MongoDB statement: %s", raw.Errors[0].Message)
	}

	if len(raw.Tree.AllStatement()) != 1 {
		return nil, nil
	}

	listener := &maskingAnalysisListener{}
	antlr.ParseTreeWalkerDefault.Walk(listener, raw.Tree)
	if listener.analysis == nil {
		return nil, nil
	}
	return listener.analysis, nil
}

type maskingAnalysisListener struct {
	*mongoparser.BaseMongoShellParserListener

	analysis *MaskingAnalysis
}

func (l *maskingAnalysisListener) EnterCollectionOperation(ctx *mongoparser.CollectionOperationContext) {
	if l.analysis != nil {
		return
	}

	collection := extractCollectionName(ctx.CollectionAccess())
	if collection == "" {
		return
	}

	chain := ctx.MethodChain()
	if chain == nil {
		return
	}
	mc := chain.CollectionMethodCall()
	if mc == nil {
		return
	}

	analysis := classifyMaskingMethod(mc)
	if analysis == nil {
		return
	}
	analysis.Collection = collection
	if len(analysis.PredicateFields) > 0 {
		slices.Sort(analysis.PredicateFields)
	}
	l.analysis = analysis
}

func classifyMaskingMethod(mc mongoparser.ICollectionMethodCallContext) *MaskingAnalysis {
	switch {
	case mc.FindMethod() != nil:
		return &MaskingAnalysis{
			API:             MaskableAPIFind,
			Operation:       "find",
			PredicateFields: extractFindPredicateFields(mc.FindMethod().Arguments()),
		}
	case mc.FindOneMethod() != nil:
		return &MaskingAnalysis{
			API:             MaskableAPIFindOne,
			Operation:       "findOne",
			PredicateFields: extractFindPredicateFields(mc.FindOneMethod().Arguments()),
		}
	case mc.CountDocumentsMethod() != nil:
		return unsupportedReadAnalysis("countDocuments")
	case mc.EstimatedDocumentCountMethod() != nil:
		return unsupportedReadAnalysis("estimatedDocumentCount")
	case mc.CollectionCountMethod() != nil:
		return unsupportedReadAnalysis("count")
	case mc.DistinctMethod() != nil:
		return unsupportedReadAnalysis("distinct")
	case mc.AggregateMethod() != nil:
		return classifyAggregateForMasking(mc.AggregateMethod())
	case mc.GetIndexesMethod() != nil:
		return unsupportedReadAnalysis("getIndexes")
	case mc.StatsMethod() != nil:
		return unsupportedReadAnalysis("stats")
	case mc.StorageSizeMethod() != nil:
		return unsupportedReadAnalysis("storageSize")
	case mc.TotalIndexSizeMethod() != nil:
		return unsupportedReadAnalysis("totalIndexSize")
	case mc.TotalSizeMethod() != nil:
		return unsupportedReadAnalysis("totalSize")
	case mc.DataSizeMethod() != nil:
		return unsupportedReadAnalysis("dataSize")
	case mc.IsCappedMethod() != nil:
		return unsupportedReadAnalysis("isCapped")
	case mc.ValidateMethod() != nil:
		return unsupportedReadAnalysis("validate")
	case mc.LatencyStatsMethod() != nil:
		return unsupportedReadAnalysis("latencyStats")
	case mc.GetShardDistributionMethod() != nil:
		return unsupportedReadAnalysis("getShardDistribution")
	case mc.GetShardVersionMethod() != nil:
		return unsupportedReadAnalysis("getShardVersion")
	case mc.AnalyzeShardKeyMethod() != nil:
		return unsupportedReadAnalysis("analyzeShardKey")
	default:
		return nil
	}
}

func unsupportedReadAnalysis(operation string) *MaskingAnalysis {
	return &MaskingAnalysis{
		API:       MaskableAPIUnsupportedRead,
		Operation: operation,
	}
}

func extractCollectionName(access mongoparser.ICollectionAccessContext) string {
	switch access := access.(type) {
	case *mongoparser.DotAccessContext:
		if id := access.Identifier(); id != nil {
			return id.GetText()
		}
	case *mongoparser.BracketAccessContext:
		if sl := access.StringLiteral(); sl != nil {
			return unquoteMongoString(sl.GetText())
		}
	case *mongoparser.GetCollectionAccessContext:
		if sl := access.StringLiteral(); sl != nil {
			return unquoteMongoString(sl.GetText())
		}
	default:
	}
	return ""
}

func unquoteMongoString(s string) string {
	if s == "" {
		return ""
	}
	if unquoted, err := strconv.Unquote(s); err == nil {
		return unquoted
	}
	return strings.Trim(s, `"'`)
}

func extractFindPredicateFields(args mongoparser.IArgumentsContext) []string {
	if args == nil {
		return nil
	}
	allArgs := args.AllArgument()
	if len(allArgs) == 0 {
		return nil
	}

	first := allArgs[0]
	if first == nil || first.Value() == nil {
		return nil
	}

	doc := extractDocumentValue(first.Value())
	if doc == nil {
		return nil
	}

	fields := make(map[string]struct{})
	collectPredicateFieldsFromDocument(doc, "", fields)
	if len(fields) == 0 {
		return nil
	}

	result := make([]string, 0, len(fields))
	for field := range fields {
		result = append(result, field)
	}
	return result
}

func extractDocumentValue(value mongoparser.IValueContext) mongoparser.IDocumentContext {
	if value == nil {
		return nil
	}
	switch value := value.(type) {
	case *mongoparser.DocumentValueContext:
		return value.Document()
	default:
		return nil
	}
}

func extractArrayValue(value mongoparser.IValueContext) mongoparser.IArrayContext {
	if value == nil {
		return nil
	}
	switch value := value.(type) {
	case *mongoparser.ArrayValueContext:
		return value.Array()
	default:
		return nil
	}
}

func collectPredicateFieldsFromDocument(doc mongoparser.IDocumentContext, prefix string, fields map[string]struct{}) {
	if doc == nil {
		return
	}
	for _, pair := range doc.AllPair() {
		collectPredicateFieldsFromPair(pair, prefix, fields)
	}
}

func collectPredicateFieldsFromPair(pair mongoparser.IPairContext, prefix string, fields map[string]struct{}) {
	if pair == nil || pair.Key() == nil {
		return
	}

	key := extractPairKey(pair.Key())
	if key == "" {
		return
	}

	if strings.HasPrefix(key, "$") {
		collectLogicalPredicateFieldsByKey(key, pair.Value(), prefix, fields)
		return
	}

	fullPath := buildPredicateFieldPath(prefix, key)
	fields[fullPath] = struct{}{}
	collectPredicateFieldValue(pair.Value(), fullPath, fields)
}

func collectLogicalPredicateFieldsByKey(key string, value mongoparser.IValueContext, prefix string, fields map[string]struct{}) {
	if !isLogicalOperator(key) {
		return
	}
	collectLogicalOperatorValue(value, prefix, fields)
}

func buildPredicateFieldPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func collectPredicateFieldValue(value mongoparser.IValueContext, fullPath string, fields map[string]struct{}) {
	if childDoc := extractDocumentValue(value); childDoc != nil {
		collectPredicateFieldsFromDocument(childDoc, fullPath, fields)
		return
	}
	if childArr := extractArrayValue(value); childArr != nil {
		collectPredicateFieldsFromArray(childArr, fullPath, fields)
	}
}

func collectPredicateFieldsFromArray(arr mongoparser.IArrayContext, prefix string, fields map[string]struct{}) {
	if arr == nil {
		return
	}
	for _, v := range arr.AllValue() {
		if childDoc := extractDocumentValue(v); childDoc != nil {
			collectPredicateFieldsFromDocument(childDoc, prefix, fields)
		}
	}
}

func collectLogicalOperatorValue(value mongoparser.IValueContext, prefix string, fields map[string]struct{}) {
	if value == nil {
		return
	}
	if doc := extractDocumentValue(value); doc != nil {
		collectPredicateFieldsFromDocument(doc, prefix, fields)
		return
	}
	if arr := extractArrayValue(value); arr != nil {
		collectPredicateFieldsFromArray(arr, prefix, fields)
	}
}

func extractPairKey(keyCtx mongoparser.IKeyContext) string {
	switch keyCtx := keyCtx.(type) {
	case *mongoparser.UnquotedKeyContext:
		if id := keyCtx.Identifier(); id != nil {
			return id.GetText()
		}
	case *mongoparser.QuotedKeyContext:
		if sl := keyCtx.StringLiteral(); sl != nil {
			return unquoteMongoString(sl.GetText())
		}
	default:
	}
	if keyCtx == nil {
		return ""
	}
	return keyCtx.GetText()
}

func isLogicalOperator(key string) bool {
	switch key {
	case "$and", "$or", "$nor":
		return true
	default:
		return false
	}
}

// shapePreservingAggregateStages contains aggregate pipeline stages whose output
// documents retain the collection's original structure (fields are not reshaped).
var shapePreservingAggregateStages = map[string]bool{
	"$match":           true,
	"$sort":            true,
	"$limit":           true,
	"$skip":            true,
	"$sample":          true,
	"$addFields":       true,
	"$set":             true,
	"$unset":           true,
	"$geoNear":         true,
	"$setWindowFields": true,
	"$fill":            true,
	"$redact":          true,
	"$unwind":          true,
}

func classifyAggregateForMasking(am mongoparser.IAggregateMethodContext) *MaskingAnalysis {
	predicateFields, joinedCollections, unsupportedStage := extractAggregatePredicateFields(am.Arguments())
	if unsupportedStage != "" {
		return &MaskingAnalysis{
			API:              MaskableAPIUnsupportedRead,
			Operation:        "aggregate",
			UnsupportedStage: unsupportedStage,
		}
	}
	return &MaskingAnalysis{
		API:               MaskableAPIAggregate,
		Operation:         "aggregate",
		PredicateFields:   predicateFields,
		JoinedCollections: joinedCollections,
	}
}

// extractAggregatePredicateFields walks the pipeline and returns:
// - predicate fields from $match stages
// - join info from $lookup/$graphLookup stages
// - the first unsupported stage name, or "" if all stages are allowed
func extractAggregatePredicateFields(args mongoparser.IArgumentsContext) ([]string, []JoinedCollection, string) {
	if args == nil {
		return nil, nil, ""
	}
	allArgs := args.AllArgument()
	if len(allArgs) == 0 {
		return nil, nil, ""
	}

	first := allArgs[0]
	if first == nil || first.Value() == nil {
		return nil, nil, ""
	}

	arr := extractArrayValue(first.Value())
	if arr == nil {
		return nil, nil, "unknown"
	}

	fields := make(map[string]struct{})
	var joinedCollections []JoinedCollection
	for _, elem := range arr.AllValue() {
		doc := extractDocumentValue(elem)
		if doc == nil {
			return nil, nil, "unknown"
		}
		pairs := doc.AllPair()
		if len(pairs) == 0 {
			continue
		}
		stageName := extractPairKey(pairs[0].Key())
		switch {
		case shapePreservingAggregateStages[stageName]:
			if stageName == "$match" {
				stageDoc := extractDocumentValue(pairs[0].Value())
				if stageDoc != nil {
					collectPredicateFieldsFromDocument(stageDoc, "", fields)
				}
			}
		case stageName == "$lookup":
			join, unsupported := extractLookupJoin(pairs[0].Value())
			if unsupported {
				return nil, nil, "$lookup"
			}
			if join != nil {
				joinedCollections = append(joinedCollections, *join)
			}
		case stageName == "$graphLookup":
			join := extractGraphLookupJoin(pairs[0].Value())
			if join != nil {
				joinedCollections = append(joinedCollections, *join)
			}
		default:
			return nil, nil, stageName
		}
	}

	var predicateFields []string
	if len(fields) > 0 {
		predicateFields = make([]string, 0, len(fields))
		for field := range fields {
			predicateFields = append(predicateFields, field)
		}
	}
	return predicateFields, joinedCollections, ""
}

// extractLookupJoin parses a $lookup stage value and returns the join info.
// Returns (nil, true) if this is a pipeline-form $lookup (unsupported).
// Returns (nil, false) if from/as cannot be extracted (treated as passthrough).
func extractLookupJoin(value mongoparser.IValueContext) (*JoinedCollection, bool) {
	doc := extractDocumentValue(value)
	if doc == nil {
		return nil, false
	}
	var from, as string
	for _, pair := range doc.AllPair() {
		key := extractPairKey(pair.Key())
		switch key {
		case "pipeline":
			// Pipeline form is not supported.
			return nil, true
		case "from":
			from = extractStringLiteralValue(pair.Value())
		case "as":
			as = extractStringLiteralValue(pair.Value())
		default:
		}
	}
	if from == "" || as == "" {
		return nil, false
	}
	return &JoinedCollection{AsField: as, Collection: from}, false
}

// extractGraphLookupJoin parses a $graphLookup stage value and returns the join info.
func extractGraphLookupJoin(value mongoparser.IValueContext) *JoinedCollection {
	doc := extractDocumentValue(value)
	if doc == nil {
		return nil
	}
	var from, as string
	for _, pair := range doc.AllPair() {
		key := extractPairKey(pair.Key())
		switch key {
		case "from":
			from = extractStringLiteralValue(pair.Value())
		case "as":
			as = extractStringLiteralValue(pair.Value())
		default:
		}
	}
	if from == "" || as == "" {
		return nil
	}
	return &JoinedCollection{AsField: as, Collection: from}
}

// extractStringLiteralValue extracts a plain string value from a Value node.
func extractStringLiteralValue(value mongoparser.IValueContext) string {
	if value == nil {
		return ""
	}
	text := value.GetText()
	if len(text) >= 2 && (text[0] == '"' || text[0] == '\'') {
		return unquoteMongoString(text)
	}
	return ""
}
