package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
)

// documentMasker masks query results for document-oriented databases
// (CosmosDB, MongoDB, Elasticsearch) that require per-engine masking logic.
type documentMasker interface {
	maskResults(
		ctx context.Context,
		stores *store.Store,
		database *store.DatabaseMessage,
		spans []*parserbase.QuerySpan,
		results []*v1pb.QueryResult,
		semanticTypeToMaskerMap map[string]masker.Masker,
		queryContext db.QueryContext,
	) error
}

// getDocumentMasker returns the appropriate documentMasker for the given engine,
// or nil if the engine does not use document-based masking.
func getDocumentMasker(engine storepb.Engine) documentMasker {
	switch engine {
	case storepb.Engine_COSMOSDB:
		return &cosmosDBMasker{}
	case storepb.Engine_MONGODB:
		return &mongoDBMasker{}
	case storepb.Engine_ELASTICSEARCH:
		return &elasticsearchMasker{}
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// preExecuteMaskingCheck runs engine-specific pre-execution checks that block
// queries with unsupported APIs. Sensitive predicate checks are handled
// post-execution on a per-statement basis by checkSensitivePredicates.
func preExecuteMaskingCheck(
	ctx context.Context,
	stores *store.Store,
	engine storepb.Engine,
	database *store.DatabaseMessage,
	spans []*parserbase.QuerySpan,
) error {
	switch engine {
	case storepb.Engine_MONGODB:
		return preExecuteMaskingCheckMongoDB(ctx, stores, database, spans)
	case storepb.Engine_ELASTICSEARCH:
		return preExecuteMaskingCheckElasticsearch(ctx, stores, database, spans)
	default:
		return nil
	}
}

func preExecuteMaskingCheckMongoDB(
	ctx context.Context,
	stores *store.Store,
	database *store.DatabaseMessage,
	spans []*parserbase.QuerySpan,
) error {
	for _, span := range spans {
		if span.MongoDBAnalysis == nil {
			continue
		}
		analysis := span.MongoDBAnalysis
		if analysis.Collection == "" {
			continue
		}

		objectSchema, err := getTableObjectSchema(ctx, stores, database.InstanceID, database.DatabaseName, analysis.Collection)
		if err != nil {
			return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get object schema for collection %q", analysis.Collection))
		}
		if objectSchema == nil {
			continue
		}

		if err := checkMongoDBRequestBlocked(analysis); err != nil {
			return connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	return nil
}

func preExecuteMaskingCheckElasticsearch(
	ctx context.Context,
	stores *store.Store,
	database *store.DatabaseMessage,
	spans []*parserbase.QuerySpan,
) error {
	for _, span := range spans {
		if span.ElasticsearchAnalysis == nil {
			continue
		}
		analysis := span.ElasticsearchAnalysis
		if analysis.API == parserbase.ElasticsearchAPIUnsupported {
			continue
		}
		indexName := analysis.Index
		if indexName == "" {
			continue
		}
		objectSchema, err := getTableObjectSchema(ctx, stores, database.InstanceID, database.DatabaseName, indexName)
		if err != nil {
			return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get object schema for index %q", indexName))
		}
		if objectSchema == nil || !objectSchemaHasSemanticTypes(objectSchema) {
			continue
		}
		if err := checkElasticsearchRequestBlocked(analysis); err != nil {
			return connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	return nil
}

// checkSensitivePredicates checks whether a span uses sensitive fields in
// predicates. Returns an error message if so, or "" if clean. This is used
// post-execution to blank individual statement results (matching SQL/CosmosDB
// behavior) rather than blocking the entire request.
func checkSensitivePredicates(
	ctx context.Context,
	stores *store.Store,
	database *store.DatabaseMessage,
	span *parserbase.QuerySpan,
) (string, error) {
	if len(span.PredicatePaths) == 0 {
		return "", nil
	}

	var objectSchema *storepb.ObjectSchema
	var err error

	if analysis := span.MongoDBAnalysis; analysis != nil && analysis.Collection != "" {
		objectSchema, err = getTableObjectSchema(ctx, stores, database.InstanceID, database.DatabaseName, analysis.Collection)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get object schema for collection %q", analysis.Collection)
		}
	} else if analysis := span.ElasticsearchAnalysis; analysis != nil && analysis.Index != "" {
		objectSchema, err = getTableObjectSchema(ctx, stores, database.InstanceID, database.DatabaseName, analysis.Index)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get object schema for index %q", analysis.Index)
		}
	}

	if objectSchema == nil {
		return "", nil
	}

	for pathStr := range span.PredicatePaths {
		semanticType := lookupSemanticTypeByDotPath(pathStr, objectSchema)
		if semanticType != "" {
			return fmt.Sprintf("using field %q tagged by semantic type %q in query predicate is not allowed", pathStr, semanticType), nil
		}
	}
	return "", nil
}

func getFirstSemanticTypeInPath(ast *parserbase.PathAST, objectSchema *storepb.ObjectSchema) string {
	if ast == nil || ast.Root == nil || objectSchema == nil {
		return ""
	}

	// Skip the first node because it always represents the container.
	astWoutContainer := parserbase.NewPathAST(ast.Root.GetNext())
	if astWoutContainer == nil || astWoutContainer.Root == nil {
		return ""
	}

	if objectSchema.SemanticType != "" {
		return objectSchema.SemanticType
	}

	os := objectSchema

	for node := astWoutContainer.Root; node != nil; node = node.GetNext() {
		if node.GetIdentifier() == "" {
			return ""
		}

		switch node := node.(type) {
		case *parserbase.ItemSelector:
			if os.Type != storepb.ObjectSchema_OBJECT {
				return ""
			}
			var valid bool
			if v := os.GetStructKind().GetProperties(); v != nil {
				if child, ok := v[node.GetIdentifier()]; ok {
					os = child
					valid = true
				}
			}
			if !valid {
				return ""
			}
		case *parserbase.ArraySelector:
			if os.Type != storepb.ObjectSchema_OBJECT {
				return ""
			}
			var valid bool
			if v := os.GetStructKind().GetProperties(); v != nil {
				if child, ok := v[node.GetIdentifier()]; ok {
					os = child
					valid = true
				}
			}
			if !valid {
				return ""
			}

			if os.Type != storepb.ObjectSchema_ARRAY {
				return ""
			}

			os = os.GetArrayKind().GetKind()
			if os == nil {
				return ""
			}
		default:
		}

		if os.SemanticType != "" {
			return os.SemanticType
		}
	}

	return ""
}

// ---------------------------------------------------------------------------
// Shared JSON masking utilities
// ---------------------------------------------------------------------------

// maskDocumentString unmarshals a JSON string, masks it recursively using
// the ObjectSchema and semantic type maskers, and marshals it back.
func maskDocumentString(document string, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (string, error) {
	if document == "" || objectSchema == nil {
		return document, nil
	}

	var parsed any
	if err := json.Unmarshal([]byte(document), &parsed); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal document")
	}

	masked, err := walkAndMaskJSONRecursive(parsed, objectSchema, semanticTypeToMasker)
	if err != nil {
		return "", errors.Wrap(err, "failed to mask document")
	}

	out, err := json.Marshal(masked)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal masked document")
	}
	return string(out), nil
}

func walkAndMaskJSON(data map[string]any, fieldPaths map[string][]*parserbase.PathAST, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (map[string]any, error) {
	result := make(map[string]any)
	for key, value := range data {
		// Check all source field paths for this key. If ANY resolves to a sensitive
		// semantic type, mask the value. This handles expressions like UPPER(c.email)
		// or CONCAT(c.name, c.email) where the result derives from sensitive fields.
		if paths, ok := fieldPaths[key]; ok && len(paths) > 0 {
			if maskedValue, masked, err := maskBySourcePaths(value, paths, objectSchema, semanticTypeToMasker); err != nil {
				return nil, err
			} else if masked {
				result[key] = maskedValue
				continue
			}
			// No source path was sensitive; use the first path's schema for recursive walk.
			ast := stripContainerFromPath(paths[0])
			o, _ := getObjectSchemaByPath(objectSchema, ast)
			fieldValue, err := walkAndMaskJSONRecursive(value, o, semanticTypeToMasker)
			if err != nil {
				return nil, err
			}
			result[key] = fieldValue
			continue
		}

		// No field paths recorded: fall back to direct key lookup in schema.
		ast := parserbase.NewPathAST(parserbase.NewItemSelector(key))
		o, parentSemanticType := getObjectSchemaByPath(objectSchema, ast)
		if parentSemanticType != "" {
			if m, ok := semanticTypeToMasker[parentSemanticType]; ok {
				maskedValue, err := applyMaskerToJSONMember(value, m)
				if err != nil {
					return nil, err
				}
				result[key] = maskedValue
				continue
			}
		}

		fieldValue, err := walkAndMaskJSONRecursive(value, o, semanticTypeToMasker)
		if err != nil {
			return nil, err
		}
		result[key] = fieldValue
	}
	return result, nil
}

// stripContainerFromPath returns a PathAST with the first node (container) removed.
func stripContainerFromPath(path *parserbase.PathAST) *parserbase.PathAST {
	if path != nil && path.Root != nil {
		return parserbase.NewPathAST(path.Root.GetNext())
	}
	return parserbase.NewPathAST(nil)
}

// maskBySourcePaths checks if any source field path resolves to a sensitive semantic type.
// Returns (maskedValue, true, nil) if masked, or (nil, false, nil) if no path is sensitive.
func maskBySourcePaths(value any, paths []*parserbase.PathAST, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (any, bool, error) {
	for _, path := range paths {
		ast := stripContainerFromPath(path)
		_, parentSemanticType := getObjectSchemaByPath(objectSchema, ast)
		if parentSemanticType != "" {
			if m, ok := semanticTypeToMasker[parentSemanticType]; ok {
				maskedValue, err := applyMaskerToJSONMember(value, m)
				if err != nil {
					return nil, false, err
				}
				return maskedValue, true, nil
			}
		}
	}
	return nil, false, nil
}

func getObjectSchemaByPath(objectSchema *storepb.ObjectSchema, path *parserbase.PathAST) (*storepb.ObjectSchema, string) {
	outer := objectSchema
	outerSemanticType := outer.SemanticType
	if outerSemanticType != "" {
		return outer, outer.SemanticType
	}
	for node := path.Root; node != nil; node = node.GetNext() {
		identifier := node.GetIdentifier()
		switch outer.Type {
		case storepb.ObjectSchema_OBJECT:
			v := outer.GetStructKind().GetProperties()
			if v == nil {
				return nil, outerSemanticType
			}
			inner, ok := v[identifier]
			if !ok {
				return nil, outerSemanticType
			}
			outer = inner
			outerSemanticType = outer.SemanticType
		case storepb.ObjectSchema_ARRAY:
			v := outer.GetArrayKind().GetKind()
			if v == nil {
				return nil, outerSemanticType
			}
			if v.Type != storepb.ObjectSchema_OBJECT {
				return nil, outerSemanticType
			}
			p := v.GetStructKind().GetProperties()
			if p == nil {
				return nil, outerSemanticType
			}
			inner, ok := p[identifier]
			if !ok {
				return nil, outerSemanticType
			}
			outer = inner
			outerSemanticType = outer.SemanticType
		default:
			// Other schema types
			return nil, outerSemanticType
		}
	}

	return outer, outerSemanticType
}

func walkAndMaskJSONRecursive(data any, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (any, error) {
	if objectSchema == nil {
		return data, nil
	}
	// Schema is ARRAY but data is a single element (e.g. field unwound by $unwind).
	// When there is no array-level semantic type, descend into the item schema so
	// that item-level masking still applies to the unwound scalar or object.
	if objectSchema.Type == storepb.ObjectSchema_ARRAY && objectSchema.SemanticType == "" {
		if _, isArray := data.([]any); !isArray {
			if itemSchema := objectSchema.GetArrayKind().GetKind(); itemSchema != nil {
				return walkAndMaskJSONRecursive(data, itemSchema, semanticTypeToMasker)
			}
			return data, nil
		}
	}
	switch data := data.(type) {
	case map[string]any:
		if objectSchema.SemanticType != "" {
			// If the semantic type is found, replace the entire value directly.
			if m, ok := semanticTypeToMasker[objectSchema.SemanticType]; ok {
				return applyMaskerToJSONMember(data, m)
			}
		} else {
			// Otherwise, recursively walk the object.
			structKind := objectSchema.GetStructKind()
			// Quick return if there is no struct kind in object schema.
			if structKind == nil {
				return data, nil
			}
			for key, value := range data {
				if childObjectSchema, ok := structKind.Properties[key]; ok {
					// Recursively walk the property if child object schema found.
					var err error
					data[key], err = walkAndMaskJSONRecursive(value, childObjectSchema, semanticTypeToMasker)
					if err != nil {
						return nil, err
					}
				}
			}
		}
		return data, nil
	case []any:
		if objectSchema.SemanticType != "" {
			// If the semantic type is found, replace the entire value directly.
			if m, ok := semanticTypeToMasker[objectSchema.SemanticType]; ok {
				return applyMaskerToJSONMember(data, m)
			}
		} else {
			arrayKind := objectSchema.GetArrayKind()
			// Quick return if there is no array kind in object schema.
			if arrayKind == nil {
				return data, nil
			}
			childObjectSchema := arrayKind.GetKind()
			if childObjectSchema == nil {
				return data, nil
			}
			// Otherwise, recursively walk the array.
			for i, value := range data {
				maskedValue, err := walkAndMaskJSONRecursive(value, childObjectSchema, semanticTypeToMasker)
				if err != nil {
					return nil, err
				}
				data[i] = maskedValue
			}
		}
	default:
		// For JSON atomic member, apply the masker if semantic type is found.
		if objectSchema.SemanticType != "" {
			if m, ok := semanticTypeToMasker[objectSchema.SemanticType]; ok {
				maskedData, err := applyMaskerToData(data, m)
				if err != nil {
					return nil, err
				}
				return maskedData, nil
			}
		}
	}
	return data, nil
}

func applyMaskerToJSONMember(data any, m masker.Masker) (any, error) {
	if rowValue, ok := getRowValueFromJSONAtomicMember(data); ok {
		return getJSONMemberFromRowValue(m.Mask(&masker.MaskData{Data: rowValue})), nil
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal JSON member")
	}
	return getJSONMemberFromRowValue(m.Mask(&masker.MaskData{
		Data: &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(jsonBytes),
			},
		},
	})), nil
}

func applyMaskerToData(data any, m masker.Masker) (any, error) {
	switch data := data.(type) {
	case map[string]any:
		// Recursively apply the masker to the object.
		for key, value := range data {
			maskedValue, err := applyMaskerToData(value, m)
			if err != nil {
				return nil, err
			}
			data[key] = maskedValue
		}
	case []any:
		// Recursively apply the masker to the array.
		for i, value := range data {
			maskedValue, err := applyMaskerToData(value, m)
			if err != nil {
				return nil, err
			}
			data[i] = maskedValue
		}
	default:
		// Apply the masker to the atomic value.
		if wrappedValue, ok := getRowValueFromJSONAtomicMember(data); ok {
			maskedValue := m.Mask(&masker.MaskData{Data: wrappedValue})
			return getJSONMemberFromRowValue(maskedValue), nil
		}
	}

	return data, nil
}

func getJSONMemberFromRowValue(rowValue *v1pb.RowValue) any {
	switch rowValue := rowValue.Kind.(type) {
	case *v1pb.RowValue_NullValue:
		return nil
	case *v1pb.RowValue_BoolValue:
		return rowValue.BoolValue
	case *v1pb.RowValue_BytesValue:
		return string(rowValue.BytesValue)
	case *v1pb.RowValue_DoubleValue:
		return rowValue.DoubleValue
	case *v1pb.RowValue_FloatValue:
		return rowValue.FloatValue
	case *v1pb.RowValue_Int32Value:
		return rowValue.Int32Value
	case *v1pb.RowValue_StringValue:
		return rowValue.StringValue
	case *v1pb.RowValue_Uint32Value:
		return rowValue.Uint32Value
	case *v1pb.RowValue_Uint64Value:
		return rowValue.Uint64Value
	}
	return nil
}

func getRowValueFromJSONAtomicMember(data any) (result *v1pb.RowValue, ok bool) {
	if data == nil {
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_NullValue{},
		}, true
	}
	switch data := data.(type) {
	case string:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{StringValue: data},
		}, true
	case float64:
		// https://pkg.go.dev/encoding/json#Unmarshal
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_DoubleValue{DoubleValue: data},
		}, true
	case bool:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_BoolValue{BoolValue: data},
		}, true
	}
	return nil, false
}

// ---------------------------------------------------------------------------
// CosmosDB
// ---------------------------------------------------------------------------

// cosmosDBMasker implements documentMasker for CosmosDB.
type cosmosDBMasker struct{}

func (*cosmosDBMasker) maskResults(
	ctx context.Context,
	stores *store.Store,
	database *store.DatabaseMessage,
	spans []*parserbase.QuerySpan,
	results []*v1pb.QueryResult,
	semanticTypeToMaskerMap map[string]masker.Masker,
	queryContext db.QueryContext,
) error {
	if len(spans) != 1 {
		return connect.NewError(connect.CodeInternal, errors.New("expected one span for CosmosDB"))
	}
	objectSchema, err := getTableObjectSchema(ctx, stores, database.InstanceID, database.DatabaseName, queryContext.Container)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.New(err.Error()))
	}
	for pathStr, predicatePath := range spans[0].PredicatePaths {
		semanticType := getFirstSemanticTypeInPath(predicatePath, objectSchema)
		if semanticType != "" {
			for _, result := range results {
				result.Error = fmt.Sprintf("using path %q tagged by semantic type %q in WHERE clause is not allowed", pathStr, semanticType)
				result.Rows = nil
				result.RowsCount = 0
			}
			return nil
		}
	}
	if objectSchema == nil {
		return nil
	}
	for _, result := range results {
		for _, row := range result.Rows {
			if len(row.Values) != 1 {
				continue
			}
			value := row.Values[0].GetStringValue()
			if value == "" {
				continue
			}
			doc := make(map[string]any)
			if err := json.Unmarshal([]byte(value), &doc); err != nil {
				// VALUE query: result is a scalar, not a JSON object.
				masked, maskErr := maskCosmosDBScalarValue(spans[0], value, objectSchema, semanticTypeToMaskerMap)
				if maskErr != nil {
					return connect.NewError(connect.CodeInternal, errors.Errorf("failed to mask scalar value: %v", maskErr))
				}
				row.Values[0] = &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{StringValue: masked},
				}
				continue
			}
			maskedDoc, err := maskCosmosDB(spans[0], doc, objectSchema, semanticTypeToMaskerMap)
			if err != nil {
				return connect.NewError(connect.CodeInternal, errors.Errorf("failed to mask document: %v", err))
			}
			maskedValue, err := json.Marshal(maskedDoc)
			if err != nil {
				return connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal masked document: %v", err))
			}
			row.Values[0] = &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: string(maskedValue),
				},
			}
		}
	}
	return nil
}

func maskCosmosDB(span *parserbase.QuerySpan, data map[string]any, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (map[string]any, error) {
	if len(span.Results) != 1 {
		return nil, errors.Errorf("expected 1 result, but got %d", len(span.Results))
	}
	return walkAndMaskJSON(data, span.Results[0].SourceFieldPaths, objectSchema, semanticTypeToMasker)
}

// maskCosmosDBScalarValue handles VALUE query results where the JSON is a scalar
// (string, number, bool) rather than an object. It checks if any projected source
// field path resolves to a sensitive semantic type and masks the raw JSON accordingly.
func maskCosmosDBScalarValue(span *parserbase.QuerySpan, rawJSON string, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (string, error) {
	if len(span.Results) != 1 {
		return rawJSON, nil
	}
	// Collect all source field paths across all projected fields.
	for _, paths := range span.Results[0].SourceFieldPaths {
		for _, path := range paths {
			ast := stripContainerFromPath(path)
			_, semanticType := getObjectSchemaByPath(objectSchema, ast)
			if semanticType != "" {
				if m, ok := semanticTypeToMasker[semanticType]; ok {
					// Parse the scalar JSON value, mask it, and re-serialize.
					var parsed any
					if unmarshalErr := json.Unmarshal([]byte(rawJSON), &parsed); unmarshalErr != nil {
						// Unparseable JSON: pass through as-is.
						return rawJSON, nil //nolint:nilerr
					}
					masked, err := applyMaskerToJSONMember(parsed, m)
					if err != nil {
						return "", err
					}
					out, err := json.Marshal(masked)
					if err != nil {
						return "", err
					}
					return string(out), nil
				}
			}
		}
	}
	return rawJSON, nil
}

// ---------------------------------------------------------------------------
// MongoDB
// ---------------------------------------------------------------------------

// mongoDBMasker implements documentMasker for MongoDB.
type mongoDBMasker struct{}

func (*mongoDBMasker) maskResults(
	ctx context.Context,
	stores *store.Store,
	database *store.DatabaseMessage,
	spans []*parserbase.QuerySpan,
	results []*v1pb.QueryResult,
	semanticTypeToMaskerMap map[string]masker.Masker,
	_ db.QueryContext,
) error {
	for i, result := range results {
		if i < len(spans) {
			if errMsg, err := checkSensitivePredicates(ctx, stores, database, spans[i]); err != nil {
				return connect.NewError(connect.CodeInternal, err)
			} else if errMsg != "" {
				result.Error = errMsg
				result.Rows = nil
				result.RowsCount = 0
				continue
			}
		}

		var analysis *parserbase.MongoDBAnalysis
		if i < len(spans) {
			analysis = spans[i].MongoDBAnalysis
		}
		if analysis == nil || analysis.Collection == "" {
			continue
		}
		if analysis.API != parserbase.MongoDBMaskableAPIFind && analysis.API != parserbase.MongoDBMaskableAPIFindOne && analysis.API != parserbase.MongoDBMaskableAPIAggregate {
			continue
		}

		objectSchema, err := getTableObjectSchema(ctx, stores, database.InstanceID, database.DatabaseName, analysis.Collection)
		if err != nil {
			return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get object schema for collection %q", analysis.Collection))
		}
		if objectSchema == nil {
			continue
		}

		if len(analysis.JoinedCollections) > 0 {
			var joined []joinedSchema
			for _, jc := range analysis.JoinedCollections {
				js, jsErr := getTableObjectSchema(ctx, stores, database.InstanceID, database.DatabaseName, jc.Collection)
				if jsErr != nil {
					return connect.NewError(connect.CodeInternal, errors.Wrapf(jsErr, "failed to get object schema for joined collection %q", jc.Collection))
				}
				joined = append(joined, joinedSchema{asField: jc.AsField, schema: js})
			}
			objectSchema = injectJoinedSchemas(objectSchema, joined)
		}

		for _, row := range result.Rows {
			if len(row.Values) == 0 {
				continue
			}
			value := row.Values[0].GetStringValue()
			if value == "" {
				continue
			}

			maskedValue, maskErr := maskDocumentString(value, objectSchema, semanticTypeToMaskerMap)
			if maskErr != nil {
				return connect.NewError(connect.CodeInternal, errors.Wrapf(maskErr, "failed to mask MongoDB response"))
			}

			row.Values[0] = &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: maskedValue,
				},
			}
		}
	}
	return nil
}

func checkMongoDBRequestBlocked(analysis *parserbase.MongoDBAnalysis) error {
	if analysis == nil {
		return nil
	}
	if analysis.API == parserbase.MongoDBMaskableAPIUnsupportedRead {
		operation := analysis.Operation
		if operation == "" {
			operation = "unknown"
		}
		if analysis.UnsupportedStage != "" {
			return errors.Errorf("MongoDB aggregate() with stage %q on collection %q is not supported for dynamic masking. Supported operations are find(), findOne(), and aggregate() with shape-preserving stages only", analysis.UnsupportedStage, analysis.Collection)
		}
		return errors.Errorf("MongoDB operation %q on collection %q is not supported for dynamic masking in this release. Supported operations are find() and findOne()", operation+"()", analysis.Collection)
	}
	return nil
}

// injectJoinedSchemas adds array-of-objects schemas for each $lookup/$graphLookup join
// into the source objectSchema so masking covers the joined field.
// The as field holds an array of joined documents, so the injected schema is ARRAY{item: joinedSchema}.
// If the source schema has no StructKind or the joined collection has no schema, the join is skipped silently.
func injectJoinedSchemas(objectSchema *storepb.ObjectSchema, joins []joinedSchema) *storepb.ObjectSchema {
	if objectSchema == nil || len(joins) == 0 {
		return objectSchema
	}
	structKind := objectSchema.GetStructKind()
	if structKind == nil {
		return objectSchema
	}

	// Clone properties map so we don't mutate the cached schema.
	props := make(map[string]*storepb.ObjectSchema, len(structKind.Properties)+len(joins))
	maps.Copy(props, structKind.Properties)
	for _, j := range joins {
		if j.schema == nil {
			continue
		}
		props[j.asField] = &storepb.ObjectSchema{
			Type: storepb.ObjectSchema_ARRAY,
			Kind: &storepb.ObjectSchema_ArrayKind_{
				ArrayKind: &storepb.ObjectSchema_ArrayKind{
					Kind: j.schema,
				},
			},
		}
	}
	return &storepb.ObjectSchema{
		Type: storepb.ObjectSchema_OBJECT,
		Kind: &storepb.ObjectSchema_StructKind_{
			StructKind: &storepb.ObjectSchema_StructKind{
				Properties: props,
			},
		},
	}
}

// joinedSchema pairs an as-field name with its resolved ObjectSchema.
type joinedSchema struct {
	asField string
	schema  *storepb.ObjectSchema
}

// ---------------------------------------------------------------------------
// Elasticsearch
// ---------------------------------------------------------------------------

// elasticsearchMasker implements documentMasker for Elasticsearch.
type elasticsearchMasker struct{}

func (*elasticsearchMasker) maskResults(
	ctx context.Context,
	stores *store.Store,
	database *store.DatabaseMessage,
	spans []*parserbase.QuerySpan,
	results []*v1pb.QueryResult,
	semanticTypeToMaskerMap map[string]masker.Masker,
	_ db.QueryContext,
) error {
	for i, result := range results {
		if i < len(spans) {
			if errMsg, err := checkSensitivePredicates(ctx, stores, database, spans[i]); err != nil {
				return connect.NewError(connect.CodeInternal, err)
			} else if errMsg != "" {
				result.Error = errMsg
				result.Rows = nil
				result.RowsCount = 0
				continue
			}
		}

		var analysis *parserbase.ElasticsearchAnalysis
		if i < len(spans) {
			analysis = spans[i].ElasticsearchAnalysis
		}
		if analysis == nil {
			continue
		}
		if analysis.API == parserbase.ElasticsearchAPIUnsupported || analysis.API == parserbase.ElasticsearchAPIBlocked {
			continue
		}

		indexName := analysis.Index
		if indexName == "" {
			continue
		}
		objectSchema, err := getTableObjectSchema(ctx, stores, database.InstanceID, database.DatabaseName, indexName)
		if err != nil {
			return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get object schema for index %q", indexName))
		}
		if objectSchema == nil || !objectSchemaHasSemanticTypes(objectSchema) {
			continue
		}

		for _, row := range result.Rows {
			for colIdx, colName := range result.ColumnNames {
				if colIdx >= len(row.Values) {
					break
				}
				value := row.Values[colIdx].GetStringValue()
				if value == "" {
					continue
				}

				var maskedValue string
				var maskErr error

				switch analysis.API {
				case parserbase.ElasticsearchAPIMaskSearch:
					switch colName {
					case "hits":
						maskedValue, maskErr = maskElasticsearchHitsColumn(value, analysis.SortFields, objectSchema, semanticTypeToMaskerMap)
					case "responses":
						maskedValue, maskErr = maskElasticsearchMSearchResponses(value, analysis.SortFields, objectSchema, semanticTypeToMaskerMap)
					default:
						continue
					}
				case parserbase.ElasticsearchAPIMaskGetDoc, parserbase.ElasticsearchAPIMaskExplain:
					if colName == "_source" {
						maskedValue, maskErr = maskDocumentString(value, objectSchema, semanticTypeToMaskerMap)
					} else {
						continue
					}
				case parserbase.ElasticsearchAPIMaskGetSource:
					maskedValue, maskErr = maskElasticsearchGetSourceColumn(colName, value, objectSchema, semanticTypeToMaskerMap)
				case parserbase.ElasticsearchAPIMaskMGet:
					if colName == "docs" {
						maskedValue, maskErr = maskElasticsearchMGetSource(value, objectSchema, semanticTypeToMaskerMap)
					} else {
						continue
					}
				default:
					continue
				}

				if maskErr != nil {
					return connect.NewError(connect.CodeInternal, errors.Wrapf(maskErr, "failed to mask ES response"))
				}

				row.Values[colIdx] = &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: maskedValue,
					},
				}
			}
		}
	}
	return nil
}

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
		masked, err := walkAndMaskJSONRecursive(source, objectSchema, semanticTypeToMasker)
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
			masked, err := walkAndMaskJSONRecursive(source, objectSchema, semanticTypeToMasker)
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
func checkElasticsearchRequestBlocked(analysis *parserbase.ElasticsearchAnalysis) error {
	if analysis.API == parserbase.ElasticsearchAPIBlocked {
		return errors.New("this Elasticsearch API is not supported when data masking is configured on the target index")
	}
	if len(analysis.BlockedFeatures) > 0 {
		names := make([]string, 0, len(analysis.BlockedFeatures))
		for _, f := range analysis.BlockedFeatures {
			names = append(names, parserbase.ElasticsearchBlockedFeatureNames[f])
		}
		return errors.Errorf("the following features are not supported when data masking is configured: %s", strings.Join(names, ", "))
	}
	return nil
}

// getTableObjectSchema retrieves the ObjectSchema for a table/collection/index/container
// by name from the database config. This is the shared implementation for all document
// databases (CosmosDB containers, MongoDB collections, Elasticsearch indices).
func getTableObjectSchema(ctx context.Context, stores *store.Store, instanceID, databaseName, tableName string) (*storepb.ObjectSchema, error) {
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

	for _, schema := range dbMetadata.GetConfig().GetSchemas() {
		for _, table := range schema.GetTables() {
			if table.GetName() == tableName {
				return table.GetObjectSchema(), nil
			}
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
