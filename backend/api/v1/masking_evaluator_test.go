package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/store"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestEvalMaskingLevelOfColumn(t *testing.T) {
	defaultDatabaseMessage := &store.DatabaseMessage{
		EffectiveEnvironmentID: "prod",
		ProjectID:              "bytebase",
		InstanceID:             "neon-host",
		DatabaseName:           "bb",
	}

	defaultProjectDatabaseDataClassificationID := "2b599739-41da-4c35-a9ff-4a73c6cfe32c"

	defaultClassification := &storepb.DataClassificationSetting{
		Configs: []*storepb.DataClassificationSetting_DataClassificationConfig{
			{
				Id: defaultProjectDatabaseDataClassificationID,
				Levels: []*storepb.DataClassificationSetting_DataClassificationConfig_Level{
					{
						Id: "S1",
					},
					{
						Id: "S2",
					},
				},
				Classification: map[string]*storepb.DataClassificationSetting_DataClassificationConfig_DataClassification{
					"1-1-1": {
						Id:    "1-1-1",
						Title: "personal",
						LevelId: func() *string {
							a := "S2"
							return &a
						}(),
					},
				},
			},
		},
	}
	fullAlgorithm := &storepb.Algorithm{
		Mask: &storepb.Algorithm_FullMask_{FullMask: &storepb.Algorithm_FullMask{Substitution: "******"}},
	}
	hashAlgorithm := &storepb.Algorithm{
		Mask: &storepb.Algorithm_Md5Mask{
			Md5Mask: &storepb.Algorithm_MD5Mask{Salt: "123"},
		},
	}
	defaultSemanticType := &storepb.SemanticTypeSetting{
		Types: []*storepb.SemanticTypeSetting_SemanticType{
			{
				Id:        "default",
				Algorithm: fullAlgorithm,
			},
			{
				Id:        "salary-amount",
				Algorithm: hashAlgorithm,
			},
		},
	}

	testCases := []struct {
		description                             string
		databaseMessage                         *store.DatabaseMessage
		databaseProjectDatabaseClassificationID string
		schemaName                              string
		tableName                               string
		columnName                              string
		columnCatalog                           *storepb.ColumnCatalog
		maskingRulePolicy                       *storepb.MaskingRulePolicy
		filteredMaskingExceptions               []*storepb.MaskingExceptionPolicy_MaskingException
		dataClassification                      *storepb.DataClassificationSetting

		want *storepb.Algorithm
	}{
		{
			description:     "Follow The Global Masking Rule",
			databaseMessage: defaultDatabaseMessage,
			schemaName:      "hiring",
			tableName:       "employees",
			columnName:      "salary",
			columnCatalog: &storepb.ColumnCatalog{
				ClassificationId: "1-1-1",
			},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{
					{
						// Classification hit.
						Condition:    &expr.Expr{Expression: `(table_name == "no_table") || (classification_level == "S2")`},
						SemanticType: "default",
					},
				},
			},
			filteredMaskingExceptions:               []*storepb.MaskingExceptionPolicy_MaskingException{},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want: fullAlgorithm,
		},
		{
			description:     "Respect The Exception",
			databaseMessage: defaultDatabaseMessage,
			schemaName:      "hiring",
			tableName:       "employees",
			columnName:      "salary",
			columnCatalog: &storepb.ColumnCatalog{
				ClassificationId: "1-1-1",
			},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{
					{
						// Classification hit.
						Condition:    &expr.Expr{Expression: `(table_name == "no_table") || (classification_level == "S2")`},
						SemanticType: "default",
					},
				},
			},
			filteredMaskingExceptions: []*storepb.MaskingExceptionPolicy_MaskingException{
				{
					Action: storepb.MaskingExceptionPolicy_MaskingException_QUERY,
					Condition: &expr.Expr{
						Expression: `(resource.instance_id == "neon-host") && (resource.database_name == "bb") && (resource.schema_name == "hiring") && (resource.table_name == "employees") && (resource.column_name == "salary")`,
					},
					Member: "users/1234",
				},
			},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want: nil,
		},
		{
			description:     "Column Catalog",
			databaseMessage: defaultDatabaseMessage,
			schemaName:      "hiring",
			tableName:       "employees",
			columnName:      "salary",
			columnCatalog: &storepb.ColumnCatalog{
				SemanticTypeId: "salary-amount",
			},
			maskingRulePolicy:                       &storepb.MaskingRulePolicy{},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want: hashAlgorithm,
		},
	}

	a := require.New(t)

	for _, tc := range testCases {
		m := newEmptyMaskingLevelEvaluator().withMaskingRulePolicy(tc.maskingRulePolicy).withDataClassificationSetting(tc.dataClassification).withSemanticTypeSetting(defaultSemanticType)
		result, err := m.evaluateMaskingAlgorithmOfColumn(tc.databaseMessage, tc.schemaName, tc.tableName, tc.columnName, tc.databaseProjectDatabaseClassificationID, tc.columnCatalog, tc.filteredMaskingExceptions)
		a.NoError(err, tc.description)
		a.Equal(tc.want, result, tc.description)
	}
}
