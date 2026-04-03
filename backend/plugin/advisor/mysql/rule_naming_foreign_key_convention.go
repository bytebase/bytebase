package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingFKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_INDEX_FK, &NamingFKConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_NAMING_INDEX_FK, &NamingFKConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_NAMING_INDEX_FK, &NamingFKConventionAdvisor{})
}

// NamingFKConventionAdvisor is the advisor checking for foreign key naming convention.
type NamingFKConventionAdvisor struct {
}

// Check checks for foreign key naming convention.
func (*NamingFKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	rule := &namingFKOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		format:       format,
		maxLength:    maxLength,
		templateList: templateList,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

// fkIndexMetaData is the metadata for foreign key.
type fkIndexMetaData struct {
	indexName string
	tableName string
	metaData  map[string]string
	line      int
}

type namingFKOmniRule struct {
	OmniBaseRule
	format       string
	maxLength    int
	templateList []string
}

func (*namingFKOmniRule) Name() string {
	return "NamingFKConventionRule"
}

func (r *namingFKOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *namingFKOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	var indexDataList []*fkIndexMetaData
	for _, constraint := range n.Constraints {
		if constraint == nil {
			continue
		}
		if metaData := r.handleConstraint(tableName, constraint, r.BaseLine+int(r.LocToLine(constraint.Loc))); metaData != nil {
			indexDataList = append(indexDataList, metaData)
		}
	}
	r.handleIndexList(indexDataList)
}

func (r *namingFKOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	tableName := ""
	if n.Table != nil {
		tableName = n.Table.Name
	}
	var indexDataList []*fkIndexMetaData
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		if cmd.Type == ast.ATAddConstraint && cmd.Constraint != nil {
			if metaData := r.handleConstraint(tableName, cmd.Constraint, r.BaseLine+int(r.LocToLine(n.Loc))); metaData != nil {
				indexDataList = append(indexDataList, metaData)
			}
		}
	}
	r.handleIndexList(indexDataList)
}

func (r *namingFKOmniRule) handleIndexList(indexDataList []*fkIndexMetaData) {
	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(r.format, r.templateList, indexData.metaData)
		if err != nil {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:  r.Level,
				Code:    code.Internal.Int32(),
				Title:   "Internal error for foreign key naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", r.TrimmedStmtText(), err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.NamingFKConventionMismatch.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Foreign key in table `%s` mismatches the naming convention, expect %q but found `%s`", indexData.tableName, regex, indexData.indexName),
				StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
			})
		}
		if r.maxLength > 0 && len(indexData.indexName) > r.maxLength {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.NamingFKConventionMismatch.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Foreign key `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", indexData.indexName, indexData.tableName, r.maxLength),
				StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
			})
		}
	}
}

func (*namingFKOmniRule) handleConstraint(tableName string, constraint *ast.Constraint, line int) *fkIndexMetaData {
	// Focus on foreign key.
	if constraint.Type != ast.ConstrForeignKey {
		return nil
	}

	indexName := constraint.Name

	referencingColumnList := constraint.Columns
	referencedTable := ""
	if constraint.RefTable != nil {
		referencedTable = constraint.RefTable.Name
	}
	referencedColumnList := constraint.RefColumns

	metaData := map[string]string{
		advisor.ReferencingTableNameTemplateToken:  tableName,
		advisor.ReferencingColumnNameTemplateToken: strings.Join(referencingColumnList, "_"),
		advisor.ReferencedTableNameTemplateToken:   referencedTable,
		advisor.ReferencedColumnNameTemplateToken:  strings.Join(referencedColumnList, "_"),
	}
	return &fkIndexMetaData{
		indexName: indexName,
		tableName: tableName,
		metaData:  metaData,
		line:      line,
	}
}
