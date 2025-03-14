package v1

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/masker"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func getFirstSemanticTypeInPath(ast *base.PathAST, objectSchema *storepb.ObjectSchema) string {
	if ast == nil || ast.Root == nil || objectSchema == nil {
		return ""
	}

	// Skip the first node because it always represents the container.
	astWoutContainer := base.NewPathAST(ast.Root.GetNext())
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
		case *base.ItemSelector:
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
		case *base.ArraySelector:
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
		}

		if os.SemanticType != "" {
			return os.SemanticType
		}
	}

	return ""
}

func maskCosmosDB(span *base.QuerySpan, data any, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (any, error) {
	if len(span.Results) == 1 && len(span.Results[0].SourceFieldPaths) == 0 {
		// SELECT * FROM c
		return walkAndMaskJSON(data, objectSchema, semanticTypeToMasker)
	}
	return nil, errors.New("unsupported statement for CosmosDB masking")
}

func walkAndMaskJSON(data any, objectSchema *storepb.ObjectSchema, semanticTypeToMasker map[string]masker.Masker) (any, error) {
	switch data := data.(type) {
	case map[string]any:
		if objectSchema.SemanticType != "" {
			// If the outer semantic type is found, apply the masker recursively to the object.
			if m, ok := semanticTypeToMasker[objectSchema.SemanticType]; ok {
				maskedData, err := applyMaskerToData(data, m)
				if err != nil {
					return nil, err
				}
				return maskedData, nil
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
					data[key], err = walkAndMaskJSON(value, childObjectSchema, semanticTypeToMasker)
					if err != nil {
						return nil, err
					}
				}
			}
		}
		return data, nil
	case []any:
		if objectSchema.SemanticType != "" {
			// If the outer semantic type is found, apply the masker recursively to the array.
			if m, ok := semanticTypeToMasker[objectSchema.SemanticType]; ok {
				maskedData, err := applyMaskerToData(data, m)
				if err != nil {
					return nil, err
				}
				return maskedData, nil
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
				maskedValue, err := walkAndMaskJSON(value, childObjectSchema, semanticTypeToMasker)
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
	// TODO: Handle NULL, VALUE_VALUE, TIMESTAMP_VALUE, TIMESTAMPTZVALUE.
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
	// TODO: Handle NULL.
	return nil, false
}
