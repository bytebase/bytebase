package redshift

import (
	"strings"

	omniredshift "github.com/bytebase/omni/redshift"
	redshiftast "github.com/bytebase/omni/redshift/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_REDSHIFT, extractChangedResources)
}

func extractChangedResources(database string, schema string, dbMetadata *model.DatabaseMetadata, _ []base.AST, statement string) (*base.ChangeSummary, error) {
	omniSummary, err := omniredshift.ExtractChangedResources(statement, database, schema)
	if err != nil {
		return nil, err
	}

	changedResources := model.NewChangedResources(dbMetadata)
	for _, table := range omniSummary.Tables {
		changedResources.AddTable(
			table.Database,
			table.Schema,
			&storepb.ChangedResourceTable{
				Name: table.Name,
			},
			table.Affected,
		)
	}

	sampleDMLs, err := extractOmniSampleDMLs(statement)
	if err != nil {
		return nil, err
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS:       sampleDMLs,
		DMLCount:         omniSummary.DMLCount,
		InsertCount:      omniSummary.InsertCount,
	}, nil
}

func extractOmniSampleDMLs(statement string) ([]string, error) {
	stmts, err := ParseRedshiftOmni(statement)
	if err != nil {
		return nil, err
	}

	var sampleDMLs []string
	for _, stmt := range stmts {
		if len(sampleDMLs) >= common.MaximumLintExplainSize {
			break
		}
		switch stmt.AST.(type) {
		case *redshiftast.InsertStmt, *redshiftast.UpdateStmt, *redshiftast.DeleteStmt, *redshiftast.MergeStmt:
			sampleDMLs = append(sampleDMLs, strings.TrimSpace(stmt.Text))
		default:
		}
	}
	return sampleDMLs, nil
}
