package common

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

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

func GetProjectID(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

func GetProjectIDDatabaseGroupID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, DatabaseGroupNamePrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

func GetProjectIDDatabaseGroupIDSchemaGroupID(name string) (string, string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, DatabaseGroupNamePrefix, SchemaGroupNamePrefix)
	if err != nil {
		return "", "", "", err
	}
	return tokens[0], tokens[1], tokens[2], nil
}

func GetProjectIDWebhookID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, WebhookIDPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

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

func TrimSuffixAndGetProjectID(name string, suffix string) (string, error) {
	trimmed, err := TrimSuffix(name, suffix)
	if err != nil {
		return "", err
	}
	return GetProjectID(trimmed)
}

func TrimSuffixAndGetInstanceDatabaseID(name string, suffix string) (string, string, error) {
	trimmed, err := TrimSuffix(name, suffix)
	if err != nil {
		return "", "", err
	}
	return GetInstanceDatabaseID(trimmed)
}

func GetEnvironmentID(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, EnvironmentNamePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

func GetInstanceID(name string) (string, error) {
	// the instance request should be instances/{instance-id}
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

func GetInstanceRoleID(name string) (string, string, error) {
	// the instance request should be instances/{instance-id}/roles/{role-name}
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, InstanceRolePrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

func GetInstanceDatabaseID(name string) (string, string, error) {
	// the instance request should be instances/{instance-id}/databases/{database-id}
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, DatabaseIDPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

func GetInstanceDatabaseIDChangeHistory(name string) (string, string, string, error) {
	// the name should be instances/{instance-id}/databases/{database-id}/changeHistories/{changeHistory-id}
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, DatabaseIDPrefix, ChangeHistoryPrefix)
	if err != nil {
		return "", "", "", err
	}
	return tokens[0], tokens[1], tokens[2], nil
}

func GetInstanceDatabaseIDSecretName(name string) (string, string, string, error) {
	// the instance request should be instances/{instance-id}/databases/{database-id}/secrets/{secret-name}
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, DatabaseIDPrefix, SecretNamePrefix)
	if err != nil {
		return "", "", "", err
	}
	return tokens[0], tokens[1], tokens[2], nil
}

func GetInstanceDatabaseIDBackupName(name string) (string, string, string, error) {
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, DatabaseIDPrefix, BackupPrefix)
	if err != nil {
		return "", "", "", err
	}
	return tokens[0], tokens[1], tokens[2], nil
}

func GetUserID(name string) (int, error) {
	return GetUIDFromName(name, UserNamePrefix)
}

func GetUserEmail(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, UserNamePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

func GetSettingName(name string) (string, error) {
	token, err := GetNameParentTokens(name, SettingNamePrefix)
	if err != nil {
		return "", err
	}
	return token[0], nil
}

func GetIdentityProviderID(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, IdentityProviderNamePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

func GetBookmarkID(name string) (int, error) {
	return GetUIDFromName(name, BookmarkPrefix)
}

func GetExternalVersionControlID(name string) (int, error) {
	return GetUIDFromName(name, ExternalVersionControlPrefix)
}

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

func GetRoleID(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, RolePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

func GetProjectResourceIDSheetID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, SheetIDPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

func GetProjectResourceIDAndSchemaDesignSheetID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, SchemaDesignPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

func TrimSuffix(name, suffix string) (string, error) {
	if !strings.HasSuffix(name, suffix) {
		return "", errors.Errorf("invalid request %q with suffix %q", name, suffix)
	}
	return strings.TrimSuffix(name, suffix), nil
}

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
