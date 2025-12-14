package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/store"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestEvalMaskingLevelOfColumn(t *testing.T) {
	environment := "prod"
	defaultDatabaseMessage := &store.DatabaseMessage{
		EffectiveEnvironmentID: &environment,
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
		filteredMaskingExemptions               []*storepb.MaskingExemptionPolicy_Exemption
		dataClassification                      *storepb.DataClassificationSetting

		want           string
		wantAlgorithm  string
		wantContext    string
		wantClassLevel string
	}{
		{
			description:     "Follow The Global Masking Rule",
			databaseMessage: defaultDatabaseMessage,
			schemaName:      "hiring",
			tableName:       "employees",
			columnName:      "salary",
			columnCatalog: &storepb.ColumnCatalog{
				Classification: "1-1-1",
			},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{
					{
						// Classification hit.
						Condition:    &expr.Expr{Expression: `(resource.table_name == "no_table") || (resource.classification_level == "S2")`},
						SemanticType: "default",
					},
				},
			},
			filteredMaskingExemptions:               []*storepb.MaskingExemptionPolicy_Exemption{},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want:           "default",
			wantAlgorithm:  "Full mask",
			wantClassLevel: "S2",
		},
		{
			description:     "Respect The Exception",
			databaseMessage: defaultDatabaseMessage,
			schemaName:      "hiring",
			tableName:       "employees",
			columnName:      "salary",
			columnCatalog: &storepb.ColumnCatalog{
				Classification: "1-1-1",
			},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{
					{
						// Classification hit.
						Condition:    &expr.Expr{Expression: `(resource.table_name == "no_table") || (resource.classification_level == "S2")`},
						SemanticType: "default",
					},
				},
			},
			filteredMaskingExemptions: []*storepb.MaskingExemptionPolicy_Exemption{
				{
					Condition: &expr.Expr{
						Expression: `(resource.instance_id == "neon-host") && (resource.database_name == "bb") && (resource.schema_name == "hiring") && (resource.table_name == "employees") && (resource.column_name == "salary")`,
					},
					Members: []string{"users/1234"},
				},
			},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want: "",
		},
		{
			description:     "Column Catalog",
			databaseMessage: defaultDatabaseMessage,
			schemaName:      "hiring",
			tableName:       "employees",
			columnName:      "salary",
			columnCatalog: &storepb.ColumnCatalog{
				SemanticType: "salary-amount",
			},
			maskingRulePolicy:                       &storepb.MaskingRulePolicy{},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want:          "salary-amount",
			wantAlgorithm: "Hash (MD5)",
		},
	}

	a := require.New(t)

	for _, tc := range testCases {
		m := newEmptyMaskingLevelEvaluator().withMaskingRulePolicy(tc.maskingRulePolicy).withDataClassificationSetting(tc.dataClassification).withSemanticTypeSetting(defaultSemanticType)
		result, err := m.evaluateSemanticTypeOfColumn(tc.databaseMessage, tc.schemaName, tc.tableName, tc.columnName, tc.databaseProjectDatabaseClassificationID, tc.columnCatalog, tc.filteredMaskingExemptions)
		a.NoError(err, tc.description)
		if tc.want == "" {
			a.Nil(result, tc.description)
		} else {
			a.NotNil(result, tc.description)
			a.Equal(tc.want, result.SemanticTypeID, tc.description)
			if tc.wantAlgorithm != "" {
				a.Equal(tc.wantAlgorithm, result.Algorithm, tc.description)
			}
			if tc.wantContext != "" {
				a.Equal(tc.wantContext, result.Context, tc.description)
			}
			if tc.wantClassLevel != "" {
				a.Equal(tc.wantClassLevel, result.ClassificationLevel, tc.description)
			}
		}
	}
}
