package tests

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (ctl *controller) adminQuery(ctx context.Context, instance *v1pb.Instance, databaseName, query string) ([]*v1pb.QueryResult, error) {
	c, err := ctl.sqlServiceClient.AdminExecute(ctx)
	if err != nil {
		return nil, err
	}
	if err := c.Send(&v1pb.AdminExecuteRequest{
		Name:               instance.Name,
		ConnectionDatabase: databaseName,
		Statement:          query,
	}); err != nil {
		return nil, err
	}
	resp, err := c.Recv()
	if err != nil {
		return nil, err
	}
	if err := c.CloseSend(); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

// GetSQLReviewResult will wait for next task SQL review task check to finish and return the task check result.
func (ctl *controller) GetSQLReviewResult(ctx context.Context, plan *v1pb.Plan) (*v1pb.PlanCheckRun, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		resp, err := ctl.rolloutServiceClient.ListPlanCheckRuns(ctx, &v1pb.ListPlanCheckRunsRequest{
			Parent: plan.Name,
		})
		if err != nil {
			return nil, err
		}
		for _, check := range resp.PlanCheckRuns {
			if check.Status == v1pb.PlanCheckRun_DONE && check.Type == v1pb.PlanCheckRun_DATABASE_STATEMENT_ADVISE {
				return check, nil
			}
		}
	}
	return nil, nil
}

func prodTemplateSQLReviewPolicyForPostgreSQL() (*v1pb.SQLReviewPolicy, error) {
	policy := &v1pb.SQLReviewPolicy{
		Name: "Prod",
		Rules: []*v1pb.SQLReviewRule{
			// Naming
			{
				Type:   string(advisor.SchemaRuleTableNaming),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleColumnNaming),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleIDXNaming),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRulePKNaming),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleUKNaming),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleFKNaming),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			// Statement
			{
				Type:   string(advisor.SchemaRuleStatementNoSelectAll),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleStatementRequireWhere),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleStatementNoLeadingWildcardLike),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleStatementDisallowCommit),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleStatementDisallowOrderBy),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleStatementMergeAlterTable),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleStatementInsertDisallowOrderByRand),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			// TABLE
			{
				Type:   string(advisor.SchemaRuleTableRequirePK),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleTableNoFK),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleTableDropNamingConvention),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleTableCommentConvention),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleTableDisallowPartition),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			// COLUMN
			{
				Type:   string(advisor.SchemaRuleRequiredColumn),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleColumnNotNull),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleColumnDisallowChangeType),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleColumnDisallowChange),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleColumnDisallowChangingOrder),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleColumnCommentConvention),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleColumnAutoIncrementMustInteger),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleColumnTypeDisallowList),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleColumnDisallowSetCharset),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleColumnMaximumCharacterLength),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleColumnAutoIncrementInitialValue),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleColumnAutoIncrementMustUnsigned),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleCurrentTimeColumnCountLimit),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			// SCHEMA
			{
				Type:   string(advisor.SchemaRuleSchemaBackwardCompatibility),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			// DATABASE
			{
				Type:   string(advisor.SchemaRuleDropEmptyDatabase),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_POSTGRES,
			},
			// INDEX
			{
				Type:   string(advisor.SchemaRuleIndexNoDuplicateColumn),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleIndexKeyNumberLimit),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleIndexPKTypeLimit),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleIndexTypeNoBlob),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleIndexTotalNumberLimit),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			// SYSTEM
			{
				Type:   string(advisor.SchemaRuleCharsetAllowlist),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
			{
				Type:   string(advisor.SchemaRuleCollationAllowlist),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_POSTGRES,
			},
		},
	}

	for _, rule := range policy.Rules {
		payload, err := advisor.SetDefaultSQLReviewRulePayload(advisor.SQLReviewRuleType(rule.Type), db.Postgres)
		if err != nil {
			return nil, err
		}
		rule.Payload = payload
	}
	return policy, nil
}

func prodTemplateSQLReviewPolicyForMySQL() (*v1pb.SQLReviewPolicy, error) {
	policy := &v1pb.SQLReviewPolicy{
		Name: "Prod",
		Rules: []*v1pb.SQLReviewRule{
			// Engine
			{
				Type:   string(advisor.SchemaRuleMySQLEngine),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			// Naming
			{
				Type:   string(advisor.SchemaRuleTableNaming),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnNaming),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleIDXNaming),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRulePKNaming),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleUKNaming),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleFKNaming),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleAutoIncrementColumnNaming),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			// Statement
			{
				Type:   string(advisor.SchemaRuleStatementNoSelectAll),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleStatementRequireWhere),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleStatementNoLeadingWildcardLike),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleStatementDisallowCommit),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleStatementDisallowLimit),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleStatementDisallowOrderBy),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleStatementMergeAlterTable),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleStatementInsertRowLimit),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleStatementInsertMustSpecifyColumn),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleStatementInsertDisallowOrderByRand),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleStatementAffectedRowLimit),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleStatementDMLDryRun),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			// TABLE
			{
				Type:   string(advisor.SchemaRuleTableRequirePK),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleTableNoFK),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleTableDropNamingConvention),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleTableCommentConvention),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleTableDisallowPartition),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			// COLUMN
			{
				Type:   string(advisor.SchemaRuleRequiredColumn),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnNotNull),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnDisallowChangeType),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnSetDefaultForNotNull),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnDisallowChange),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnDisallowChangingOrder),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnCommentConvention),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnAutoIncrementMustInteger),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnTypeDisallowList),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnDisallowSetCharset),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnMaximumCharacterLength),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnAutoIncrementInitialValue),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnAutoIncrementMustUnsigned),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleCurrentTimeColumnCountLimit),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleColumnRequireDefault),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			// SCHEMA
			{
				Type:   string(advisor.SchemaRuleSchemaBackwardCompatibility),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			// DATABASE
			{
				Type:   string(advisor.SchemaRuleDropEmptyDatabase),
				Level:  v1pb.SQLReviewRuleLevel_ERROR,
				Engine: v1pb.Engine_MYSQL,
			},
			// INDEX
			{
				Type:   string(advisor.SchemaRuleIndexNoDuplicateColumn),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleIndexKeyNumberLimit),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleIndexPKTypeLimit),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleIndexTypeNoBlob),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleIndexTotalNumberLimit),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			// SYSTEM
			{
				Type:   string(advisor.SchemaRuleCharsetAllowlist),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
			{
				Type:   string(advisor.SchemaRuleCollationAllowlist),
				Level:  v1pb.SQLReviewRuleLevel_WARNING,
				Engine: v1pb.Engine_MYSQL,
			},
		},
	}

	for _, rule := range policy.Rules {
		payload, err := advisor.SetDefaultSQLReviewRulePayload(advisor.SQLReviewRuleType(rule.Type), db.Postgres)
		if err != nil {
			return nil, err
		}
		rule.Payload = payload
	}
	return policy, nil
}

type schemaDiffRequest struct {
	EngineType   parser.EngineType `json:"engineType"`
	SourceSchema string            `json:"sourceSchema"`
	TargetSchema string            `json:"targetSchema"`
}

// getSchemaDiff gets the schema diff.
func (ctl *controller) getSchemaDiff(schemaDiff schemaDiffRequest) (string, error) {
	buf, err := json.Marshal(&schemaDiff)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal schemaDiffRequest")
	}

	body, err := ctl.postOpenAPI("/sql/schema/diff", strings.NewReader(string(buf)))
	if err != nil {
		return "", err
	}

	diff, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	diffString := ""
	if err := json.Unmarshal(diff, &diffString); err != nil {
		return "", err
	}
	return diffString, nil
}
