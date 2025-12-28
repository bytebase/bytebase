//nolint:revive
package common

import (
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
	RolloutPrefix              = "rollouts/"
	StagePrefix                = "stages/"
	TaskPrefix                 = "tasks/"
	TaskRunPrefix              = "taskRuns/"
	PlanPrefix                 = "plans/"
	PlanCheckRunPrefix         = "planCheckRuns/"
	SpecPrefix                 = "specs/"
	RolePrefix                 = "roles/"
	WebhookIDPrefix            = "webhooks/"
	SheetIDPrefix              = "sheets/"
	WorksheetIDPrefix          = "worksheets/"
	DatabaseGroupNamePrefix    = "databaseGroups/"
	SchemaNamePrefix           = "schemas/"
	TableNamePrefix            = "tables/"
	ChangelogPrefix            = "changelogs/"
	IssueNamePrefix            = "issues/"
	IssueCommentNamePrefix     = "issueComments/"
	PipelineNamePrefix         = "pipelines/"
	LogNamePrefix              = "logs/"
	BranchPrefix               = "branches/"
	DeploymentConfigPrefix     = "deploymentConfigs/"
	AuditLogPrefix             = "auditLogs/"
	GroupPrefix                = "groups/"
	ReviewConfigPrefix         = "reviewConfigs/"
	ReleaseNamePrefix          = "releases/"
	FileNamePrefix             = "files/"
	RevisionNamePrefix         = "revisions/"

	SchemaSuffix    = "/schema"
	SDLSchemaSuffix = "/sdlSchema"
	MetadataSuffix  = "/metadata"
	CatalogSuffix   = "/catalog"

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

// GetProjectIDWebhookID returns the project ID and webhook ID from a resource name.
func GetProjectIDWebhookID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, WebhookIDPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
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

// GetInstanceDatabaseID returns the instance ID and database ID from a resource name.
func GetInstanceDatabaseID(name string) (string, string, error) {
	// the instance request should be instances/{instance-id}/databases/{database-id}
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, DatabaseIDPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
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

// GetProjectIDPlanIDFromPlanCheckRun returns the project ID and plan ID from a plan check run singleton resource name.
// Format: projects/{project}/plans/{plan}/planCheckRun
func GetProjectIDPlanIDFromPlanCheckRun(name string) (string, int64, error) {
	// Remove the trailing "/planCheckRun" suffix
	if !strings.HasSuffix(name, "/planCheckRun") {
		return "", 0, errors.Errorf("invalid plan check run name %q, expected suffix /planCheckRun", name)
	}
	planName := strings.TrimSuffix(name, "/planCheckRun")
	projectID, planID, err := GetProjectIDPlanID(planName)
	if err != nil {
		return "", 0, err
	}
	return projectID, int64(planID), nil
}

// GetProjectIDPlanIDFromRolloutName returns the project ID and plan ID from a resource name.
func GetProjectIDPlanIDFromRolloutName(name string) (string, int64, error) {
	if !strings.HasSuffix(name, "/rollout") {
		return "", 0, errors.Errorf("invalid rollout name %q, expected suffix /rollout", name)
	}
	planName := strings.TrimSuffix(name, "/rollout")
	return GetProjectIDPlanID(planName)
}

// GetProjectIDPlanIDMaybeStageID returns the project ID, plan ID, and maybe stage ID from a resource name.
func GetProjectIDPlanIDMaybeStageID(name string) (string, int64, *string, error) {
	parts := strings.Split(name, "/rollout")
	if len(parts) != 2 {
		return "", 0, nil, errors.Errorf("invalid rollout stage name %q", name)
	}

	projectID, planID, err := GetProjectIDPlanID(parts[0])
	if err != nil {
		return "", 0, nil, err
	}

	// suffix should be /stages/{stage}
	suffixParts := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
	if len(suffixParts) != 2 || suffixParts[0]+"/" != StagePrefix {
		return "", 0, nil, errors.Errorf("invalid stage suffix %q", parts[1])
	}

	var maybeStageID *string
	if suffixParts[1] != "-" {
		maybeStageID = &suffixParts[1]
	}
	return projectID, planID, maybeStageID, nil
}

// GetProjectIDPlanIDStageIDMaybeTaskID returns the project ID, plan ID, and maybe stage ID and maybe task ID from a resource name.
func GetProjectIDPlanIDStageIDMaybeTaskID(name string) (string, int64, string, *int, error) {
	parts := strings.Split(name, "/rollout")
	if len(parts) != 2 {
		return "", 0, "", nil, errors.Errorf("invalid rollout task name %q", name)
	}

	projectID, planID, err := GetProjectIDPlanID(parts[0])
	if err != nil {
		return "", 0, "", nil, err
	}

	// suffix should be /stages/{stage}/tasks/{task}
	suffixParts := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
	if len(suffixParts) != 4 || suffixParts[0]+"/" != StagePrefix || suffixParts[2]+"/" != TaskPrefix {
		return "", 0, "", nil, errors.Errorf("invalid task suffix %q", parts[1])
	}

	stageID := suffixParts[1]
	var maybeTaskID *int
	if suffixParts[3] != "-" {
		taskID, err := strconv.Atoi(suffixParts[3])
		if err != nil {
			return "", 0, "", nil, errors.Errorf("invalid task ID %q", suffixParts[3])
		}
		maybeTaskID = &taskID
	}
	return projectID, planID, stageID, maybeTaskID, nil
}

// GetProjectIDPlanIDMaybeStageIDMaybeTaskID returns the project ID, plan ID, and maybe stage ID and maybe task ID from a resource name.
func GetProjectIDPlanIDMaybeStageIDMaybeTaskID(name string) (string, int64, *string, *int, error) {
	parts := strings.Split(name, "/rollout")
	if len(parts) != 2 {
		return "", 0, nil, nil, errors.Errorf("invalid rollout task name %q", name)
	}

	projectID, planID, err := GetProjectIDPlanID(parts[0])
	if err != nil {
		return "", 0, nil, nil, err
	}

	// suffix should be /stages/{stage}/tasks/{task}
	suffixParts := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
	if len(suffixParts) != 4 || suffixParts[0]+"/" != StagePrefix || suffixParts[2]+"/" != TaskPrefix {
		return "", 0, nil, nil, errors.Errorf("invalid task suffix %q", parts[1])
	}

	var maybeStageID *string
	if suffixParts[1] != "-" {
		maybeStageID = &suffixParts[1]
	}
	var maybeTaskID *int
	if suffixParts[3] != "-" {
		taskID, err := strconv.Atoi(suffixParts[3])
		if err != nil {
			return "", 0, nil, nil, errors.Errorf("invalid task ID %q", suffixParts[3])
		}
		maybeTaskID = &taskID
	}
	return projectID, planID, maybeStageID, maybeTaskID, nil
}

// GetProjectIDPlanIDStageIDTaskID returns the project ID, plan ID, stage ID, and task ID from a resource name.
func GetProjectIDPlanIDStageIDTaskID(name string) (string, int64, string, int, error) {
	parts := strings.Split(name, "/rollout")
	if len(parts) != 2 {
		return "", 0, "", 0, errors.Errorf("invalid rollout task name %q", name)
	}

	projectID, planID, err := GetProjectIDPlanID(parts[0])
	if err != nil {
		return "", 0, "", 0, err
	}

	// suffix should be /stages/{stage}/tasks/{task}
	suffixParts := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
	if len(suffixParts) != 4 || suffixParts[0]+"/" != StagePrefix || suffixParts[2]+"/" != TaskPrefix {
		return "", 0, "", 0, errors.Errorf("invalid task suffix %q", parts[1])
	}

	stageID := suffixParts[1]
	taskID, err := strconv.Atoi(suffixParts[3])
	if err != nil {
		return "", 0, "", 0, errors.Errorf("invalid task ID %q", suffixParts[3])
	}
	return projectID, planID, stageID, taskID, nil
}

// GetProjectIDPlanIDStageIDTaskIDTaskRunID returns the project ID, plan ID, stage ID, task ID and task run ID from a resource name.
func GetProjectIDPlanIDStageIDTaskIDTaskRunID(name string) (string, int64, string, int, int, error) {
	parts := strings.Split(name, "/rollout")
	if len(parts) != 2 {
		return "", 0, "", 0, 0, errors.Errorf("invalid rollout task run name %q", name)
	}

	projectID, planID, err := GetProjectIDPlanID(parts[0])
	if err != nil {
		return "", 0, "", 0, 0, err
	}

	// suffix should be /stages/{stage}/tasks/{task}/taskRuns/{taskRun}
	suffixParts := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
	if len(suffixParts) != 6 || suffixParts[0]+"/" != StagePrefix || suffixParts[2]+"/" != TaskPrefix || suffixParts[4]+"/" != TaskRunPrefix {
		return "", 0, "", 0, 0, errors.Errorf("invalid task run suffix %q", parts[1])
	}

	stageID := suffixParts[1]
	taskID, err := strconv.Atoi(suffixParts[3])
	if err != nil {
		return "", 0, "", 0, 0, errors.Errorf("invalid task ID %q", suffixParts[3])
	}
	taskRunID, err := strconv.Atoi(suffixParts[5])
	if err != nil {
		return "", 0, "", 0, 0, errors.Errorf("invalid task run ID %q", suffixParts[5])
	}
	return projectID, planID, stageID, taskID, taskRunID, nil
}

// GetRoleID returns the role ID from a resource name.
func GetRoleID(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, RolePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

// GetProjectResourceIDSheetSha256 returns the project ID and sheet SHA256 from a resource name.
func GetProjectResourceIDSheetSha256(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, SheetIDPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
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

func FormatUserEmail(email string) string {
	return fmt.Sprintf("%s%s", UserNamePrefix, email)
}

func FormatGroupEmail(email string) string {
	return fmt.Sprintf("%s%s", GroupPrefix, email)
}

// IsWorkloadIdentityEmail checks if the email is a workload identity email.
func IsWorkloadIdentityEmail(email string) bool {
	return strings.HasSuffix(email, WorkloadIdentityEmailSuffix)
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

func FormatSheet(projectID string, sheetSha256 string) string {
	return fmt.Sprintf("%s/%s%s", FormatProject(projectID), SheetIDPrefix, sheetSha256)
}

func FormatIssue(projectID string, issueUID int) string {
	return fmt.Sprintf("%s/%s%d", FormatProject(projectID), IssueNamePrefix, issueUID)
}

func FormatRollout(projectID string, planUID int64) string {
	return fmt.Sprintf("%s/rollout", FormatPlan(projectID, planUID))
}

// EmptyStageID is the placeholder used for stages without environment or with deleted environments.
const EmptyStageID = "-"

// FormatStageID returns the stage ID, using EmptyStageID placeholder if environment is empty.
func FormatStageID(environment string) string {
	if environment == "" {
		return EmptyStageID
	}
	return environment
}

// stageID is task environmentID.
func FormatStage(projectID string, planUID int64, stageID string) string {
	return fmt.Sprintf("%s/%s%s", FormatRollout(projectID, planUID), StagePrefix, stageID)
}

// stageID is task environmentID.
func FormatTask(projectID string, planUID int64, stageID string, taskUID int) string {
	// stageUID is now environmentID
	return fmt.Sprintf("%s/%s%d", FormatStage(projectID, planUID, stageID), TaskPrefix, taskUID)
}

// stageID is task environmentID.
func FormatTaskRun(projectID string, planUID int64, stageID string, taskUID, taskRunUID int) string {
	return fmt.Sprintf("%s/%s%d", FormatTask(projectID, planUID, stageID, taskUID), TaskRunPrefix, taskRunUID)
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

// FormatPlanCheckRun formats a plan check run singleton resource name.
// Format: projects/{project}/plans/{plan}/planCheckRun
func FormatPlanCheckRun(projectID string, planUID int64) string {
	return fmt.Sprintf("%s/planCheckRun", FormatPlan(projectID, planUID))
}

func FormatSpec(projectID string, planUID int64, specID string) string {
	return fmt.Sprintf("%s/%s%s", FormatPlan(projectID, planUID), SpecPrefix, specID)
}

func GetPolicyResourceTypeAndResource(requestName string) (storepb.Policy_Resource, *string, error) {
	if requestName == "" {
		return storepb.Policy_WORKSPACE, nil, nil
	}

	if strings.HasPrefix(requestName, ProjectNamePrefix) {
		projectID, err := GetProjectID(requestName)
		if err != nil {
			return storepb.Policy_RESOURCE_UNSPECIFIED, nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if projectID == "-" {
			return storepb.Policy_PROJECT, nil, nil
		}
		return storepb.Policy_PROJECT, &requestName, nil
	}

	if strings.HasPrefix(requestName, EnvironmentNamePrefix) {
		// environment policy request name should be environments/{environment id}
		environmentID, err := GetEnvironmentID(requestName)
		if err != nil {
			return storepb.Policy_RESOURCE_UNSPECIFIED, nil, err
		}
		if environmentID == "-" {
			return storepb.Policy_ENVIRONMENT, nil, nil
		}
		return storepb.Policy_ENVIRONMENT, &requestName, nil
	}

	return storepb.Policy_RESOURCE_UNSPECIFIED, nil, errors.Errorf("unknown request name %s", requestName)
}
