package tests

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
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

// sqlReviewTaskCheckRunFinished will return SQL review task check result for next task.
// If the SQL review task check is not done, return nil, false, nil.
func (*controller) sqlReviewTaskCheckRunFinished(issue *api.Issue) ([]api.TaskCheckResult, bool, error) {
	var result []api.TaskCheckResult
	var latestTs int64
	for _, stage := range issue.Pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status == api.TaskPendingApproval {
				for _, taskCheck := range task.TaskCheckRunList {
					if taskCheck.Type == api.TaskCheckDatabaseStatementAdvise {
						switch taskCheck.Status {
						case api.TaskCheckRunRunning:
							return nil, false, nil
						case api.TaskCheckRunDone:
							// return the latest result
							if latestTs != 0 && latestTs > taskCheck.UpdatedTs {
								continue
							}
							checkResult := &api.TaskCheckRunResultPayload{}
							if err := json.Unmarshal([]byte(taskCheck.Result), checkResult); err != nil {
								return nil, false, err
							}
							result = checkResult.ResultList
						}
					}
				}
				return result, true, nil
			}
		}
	}
	return nil, true, nil
}

// GetSQLReviewResult will wait for next task SQL review task check to finish and return the task check result.
func (ctl *controller) GetSQLReviewResult(id int) ([]api.TaskCheckResult, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		issue, err := ctl.getIssue(id)
		if err != nil {
			return nil, err
		}

		status, err := getNextTaskStatus(issue)
		if err != nil {
			return nil, err
		}

		if status != api.TaskPendingApproval {
			return nil, errors.Errorf("the status of issue %v is not pending approval", id)
		}

		result, yes, err := ctl.sqlReviewTaskCheckRunFinished(issue)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get SQL review result for issue %v", id)
		}
		if yes {
			return result, nil
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
				Type:  string(advisor.SchemaRuleTableNaming),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnNaming),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleIDXNaming),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRulePKNaming),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleUKNaming),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleFKNaming),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			// Statement
			{
				Type:  string(advisor.SchemaRuleStatementNoSelectAll),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleStatementRequireWhere),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleStatementNoLeadingWildcardLike),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleStatementDisallowCommit),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleStatementDisallowOrderBy),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleStatementMergeAlterTable),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleStatementInsertDisallowOrderByRand),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			// TABLE
			{
				Type:  string(advisor.SchemaRuleTableRequirePK),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleTableNoFK),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleTableDropNamingConvention),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleTableCommentConvention),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleTableDisallowPartition),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			// COLUMN
			{
				Type:  string(advisor.SchemaRuleRequiredColumn),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnNotNull),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnDisallowChangeType),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnDisallowChange),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnDisallowChangingOrder),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnCommentConvention),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnAutoIncrementMustInteger),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleColumnTypeDisallowList),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleColumnDisallowSetCharset),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnMaximumCharacterLength),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnAutoIncrementInitialValue),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnAutoIncrementMustUnsigned),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleCurrentTimeColumnCountLimit),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			// SCHEMA
			{
				Type:  string(advisor.SchemaRuleSchemaBackwardCompatibility),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			// DATABASE
			{
				Type:  string(advisor.SchemaRuleDropEmptyDatabase),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			// INDEX
			{
				Type:  string(advisor.SchemaRuleIndexNoDuplicateColumn),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleIndexKeyNumberLimit),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleIndexPKTypeLimit),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleIndexTypeNoBlob),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleIndexTotalNumberLimit),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			// SYSTEM
			{
				Type:  string(advisor.SchemaRuleCharsetAllowlist),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleCollationAllowlist),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
		},
	}

	for _, rule := range policy.Rules {
		payload, err := advisor.SetDefaultSQLReviewRulePayload(advisor.SQLReviewRuleType(rule.Type))
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
				Type:  string(advisor.SchemaRuleMySQLEngine),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			// Naming
			{
				Type:  string(advisor.SchemaRuleTableNaming),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnNaming),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleIDXNaming),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRulePKNaming),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleUKNaming),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleFKNaming),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleAutoIncrementColumnNaming),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			// Statement
			{
				Type:  string(advisor.SchemaRuleStatementNoSelectAll),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleStatementRequireWhere),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleStatementNoLeadingWildcardLike),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleStatementDisallowCommit),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleStatementDisallowLimit),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleStatementDisallowOrderBy),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleStatementMergeAlterTable),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleStatementInsertRowLimit),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleStatementInsertMustSpecifyColumn),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleStatementInsertDisallowOrderByRand),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleStatementAffectedRowLimit),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleStatementDMLDryRun),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			// TABLE
			{
				Type:  string(advisor.SchemaRuleTableRequirePK),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleTableNoFK),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleTableDropNamingConvention),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleTableCommentConvention),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleTableDisallowPartition),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			// COLUMN
			{
				Type:  string(advisor.SchemaRuleRequiredColumn),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnNotNull),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnDisallowChangeType),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnSetDefaultForNotNull),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnDisallowChange),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnDisallowChangingOrder),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnCommentConvention),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnAutoIncrementMustInteger),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleColumnTypeDisallowList),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			{
				Type:  string(advisor.SchemaRuleColumnDisallowSetCharset),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnMaximumCharacterLength),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnAutoIncrementInitialValue),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnAutoIncrementMustUnsigned),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleCurrentTimeColumnCountLimit),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleColumnRequireDefault),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			// SCHEMA
			{
				Type:  string(advisor.SchemaRuleSchemaBackwardCompatibility),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			// DATABASE
			{
				Type:  string(advisor.SchemaRuleDropEmptyDatabase),
				Level: v1pb.SQLReviewRuleLevel_ERROR,
			},
			// INDEX
			{
				Type:  string(advisor.SchemaRuleIndexNoDuplicateColumn),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleIndexKeyNumberLimit),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleIndexPKTypeLimit),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleIndexTypeNoBlob),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleIndexTotalNumberLimit),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			// SYSTEM
			{
				Type:  string(advisor.SchemaRuleCharsetAllowlist),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
			{
				Type:  string(advisor.SchemaRuleCollationAllowlist),
				Level: v1pb.SQLReviewRuleLevel_WARNING,
			},
		},
	}

	for _, rule := range policy.Rules {
		payload, err := advisor.SetDefaultSQLReviewRulePayload(advisor.SQLReviewRuleType(rule.Type))
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
