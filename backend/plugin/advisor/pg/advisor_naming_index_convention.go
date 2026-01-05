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
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_INDEX_IDX, &NamingIndexConventionAdvisor{})
}

// NamingIndexConventionAdvisor is the advisor checking for index naming convention.
type NamingIndexConventionAdvisor struct {
}

// Check checks for index naming convention.
func (*NamingIndexConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	rule := &namingIndexConventionRule{
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

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

type namingIndexConventionRule struct {
	BaseRule

	format           string
	maxLength        int
	templateList     []string
	originalMetadata *model.DatabaseMetadata
}

// Name returns the rule name.
func (*namingIndexConventionRule) Name() string {
	return "naming-index-convention"
}

// OnEnter handles entering a parse tree node.
func (r *namingIndexConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Indexstmt":
		r.handleIndexstmt(ctx)
	case "Renamestmt":
		r.handleRenamestmt(ctx)
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit handles exiting a parse tree node.
func (*namingIndexConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleIndexstmt checks CREATE INDEX statements
func (r *namingIndexConventionRule) handleIndexstmt(ctx antlr.ParserRuleContext) {
	indexstmtCtx, ok := ctx.(*parser.IndexstmtContext)
	if !ok {
		return
	}
	if !isTopLevel(indexstmtCtx.GetParent()) {
		return
	}

	// Check if this is a UNIQUE index - if so, skip it
	if indexstmtCtx.Opt_unique() != nil && indexstmtCtx.Opt_unique().UNIQUE() != nil {
		return
	}

	// Get index name
	indexName := ""
	if indexstmtCtx.Name() != nil {
		indexName = pgparser.NormalizePostgreSQLName(indexstmtCtx.Name())
	}
	if indexName == "" {
		return
	}

	// Get table name
	tableName := ""
	if indexstmtCtx.Relation_expr() != nil && indexstmtCtx.Relation_expr().Qualified_name() != nil {
		tableName = extractTableName(indexstmtCtx.Relation_expr().Qualified_name())
	}
	if tableName == "" {
		return
	}

	// Get column list
	var columnList []string
	if indexstmtCtx.Index_params() != nil {
		allParams := indexstmtCtx.Index_params().AllIndex_elem()
		for _, param := range allParams {
			if param.Colid() != nil {
				colName := pgparser.NormalizePostgreSQLColid(param.Colid())
				columnList = append(columnList, colName)
			}
		}
	}

	r.checkIndexName(indexName, tableName, columnList, indexstmtCtx.GetStart().GetLine())
}

// handleRenamestmt checks ALTER INDEX ... RENAME TO statements
func (r *namingIndexConventionRule) handleRenamestmt(ctx antlr.ParserRuleContext) {
	renamestmtCtx, ok := ctx.(*parser.RenamestmtContext)
	if !ok {
		return
	}
	if !isTopLevel(renamestmtCtx.GetParent()) {
		return
	}

	// Check for ALTER INDEX ... RENAME TO
	if renamestmtCtx.INDEX() != nil && renamestmtCtx.TO() != nil {
		allNames := renamestmtCtx.AllName()
		if len(allNames) < 1 {
			return
		}

		// Get old index name from qualified_name
		var oldIndexName string
		if renamestmtCtx.Qualified_name() != nil {
			parts := pgparser.NormalizePostgreSQLQualifiedName(renamestmtCtx.Qualified_name())
			if len(parts) > 0 {
				oldIndexName = parts[len(parts)-1]
			}
		}

		// Get new index name from the name after TO
		newIndexName := pgparser.NormalizePostgreSQLName(allNames[0])

		// Look up the index in catalog to determine if it's a regular index (not unique, not PK)
		if r.originalMetadata != nil && oldIndexName != "" {
			tableName, index := r.findIndex("", "", oldIndexName)
			if index != nil {
				// Only check if it's a regular index (not unique, not primary)
				if !index.GetProto().GetUnique() && !index.GetProto().GetPrimary() {
					r.checkIndexName(newIndexName, tableName, index.GetProto().GetExpressions(), renamestmtCtx.GetStart().GetLine())
				}
			}
		}
	}
}

func (r *namingIndexConventionRule) checkIndexName(indexName, tableName string, columnList []string, line int) {
	metaData := map[string]string{
		advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
		advisor.TableNameTemplateToken:  tableName,
	}

	regex, err := getTemplateRegexp(r.format, r.templateList, metaData)
	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.Internal.Int32(),
			Title:   "Internal error for index naming convention rule",
			Content: fmt.Sprintf("Failed to compile regex: %v", err),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
		return
	}

	if !regex.MatchString(indexName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NamingIndexConventionMismatch.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("Index in table %q mismatches the naming convention, expect %q but found %q", tableName, regex, indexName),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}

	if r.maxLength > 0 && len(indexName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NamingIndexConventionMismatch.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("Index %q in table %q mismatches the naming convention, its length should be within %d characters", indexName, tableName, r.maxLength),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}

// findIndex returns index found in catalogs, nil if not found.
func (r *namingIndexConventionRule) findIndex(schemaName string, tableName string, indexName string) (string, *model.IndexMetadata) {
	if r.originalMetadata == nil {
		return "", nil
	}
	schema := r.originalMetadata.GetSchemaMetadata(normalizeSchemaName(schemaName))
	if schema == nil {
		return "", nil
	}
	if tableName != "" {
		index := schema.GetTable(tableName).GetIndex(indexName)
		if index != nil {
			return tableName, index
		}
		return "", nil
	}
	// tableName is empty, search all tables
	index := schema.GetIndex(indexName)
	if index != nil {
		return index.GetTableProto().Name, index
	}
	return "", nil
}
