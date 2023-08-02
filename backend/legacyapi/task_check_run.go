package api

import (
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorDB "github.com/bytebase/bytebase/backend/plugin/advisor/db"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TaskCheckRunStatus is the status of a task check run.
type TaskCheckRunStatus string

const (
	// TaskCheckRunUnknown is the task check run status for UNKNOWN.
	TaskCheckRunUnknown TaskCheckRunStatus = "UNKNOWN"
	// TaskCheckRunRunning is the task check run status for RUNNING.
	TaskCheckRunRunning TaskCheckRunStatus = "RUNNING"
	// TaskCheckRunDone is the task check run status for DONE.
	TaskCheckRunDone TaskCheckRunStatus = "DONE"
	// TaskCheckRunFailed is the task check run status for FAILED.
	TaskCheckRunFailed TaskCheckRunStatus = "FAILED"
	// TaskCheckRunCanceled is the task check run status for CANCELED.
	TaskCheckRunCanceled TaskCheckRunStatus = "CANCELED"
)

// TaskCheckStatus is the status of a task check.
type TaskCheckStatus string

const (
	// TaskCheckStatusSuccess is the task check status for SUCCESS.
	TaskCheckStatusSuccess TaskCheckStatus = "SUCCESS"
	// TaskCheckStatusWarn is the task check status for WARN.
	TaskCheckStatusWarn TaskCheckStatus = "WARN"
	// TaskCheckStatusError is the task check status for ERROR.
	TaskCheckStatusError TaskCheckStatus = "ERROR"
)

func (t TaskCheckStatus) level() int {
	switch t {
	case TaskCheckStatusSuccess:
		return 2
	case TaskCheckStatusWarn:
		return 1
	case TaskCheckStatusError:
		return 0
	}
	return -1
}

// LessThan helps judge if a task check status doesn't meet the minimum requirement.
// For example, ERROR is LessThan WARN.
func (t TaskCheckStatus) LessThan(r TaskCheckStatus) bool {
	return t.level() < r.level()
}

// TaskCheckType is the type of a taskCheck.
type TaskCheckType string

const (
	// TaskCheckDatabaseStatementFakeAdvise is the task check type for fake advise.
	TaskCheckDatabaseStatementFakeAdvise TaskCheckType = "bb.task-check.database.statement.fake-advise"
	// TaskCheckDatabaseStatementCompatibility is the task check type for statement compatibility.
	TaskCheckDatabaseStatementCompatibility TaskCheckType = "bb.task-check.database.statement.compatibility"
	// TaskCheckDatabaseStatementAdvise is the task check type for schema system review policy.
	TaskCheckDatabaseStatementAdvise TaskCheckType = "bb.task-check.database.statement.advise"
	// TaskCheckDatabaseStatementType is the task check type for statement type.
	TaskCheckDatabaseStatementType TaskCheckType = "bb.task-check.database.statement.type"
	// TaskCheckDatabaseStatementTypeReport is the task check type for statement type report.
	TaskCheckDatabaseStatementTypeReport TaskCheckType = "bb.task-check.database.statement.type.report"
	// TaskCheckDatabaseStatementAffectedRowsReport is the task check type for statement affected rows.
	TaskCheckDatabaseStatementAffectedRowsReport TaskCheckType = "bb.task-check.database.statement.affected-rows.report"
	// TaskCheckDatabaseConnect is the task check type for database connection.
	TaskCheckDatabaseConnect TaskCheckType = "bb.task-check.database.connect"
	// TaskCheckGhostSync is the task check type for the gh-ost sync task.
	TaskCheckGhostSync TaskCheckType = "bb.task-check.database.ghost.sync"
	// TaskCheckPITRMySQL is the task check type for MySQL PITR.
	TaskCheckPITRMySQL TaskCheckType = "bb.task-check.pitr.mysql"
)

// Namespace is the namespace for task check result.
type Namespace string

const (
	// AdvisorNamespace is task check result namespace for advisor.
	AdvisorNamespace Namespace = "bb.advisor"
	// BBNamespace is task check result namespace for bytebase.
	BBNamespace Namespace = "bb.core"
)

// TaskCheckResult is the result of task checks.
type TaskCheckResult struct {
	Namespace        Namespace       `json:"namespace,omitempty"`
	Code             int             `json:"code,omitempty"`
	Status           TaskCheckStatus `json:"status,omitempty"`
	Title            string          `json:"title,omitempty"`
	Content          string          `json:"content,omitempty"`
	Line             int             `json:"line,omitempty"`
	Column           int             `json:"column"`
	Details          string          `json:"details,omitempty"`
	ChangedResources string          `json:"changedResources,omitempty"`
}

// TaskCheckRunResultPayload is the result payload of a task check run.
type TaskCheckRunResultPayload struct {
	Detail     string            `json:"detail,omitempty"`
	ResultList []TaskCheckResult `json:"resultList,omitempty"`
}

// TaskCheckRun is the API message for task check run.
type TaskCheckRun struct {
	ID int `jsonapi:"primary,taskCheckRun"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	TaskID int `jsonapi:"attr,taskId"`

	// Domain specific fields
	Status  TaskCheckRunStatus `jsonapi:"attr,status"`
	Type    TaskCheckType      `jsonapi:"attr,type"`
	Code    common.Code        `jsonapi:"attr,code"`
	Comment string             `jsonapi:"attr,comment"`
	Result  string             `jsonapi:"attr,result"`
	Payload string             `jsonapi:"attr,payload"`
}

// IsSQLReviewSupported checks the engine type if SQL review supports it.
func IsSQLReviewSupported(dbType db.Type) bool {
	if dbType == db.Postgres || dbType == db.MySQL || dbType == db.TiDB || dbType == db.MariaDB || dbType == db.Oracle || dbType == db.OceanBase || dbType == db.Snowflake || dbType == db.MSSQL {
		advisorDB, err := advisorDB.ConvertToAdvisorDBType(string(dbType))
		if err != nil {
			return false
		}

		return advisor.IsSQLReviewSupported(advisorDB)
	}

	return false
}

// IsStatementTypeCheckSupported checks the engine type if statement type check supports it.
func IsStatementTypeCheckSupported(dbType db.Type) bool {
	switch dbType {
	case db.Postgres, db.TiDB, db.MySQL, db.MariaDB, db.OceanBase:
		return true
	default:
		return false
	}
}

// IsTaskCheckReportSupported checks if the task report supports the engine type.
func IsTaskCheckReportSupported(dbType db.Type) bool {
	switch dbType {
	case db.Postgres, db.MySQL, db.MariaDB, db.OceanBase:
		return true
	default:
		return false
	}
}

// IsTaskCheckReportNeededForTaskType checks if the task report is needed for the task type.
func IsTaskCheckReportNeededForTaskType(taskType TaskType) bool {
	switch taskType {
	case TaskDatabaseSchemaUpdate, TaskDatabaseSchemaUpdateSDL, TaskDatabaseSchemaUpdateGhostSync, TaskDatabaseDataUpdate:
		return true
	default:
		return false
	}
}
