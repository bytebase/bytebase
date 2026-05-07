package tidb

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/tidb/ast"
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
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_INDEX_FK, &NamingFKConventionAdvisor{})
}

// NamingFKConventionAdvisor is the advisor checking for foreign key naming convention.
type NamingFKConventionAdvisor struct {
}

// Check checks for foreign key naming convention.
func (*NamingFKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

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

	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		var indexDataList []*indexMetaData
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			indexDataList = collectFKCreateTable(ostmt, n)
		case *ast.AlterTableStmt:
			indexDataList = collectFKAlterTable(ostmt, n)
		default:
		}

		for _, indexData := range indexDataList {
			regex, err := getTemplateRegexp(format, templateList, indexData.metaData)
			if err != nil {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  level,
					Code:    code.Internal.Int32(),
					Title:   "Internal error for foreign key naming convention rule",
					Content: fmt.Sprintf("%q meet internal error %q", ostmt.TrimmedText(), err.Error()),
				})
				continue
			}
			if !regex.MatchString(indexData.indexName) {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          code.NamingFKConventionMismatch.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("Foreign key in table `%s` mismatches the naming convention, expect %q but found `%s`", indexData.tableName, regex, indexData.indexName),
					StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
				})
			}
			if maxLength > 0 && len(indexData.indexName) > maxLength {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          code.NamingFKConventionMismatch.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("Foreign key `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", indexData.indexName, indexData.tableName, maxLength),
					StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
				})
			}
		}
	}

	return adviceList, nil
}

func collectFKCreateTable(ostmt OmniStmt, n *ast.CreateTableStmt) []*indexMetaData {
	if n.Table == nil {
		return nil
	}
	tableName := n.Table.Name
	var res []*indexMetaData
	for _, constraint := range n.Constraints {
		if metaData := buildFKMetaData(tableName, constraint, ostmt.AbsoluteLine(constraint.Loc.Start)); metaData != nil {
			res = append(res, metaData)
		}
	}
	return res
}

func collectFKAlterTable(ostmt OmniStmt, n *ast.AlterTableStmt) []*indexMetaData {
	if n.Table == nil {
		return nil
	}
	tableName := n.Table.Name
	stmtLine := ostmt.FirstTokenLine()
	var res []*indexMetaData
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		// FOREIGN KEY definitions land on ATAddConstraint only — no
		// ATAddIndex equivalent (mirroring mysql omni rule).
		if cmd.Type != ast.ATAddConstraint {
			continue
		}
		if metaData := buildFKMetaData(tableName, cmd.Constraint, stmtLine); metaData != nil {
			res = append(res, metaData)
		}
	}
	return res
}

func buildFKMetaData(tableName string, constraint *ast.Constraint, line int) *indexMetaData {
	if constraint == nil || constraint.Type != ast.ConstrForeignKey {
		return nil
	}
	referencingColumnList := constraint.Columns
	referencedTable := ""
	// constraint.RefTable can be nil even on FK constraints in pathological
	// inputs; guard defensively per the mysql omni analog.
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
	return &indexMetaData{
		indexName: constraint.Name,
		tableName: tableName,
		metaData:  metaData,
		line:      line,
	}
}
