package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestCheckSDLDropOperations(t *testing.T) {
	tests := []struct {
		name         string
		diff         *schema.MetadataDiff
		wantWarnings int
		wantMessages []string
	}{
		{
			name:         "No DROP operations",
			diff:         &schema.MetadataDiff{},
			wantWarnings: 0,
		},
		{
			name: "DROP TABLE",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					{
						Action:     schema.MetadataDiffActionDrop,
						SchemaName: "public",
						TableName:  "users",
					},
				},
			},
			wantWarnings: 1,
			wantMessages: []string{"table 'public.users'"},
		},
		{
			name: "DROP COLUMN",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					{
						Action:     schema.MetadataDiffActionAlter,
						SchemaName: "public",
						TableName:  "users",
						ColumnChanges: []*schema.ColumnDiff{
							{
								Action: schema.MetadataDiffActionDrop,
								OldColumn: &storepb.ColumnMetadata{
									Name: "email",
								},
							},
						},
					},
				},
			},
			wantWarnings: 1,
			wantMessages: []string{"column 'email'"},
		},
		{
			name: "Multiple DROPs",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					{
						Action:     schema.MetadataDiffActionDrop,
						SchemaName: "public",
						TableName:  "orders",
					},
				},
				ViewChanges: []*schema.ViewDiff{
					{
						Action:     schema.MetadataDiffActionDrop,
						SchemaName: "public",
						ViewName:   "active_users",
					},
				},
			},
			wantWarnings: 2,
			wantMessages: []string{"table 'public.orders'", "view 'public.active_users'"},
		},
		{
			name: "DROP MATERIALIZED VIEW",
			diff: &schema.MetadataDiff{
				MaterializedViewChanges: []*schema.MaterializedViewDiff{
					{
						Action:               schema.MetadataDiffActionDrop,
						SchemaName:           "public",
						MaterializedViewName: "sales_summary",
					},
				},
			},
			wantWarnings: 1,
			wantMessages: []string{"materialized view 'public.sales_summary'"},
		},
		{
			name: "DROP CONSTRAINT",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					{
						Action:     schema.MetadataDiffActionAlter,
						SchemaName: "public",
						TableName:  "users",
						ForeignKeyChanges: []*schema.ForeignKeyDiff{
							{
								Action: schema.MetadataDiffActionDrop,
								OldForeignKey: &storepb.ForeignKeyMetadata{
									Name: "fk_users_teams",
								},
							},
						},
					},
				},
			},
			wantWarnings: 1,
			wantMessages: []string{"foreign key constraint 'fk_users_teams'"},
		},
		{
			name: "DROP SEQUENCE",
			diff: &schema.MetadataDiff{
				SequenceChanges: []*schema.SequenceDiff{
					{
						Action:       schema.MetadataDiffActionDrop,
						SchemaName:   "public",
						SequenceName: "order_id_seq",
					},
				},
			},
			wantWarnings: 1,
			wantMessages: []string{"sequence 'public.order_id_seq'"},
		},
		{
			name: "DROP FUNCTION",
			diff: &schema.MetadataDiff{
				FunctionChanges: []*schema.FunctionDiff{
					{
						Action:       schema.MetadataDiffActionDrop,
						SchemaName:   "public",
						FunctionName: "calculate_total",
					},
				},
			},
			wantWarnings: 1,
			wantMessages: []string{"function 'public.calculate_total'"},
		},
		{
			name: "DROP PROCEDURE",
			diff: &schema.MetadataDiff{
				ProcedureChanges: []*schema.ProcedureDiff{
					{
						Action:        schema.MetadataDiffActionDrop,
						SchemaName:    "public",
						ProcedureName: "log_message",
					},
				},
			},
			wantWarnings: 1,
			wantMessages: []string{"procedure 'public.log_message'"},
		},
		{
			name: "DROP ENUM TYPE",
			diff: &schema.MetadataDiff{
				EnumTypeChanges: []*schema.EnumTypeDiff{
					{
						Action:       schema.MetadataDiffActionDrop,
						SchemaName:   "public",
						EnumTypeName: "status",
					},
				},
			},
			wantWarnings: 1,
			wantMessages: []string{"enum type 'public.status'"},
		},
		{
			name: "Complex diff with mix of CREATE and DROP",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "public",
						TableName:  "products",
					},
					{
						Action:     schema.MetadataDiffActionDrop,
						SchemaName: "public",
						TableName:  "old_products",
					},
					{
						Action:     schema.MetadataDiffActionAlter,
						SchemaName: "public",
						TableName:  "users",
						ColumnChanges: []*schema.ColumnDiff{
							{
								Action: schema.MetadataDiffActionDrop,
								OldColumn: &storepb.ColumnMetadata{
									Name: "legacy_field",
								},
							},
						},
					},
				},
				ViewChanges: []*schema.ViewDiff{
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "public",
						ViewName:   "active_products",
					},
					{
						Action:     schema.MetadataDiffActionDrop,
						SchemaName: "public",
						ViewName:   "inactive_products",
					},
				},
			},
			wantWarnings: 3,
			wantMessages: []string{"table 'public.old_products'", "column 'legacy_field'", "view 'public.inactive_products'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			advices := CheckSDLDropOperations(tt.diff)

			// Check number of warnings
			warningCount := 0
			for _, advice := range advices {
				if advice.Status == storepb.Advice_WARNING {
					warningCount++
					assert.Equal(t, code.SDLDropOperation.Int32(), advice.Code)
					assert.Equal(t, "DROP operation detected", advice.Title)
				}
			}
			assert.Equal(t, tt.wantWarnings, warningCount,
				"Expected %d warnings, got %d", tt.wantWarnings, warningCount)

			// Check messages mentioned in warnings
			if len(tt.wantMessages) > 0 {
				for i, advice := range advices {
					if i < len(tt.wantMessages) {
						assert.Contains(t, advice.Content, tt.wantMessages[i],
							"Warning should mention '%s'", tt.wantMessages[i])
					}
				}
			}
		})
	}
}

func TestCheckSDLDropOperations_Nil(t *testing.T) {
	advices := CheckSDLDropOperations(nil)
	require.Nil(t, advices)
}

func TestCheckSDLDropOperations_EmptyDiff(t *testing.T) {
	advices := CheckSDLDropOperations(&schema.MetadataDiff{})
	assert.Empty(t, advices)
}

func TestCheckSDLDropOperations_AlterOperations(t *testing.T) {
	tests := []struct {
		name            string
		diff            *schema.MetadataDiff
		wantInfoCount   int
		wantMessages    []string
		wantCodeMatches []int32
	}{
		{
			name: "ALTER FUNCTION (CREATE OR REPLACE)",
			diff: &schema.MetadataDiff{
				FunctionChanges: []*schema.FunctionDiff{
					{
						Action:       schema.MetadataDiffActionAlter,
						SchemaName:   "public",
						FunctionName: "calculate_total",
					},
				},
			},
			wantInfoCount:   1,
			wantMessages:    []string{"Function 'public.calculate_total'"},
			wantCodeMatches: []int32{code.SDLReplaceOperation.Int32()},
		},
		{
			name: "ALTER PROCEDURE (CREATE OR REPLACE)",
			diff: &schema.MetadataDiff{
				ProcedureChanges: []*schema.ProcedureDiff{
					{
						Action:        schema.MetadataDiffActionAlter,
						SchemaName:    "public",
						ProcedureName: "process_orders",
					},
				},
			},
			wantInfoCount:   1,
			wantMessages:    []string{"Procedure 'public.process_orders'"},
			wantCodeMatches: []int32{code.SDLReplaceOperation.Int32()},
		},
		{
			name: "ALTER TRIGGER (CREATE OR REPLACE)",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					{
						Action:     schema.MetadataDiffActionAlter,
						SchemaName: "public",
						TableName:  "users",
						TriggerChanges: []*schema.TriggerDiff{
							{
								Action:      schema.MetadataDiffActionAlter,
								TriggerName: "update_timestamp",
							},
						},
					},
				},
			},
			wantInfoCount:   1,
			wantMessages:    []string{"Trigger 'update_timestamp'"},
			wantCodeMatches: []int32{code.SDLReplaceOperation.Int32()},
		},
		{
			name: "Multiple ALTER operations",
			diff: &schema.MetadataDiff{
				FunctionChanges: []*schema.FunctionDiff{
					{
						Action:       schema.MetadataDiffActionAlter,
						SchemaName:   "public",
						FunctionName: "func1",
					},
					{
						Action:       schema.MetadataDiffActionAlter,
						SchemaName:   "public",
						FunctionName: "func2",
					},
				},
				ProcedureChanges: []*schema.ProcedureDiff{
					{
						Action:        schema.MetadataDiffActionAlter,
						SchemaName:    "public",
						ProcedureName: "proc1",
					},
				},
			},
			wantInfoCount:   3,
			wantMessages:    []string{"Function 'public.func1'", "Function 'public.func2'", "Procedure 'public.proc1'"},
			wantCodeMatches: []int32{code.SDLReplaceOperation.Int32(), code.SDLReplaceOperation.Int32(), code.SDLReplaceOperation.Int32()},
		},
		{
			name: "Mix of DROP and ALTER operations",
			diff: &schema.MetadataDiff{
				FunctionChanges: []*schema.FunctionDiff{
					{
						Action:       schema.MetadataDiffActionDrop,
						SchemaName:   "public",
						FunctionName: "old_func",
					},
					{
						Action:       schema.MetadataDiffActionAlter,
						SchemaName:   "public",
						FunctionName: "modified_func",
					},
				},
			},
			wantInfoCount: 2, // 1 DROP (WARNING) + 1 ALTER (SUCCESS with info message)
			wantMessages:  []string{"function 'public.old_func'", "Function 'public.modified_func'"},
			wantCodeMatches: []int32{
				code.SDLDropOperation.Int32(),
				code.SDLReplaceOperation.Int32(),
			},
		},
		{
			name: "ALTER with CREATE - no info (CREATE doesn't trigger ALTER check)",
			diff: &schema.MetadataDiff{
				FunctionChanges: []*schema.FunctionDiff{
					{
						Action:       schema.MetadataDiffActionCreate,
						SchemaName:   "public",
						FunctionName: "new_func",
					},
				},
			},
			wantInfoCount: 0,
			wantMessages:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			advices := CheckSDLDropOperations(tt.diff)

			// Check total count
			assert.Equal(t, tt.wantInfoCount, len(advices),
				"Expected %d advices, got %d", tt.wantInfoCount, len(advices))

			// Check messages are present
			for i, expectedMsg := range tt.wantMessages {
				if i < len(advices) {
					assert.Contains(t, advices[i].Content, expectedMsg,
						"Advice %d should mention '%s'", i, expectedMsg)
				}
			}

			// Check codes match
			for i, expectedCode := range tt.wantCodeMatches {
				if i < len(advices) {
					assert.Equal(t, expectedCode, advices[i].Code,
						"Advice %d should have code %d", i, expectedCode)
				}
			}

			// For ALTER operations specifically, verify they use WARNING status
			for _, advice := range advices {
				if advice.Code == code.SDLReplaceOperation.Int32() {
					assert.Equal(t, storepb.Advice_WARNING, advice.Status,
						"ALTER operations should use WARNING status")
					assert.Equal(t, "CREATE OR REPLACE operation detected", advice.Title,
						"ALTER operations should have correct title")
				}
			}
		})
	}
}

func TestCheckSDLDropOperations_OnlyAlterWarnings(t *testing.T) {
	// Test that ALTER operations generate WARNINGs with appropriate messaging
	diff := &schema.MetadataDiff{
		FunctionChanges: []*schema.FunctionDiff{
			{
				Action:       schema.MetadataDiffActionAlter,
				SchemaName:   "public",
				FunctionName: "test_func",
			},
		},
	}

	advices := CheckSDLDropOperations(diff)

	require.Len(t, advices, 1)
	assert.Equal(t, storepb.Advice_WARNING, advices[0].Status,
		"ALTER operations should generate WARNINGs")
	assert.Equal(t, code.SDLReplaceOperation.Int32(), advices[0].Code)
	assert.Contains(t, advices[0].Content, "will be replaced")
	assert.Contains(t, advices[0].Content, "dependent objects")
}
