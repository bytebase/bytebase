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
	return sqlResultSet.Data, nil
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

func (ctl *controller) adminQuery(instance *api.Instance, databaseName, query string) (string, error) {
	sqlResultSet, err := ctl.adminExecuteSQL(api.SQLExecute{
		InstanceID:   instance.ID,
		DatabaseName: databaseName,
		Statement:    query,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to execute SQL")
	}
	if sqlResultSet.Error != "" {
		return "", errors.Errorf("expect SQL result has no error, got %q", sqlResultSet.Error)
	}
	return sqlResultSet.Data, nil
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
				Type:  advisor.PostgreSQLSchemaRuleTableNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleColumnNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleIDXNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.PostgreSQLSchemaRulePKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleUKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleFKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// Statement
			{
				Type:  advisor.PostgreSQLSchemaRuleStatementNoSelectAll,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleStatementRequireWhere,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleStatementNoLeadingWildcardLike,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleStatementDisallowCommit,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleStatementMergeAlterTable,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleStatementInsertDisallowOrderByRand,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// TABLE
			{
				Type:  advisor.PostgreSQLSchemaRuleTableRequirePK,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleTableNoFK,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleTableDropNamingConvention,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleTableDisallowPartition,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// COLUMN
			{
				Type:  advisor.PostgreSQLSchemaRuleRequiredColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleColumnNotNull,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleColumnDisallowChangeType,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleColumnTypeDisallowList,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleColumnMaximumCharacterLength,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// SCHEMA
			{
				Type:  advisor.PostgreSQLSchemaRuleSchemaBackwardCompatibility,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// INDEX
			{
				Type:  advisor.PostgreSQLSchemaRuleIndexNoDuplicateColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleIndexKeyNumberLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleIndexTotalNumberLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// SYSTEM
			{
				Type:  advisor.PostgreSQLSchemaRuleCharsetAllowlist,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.PostgreSQLSchemaRuleCollationAllowlist,
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
				Type:  advisor.MySQLSchemaRuleMySQLEngine,
				Level: advisor.SchemaRuleLevelError,
			},
			// Naming
			{
				Type:  advisor.MySQLSchemaRuleTableNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleIDXNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleUKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleFKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleAutoIncrementColumnNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// Statement
			{
				Type:  advisor.MySQLSchemaRuleStatementNoSelectAll,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.MySQLSchemaRuleStatementRequireWhere,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.MySQLSchemaRuleStatementNoLeadingWildcardLike,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.MySQLSchemaRuleStatementDisallowCommit,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.MySQLSchemaRuleStatementDisallowLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleStatementDisallowOrderBy,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleStatementMergeAlterTable,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleStatementInsertRowLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleStatementInsertMustSpecifyColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleStatementInsertDisallowOrderByRand,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleStatementAffectedRowLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleStatementDMLDryRun,
				Level: advisor.SchemaRuleLevelError,
			},
			// TABLE
			{
				Type:  advisor.MySQLSchemaRuleTableRequirePK,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.MySQLSchemaRuleTableNoFK,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.MySQLSchemaRuleTableDropNamingConvention,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.MySQLSchemaRuleTableCommentConvention,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleTableDisallowPartition,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// COLUMN
			{
				Type:  advisor.MySQLSchemaRuleRequiredColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnNotNull,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnDisallowChangeType,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnSetDefaultForNotNull,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnDisallowChange,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnDisallowChangingOrder,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnCommentConvention,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnAutoIncrementMustInteger,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnTypeDisallowList,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnDisallowSetCharset,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnMaximumCharacterLength,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnAutoIncrementInitialValue,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnAutoIncrementMustUnsigned,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleCurrentTimeColumnCountLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleColumnRequireDefault,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// SCHEMA
			{
				Type:  advisor.MySQLSchemaRuleSchemaBackwardCompatibility,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// DATABASE
			{
				Type:  advisor.MySQLSchemaRuleDropEmptyDatabase,
				Level: advisor.SchemaRuleLevelError,
			},
			// INDEX
			{
				Type:  advisor.MySQLSchemaRuleIndexNoDuplicateColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleIndexKeyNumberLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleIndexPKTypeLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleIndexTypeNoBlob,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleIndexTotalNumberLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// SYSTEM
			{
				Type:  advisor.MySQLSchemaRuleCharsetAllowlist,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.MySQLSchemaRuleCollationAllowlist,
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
