package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	expr "google.golang.org/genproto/googleapis/type/expr"

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

	testCases := []struct {
		description                             string
		databaseMessage                         *store.DatabaseMessage
		databaseProjectDatabaseClassificationID string
		schemaName                              string
		tableName                               string
		columnName                              string
		columnClassification                    string
		maskingPolicyMap                        map[maskingPolicyKey]*storepb.MaskData
		maskingRulePolicy                       *storepb.MaskingRulePolicy
		filteredMaskingExceptions               []*storepb.MaskingExceptionPolicy_MaskingException
		dataClassification                      *storepb.DataClassificationSetting

		want storepb.MaskingLevel
	}{
		{
			description:          "Follow The Global Masking Rule If Column Masking Policy Is Default",
			databaseMessage:      defaultDatabaseMessage,
			schemaName:           "hiring",
			tableName:            "employees",
			columnName:           "salary",
			columnClassification: "1-1-1",
			maskingPolicyMap:     map[maskingPolicyKey]*storepb.MaskData{},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{
					{
						// Classification hit.
						Condition:    &expr.Expr{Expression: `(table_name == "no_table") || (classification_level == "S2")`},
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
				},
			},
			filteredMaskingExceptions:               []*storepb.MaskingExceptionPolicy_MaskingException{},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want: storepb.MaskingLevel_FULL,
		},
		{
			description:          "Follow The Global Masking Rule If Column Masking Policy Is Default And Respect The Exception",
			databaseMessage:      defaultDatabaseMessage,
			schemaName:           "hiring",
			tableName:            "employees",
			columnName:           "salary",
			columnClassification: "1-1-1",
			maskingPolicyMap:     map[maskingPolicyKey]*storepb.MaskData{},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{
					{
						// Classification hit.
						Condition:    &expr.Expr{Expression: `(table_name == "no_table") || (classification_level == "S2")`},
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
				},
			},
			filteredMaskingExceptions: []*storepb.MaskingExceptionPolicy_MaskingException{
				{
					Action: storepb.MaskingExceptionPolicy_MaskingException_QUERY,
					Condition: &expr.Expr{
						Expression: `(resource.instance_id == "neon-host") && (resource.database_name == "bb") && (resource.schema_name == "hiring") && (resource.table_name == "employees") && (resource.column_name == "salary")`,
					},
					Member:       "zp@bytebase.com",
					MaskingLevel: storepb.MaskingLevel_PARTIAL,
				},
			},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want: storepb.MaskingLevel_PARTIAL,
		},
		{
			description:          "Only Find The Lower Level in Exception",
			databaseMessage:      defaultDatabaseMessage,
			schemaName:           "hiring",
			tableName:            "employees",
			columnName:           "salary",
			columnClassification: "1-1-1",
			maskingPolicyMap:     map[maskingPolicyKey]*storepb.MaskData{},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{
					{
						// Classification hit.
						Condition:    &expr.Expr{Expression: `(table_name == "no_table") || (classification_level == "S2")`},
						MaskingLevel: storepb.MaskingLevel_PARTIAL,
					},
				},
			},
			filteredMaskingExceptions: []*storepb.MaskingExceptionPolicy_MaskingException{
				{
					// Hit, but MaskingLevel_FULL > MaskingLevel_PARTIAL, do not replace the rule.
					Action: storepb.MaskingExceptionPolicy_MaskingException_QUERY,
					Condition: &expr.Expr{
						Expression: `(resource.instance_id == "neon-host") && (resource.database_name == "bb") && (resource.schema_name == "hiring") && (resource.table_name == "employees") && (resource.column_name == "salary")`,
					},
					Member:       "zp@bytebase.com",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
			},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want: storepb.MaskingLevel_PARTIAL,
		},
		{
			description:          "Respect The Column Masking Policy",
			databaseMessage:      defaultDatabaseMessage,
			schemaName:           "hiring",
			tableName:            "employees",
			columnName:           "salary",
			columnClassification: "1-1-1",
			maskingPolicyMap: map[maskingPolicyKey]*storepb.MaskData{
				{
					schema: "hiring",
					table:  "employees",
					column: "salary",
				}: {
					Schema:       "hiring",
					Table:        "employees",
					Column:       "salary",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
			},
			maskingRulePolicy: &storepb.MaskingRulePolicy{},
			filteredMaskingExceptions: []*storepb.MaskingExceptionPolicy_MaskingException{
				{
					// Hit, and MaskingLevel_PARTIAL < MaskingLevel_FULL.
					Action: storepb.MaskingExceptionPolicy_MaskingException_QUERY,
					Condition: &expr.Expr{
						Expression: `(resource.instance_id == "neon-host") && (resource.database_name == "bb") && (resource.schema_name == "hiring") && (resource.table_name == "employees") && (resource.column_name == "salary")`,
					},
					Member:       "zp@bytebase.com",
					MaskingLevel: storepb.MaskingLevel_PARTIAL,
				},
			},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want: storepb.MaskingLevel_PARTIAL,
		},
	}

	a := require.New(t)

	for _, tc := range testCases {
		m := newEmptyMaskingLevelEvaluator().withMaskingRulePolicy(tc.maskingRulePolicy).withDataClassificationSetting(tc.dataClassification)
		result, err := m.evaluateMaskingLevelOfColumn(tc.databaseMessage, tc.schemaName, tc.tableName, tc.columnName, tc.columnClassification, tc.databaseProjectDatabaseClassificationID, tc.maskingPolicyMap, tc.filteredMaskingExceptions)
		a.NoError(err, tc.description)
		a.Equal(tc.want, result, tc.description)
	}
}
