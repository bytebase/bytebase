package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser"
)

// executeSQL executes a SQL query on the database.
func (ctl *controller) executeSQL(sqlExecute api.SQLExecute) (*api.SQLResultSet, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &sqlExecute); err != nil {
		return nil, errors.Wrap(err, "failed to marshal sqlExecute")
	}

	body, err := ctl.post("/sql/execute", buf)
	if err != nil {
		return nil, err
	}

	sqlResultSet := new(api.SQLResultSet)
	if err = jsonapi.UnmarshalPayload(body, sqlResultSet); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal sqlResultSet response")
	}
	return sqlResultSet, nil
}

func (ctl *controller) query(instance *api.Instance, databaseName, query string) (string, error) {
	sqlResultSet, err := ctl.executeSQL(api.SQLExecute{
		InstanceID:   instance.ID,
		DatabaseName: databaseName,
		Statement:    query,
		Readonly:     true,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to execute SQL")
	}
	if sqlResultSet.Error != "" {
		return "", errors.Errorf("expect SQL result has no error, got %q", sqlResultSet.Error)
	}
	// TODO(zp): optimize here
	return sqlResultSet.SingleSQLResultList[0].Data, nil
}

// adminExecuteSQL executes a SQL query on the database.
func (ctl *controller) adminExecuteSQL(sqlExecute api.SQLExecute) (*api.SQLResultSet, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &sqlExecute); err != nil {
		return nil, errors.Wrap(err, "failed to marshal sqlExecute")
	}

	body, err := ctl.post("/sql/execute/admin", buf)
	if err != nil {
		return nil, err
	}

	sqlResultSet := new(api.SQLResultSet)
	if err = jsonapi.UnmarshalPayload(body, sqlResultSet); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal sqlResultSet response")
	}
	return sqlResultSet, nil
}

func (ctl *controller) adminQuery(instance *api.Instance, databaseName, query string) ([]api.SingleSQLResult, error) {
	sqlResultSet, err := ctl.adminExecuteSQL(api.SQLExecute{
		InstanceID:   instance.ID,
		DatabaseName: databaseName,
		Statement:    query,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute SQL")
	}
	if sqlResultSet.Error != "" {
		return nil, errors.Errorf("expect SQL result has no error, got %q", sqlResultSet.Error)
	}
	return sqlResultSet.SingleSQLResultList, nil
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

func prodTemplateSQLReviewPolicyForPostgreSQL() (string, error) {
	policy := advisor.SQLReviewPolicy{
		Name: "Prod",
		RuleList: []*advisor.SQLReviewRule{
			// Naming
			{
				Type:  advisor.SchemaRuleTableNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIDXNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRulePKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleUKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleFKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// Statement
			{
				Type:  advisor.SchemaRuleStatementNoSelectAll,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleStatementRequireWhere,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleStatementNoLeadingWildcardLike,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleStatementDisallowCommit,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleStatementDisallowOrderBy,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementMergeAlterTable,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementInsertDisallowOrderByRand,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// TABLE
			{
				Type:  advisor.SchemaRuleTableRequirePK,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleTableNoFK,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleTableDropNamingConvention,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleTableCommentConvention,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleTableDisallowPartition,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// COLUMN
			{
				Type:  advisor.SchemaRuleRequiredColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnNotNull,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnDisallowChangeType,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnDisallowChange,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnDisallowChangingOrder,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnCommentConvention,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnAutoIncrementMustInteger,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleColumnTypeDisallowList,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleColumnDisallowSetCharset,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnMaximumCharacterLength,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnAutoIncrementInitialValue,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnAutoIncrementMustUnsigned,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleCurrentTimeColumnCountLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// SCHEMA
			{
				Type:  advisor.SchemaRuleSchemaBackwardCompatibility,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// DATABASE
			{
				Type:  advisor.SchemaRuleDropEmptyDatabase,
				Level: advisor.SchemaRuleLevelError,
			},
			// INDEX
			{
				Type:  advisor.SchemaRuleIndexNoDuplicateColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIndexKeyNumberLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIndexPKTypeLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIndexTypeNoBlob,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIndexTotalNumberLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// SYSTEM
			{
				Type:  advisor.SchemaRuleCharsetAllowlist,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleCollationAllowlist,
				Level: advisor.SchemaRuleLevelWarning,
			},
		},
	}

	return templateSQLReviewPolicy(policy)
}

func prodTemplateSQLReviewPolicyForMySQL() (string, error) {
	policy := advisor.SQLReviewPolicy{
		Name: "Prod",
		RuleList: []*advisor.SQLReviewRule{
			// Engine
			{
				Type:  advisor.SchemaRuleMySQLEngine,
				Level: advisor.SchemaRuleLevelError,
			},
			// Naming
			{
				Type:  advisor.SchemaRuleTableNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIDXNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRulePKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleUKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleFKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleAutoIncrementColumnNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// Statement
			{
				Type:  advisor.SchemaRuleStatementNoSelectAll,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleStatementRequireWhere,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleStatementNoLeadingWildcardLike,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleStatementDisallowCommit,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleStatementDisallowLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementDisallowOrderBy,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementMergeAlterTable,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementInsertRowLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementInsertMustSpecifyColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementInsertDisallowOrderByRand,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementAffectedRowLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementDMLDryRun,
				Level: advisor.SchemaRuleLevelError,
			},
			// TABLE
			{
				Type:  advisor.SchemaRuleTableRequirePK,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleTableNoFK,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleTableDropNamingConvention,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleTableCommentConvention,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleTableDisallowPartition,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// COLUMN
			{
				Type:  advisor.SchemaRuleRequiredColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnNotNull,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnDisallowChangeType,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnSetDefaultForNotNull,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnDisallowChange,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnDisallowChangingOrder,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnCommentConvention,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnAutoIncrementMustInteger,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleColumnTypeDisallowList,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleColumnDisallowSetCharset,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnMaximumCharacterLength,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnAutoIncrementInitialValue,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnAutoIncrementMustUnsigned,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleCurrentTimeColumnCountLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnRequireDefault,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// SCHEMA
			{
				Type:  advisor.SchemaRuleSchemaBackwardCompatibility,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// DATABASE
			{
				Type:  advisor.SchemaRuleDropEmptyDatabase,
				Level: advisor.SchemaRuleLevelError,
			},
			// INDEX
			{
				Type:  advisor.SchemaRuleIndexNoDuplicateColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIndexKeyNumberLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIndexPKTypeLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIndexTypeNoBlob,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIndexTotalNumberLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// SYSTEM
			{
				Type:  advisor.SchemaRuleCharsetAllowlist,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleCollationAllowlist,
				Level: advisor.SchemaRuleLevelWarning,
			},
		},
	}

	return templateSQLReviewPolicy(policy)
}

// templateSQLReviewPolicy returns the default SQL review policy.
func templateSQLReviewPolicy(policy advisor.SQLReviewPolicy) (string, error) {
	for _, rule := range policy.RuleList {
		payload, err := advisor.SetDefaultSQLReviewRulePayload(rule.Type)
		if err != nil {
			return "", err
		}
		rule.Payload = payload
	}

	s, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}
	return string(s), nil
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
