package common

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// nolint:revive
const (
	WorkspacePrefix            = "workspaces/"
	ProjectNamePrefix          = "projects/"
	EnvironmentNamePrefix      = "environments/"
	InstanceNamePrefix         = "instances/"
	PolicyNamePrefix           = "policies/"
	DatabaseIDPrefix           = "databases/"
	InstanceRolePrefix         = "roles/"
	UserNamePrefix             = "users/"
	IdentityProviderNamePrefix = "idps/"
	SettingNamePrefix          = "settings/"
	VCSProviderPrefix          = "vcsProviders/"
	RiskPrefix                 = "risks/"
	RolloutPrefix              = "rollouts/"
	StagePrefix                = "stages/"
	TaskPrefix                 = "tasks/"
	TaskRunPrefix              = "taskRuns/"
	PlanPrefix                 = "plans/"
	PlanCheckRunPrefix         = "planCheckRuns/"
	RolePrefix                 = "roles/"
	SecretNamePrefix           = "secrets/"
	WebhookIDPrefix            = "webhooks/"
	SheetIDPrefix              = "sheets/"
	WorksheetIDPrefix          = "worksheets/"
	DatabaseGroupNamePrefix    = "databaseGroups/"
	SchemaNamePrefix           = "schemas/"
	TableNamePrefix            = "tables/"
	ChangeHistoryPrefix        = "changeHistories/"
	ChangelogPrefix            = "changelogs/"
	IssueNamePrefix            = "issues/"
	IssueCommentNamePrefix     = "issueComments/"
	PipelineNamePrefix         = "pipelines/"
	LogNamePrefix              = "logs/"
	BranchPrefix               = "branches/"
	DeploymentConfigPrefix     = "deploymentConfigs/"
	ChangelistsPrefix          = "changelists/"
	VCSConnectorPrefix         = "vcsConnectors/"
	AuditLogPrefix             = "auditLogs/"
	GroupPrefix                = "groups/"
	ReviewConfigPrefix         = "reviewConfigs/"
	ReleaseNamePrefix          = "releases/"
	FileNamePrefix             = "files/"
	RevisionNamePrefix         = "revisions/"

	SchemaSuffix     = "/schema"
	MetadataSuffix   = "/metadata"
	CatalogSuffix    = "/catalog"
	GitOpsInfoSuffix = "/gitOpsInfo"

	UserBindingPrefix  = "user:"
	GroupBindingPrefix = "group:"
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

// GetSchemaTableName returns the schema and table names from a resource name.
func GetSchemaTableName(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, SchemaNamePrefix, TableNamePrefix)
	if err != nil {
		return "", "", err
	}
	if tokens[0] == "-" {
		tokens[0] = ""
	}
	return tokens[0], tokens[1], nil
}

// GetProjectIDWebhookID returns the project ID and webhook ID from a resource name.
func GetProjectIDWebhookID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, WebhookIDPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

func GetProjectIDDeploymentConfigID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, DeploymentConfigPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

func GetProjectIDChangelistID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, ChangelistsPrefix)
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

// GetInstanceDatabaseRevisionID returns the instance ID, database ID, and revision UID from a resource name.
func GetInstanceDatabaseRevisionID(name string) (string, string, int64, error) {
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, DatabaseIDPrefix, RevisionNamePrefix)
	if err != nil {
		return "", "", 0, err
	}
	revisionUID, err := strconv.ParseInt(tokens[2], 10, 64)
	if err != nil {
		return "", "", 0, errors.Wrapf(err, "failed to convert %q to int64", tokens[2])
	}
	return tokens[0], tokens[1], revisionUID, nil
}

// GetInstanceDatabaseChangelogUID returns the instance ID, database ID, and changelog UID from a resource name.
func GetInstanceDatabaseChangelogUID(name string) (string, string, int64, error) {
	// the name should be instances/{instance-id}/databases/{database-id}/changelogs/{changelog-id}
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, DatabaseIDPrefix, ChangelogPrefix)
	if err != nil {
		return "", "", 0, err
	}
	changelogUID, err := strconv.ParseInt(tokens[2], 10, 64)
	if err != nil {
		return "", "", 0, errors.Wrapf(err, "failed to convert %q to int64", tokens[2])
	}
	return tokens[0], tokens[1], changelogUID, nil
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

// GetVCSProviderID returns the VCS provider ID from a resource name.
func GetVCSProviderID(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, VCSProviderPrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
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

// GetProjectIDIssueUID returns the project ID and issue UID from the issue name.
func GetProjectIDIssueUID(name string) (string, int, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, IssueNamePrefix)
	if err != nil {
		return "", 0, err
	}
	issueUID, err := strconv.Atoi(tokens[1])
	if err != nil {
		return "", 0, errors.Errorf("invalid issue ID %q", tokens[1])
	}
	return tokens[0], issueUID, nil
}

// GetProjectIDIssueUIDIssueCommentUID returns the project ID, issue UID and issue comment UID from the issue comment name.
func GetProjectIDIssueUIDIssueCommentUID(name string) (string, int, int, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, IssueNamePrefix, IssueCommentNamePrefix)
	if err != nil {
		return "", 0, 0, err
	}
	issueUID, err := strconv.Atoi(tokens[1])
	if err != nil {
		return "", 0, 0, errors.Errorf("invalid issue ID %q", tokens[1])
	}
	issueCommentUID, err := strconv.Atoi(tokens[2])
	if err != nil {
		return "", 0, 0, errors.Errorf("invalid issue comment ID %q", tokens[2])
	}
	return tokens[0], issueUID, issueCommentUID, nil
}

// GetIssueID returns the issue ID from a resource name.
func GetIssueID(name string) (int, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, IssueNamePrefix)
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

// GetProjectIDPlanID returns the project ID and plan ID from a resource name.
func GetProjectIDPlanID(name string) (string, int64, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, PlanPrefix)
	if err != nil {
		return "", 0, err
	}
	planID, err := strconv.ParseInt(tokens[1], 10, 64)
	if err != nil {
		return "", 0, errors.Errorf("invalid plan ID %q", tokens[1])
	}
	return tokens[0], planID, nil
}

// GetProjectIDPlanIDPlanCheckRunID returns the project ID, plan ID and plan check run ID from a resource name.
func GetProjectIDPlanIDPlanCheckRunID(name string) (string, int, int, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, PlanPrefix, PlanCheckRunPrefix)
	if err != nil {
		return "", 0, 0, err
	}
	planID, err := strconv.Atoi(tokens[1])
	if err != nil {
		return "", 0, 0, errors.Errorf("invalid plan ID %q", tokens[1])
	}
	planCheckRunID, err := strconv.Atoi(tokens[2])
	if err != nil {
		return "", 0, 0, errors.Errorf("invalid plan check run ID %q", tokens[2])
	}
	return tokens[0], planID, planCheckRunID, nil
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

// GetProjectIDRolloutIDStageIDTaskIDTaskRunID returns the project ID, rollout ID, stage ID, task ID and task run ID from a resource name.
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

// GetProjectResourceIDSheetUID returns the project ID and sheet UID from a resource name.
func GetProjectResourceIDSheetUID(name string) (string, int, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, SheetIDPrefix)
	if err != nil {
		return "", 0, err
	}
	sheetUID, err := strconv.Atoi(tokens[1])
	if err != nil {
		return "", 0, errors.Wrapf(err, "failed to convert sheet uid %q to int", tokens[1])
	}
	return tokens[0], sheetUID, nil
}

// GetWorksheetUID returns the worksheet UID from a resource name.
func GetWorksheetUID(name string) (int, error) {
	tokens, err := GetNameParentTokens(name, WorksheetIDPrefix)
	if err != nil {
		return 0, err
	}
	sheetUID, err := strconv.Atoi(tokens[0])
	if err != nil {
		return 0, errors.Wrapf(err, "failed to convert worksheet uid %q to int", tokens[1])
	}
	return sheetUID, nil
}

// GetReviewConfigID returns the review config id from a resource name.
func GetReviewConfigID(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, ReviewConfigPrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

func GetProjectReleaseUID(name string) (string, int64, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, ReleaseNamePrefix)
	if err != nil {
		return "", 0, err
	}
	releaseUID, err := strconv.ParseInt(tokens[1], 10, 64)
	if err != nil {
		return "", 0, errors.Wrapf(err, "failed to convert %q to int64", tokens[1])
	}
	return tokens[0], releaseUID, nil
}

func GetProjectReleaseUIDFile(name string) (string, int64, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, ReleaseNamePrefix, FileNamePrefix)
	if err != nil {
		return "", 0, "", err
	}
	releaseUID, err := strconv.ParseInt(tokens[1], 10, 64)
	if err != nil {
		return "", 0, "", errors.Wrapf(err, "failed to convert %q to int64", tokens[1])
	}
	return tokens[0], releaseUID, tokens[2], nil
}

var branchRegexp = regexp.MustCompile("^projects/([^/]+)/branches/(.+)$")

// GetProjectAndBranchID returns the project and branch ID from a resource name.
func GetProjectAndBranchID(name string) (string, string, error) {
	matches := branchRegexp.FindStringSubmatch(name)
	if len(matches) != 3 {
		return "", "", errors.Errorf("invalid branch name %q", name)
	}
	return matches[1], matches[2], nil
}

// GetProjectVCSConnectorID returns the workspace, project, and VCS connector ID from a resource name.
func GetProjectVCSConnectorID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, VCSConnectorPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

// GetWorkspaceProjectVCSConnectorID returns the workspace, project, and VCS connector ID from a resource name.
func GetWorkspaceProjectVCSConnectorID(name string) (string, string, string, error) {
	tokens, err := GetNameParentTokens(name, WorkspacePrefix, ProjectNamePrefix, VCSConnectorPrefix)
	if err != nil {
		return "", "", "", err
	}
	return tokens[0], tokens[1], tokens[2], nil
}

// GetGroupEmail returns the group email.
func GetGroupEmail(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, GroupPrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
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
		tokens = append(tokens, parts[2*i+1])
	}
	return tokens, nil
}

func FormatWorkspace(id string) string {
	return fmt.Sprintf("%s%s", WorkspacePrefix, id)
}

func FormatProject(id string) string {
	return fmt.Sprintf("%s%s", ProjectNamePrefix, id)
}

func FormatDeploymentConfig(parent string) string {
	return fmt.Sprintf("%s/%s%s", parent, DeploymentConfigPrefix, "default")
}

func FormatUserEmail(email string) string {
	return fmt.Sprintf("%s%s", UserNamePrefix, email)
}

func FormatUserUID(uid int) string {
	return fmt.Sprintf("%s%d", UserNamePrefix, uid)
}

func FormatGroupEmail(email string) string {
	return fmt.Sprintf("%s%s", GroupPrefix, email)
}

func FormatReviewConfig(id string) string {
	return fmt.Sprintf("%s%s", ReviewConfigPrefix, id)
}

func FormatEnvironment(resourceID string) string {
	return fmt.Sprintf("%s%s", EnvironmentNamePrefix, resourceID)
}

func FormatInstance(resourceID string) string {
	return fmt.Sprintf("%s%s", InstanceNamePrefix, resourceID)
}

func FormatDatabase(instance string, database string) string {
	return fmt.Sprintf("%s/%s%s", FormatInstance(instance), DatabaseIDPrefix, database)
}

func FormatRole(role string) string {
	return fmt.Sprintf("%s%s", RolePrefix, role)
}

func FormatSheet(projectID string, sheetUID int) string {
	return fmt.Sprintf("%s/%s%d", FormatProject(projectID), SheetIDPrefix, sheetUID)
}

func FormatIssue(projectID string, issueUID int) string {
	return fmt.Sprintf("%s/%s%d", FormatProject(projectID), IssueNamePrefix, issueUID)
}

func FormatRollout(projectID string, pipelineUID int) string {
	return fmt.Sprintf("%s/%s%d", FormatProject(projectID), RolloutPrefix, pipelineUID)
}

func FormatStage(projectID string, pipelineUID, stageUID int) string {
	return fmt.Sprintf("%s/%s%d", FormatRollout(projectID, pipelineUID), StagePrefix, stageUID)
}

func FormatTask(projectID string, pipelineUID, stageUID, taskUID int) string {
	return fmt.Sprintf("%s/%s%d", FormatStage(projectID, pipelineUID, stageUID), TaskPrefix, taskUID)
}

func FormatTaskRun(projectID string, pipelineUID, stageUID, taskUID, taskRunUID int) string {
	return fmt.Sprintf("%s/%s%d", FormatTask(projectID, pipelineUID, stageUID, taskUID), TaskRunPrefix, taskRunUID)
}

func FormatBranchResourceID(projectID string, branchID string) string {
	return fmt.Sprintf("%s/%s%s", FormatProject(projectID), BranchPrefix, branchID)
}

func FormatReleaseName(projectID string, releaseUID int64) string {
	return fmt.Sprintf("%s/%s%d", FormatProject(projectID), ReleaseNamePrefix, releaseUID)
}

func FormatReleaseFile(release string, fileID string) string {
	return fmt.Sprintf("%s/%s%s", release, FileNamePrefix, fileID)
}

func FormatRevision(instanceID, databaseID string, revisionUID int64) string {
	return fmt.Sprintf("%s/%s%d", FormatDatabase(instanceID, databaseID), RevisionNamePrefix, revisionUID)
}

func FormatChangelog(instanceID, databaseID string, changelogUID int64) string {
	return fmt.Sprintf("%s/%s%d", FormatDatabase(instanceID, databaseID), ChangelogPrefix, changelogUID)
}

func FormatPlan(projectID string, planUID int64) string {
	return fmt.Sprintf("%s/%s%d", FormatProject(projectID), PlanPrefix, planUID)
}

func FormatPlanCheckRun(projectID string, planUID, runUID int64) string {
	return fmt.Sprintf("%s/%s%d", FormatPlan(projectID, planUID), PlanCheckRunPrefix, runUID)
}
