package mssql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*MigrationCompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, &MigrationCompatibilityAdvisor{})
}

// MigrationCompatibilityAdvisor is the advisor checking for migration compatibility.
type MigrationCompatibilityAdvisor struct {
}

// Check checks for migration compatibility.
func (*MigrationCompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &migrationCompatibilityOmniRule{
		OmniBaseRule:    OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
		currentDatabase: checkCtx.CurrentDatabase,
		newTables:       make(map[string]any),
		newSchemas:      make(map[string]any),
		newDatabases:    make(map[string]any),
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type migrationCompatibilityOmniRule struct {
	OmniBaseRule

	currentDatabase string
	newTables       map[string]any
	newSchemas      map[string]any
	newDatabases    map[string]any
}

func (*migrationCompatibilityOmniRule) Name() string {
	return "MigrationCompatibilityOmniRule"
}

func (r *migrationCompatibilityOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.handleCreateTable(n)
	case *ast.CreateSchemaStmt:
		r.handleCreateSchema(n)
	case *ast.CreateDatabaseStmt:
		r.handleCreateDatabase(n)
	case *ast.DropStmt:
		r.handleDrop(n)
	case *ast.AlterTableStmt:
		r.handleAlterTable(n)
	case *ast.ExecStmt:
		r.handleExec(n)
	default:
	}
}

func (r *migrationCompatibilityOmniRule) handleCreateTable(n *ast.CreateTableStmt) {
	if n.Name == nil {
		return
	}
	norm := normalizeTableRef(n.Name, r.currentDatabase, "dbo")
	r.newTables[norm] = nil
}

func (r *migrationCompatibilityOmniRule) handleCreateSchema(n *ast.CreateSchemaStmt) {
	schemaName := strings.ToLower(n.Name)
	if schemaName == "" {
		schemaName = strings.ToLower(n.Authorization)
	}
	norm := fmt.Sprintf("%s.%s", r.currentDatabase, schemaName)
	r.newSchemas[norm] = nil
}

func (r *migrationCompatibilityOmniRule) handleCreateDatabase(n *ast.CreateDatabaseStmt) {
	r.newDatabases[strings.ToLower(n.Name)] = nil
}

func (r *migrationCompatibilityOmniRule) handleDrop(n *ast.DropStmt) {
	switch n.ObjectType {
	case ast.DropTable:
		if n.Names == nil {
			return
		}
		for _, item := range n.Names.Items {
			ref, ok := item.(*ast.TableRef)
			if !ok {
				continue
			}
			norm := normalizeTableRef(ref, r.currentDatabase, "dbo")
			if _, ok := r.newTables[norm]; !ok {
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          advisorcode.CompatibilityDropSchema.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("Drop table %s may cause incompatibility with the existing data and code", norm),
					StartPosition: &storepb.Position{Line: r.LocToLine(n.Loc)},
				})
			}
			delete(r.newTables, norm)
		}
	case ast.DropSchema:
		if n.Names == nil {
			return
		}
		for _, item := range n.Names.Items {
			ref, ok := item.(*ast.TableRef)
			if !ok {
				continue
			}
			schemaName := strings.ToLower(ref.Object)
			norm := fmt.Sprintf("%s.%s", r.currentDatabase, schemaName)
			if _, ok := r.newSchemas[norm]; !ok {
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          advisorcode.CompatibilityDropSchema.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("Drop schema %s may cause incompatibility with the existing data and code", norm),
					StartPosition: &storepb.Position{Line: r.LocToLine(n.Loc)},
				})
			}
			delete(r.newSchemas, norm)
		}
	case ast.DropDatabase:
		if n.Names == nil {
			return
		}
		for _, item := range n.Names.Items {
			ref, ok := item.(*ast.TableRef)
			if !ok {
				continue
			}
			dbName := strings.ToLower(ref.Object)
			if _, ok := r.newDatabases[dbName]; !ok {
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          advisorcode.CompatibilityDropSchema.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("Drop database %s may cause incompatibility with the existing data and code", dbName),
					StartPosition: &storepb.Position{Line: r.LocToLine(n.Loc)},
				})
			}
			delete(r.newDatabases, dbName)
		}
	default:
	}
}

func (r *migrationCompatibilityOmniRule) handleAlterTable(n *ast.AlterTableStmt) {
	if n.Name == nil {
		return
	}
	norm := normalizeTableRef(n.Name, r.currentDatabase, "dbo")
	if _, ok := r.newTables[norm]; ok {
		return
	}

	if n.Actions == nil {
		return
	}
	for _, item := range n.Actions.Items {
		action, ok := item.(*ast.AlterTableAction)
		if !ok {
			continue
		}

		switch action.Type {
		case ast.ATDropColumn:
			colName := strings.ToLower(action.ColName)
			// Also handle multiple column drops via Names list.
			var colNames []string
			if colName != "" {
				colNames = append(colNames, colName)
			}
			if action.Names != nil {
				for _, nameItem := range action.Names.Items {
					if ref, ok := nameItem.(*ast.TableRef); ok {
						colNames = append(colNames, strings.ToLower(ref.Object))
					}
				}
			}
			if len(colNames) > 0 {
				placeholder := strings.Join(colNames, ", ")
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          advisorcode.CompatibilityDropSchema.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("Drop column %s may cause incompatibility with the existing data and code", placeholder),
					StartPosition: &storepb.Position{Line: r.LocToLine(action.Loc)},
				})
			}
		case ast.ATAlterColumn:
			colName := ""
			if action.Column != nil {
				colName = strings.ToLower(action.Column.Name)
			} else if action.ColName != "" {
				colName = strings.ToLower(action.ColName)
			}
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          advisorcode.CompatibilityAlterColumn.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Alter COLUMN %s may cause incompatibility with the existing data and code", colName),
				StartPosition: &storepb.Position{Line: r.LocToLine(action.Loc)},
			})
		case ast.ATAddConstraint:
			if action.Constraint == nil {
				continue
			}
			// WITH NOCHECK variant — the omni parser may use ATAddConstraint
			// with WithCheck="NOCHECK" instead of ATNocheckConstraint.
			if strings.EqualFold(action.WithCheck, "NOCHECK") {
				switch action.Constraint.Type {
				case ast.ConstraintForeignKey:
					r.AddAdvice(&storepb.Advice{
						Status:        r.Level,
						Code:          advisorcode.CompatibilityAddForeignKey.Int32(),
						Title:         r.Title,
						Content:       "Add FOREIGN KEY WITH NO CHECK may cause incompatibility with the existing data and code",
						StartPosition: &storepb.Position{Line: r.LocToLine(action.Loc)},
					})
				case ast.ConstraintCheck:
					r.AddAdvice(&storepb.Advice{
						Status:        r.Level,
						Code:          advisorcode.CompatibilityAddForeignKey.Int32(),
						Title:         r.Title,
						Content:       "Add CHECK WITH NO CHECK may cause incompatibility with the existing data and code",
						StartPosition: &storepb.Position{Line: r.LocToLine(action.Loc)},
					})
				default:
				}
				continue
			}
			// WITH CHECK ADD is safe — skip it.
			if strings.EqualFold(action.WithCheck, "CHECK") {
				continue
			}
			c := advisorcode.Ok
			operation := ""
			switch action.Constraint.Type {
			case ast.ConstraintPrimaryKey:
				c = advisorcode.CompatibilityAddPrimaryKey
				operation = "Add PRIMARY KEY"
			case ast.ConstraintUnique:
				c = advisorcode.CompatibilityAddUniqueKey
				operation = "Add UNIQUE KEY"
			case ast.ConstraintCheck:
				c = advisorcode.CompatibilityAddCheck
				operation = "Add CHECK"
			case ast.ConstraintForeignKey:
				c = advisorcode.CompatibilityAddForeignKey
				operation = "Add FOREIGN KEY"
			default:
				continue
			}
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          c.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("%s may cause incompatibility with the existing data and code", operation),
				StartPosition: &storepb.Position{Line: r.LocToLine(n.Loc)},
			})
		case ast.ATNocheckConstraint:
			// Legacy fallback — omni may still use this in some cases.
			if action.Constraint != nil {
				switch action.Constraint.Type {
				case ast.ConstraintForeignKey:
					r.AddAdvice(&storepb.Advice{
						Status:        r.Level,
						Code:          advisorcode.CompatibilityAddForeignKey.Int32(),
						Title:         r.Title,
						Content:       "Add FOREIGN KEY WITH NO CHECK may cause incompatibility with the existing data and code",
						StartPosition: &storepb.Position{Line: r.LocToLine(action.Loc)},
					})
				case ast.ConstraintCheck:
					r.AddAdvice(&storepb.Advice{
						Status:        r.Level,
						Code:          advisorcode.CompatibilityAddForeignKey.Int32(),
						Title:         r.Title,
						Content:       "Add CHECK WITH NO CHECK may cause incompatibility with the existing data and code",
						StartPosition: &storepb.Position{Line: r.LocToLine(action.Loc)},
					})
				default:
				}
			}
		default:
		}
	}
}

func (r *migrationCompatibilityOmniRule) handleExec(n *ast.ExecStmt) {
	if n.Name == nil {
		return
	}
	if n.Name.Schema != "" {
		return
	}
	if strings.ToLower(n.Name.Object) != "sp_rename" {
		return
	}
	// Check that there is at least one argument.
	if n.Args == nil || n.Args.Len() == 0 {
		return
	}
	firstArg, ok := n.Args.Items[0].(*ast.ExecArg)
	if !ok {
		return
	}
	// Check that the first argument is a string literal.
	if lit, ok := firstArg.Value.(*ast.Literal); ok && lit.Type == ast.LitString {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          advisorcode.CompatibilityRenameTable.Int32(),
			Title:         r.Title,
			Content:       "sp_rename may cause incompatibility with the existing data and code, and break scripts and stored procedures.",
			StartPosition: &storepb.Position{Line: r.LocToLine(n.Loc)},
		})
	}
}
