package common

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// nolint:revive
const (
	ProjectNamePrefix            = "projects/"
	EnvironmentNamePrefix        = "environments/"
	InstanceNamePrefix           = "instances/"
	PolicyNamePrefix             = "policies/"
	DatabaseIDPrefix             = "databases/"
	InstanceRolePrefix           = "roles/"
	UserNamePrefix               = "users/"
	IdentityProviderNamePrefix   = "idps/"
	SettingNamePrefix            = "settings/"
	BackupPrefix                 = "backups/"
	BookmarkPrefix               = "bookmarks/"
	ExternalVersionControlPrefix = "externalVersionControls/"
	RiskPrefix                   = "risks/"
	IssuePrefix                  = "issues/"
	RolloutPrefix                = "rollouts/"
	StagePrefix                  = "stages/"
	TaskPrefix                   = "tasks/"
	TaskRunPrefix                = "taskRuns/"
	PlanPrefix                   = "plans/"
	PlanCheckRunPrefix           = "planCheckRuns/"
	RolePrefix                   = "roles/"
	SecretNamePrefix             = "secrets/"
	WebhookIDPrefix              = "webhooks/"
	SheetIDPrefix                = "sheets/"
	DatabaseGroupNamePrefix      = "databaseGroups/"
	SchemaGroupNamePrefix        = "schemaGroups/"
	ChangeHistoryPrefix          = "changeHistories/"
	IssueNamePrefix              = "issues/"
	PipelineNamePrefix           = "pipelines/"
	LogNamePrefix                = "logs/"
	InboxNamePrefix              = "inbox/"
	SchemaDesignPrefix           = "schemaDesigns/"

	DeploymentConfigSuffix = "/deploymentConfig"
	BackupSettingSuffix    = "/backupSetting"
	SchemaSuffix           = "/schema"
	MetadataSuffix         = "/metadata"
	GitOpsInfoSuffix       = "/gitOpsInfo"
)

// GetProjectID returns the project ID from a resource name.
func GetProjectID(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

// GetProjectIDDatabaseGroupID returns the project ID and database group ID from a resource name.
func GetProjectIDDatabaseGroupID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, DatabaseGroupNamePrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

// GetProjectIDDatabaseGroupIDSchemaGroupID returns the project ID, database group ID, and schema group ID from a resource name.
func GetProjectIDDatabaseGroupIDSchemaGroupID(name string) (string, string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, DatabaseGroupNamePrefix, SchemaGroupNamePrefix)
	if err != nil {
		return "", "", "", err
	}
	return tokens[0], tokens[1], tokens[2], nil
}

// GetProjectIDWebhookID returns the project ID and webhook ID from a resource name.
func GetProjectIDWebhookID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, WebhookIDPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

// GetUIDFromName returns the UID from a resource name.
func GetUIDFromName(name, prefix string) (int, error) {
	tokens, err := GetNameParentTokens(name, prefix)
	if err != nil {
		return 0, err
	}
	uid, err := strconv.Atoi(tokens[0])
	if err != nil {
		return 0, errors.Errorf("invalid ID %q", tokens[0])
	}
	return uid, nil
}

// TrimSuffixAndGetProjectID trims the suffix from the name and returns the project ID.
func TrimSuffixAndGetProjectID(name string, suffix string) (string, error) {
	trimmed, err := TrimSuffix(name, suffix)
	if err != nil {
		return "", err
	}
	return GetProjectID(trimmed)
}

// TrimSuffixAndGetInstanceDatabaseID trims the suffix from the name and returns the instance ID and database ID.
func TrimSuffixAndGetInstanceDatabaseID(name string, suffix string) (string, string, error) {
	trimmed, err := TrimSuffix(name, suffix)
	if err != nil {
		return "", "", err
	}
	return GetInstanceDatabaseID(trimmed)
}

// GetEnvironmentID returns the environment ID from a resource name.
func GetEnvironmentID(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, EnvironmentNamePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

// GetInstanceID returns the instance ID from a resource name.
func GetInstanceID(name string) (string, error) {
	// the instance request should be instances/{instance-id}
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

// GetInstanceRoleID returns the instance ID and instance role name from a resource name.
func GetInstanceRoleID(name string) (string, string, error) {
	// the instance request should be instances/{instance-id}/roles/{role-name}
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, InstanceRolePrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

// GetInstanceDatabaseID returns the instance ID and database ID from a resource name.
func GetInstanceDatabaseID(name string) (string, string, error) {
	// the instance request should be instances/{instance-id}/databases/{database-id}
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, DatabaseIDPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

// GetInstanceDatabaseIDChangeHistory returns the instance ID, database ID, and change history ID from a resource name.
func GetInstanceDatabaseIDChangeHistory(name string) (string, string, string, error) {
	// the name should be instances/{instance-id}/databases/{database-id}/changeHistories/{changeHistory-id}
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, DatabaseIDPrefix, ChangeHistoryPrefix)
	if err != nil {
		return "", "", "", err
	}
	return tokens[0], tokens[1], tokens[2], nil
}

// GetInstanceDatabaseIDSecretName returns the instance ID, database ID, and secret name from a resource name.
func GetInstanceDatabaseIDSecretName(name string) (string, string, string, error) {
	// the instance request should be instances/{instance-id}/databases/{database-id}/secrets/{secret-name}
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, DatabaseIDPrefix, SecretNamePrefix)
	if err != nil {
		return "", "", "", err
	}
	return tokens[0], tokens[1], tokens[2], nil
}

// GetInstanceDatabaseIDBackupName returns the instance ID, database ID, and backup name from a resource name.
func GetInstanceDatabaseIDBackupName(name string) (string, string, string, error) {
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, DatabaseIDPrefix, BackupPrefix)
	if err != nil {
		return "", "", "", err
	}
	return tokens[0], tokens[1], tokens[2], nil
}

// GetUserID returns the user ID from a resource name.
func GetUserID(name string) (int, error) {
	return GetUIDFromName(name, UserNamePrefix)
}

// GetUserEmail returns the user email from a resource name.
func GetUserEmail(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, UserNamePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

// GetSettingName returns the setting name from a resource name.
func GetSettingName(name string) (string, error) {
	token, err := GetNameParentTokens(name, SettingNamePrefix)
	if err != nil {
		return "", err
	}
	return token[0], nil
}

// GetIdentityProviderID returns the identity provider ID from a resource name.
func GetIdentityProviderID(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, IdentityProviderNamePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

// GetBookmarkID returns the bookmark ID from a resource name.
func GetBookmarkID(name string) (int, error) {
	return GetUIDFromName(name, BookmarkPrefix)
}

// GetExternalVersionControlID returns the external version control ID from a resource name.
func GetExternalVersionControlID(name string) (int, error) {
	return GetUIDFromName(name, ExternalVersionControlPrefix)
}

// GetRiskID returns the risk ID from a resource name.
func GetRiskID(name string) (int64, error) {
	tokens, err := GetNameParentTokens(name, RiskPrefix)
	if err != nil {
		return 0, err
	}
	riskID, err := strconv.ParseInt(tokens[0], 10, 64)
	if err != nil {
		return 0, errors.Errorf("invalid risk ID %q", tokens[0])
	}
	return riskID, nil
}

// GetIssueID returns the issue ID from a resource name.
func GetIssueID(name string) (int, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, IssuePrefix)
	if err != nil {
		return 0, err
	}
	issueID, err := strconv.Atoi(tokens[1])
	if err != nil {
		return 0, errors.Errorf("invalid issue ID %q", tokens[1])
	}
	return issueID, nil
}

// GetTaskID returns the task ID from a resource name.
func GetTaskID(name string) (int, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, RolloutPrefix, StagePrefix, TaskPrefix)
	if err != nil {
		return 0, err
	}
	taskID, err := strconv.Atoi(tokens[3])
	if err != nil {
		return 0, errors.Errorf("invalid task ID %q", tokens[1])
	}
	return taskID, nil
}

// GetPlanID returns the plan ID from a resource name.
func GetPlanID(name string) (int64, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, PlanPrefix)
	if err != nil {
		return 0, err
	}
	planID, err := strconv.ParseInt(tokens[1], 10, 64)
	if err != nil {
		return 0, errors.Errorf("invalid plan ID %q", tokens[1])
	}
	return planID, nil
}

// GetProjectIDRolloutID returns the project ID and rollout ID from a resource name.
func GetProjectIDRolloutID(name string) (string, int, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, RolloutPrefix)
	if err != nil {
		return "", 0, err
	}
	rolloutID, err := strconv.Atoi(tokens[1])
	if err != nil {
		return "", 0, errors.Errorf("invalid rollout ID %q", tokens[1])
	}
	return tokens[0], rolloutID, nil
}

// GetProjectIDRolloutIDMaybeStageID returns the project ID, rollout ID, and maybe stage ID from a resource name.
func GetProjectIDRolloutIDMaybeStageID(name string) (string, int, *int, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, RolloutPrefix, StagePrefix)
	if err != nil {
		return "", 0, nil, err
	}
	rolloutID, err := strconv.Atoi(tokens[1])
	if err != nil {
		return "", 0, nil, errors.Errorf("invalid rollout ID %q", tokens[1])
	}
	var maybeStageID *int
	if tokens[2] != "-" {
		stageID, err := strconv.Atoi(tokens[2])
		if err != nil {
			return "", 0, nil, errors.Errorf("invalid stage ID %q", tokens[2])
		}
		maybeStageID = &stageID
	}
	return tokens[0], rolloutID, maybeStageID, nil
}

// GetProjectIDRolloutIDStageIDMaybeTaskID returns the project ID, rollout ID, and maybe stage ID and maybe task ID from a resource name.
func GetProjectIDRolloutIDStageIDMaybeTaskID(name string) (string, int, int, *int, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, RolloutPrefix, StagePrefix, TaskPrefix)
	if err != nil {
		return "", 0, 0, nil, err
	}
	rolloutID, err := strconv.Atoi(tokens[1])
	if err != nil {
		return "", 0, 0, nil, errors.Errorf("invalid rollout ID %q", tokens[1])
	}
	stageID, err := strconv.Atoi(tokens[2])
	if err != nil {
		return "", 0, 0, nil, errors.Errorf("invalid stage ID %q", tokens[2])
	}
	var maybeTaskID *int
	if tokens[3] != "-" {
		taskID, err := strconv.Atoi(tokens[3])
		if err != nil {
			return "", 0, 0, nil, errors.Errorf("invalid task ID %q", tokens[3])
		}
		maybeTaskID = &taskID
	}
	return tokens[0], rolloutID, stageID, maybeTaskID, nil
}

// GetProjectIDRolloutIDMaybeStageIDMaybeTaskID returns the project ID, rollout ID, and maybe stage ID and maybe task ID from a resource name.
func GetProjectIDRolloutIDMaybeStageIDMaybeTaskID(name string) (string, int, *int, *int, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, RolloutPrefix, StagePrefix, TaskPrefix)
	if err != nil {
		return "", 0, nil, nil, err
	}
	rolloutID, err := strconv.Atoi(tokens[1])
	if err != nil {
		return "", 0, nil, nil, errors.Errorf("invalid rollout ID %q", tokens[1])
	}
	var maybeStageID, maybeTaskID *int
	if tokens[2] != "-" {
		stageID, err := strconv.Atoi(tokens[2])
		if err != nil {
			return "", 0, nil, nil, errors.Errorf("invalid stage ID %q", tokens[2])
		}
		maybeStageID = &stageID
	}
	if tokens[3] != "-" {
		taskID, err := strconv.Atoi(tokens[3])
		if err != nil {
			return "", 0, nil, nil, errors.Errorf("invalid task ID %q", tokens[3])
		}
		maybeTaskID = &taskID
	}
	return tokens[0], rolloutID, maybeStageID, maybeTaskID, nil
}

// GetProjectIDRolloutIDStageIDTaskID returns the project ID, rollout ID, stage ID, and task ID from a resource name.
func GetProjectIDRolloutIDStageIDTaskID(name string) (string, int, int, int, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, RolloutPrefix, StagePrefix, TaskPrefix)
	if err != nil {
		return "", 0, 0, 0, err
	}
	rolloutID, err := strconv.Atoi(tokens[1])
	if err != nil {
		return "", 0, 0, 0, errors.Errorf("invalid rollout ID %q", tokens[1])
	}
	stageID, err := strconv.Atoi(tokens[2])
	if err != nil {
		return "", 0, 0, 0, errors.Errorf("invalid stage ID %q", tokens[2])
	}

	taskID, err := strconv.Atoi(tokens[3])
	if err != nil {
		return "", 0, 0, 0, errors.Errorf("invalid task ID %q", tokens[3])
	}
	return tokens[0], rolloutID, stageID, taskID, nil
}

// GetProjectIDRolloutIDStageIDTaskID returns the project ID, rollout ID, stage ID, and task ID from a resource name.
func GetProjectIDRolloutIDStageIDTaskIDTaskRunID(name string) (string, int, int, int, int, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, RolloutPrefix, StagePrefix, TaskPrefix, TaskRunPrefix)
	if err != nil {
		return "", 0, 0, 0, 0, err
	}
	rolloutID, err := strconv.Atoi(tokens[1])
	if err != nil {
		return "", 0, 0, 0, 0, errors.Errorf("invalid rollout ID %q", tokens[1])
	}
	stageID, err := strconv.Atoi(tokens[2])
	if err != nil {
		return "", 0, 0, 0, 0, errors.Errorf("invalid stage ID %q", tokens[2])
	}

	taskID, err := strconv.Atoi(tokens[3])
	if err != nil {
		return "", 0, 0, 0, 0, errors.Errorf("invalid task ID %q", tokens[3])
	}
	taskRunID, err := strconv.Atoi(tokens[4])
	if err != nil {
		return "", 0, 0, 0, 0, errors.Errorf("invalid task run ID %q", tokens[4])
	}
	return tokens[0], rolloutID, stageID, taskID, taskRunID, nil
}

// GetRoleID returns the role ID from a resource name.
func GetRoleID(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, RolePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

// GetProjectResourceIDSheetID returns the project ID and sheet ID from a resource name.
func GetProjectResourceIDSheetID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, SheetIDPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

// GetProjectResourceIDAndSchemaDesignSheetID returns the project ID and schema design sheet ID from a resource name.
func GetProjectResourceIDAndSchemaDesignSheetID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, SchemaDesignPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

// TrimSuffix trims the suffix from the name and returns the trimmed name.
func TrimSuffix(name, suffix string) (string, error) {
	if !strings.HasSuffix(name, suffix) {
		return "", errors.Errorf("invalid request %q with suffix %q", name, suffix)
	}
	return strings.TrimSuffix(name, suffix), nil
}

// GetNameParentTokens returns the tokens from a resource name.
func GetNameParentTokens(name string, tokenPrefixes ...string) ([]string, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2*len(tokenPrefixes) {
		return nil, errors.Errorf("invalid request %q", name)
	}

	var tokens []string
	for i, tokenPrefix := range tokenPrefixes {
		if fmt.Sprintf("%s/", parts[2*i]) != tokenPrefix {
			return nil, errors.Errorf("invalid prefix %q in request %q", tokenPrefix, name)
		}
		if parts[2*i+1] == "" {
			return nil, errors.Errorf("invalid request %q with empty prefix %q", name, tokenPrefix)
		}
		tokens = append(tokens, parts[2*i+1])
	}
	return tokens, nil
}
