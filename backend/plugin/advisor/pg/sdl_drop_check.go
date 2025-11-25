package pg

import (
	"fmt"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// CheckSDLDropOperations analyzes MetadataDiff for DROP and ALTER operations
// and returns warnings for each DROP detected and info messages for ALTER (CREATE OR REPLACE) operations.
func CheckSDLDropOperations(diff *schema.MetadataDiff) []*storepb.Advice {
	if diff == nil {
		return nil
	}

	var advices []*storepb.Advice

	// Check schema drops
	for _, schemaDiff := range diff.SchemaChanges {
		if schemaDiff.Action == schema.MetadataDiffActionDrop {
			advices = append(advices, &storepb.Advice{
				Status: storepb.Advice_WARNING,
				Code:   code.SDLDropOperation.Int32(),
				Title:  "DROP operation detected",
				Content: fmt.Sprintf(
					"Dropping schema '%s' will result in data loss.\n\n"+
						"This operation cannot be undone. Please ensure:\n"+
						"- You have a backup of the data\n"+
						"- This change is intentional\n"+
						"- All dependent objects are handled",
					schemaDiff.SchemaName,
				),
			})
		}
	}

	// Check table drops
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionDrop {
			advices = append(advices, &storepb.Advice{
				Status: storepb.Advice_WARNING,
				Code:   code.SDLDropOperation.Int32(),
				Title:  "DROP operation detected",
				Content: fmt.Sprintf(
					"Dropping table '%s.%s' will result in data loss.\n\n"+
						"This operation cannot be undone. Please ensure:\n"+
						"- You have a backup of the data\n"+
						"- This change is intentional\n"+
						"- All dependent objects are handled",
					tableDiff.SchemaName, tableDiff.TableName,
				),
			})
		}

		// Check column drops within table
		for _, colDiff := range tableDiff.ColumnChanges {
			if colDiff.Action == schema.MetadataDiffActionDrop {
				columnName := ""
				if colDiff.OldColumn != nil {
					columnName = colDiff.OldColumn.Name
				}
				advices = append(advices, &storepb.Advice{
					Status: storepb.Advice_WARNING,
					Code:   code.SDLDropOperation.Int32(),
					Title:  "DROP operation detected",
					Content: fmt.Sprintf(
						"Dropping column '%s' from table '%s.%s' will result in data loss.\n\n"+
							"This operation cannot be undone. Please ensure:\n"+
							"- You have a backup of the data\n"+
							"- This change is intentional",
						columnName, tableDiff.SchemaName, tableDiff.TableName,
					),
				})
			}
		}

		// Check constraint drops within table
		for _, fkDiff := range tableDiff.ForeignKeyChanges {
			if fkDiff.Action == schema.MetadataDiffActionDrop {
				fkName := ""
				if fkDiff.OldForeignKey != nil {
					fkName = fkDiff.OldForeignKey.Name
				}
				advices = append(advices, &storepb.Advice{
					Status: storepb.Advice_WARNING,
					Code:   code.SDLDropOperation.Int32(),
					Title:  "DROP operation detected",
					Content: fmt.Sprintf(
						"Dropping foreign key constraint '%s' from table '%s.%s'.\n\n"+
							"This change may affect referential integrity. Please ensure this is intentional.",
						fkName, tableDiff.SchemaName, tableDiff.TableName,
					),
				})
			}
		}

		for _, ckDiff := range tableDiff.CheckConstraintChanges {
			if ckDiff.Action == schema.MetadataDiffActionDrop {
				ckName := ""
				if ckDiff.OldCheckConstraint != nil {
					ckName = ckDiff.OldCheckConstraint.Name
				}
				advices = append(advices, &storepb.Advice{
					Status: storepb.Advice_WARNING,
					Code:   code.SDLDropOperation.Int32(),
					Title:  "DROP operation detected",
					Content: fmt.Sprintf(
						"Dropping check constraint '%s' from table '%s.%s'.\n\n"+
							"This change may allow previously invalid data. Please ensure this is intentional.",
						ckName, tableDiff.SchemaName, tableDiff.TableName,
					),
				})
			}
		}
	}

	// Check view drops
	for _, viewDiff := range diff.ViewChanges {
		if viewDiff.Action == schema.MetadataDiffActionDrop {
			advices = append(advices, &storepb.Advice{
				Status: storepb.Advice_WARNING,
				Code:   code.SDLDropOperation.Int32(),
				Title:  "DROP operation detected",
				Content: fmt.Sprintf(
					"Dropping view '%s.%s' will affect dependent objects.\n\n"+
						"Please ensure:\n"+
						"- All dependent views, functions, and queries are updated\n"+
						"- This change is intentional",
					viewDiff.SchemaName, viewDiff.ViewName,
				),
			})
		}
	}

	// Check materialized view drops
	for _, mvDiff := range diff.MaterializedViewChanges {
		if mvDiff.Action == schema.MetadataDiffActionDrop {
			advices = append(advices, &storepb.Advice{
				Status: storepb.Advice_WARNING,
				Code:   code.SDLDropOperation.Int32(),
				Title:  "DROP operation detected",
				Content: fmt.Sprintf(
					"Dropping materialized view '%s.%s' will result in data loss.\n\n"+
						"This operation cannot be undone. Please ensure:\n"+
						"- You have a backup of the data\n"+
						"- This change is intentional\n"+
						"- All dependent objects are handled",
					mvDiff.SchemaName, mvDiff.MaterializedViewName,
				),
			})
		}
	}

	// Check function drops
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionDrop {
			advices = append(advices, &storepb.Advice{
				Status: storepb.Advice_WARNING,
				Code:   code.SDLDropOperation.Int32(),
				Title:  "DROP operation detected",
				Content: fmt.Sprintf(
					"Dropping function '%s.%s' will affect dependent objects.\n\n"+
						"Please ensure:\n"+
						"- All dependent triggers, views, and functions are updated\n"+
						"- This change is intentional",
					funcDiff.SchemaName, funcDiff.FunctionName,
				),
			})
		}
	}

	// Check procedure drops
	for _, procDiff := range diff.ProcedureChanges {
		if procDiff.Action == schema.MetadataDiffActionDrop {
			advices = append(advices, &storepb.Advice{
				Status: storepb.Advice_WARNING,
				Code:   code.SDLDropOperation.Int32(),
				Title:  "DROP operation detected",
				Content: fmt.Sprintf(
					"Dropping procedure '%s.%s' will affect dependent objects.\n\n"+
						"Please ensure:\n"+
						"- All dependent code and queries are updated\n"+
						"- This change is intentional",
					procDiff.SchemaName, procDiff.ProcedureName,
				),
			})
		}
	}

	// Check sequence drops
	for _, seqDiff := range diff.SequenceChanges {
		if seqDiff.Action == schema.MetadataDiffActionDrop {
			advices = append(advices, &storepb.Advice{
				Status: storepb.Advice_WARNING,
				Code:   code.SDLDropOperation.Int32(),
				Title:  "DROP operation detected",
				Content: fmt.Sprintf(
					"Dropping sequence '%s.%s' may affect auto-increment columns.\n\n"+
						"This operation cannot be undone. Please ensure:\n"+
						"- No columns depend on this sequence\n"+
						"- This change is intentional",
					seqDiff.SchemaName, seqDiff.SequenceName,
				),
			})
		}
	}

	// Check enum type drops
	for _, enumDiff := range diff.EnumTypeChanges {
		if enumDiff.Action == schema.MetadataDiffActionDrop {
			advices = append(advices, &storepb.Advice{
				Status: storepb.Advice_WARNING,
				Code:   code.SDLDropOperation.Int32(),
				Title:  "DROP operation detected",
				Content: fmt.Sprintf(
					"Dropping enum type '%s.%s' will affect columns using this type.\n\n"+
						"This operation cannot be undone. Please ensure:\n"+
						"- No columns use this enum type\n"+
						"- This change is intentional",
					enumDiff.SchemaName, enumDiff.EnumTypeName,
				),
			})
		}
	}

	// Check function ALTER operations (CREATE OR REPLACE)
	for _, funcDiff := range diff.FunctionChanges {
		if funcDiff.Action == schema.MetadataDiffActionAlter {
			advices = append(advices, &storepb.Advice{
				Status: storepb.Advice_WARNING,
				Code:   code.SDLReplaceOperation.Int32(),
				Title:  "CREATE OR REPLACE operation detected",
				Content: fmt.Sprintf(
					"Function '%s.%s' definition will be replaced.\n\n"+
						"The old function logic will be overwritten but dependent objects (triggers, views) will be preserved.\n\n"+
						"Please ensure:\n"+
						"- The new function logic is correct\n"+
						"- Dependent triggers, views, and applications are compatible with the changes\n"+
						"- You have tested the changes in a non-production environment",
					funcDiff.SchemaName, funcDiff.FunctionName,
				),
			})
		}
	}

	// Check procedure ALTER operations (CREATE OR REPLACE)
	for _, procDiff := range diff.ProcedureChanges {
		if procDiff.Action == schema.MetadataDiffActionAlter {
			advices = append(advices, &storepb.Advice{
				Status: storepb.Advice_WARNING,
				Code:   code.SDLReplaceOperation.Int32(),
				Title:  "CREATE OR REPLACE operation detected",
				Content: fmt.Sprintf(
					"Procedure '%s.%s' definition will be replaced.\n\n"+
						"The old procedure logic will be overwritten but dependent objects will be preserved.\n\n"+
						"Please ensure:\n"+
						"- The new procedure logic is correct\n"+
						"- Dependent code and applications are compatible with the changes\n"+
						"- You have tested the changes in a non-production environment",
					procDiff.SchemaName, procDiff.ProcedureName,
				),
			})
		}
	}

	// Check trigger ALTER operations (CREATE OR REPLACE)
	for _, tableDiff := range diff.TableChanges {
		for _, triggerDiff := range tableDiff.TriggerChanges {
			if triggerDiff.Action == schema.MetadataDiffActionAlter {
				advices = append(advices, &storepb.Advice{
					Status: storepb.Advice_WARNING,
					Code:   code.SDLReplaceOperation.Int32(),
					Title:  "CREATE OR REPLACE operation detected",
					Content: fmt.Sprintf(
						"Trigger '%s' on table '%s.%s' will be replaced.\n\n"+
							"The old trigger logic will be overwritten.\n\n"+
							"Please ensure:\n"+
							"- The new trigger logic is correct\n"+
							"- The trigger function reference is valid\n"+
							"- You have tested the changes in a non-production environment",
						triggerDiff.TriggerName, tableDiff.SchemaName, tableDiff.TableName,
					),
				})
			}
		}
	}

	return advices
}
