package v1

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	esparser "github.com/bytebase/bytebase/backend/plugin/parser/elasticsearch"
	"github.com/bytebase/bytebase/backend/store"
)

// lookupSemanticTypeByDotPath splits a dot-delimited field path and walks the ObjectSchema
// to find the semantic type at the leaf.
func lookupSemanticTypeByDotPath(dotPath string, objectSchema *storepb.ObjectSchema) string {
	parts := strings.Split(dotPath, ".")
	current := objectSchema
	for _, part := range parts {
		if current == nil {
			return ""
		}
		if current.SemanticType != "" {
			return current.SemanticType
		}
		structKind := current.GetStructKind()
		if structKind == nil {
			return ""
		}
		props := structKind.GetProperties()
		if props == nil {
			return ""
		}
		child, ok := props[part]
		if !ok {
			return ""
		}
		current = child
	}
	if current != nil {
		return current.SemanticType
	}
	return ""
}

// maskElasticsearchSourceObject masks a single _source JSON object using
// the ObjectSchema and semantic type maskers. It delegates to walkAndMaskJSONRecursive
// from catalog_masking.go: when a field has a semantic type, the entire value is
// replaced directly regardless of whether it is a primitive, object, or array.
func maskElasticsearchSourceObject(source map[string]any, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (map[string]any, error) {
	if objectSchema == nil {
		return source, nil
	}
	masked, err := walkAndMaskJSONRecursive(source, objectSchema, semanticTypeToMasker)
	if err != nil {
		return nil, err
	}
	result, ok := masked.(map[string]any)
	if !ok {
		return source, nil
	}
	return result, nil
}

// maskElasticsearchHitFields masks the "fields" section of a hit.
// Fields use dot-path keys with array values.
func maskElasticsearchHitFields(hitMap map[string]any, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) error {
	fieldsVal, ok := hitMap["fields"]
	if !ok {
		return nil
	}
	fieldsMap, ok := fieldsVal.(map[string]any)
	if !ok {
		return nil
	}
	for dotPath, values := range fieldsMap {
		semanticType := lookupSemanticTypeByDotPath(dotPath, objectSchema)
		if semanticType == "" {
			continue
		}
		m, ok := semanticTypeToMasker[semanticType]
		if !ok {
			continue
		}
		arr, ok := values.([]any)
		if !ok {
			continue
		}
		for i, val := range arr {
			masked, err := applyMaskerToData(val, m)
			if err != nil {
				return err
			}
			arr[i] = masked
		}
		fieldsMap[dotPath] = arr
	}
	hitMap["fields"] = fieldsMap
	return nil
}

// maskElasticsearchHitHighlight masks the "highlight" section of a hit.
// Highlight uses dot-path keys with array values (HTML fragment strings).
func maskElasticsearchHitHighlight(hitMap map[string]any, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) error {
	highlightVal, ok := hitMap["highlight"]
	if !ok {
		return nil
	}
	highlightMap, ok := highlightVal.(map[string]any)
	if !ok {
		return nil
	}
	for dotPath, values := range highlightMap {
		semanticType := lookupSemanticTypeByDotPath(dotPath, objectSchema)
		if semanticType == "" {
			continue
		}
		m, ok := semanticTypeToMasker[semanticType]
		if !ok {
			continue
		}
		arr, ok := values.([]any)
		if !ok {
			continue
		}
		for i, val := range arr {
			masked, err := applyMaskerToData(val, m)
			if err != nil {
				return err
			}
			arr[i] = masked
		}
		highlightMap[dotPath] = arr
	}
	hitMap["highlight"] = highlightMap
	return nil
}

// maskElasticsearchHitSort masks the "sort" array of a hit based on the sort fields from the request.
// sortFields is the ordered list of field names used in the sort clause (from request analysis).
func maskElasticsearchHitSort(hitMap map[string]any, sortFields []string, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) error {
	sortVal, ok := hitMap["sort"]
	if !ok {
		return nil
	}
	sortArr, ok := sortVal.([]any)
	if !ok {
		return nil
	}
	for i, val := range sortArr {
		if i >= len(sortFields) {
			break
		}
		fieldName := sortFields[i]
		if fieldName == "" {
			continue
		}
		semanticType := lookupSemanticTypeByDotPath(fieldName, objectSchema)
		if semanticType == "" {
			continue
		}
		m, ok := semanticTypeToMasker[semanticType]
		if !ok {
			continue
		}
		masked, err := applyMaskerToData(val, m)
		if err != nil {
			return err
		}
		sortArr[i] = masked
	}
	hitMap["sort"] = sortArr
	return nil
}

// maskElasticsearchSingleHit masks _source, fields, highlight, sort, and inner_hits for a single hit.
func maskElasticsearchSingleHit(hitMap map[string]any, sortFields []string, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) error {
	if source, ok := hitMap["_source"].(map[string]any); ok {
		masked, err := maskElasticsearchSourceObject(source, objectSchema, semanticTypeToMasker)
		if err != nil {
			return err
		}
		hitMap["_source"] = masked
	}
	if err := maskElasticsearchHitFields(hitMap, objectSchema, semanticTypeToMasker); err != nil {
		return err
	}
	if err := maskElasticsearchHitHighlight(hitMap, objectSchema, semanticTypeToMasker); err != nil {
		return err
	}
	if err := maskElasticsearchHitSort(hitMap, sortFields, objectSchema, semanticTypeToMasker); err != nil {
		return err
	}
	return maskElasticsearchHitInnerHits(hitMap, sortFields, objectSchema, semanticTypeToMasker)
}

// maskElasticsearchHitInnerHits masks the inner_hits section of a hit.
// inner_hits.<name>.hits.hits[] has the same structure as outer hits,
// so masking is applied recursively using the same ObjectSchema.
func maskElasticsearchHitInnerHits(hitMap map[string]any, sortFields []string, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) error {
	innerHitsVal, ok := hitMap["inner_hits"]
	if !ok {
		return nil
	}
	innerHitsMap, ok := innerHitsVal.(map[string]any)
	if !ok {
		return nil
	}
	for name, ihVal := range innerHitsMap {
		if err := maskElasticsearchInnerHitGroup(name, ihVal, sortFields, objectSchema, semanticTypeToMasker); err != nil {
			return err
		}
	}
	return nil
}

// maskElasticsearchInnerHitGroup masks a single named inner_hits group (e.g. inner_hits.comments).
func maskElasticsearchInnerHitGroup(name string, ihVal any, sortFields []string, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) error {
	ihObj, ok := ihVal.(map[string]any)
	if !ok {
		return nil
	}
	hitsVal, ok := ihObj["hits"]
	if !ok {
		return nil
	}
	hitsObj, ok := hitsVal.(map[string]any)
	if !ok {
		return nil
	}
	hitsArray, ok := hitsObj["hits"].([]any)
	if !ok {
		return nil
	}
	for i, innerHit := range hitsArray {
		innerHitMap, ok := innerHit.(map[string]any)
		if !ok {
			continue
		}
		if err := maskElasticsearchSingleHit(innerHitMap, sortFields, objectSchema, semanticTypeToMasker); err != nil {
			return errors.Wrapf(err, "failed to mask inner_hits.%s hit %d", name, i)
		}
		hitsArray[i] = innerHitMap
	}
	return nil
}

// maskElasticsearchHitsColumn masks _source, fields, highlight, sort, and inner_hits in each hit
// within a hits column value. The hitsColumnJSON is the JSON string from the "hits" column,
// e.g. {"total":{"value":1},"hits":[{"_source":{...},"fields":{...},"highlight":{...},"sort":[...]}]}.
// sortFields is the ordered list of field names from the request's sort clause.
func maskElasticsearchHitsColumn(hitsColumnJSON string, sortFields []string, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (string, error) {
	var hitsObj map[string]any
	if err := json.Unmarshal([]byte(hitsColumnJSON), &hitsObj); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal hits column")
	}

	hitsArray, ok := hitsObj["hits"].([]any)
	if !ok {
		return hitsColumnJSON, nil
	}

	for i, hit := range hitsArray {
		hitMap, ok := hit.(map[string]any)
		if !ok {
			continue
		}
		if err := maskElasticsearchSingleHit(hitMap, sortFields, objectSchema, semanticTypeToMasker); err != nil {
			return "", errors.Wrapf(err, "failed to mask hit %d", i)
		}
		hitsArray[i] = hitMap
	}
	hitsObj["hits"] = hitsArray

	out, err := json.Marshal(hitsObj)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal masked hits column")
	}
	return string(out), nil
}

// maskElasticsearchDocSource masks the _source column value for a GET _doc response.
// The sourceColumnJSON is the JSON string from the "_source" column, which is the raw document.
func maskElasticsearchDocSource(sourceColumnJSON string, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (string, error) {
	var source map[string]any
	if err := json.Unmarshal([]byte(sourceColumnJSON), &source); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal _source column")
	}

	masked, err := maskElasticsearchSourceObject(source, objectSchema, semanticTypeToMasker)
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(masked)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal masked _source column")
	}
	return string(out), nil
}

// maskElasticsearchGetSourceColumn masks a single field value from a GET _source response.
// In _source responses, each top-level document field becomes a separate column.
// fieldName is the column name (= field name), valueJSON is the marshaled value.
func maskElasticsearchGetSourceColumn(fieldName, valueJSON string, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (string, error) {
	var childSchema *storepb.ObjectSchema
	if structKind := objectSchema.GetStructKind(); structKind != nil {
		if props := structKind.GetProperties(); props != nil {
			childSchema = props[fieldName]
		}
	}
	if childSchema == nil {
		return valueJSON, nil
	}

	var value any
	if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal field %q", fieldName)
	}

	masked, err := walkAndMaskJSONRecursive(value, childSchema, semanticTypeToMasker)
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(masked)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal masked field %q", fieldName)
	}
	return string(out), nil
}

// maskElasticsearchMGetSource masks _source in each doc within a _mget response.
// The docsColumnJSON is the JSON string from the "docs" column.
func maskElasticsearchMGetSource(docsColumnJSON string, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (string, error) {
	var docs []any
	if err := json.Unmarshal([]byte(docsColumnJSON), &docs); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal docs column")
	}

	for i, doc := range docs {
		docMap, ok := doc.(map[string]any)
		if !ok {
			continue
		}
		if source, ok := docMap["_source"].(map[string]any); ok {
			masked, err := maskElasticsearchSourceObject(source, objectSchema, semanticTypeToMasker)
			if err != nil {
				return "", errors.Wrapf(err, "failed to mask _source in doc %d", i)
			}
			docMap["_source"] = masked
		}
		docs[i] = docMap
	}

	out, err := json.Marshal(docs)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal masked docs column")
	}
	return string(out), nil
}

// maskElasticsearchMSearchResponses masks the "responses" column from a _msearch response.
// Each element in the responses array is a full search response with hits.
func maskElasticsearchMSearchResponses(responsesColumnJSON string, sortFields []string, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (string, error) {
	var responses []any
	if err := json.Unmarshal([]byte(responsesColumnJSON), &responses); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal responses column")
	}

	for i, resp := range responses {
		respMap, ok := resp.(map[string]any)
		if !ok {
			continue
		}
		if err := maskElasticsearchMSearchSingleResponse(respMap, sortFields, objectSchema, semanticTypeToMasker); err != nil {
			return "", errors.Wrapf(err, "failed to mask response %d", i)
		}
	}

	out, err := json.Marshal(responses)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal masked responses column")
	}
	return string(out), nil
}

// maskElasticsearchMSearchSingleResponse masks the hits in a single _msearch response element.
func maskElasticsearchMSearchSingleResponse(respMap map[string]any, sortFields []string, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) error {
	hitsVal, ok := respMap["hits"]
	if !ok {
		return nil
	}
	hitsObj, ok := hitsVal.(map[string]any)
	if !ok {
		return nil
	}
	hitsArray, ok := hitsObj["hits"].([]any)
	if !ok {
		return nil
	}
	for j, hit := range hitsArray {
		hitMap, ok := hit.(map[string]any)
		if !ok {
			continue
		}
		if err := maskElasticsearchSingleHit(hitMap, sortFields, objectSchema, semanticTypeToMasker); err != nil {
			return errors.Wrapf(err, "failed to mask hit %d", j)
		}
		hitsArray[j] = hitMap
	}
	return nil
}

// checkElasticsearchRequestBlocked returns an error if the request must be blocked
// because masking policies exist on the target index.
func checkElasticsearchRequestBlocked(analysis *esparser.RequestAnalysis) error {
	if analysis.API == esparser.APIBlocked {
		return errors.New("this Elasticsearch API is not supported when data masking is configured on the target index")
	}
	if len(analysis.BlockedFeatures) > 0 {
		names := make([]string, 0, len(analysis.BlockedFeatures))
		for _, f := range analysis.BlockedFeatures {
			names = append(names, esparser.BlockedFeatureNames[f])
		}
		return errors.Errorf("the following features are not supported when data masking is configured: %s", strings.Join(names, ", "))
	}
	return nil
}

// getElasticsearchIndexObjectSchema retrieves the ObjectSchema for an ES index from the database config.
func getElasticsearchIndexObjectSchema(ctx context.Context, stores *store.Store, instanceID string, databaseName string, indexName string) (*storepb.ObjectSchema, error) {
	dbMetadata, err := stores.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database schema: %q", databaseName)
	}

	if dbMetadata == nil {
		return nil, nil
	}

	schemas := dbMetadata.GetConfig().GetSchemas()
	if len(schemas) == 0 {
		return nil, nil
	}

	schema := schemas[0]
	tables := schema.GetTables()
	for _, table := range tables {
		if table.GetName() == indexName {
			return table.GetObjectSchema(), nil
		}
	}

	return nil, nil
}

// objectSchemaHasSemanticTypes recursively checks if any node in the ObjectSchema has a semantic type set.
func objectSchemaHasSemanticTypes(os *storepb.ObjectSchema) bool {
	if os == nil {
		return false
	}
	if os.SemanticType != "" {
		return true
	}
	if sk := os.GetStructKind(); sk != nil {
		for _, prop := range sk.GetProperties() {
			if objectSchemaHasSemanticTypes(prop) {
				return true
			}
		}
	}
	if ak := os.GetArrayKind(); ak != nil {
		if objectSchemaHasSemanticTypes(ak.GetKind()) {
			return true
		}
	}
	return false
}
