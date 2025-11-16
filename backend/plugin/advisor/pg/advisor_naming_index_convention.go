package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleIDXNaming, &NamingIndexConventionAdvisor{})
}

// NamingIndexConventionAdvisor is the advisor checking for index naming convention.
type NamingIndexConventionAdvisor struct {
}

// Check checks for index naming convention.
func (*NamingIndexConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, templateList, maxLength, err := advisor.UnmarshalNamingRulePayloadAsTemplate(advisor.SQLReviewRuleType(checkCtx.Rule.Type), checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	rule := &namingIndexConventionRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		format:        format,
		maxLength:     maxLength,
		templateList:  templateList,
		originCatalog: checkCtx.OriginCatalog,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type namingIndexConventionRule struct {
	BaseRule

	format        string
	maxLength     int
	templateList  []string
	originCatalog *catalog.DatabaseState
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
		if r.originCatalog != nil && oldIndexName != "" {
			tableName, index := r.findIndex("", "", oldIndexName)
			if index != nil {
				// Only check if it's a regular index (not unique, not primary)
				if !index.Unique() && !index.Primary() {
					r.checkIndexName(newIndexName, tableName, index.ExpressionList(), renamestmtCtx.GetStart().GetLine())
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
func (r *namingIndexConventionRule) findIndex(schemaName string, tableName string, indexName string) (string, *catalog.IndexState) {
	if r.originCatalog == nil {
		return "", nil
	}
	return r.originCatalog.GetIndex(normalizeSchemaName(schemaName), tableName, indexName)
}
