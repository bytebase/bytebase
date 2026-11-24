// Package redshift implements the SQL advisor rules for Redshift.
package redshift

import (
	redshiftv1review "github.com/bytebase/omni/redshift/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	parserredshift "github.com/bytebase/bytebase/backend/plugin/parser/redshift"
)

func init() {
	e := advisor.NewV1Engine(advisor.NodeExtractor(parserredshift.GetNode), nil, advisor.V1PosContentStart)

	// V1 anchored every finding on where the splitter starts the statement,
	// which can include leading whitespace or comments before the node.
	advisor.RegisterRules(storepb.Engine_REDSHIFT, map[storepb.SQLReviewRule_Type]advisor.Advisor{
		storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION: e.NamingRule(
			redshiftv1review.TableDropNaming,
			code.TableDropNamingConventionMismatch,
			"naming_payload is required for table drop naming convention rule",
			"drop table",
			false,
			advisor.BacktickName,
			e.PosStmtStart,
		),
	})
}
