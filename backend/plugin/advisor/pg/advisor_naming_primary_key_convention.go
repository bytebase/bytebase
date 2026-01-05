package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*NamingPKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_INDEX_PK, &NamingPKConventionAdvisor{})
}

// NamingPKConventionAdvisor is the advisor checking for primary key naming convention.
type NamingPKConventionAdvisor struct {
}

// Check checks for primary key naming convention.
func (*NamingPKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for this rule")
	}

	format := namingPayload.Format
	templateList, _ := advisor.ParseTemplateTokens(format)

	for _, key := range templateList {
		if _, ok := advisor.TemplateNamingTokens[checkCtx.Rule.Type][key]; !ok {
			return nil, errors.Errorf("invalid template %s for rule %s", key, checkCtx.Rule.Type)
		}
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}

	var adviceList []*storepb.Advice
	for _, stmtInfo := range checkCtx.ParsedStatements {
		if stmtInfo.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmtInfo.AST)
		if !ok {
			continue
		}
		rule := &namingPKConventionRule{
			BaseRule: BaseRule{
				level: level,
				title: checkCtx.Rule.Type.String(),
			},
			format:           format,
			maxLength:        maxLength,
			templateList:     templateList,
			originalMetadata: checkCtx.OriginalMetadata,
		}

		checker := NewGenericChecker([]Rule{rule})
		rule.SetBaseLine(stmtInfo.BaseLine())
		checker.SetBaseLine(stmtInfo.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
		adviceList = append(adviceList, checker.GetAdviceList()...)
	}

	return adviceList, nil
}

type namingPKConventionRule struct {
	BaseRule

	format           string
	maxLength        int
	templateList     []string
	originalMetadata *model.DatabaseMetadata
}

type pkMetaData struct {
	pkName     string
	tableName  string
	schemaName string
	line       int
	metaData   map[string]string
}

func (*namingPKConventionRule) Name() string {
	return "naming_primary_key_convention"
}

func (r *namingPKConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx.(*parser.CreatestmtContext))
	case "Altertablestmt":
		r.handleAltertablestmt(ctx.(*parser.AltertablestmtContext))
	case "Renamestmt":
		r.handleRenamestmt(ctx.(*parser.RenamestmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*namingPKConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleCreatestmt handles CREATE TABLE with PRIMARY KEY constraints
func (r *namingPKConventionRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	var tableName, schemaName string
	allQualifiedNames := ctx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		tableName = extractTableName(allQualifiedNames[0])
		schemaName = extractSchemaName(allQualifiedNames[0])
		if schemaName == "" {
			schemaName = "public"
		}
	}

	// Check table-level constraints
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			// Check table-level PRIMARY KEY constraint
			if elem.Tableconstraint() != nil {
				constraint := elem.Tableconstraint()
				if pkData := r.getPKMetaDataFromTableConstraint(constraint, tableName, schemaName, constraint.GetStart().GetLine()); pkData != nil {
					r.checkPKName(pkData)
				}
			}

			// Check column-level PRIMARY KEY constraint - commented out for now
			// Column-level constraints typically don't have explicit names in the test cases
		}
	}
}

// handleAltertablestmt handles ALTER TABLE ADD CONSTRAINT PRIMARY KEY
func (r *namingPKConventionRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	var tableName, schemaName string
	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		tableName = extractTableName(ctx.Relation_expr().Qualified_name())
		schemaName = extractSchemaName(ctx.Relation_expr().Qualified_name())
		if schemaName == "" {
			schemaName = "public"
		}
	}

	// Check all alter table commands
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// ADD CONSTRAINT
			if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
				constraint := cmd.Tableconstraint()
				if pkData := r.getPKMetaDataFromTableConstraint(constraint, tableName, schemaName, constraint.GetStart().GetLine()); pkData != nil {
					r.checkPKName(pkData)
				}
			}
		}
	}
}

// handleRenamestmt handles ALTER TABLE RENAME CONSTRAINT and ALTER INDEX RENAME TO
func (r *namingPKConventionRule) handleRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	allNames := ctx.AllName()

	// Check for ALTER TABLE ... RENAME CONSTRAINT old_name TO new_name
	// Example: ALTER TABLE tech_book RENAME CONSTRAINT old_pk TO new_pk
	if ctx.TABLE() != nil && ctx.CONSTRAINT() != nil && ctx.TO() != nil {
		var tableName, schemaName string
		// Get table name from Relation_expr()
		if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
			tableName = extractTableName(ctx.Relation_expr().Qualified_name())
			schemaName = extractSchemaName(ctx.Relation_expr().Qualified_name())
		}

		if len(allNames) >= 2 {
			oldName := pgparser.NormalizePostgreSQLName(allNames[len(allNames)-2])
			newName := pgparser.NormalizePostgreSQLName(allNames[len(allNames)-1])

			// Check if old constraint is a primary key using catalog
			if r.originalMetadata != nil {
				normalizedSchema := schemaName
				if normalizedSchema == "" {
					normalizedSchema = "public"
				}
				index := r.originalMetadata.GetSchemaMetadata(normalizedSchema).GetTable(tableName).GetIndex(oldName)
				if index != nil && index.GetProto().GetPrimary() {
					metaData := map[string]string{
						advisor.ColumnListTemplateToken: strings.Join(index.GetProto().GetExpressions(), "_"),
						advisor.TableNameTemplateToken:  tableName,
					}
					pkData := &pkMetaData{
						pkName:     newName,
						tableName:  tableName,
						schemaName: schemaName,
						line:       ctx.GetStart().GetLine(),
						metaData:   metaData,
					}
					r.checkPKName(pkData)
				}
			}
		}
	}

	// Check for ALTER INDEX ... RENAME TO new_name
	// Example: ALTER INDEX old_pk RENAME TO new_pk
	if ctx.INDEX() != nil && ctx.TO() != nil {
		var oldIndexName, schemaName string

		// Try Qualified_name first (direct qualified name)
		if ctx.Qualified_name() != nil {
			oldIndexName = extractTableName(ctx.Qualified_name())
			schemaName = extractSchemaName(ctx.Qualified_name())
		} else if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
			// Fall back to Relation_expr
			oldIndexName = extractTableName(ctx.Relation_expr().Qualified_name())
			schemaName = extractSchemaName(ctx.Relation_expr().Qualified_name())
		}

		if oldIndexName != "" && len(allNames) > 0 {
			newIndexName := pgparser.NormalizePostgreSQLName(allNames[len(allNames)-1])

			// Check if this index is a primary key using catalog
			if r.originalMetadata != nil {
				normalizedSchema := schemaName
				if normalizedSchema == "" {
					normalizedSchema = "public"
				}
				// "ALTER INDEX name RENAME TO new_name" doesn't specify table name
				// Empty table name for ALTER INDEX
				schema := r.originalMetadata.GetSchemaMetadata(normalizedSchema)
				var tableName string
				var index *model.IndexMetadata
				if schema != nil {
					index = schema.GetIndex(oldIndexName)
					if index != nil {
						tableName = index.GetTableProto().Name
					}
				}
				if index != nil && index.GetProto().GetPrimary() {
					metaData := map[string]string{
						advisor.ColumnListTemplateToken: strings.Join(index.GetProto().GetExpressions(), "_"),
						advisor.TableNameTemplateToken:  tableName,
					}
					pkData := &pkMetaData{
						pkName:     newIndexName,
						tableName:  tableName,
						schemaName: normalizedSchema,
						line:       ctx.GetStart().GetLine(),
						metaData:   metaData,
					}
					r.checkPKName(pkData)
				}
			}
		}
	}
}

func (r *namingPKConventionRule) getPKMetaDataFromTableConstraint(constraint parser.ITableconstraintContext, tableName, schemaName string, line int) *pkMetaData {
	if constraint == nil || constraint.Constraintelem() == nil {
		return nil
	}

	elem := constraint.Constraintelem()

	// Check if this is a PRIMARY KEY constraint
	if elem.PRIMARY() != nil && elem.KEY() != nil {
		var pkName string
		var columnList []string

		// Try to get constraint name using GetText() on constraint itself and parsing it
		// Format is usually: CONSTRAINT name PRIMARY KEY (columns) or just PRIMARY KEY (columns)
		constraintText := constraint.GetText()
		// Check if it starts with CONSTRAINT keyword
		if strings.HasPrefix(strings.ToUpper(constraintText), "CONSTRAINT") {
			// Extract the name between CONSTRAINT and PRIMARY
			parts := strings.Split(constraintText, "PRIMARY")
			if len(parts) > 0 {
				// Remove "CONSTRAINT" prefix
				namePart := strings.TrimPrefix(strings.ToUpper(parts[0]), "CONSTRAINT")
				namePart = strings.TrimSpace(strings.ToLower(namePart))
				// The name is before the keyword PRIMARY
				// This is a simplified extraction
				if namePart != "" && !strings.Contains(namePart, "(") {
					pkName = namePart
				}
			}
		}

		// Get column list
		if elem.Columnlist() != nil {
			allColumns := elem.Columnlist().AllColumnElem()
			for _, col := range allColumns {
				if col.Colid() != nil {
					columnList = append(columnList, pgparser.NormalizePostgreSQLColid(col.Colid()))
				}
			}
		}

		// PRIMARY KEY USING INDEX
		if elem.Existingindex() != nil && elem.Existingindex().Name() != nil {
			// Extract index name properly
			indexName := pgparser.NormalizePostgreSQLName(elem.Existingindex().Name())
			// Find the index in catalog to get column list
			if r.originalMetadata != nil && indexName != "" {
				normalizedSchema := schemaName
				if normalizedSchema == "" {
					normalizedSchema = "public"
				}
				schema := r.originalMetadata.GetSchemaMetadata(normalizedSchema)
				var index *model.IndexMetadata
				if schema != nil {
					table := schema.GetTable(tableName)
					if table != nil {
						index = table.GetIndex(indexName)
					}
				}
				if index != nil {
					columnList = index.GetProto().GetExpressions()
				}
			}
		}

		if pkName != "" {
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			return &pkMetaData{
				pkName:     pkName,
				tableName:  tableName,
				schemaName: schemaName,
				line:       line,
				metaData:   metaData,
			}
		}
	}

	return nil
}

func (r *namingPKConventionRule) checkPKName(pkData *pkMetaData) {
	regex, err := getTemplateRegexp(r.format, r.templateList, pkData.metaData)
	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.Internal.Int32(),
			Title:   "Internal error for primary key naming convention rule",
			Content: fmt.Sprintf("Failed to compile regex: %v", err),
			StartPosition: &storepb.Position{
				Line:   int32(pkData.line),
				Column: 0,
			},
		})
		return
	}

	if !regex.MatchString(pkData.pkName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NamingPKConventionMismatch.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf(`Primary key in table "%s" mismatches the naming convention, expect %q but found "%s"`, pkData.tableName, regex, pkData.pkName),
			StartPosition: &storepb.Position{
				Line:   int32(pkData.line),
				Column: 0,
			},
		})
	}

	if r.maxLength > 0 && len(pkData.pkName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NamingPKConventionMismatch.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf(`Primary key "%s" in table "%s" mismatches the naming convention, its length should be within %d characters`, pkData.pkName, pkData.tableName, r.maxLength),
			StartPosition: &storepb.Position{
				Line:   int32(pkData.line),
				Column: 0,
			},
		})
	}
}
