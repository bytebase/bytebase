package mcp

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/blang/semver/v4"
)

// The GitOps workflow files, as the GitOps page builds them in
// frontend/src/routes/project/gitopsWorkflowYaml.ts. A change to the YAML is
// made in both, and the cases in
// frontend/src/routes/project/gitops-workflow-cases fail the side that missed it.

const (
	workflowProviderGitHub = "github"
	workflowProviderGitLab = "gitlab"

	defaultMigrationFolder = "migrations"

	gitlabWorkflowPath = "bytebase.gitlab-ci.yml"
	// gitlabWorkflowInclude is the line the owner adds to the project's own
	// .gitlab-ci.yml.
	gitlabWorkflowInclude = "include:\n  - local: " + gitlabWorkflowPath
)

// workflowParams mirrors WorkflowParams in gitopsWorkflowYaml.ts. The JSON tags
// decode the shared cases.
type workflowParams struct {
	Provider              string   `json:"provider"`
	BytebaseURL           string   `json:"bytebaseUrl"`
	ActionImage           string   `json:"actionImage"`
	WorkloadIdentityEmail string   `json:"workloadIdentityEmail"`
	ProjectID             string   `json:"projectId"`
	Targets               []string `json:"targets"`
	// Environments are environment IDs in deploy order, one deploy job each.
	Environments []string `json:"environments"`
	Folder       string   `json:"folder"`
}

// WorkflowFile is one file the owner adds to the repository.
type WorkflowFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// bytebaseActionImage is the action image a server of this version needs:
// bytebase-action exits when its version is incompatible with the server's. A
// Cloud server needs the :cloud image, and a self-hosted server rejects a newer
// minor release, which the untagged image can be.
func bytebaseActionImage(serverVersion string) string {
	if strings.HasPrefix(serverVersion, "cloud") {
		return "bytebase/bytebase-action:cloud"
	}
	version, err := semver.Parse(strings.TrimPrefix(strings.TrimSpace(serverVersion), "v"))
	if err != nil {
		return "bytebase/bytebase-action"
	}
	return "bytebase/bytebase-action:" + version.String()
}

var migrationFolderPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)*$`)

// isValidMigrationFolder accepts a relative folder path: segments of letters,
// digits, ".", "_" and "-", joined by "/", with no "." or ".." segment.
func isValidMigrationFolder(folder string) bool {
	if !migrationFolderPattern.MatchString(folder) {
		return false
	}
	for _, part := range strings.Split(folder, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}

// Database names are free-form. A target list outside this set is quoted where
// a shell or GitLab would otherwise split or expand it.
var safeWord = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

// shellWord returns the value as one shell word.
func shellWord(value string) string {
	if safeWord.MatchString(value) {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// gitlabVariable returns the value as a GitLab variable. GitLab expands "$" in
// a variable's value, and reads "$$" as one "$".
func gitlabVariable(value string) string {
	if safeWord.MatchString(value) {
		return value
	}
	var sb strings.Builder
	encoder := json.NewEncoder(&sb)
	// JSON.stringify, which the page uses, leaves <, > and & as they are.
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(strings.ReplaceAll(value, "$", "$$"))
	return strings.TrimSuffix(sb.String(), "\n")
}

// generateWorkflowFiles returns every file for the provider, in the order the
// GitOps page shows them.
func generateWorkflowFiles(p workflowParams) []WorkflowFile {
	fill := strings.NewReplacer(
		"@@BYTEBASE_URL@@", p.BytebaseURL,
		"@@ACTION_IMAGE@@", p.ActionImage,
		"@@WORKLOAD_IDENTITY@@", p.WorkloadIdentityEmail,
		"@@PROJECT_ID@@", p.ProjectID,
		"@@TARGETS@@", shellWord(strings.Join(p.Targets, ",")),
		"@@TARGETS_VARIABLE@@", gitlabVariable(strings.Join(p.Targets, ",")),
		"@@FILE_PATTERN@@", p.Folder+"/*.sql",
	)
	if p.Provider == workflowProviderGitLab {
		content := fill.Replace(gitlabWorkflow) + deployJobs(fill.Replace(gitlabDeployJob), "bytebase-create-rollout", "bytebase-deploy-to-", p.Environments)
		return []WorkflowFile{{Path: gitlabWorkflowPath, Content: content}}
	}
	return []WorkflowFile{
		{Path: ".github/workflows/bytebase-review.yml", Content: fill.Replace(githubReviewWorkflow)},
		{
			Path:    ".github/workflows/bytebase-deploy.yml",
			Content: fill.Replace(githubDeployWorkflow) + deployJobs(fill.Replace(githubDeployJob), "create-rollout", "deploy-to-", p.Environments),
		},
	}
}

// deployJobs renders one deploy job per environment. Each needs the rollout
// job and the deploy job of the environment before it.
func deployJobs(job, rolloutJob, deployJobPrefix string, environments []string) string {
	var sb strings.Builder
	for i, environment := range environments {
		needs := rolloutJob
		if i > 0 {
			needs += ", " + deployJobPrefix + environments[i-1]
		}
		sb.WriteString(strings.NewReplacer("@@ENVIRONMENT@@", environment, "@@NEEDS@@", needs).Replace(job))
	}
	return sb.String()
}

const githubReviewWorkflow = `name: Bytebase SQL review
on:
  pull_request:
    paths: ["@@FILE_PATTERN@@"]
jobs:
  sql-review:
    # Reviews pull requests into the default branch, the branch that deploys.
    if: github.base_ref == github.event.repository.default_branch
    permissions:
      contents: read
      id-token: write
      pull-requests: write
    runs-on: ubuntu-latest
    container:
      image: @@ACTION_IMAGE@@
    env:
      BYTEBASE_URL: "@@BYTEBASE_URL@@"
      BYTEBASE_WORKLOAD_IDENTITY: @@WORKLOAD_IDENTITY@@
    steps:
      - uses: actions/checkout@v4
      - name: SQL review
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          bytebase-action check \
            --url=$BYTEBASE_URL \
            --project=projects/@@PROJECT_ID@@ \
            --targets=@@TARGETS@@ \
            --file-pattern=@@FILE_PATTERN@@
`

const githubDeployWorkflow = `name: Bytebase deploy
on:
  push:
    paths: ["@@FILE_PATTERN@@"]
env:
  BYTEBASE_URL: "@@BYTEBASE_URL@@"
  BYTEBASE_WORKLOAD_IDENTITY: @@WORKLOAD_IDENTITY@@
  BYTEBASE_PROJECT: projects/@@PROJECT_ID@@
jobs:
  create-rollout:
    # Deploys what is merged into the default branch.
    if: github.ref_name == github.event.repository.default_branch
    permissions:
      contents: read
      id-token: write
    runs-on: ubuntu-latest
    container:
      image: @@ACTION_IMAGE@@
    outputs:
      bytebase-plan: ${{ steps.set-output.outputs.plan }}
    steps:
      - uses: actions/checkout@v4
      - name: Create rollout
        run: |
          bytebase-action rollout \
            --url=$BYTEBASE_URL \
            --project=$BYTEBASE_PROJECT \
            --targets=@@TARGETS@@ \
            --file-pattern=@@FILE_PATTERN@@ \
            --output=${{ runner.temp }}/bytebase-metadata.json
      - name: Set output
        id: set-output
        run: |
          PLAN=$(jq -r .plan ${{ runner.temp }}/bytebase-metadata.json)
          echo "plan=$PLAN" >> $GITHUB_OUTPUT
`

const githubDeployJob = `  deploy-to-@@ENVIRONMENT@@:
    needs: [@@NEEDS@@]
    permissions:
      contents: read
      id-token: write
    runs-on: ubuntu-latest
    environment: @@ENVIRONMENT@@
    container:
      image: @@ACTION_IMAGE@@
    steps:
      - uses: actions/checkout@v4
      - name: Deploy to @@ENVIRONMENT@@
        run: |
          bytebase-action rollout \
            --url=$BYTEBASE_URL \
            --project=$BYTEBASE_PROJECT \
            --target-stage=environments/@@ENVIRONMENT@@ \
            --plan=${{ needs.create-rollout.outputs.bytebase-plan }}
`

const gitlabWorkflow = `# Bytebase review and deploy jobs. They run in the default stages test and deploy.
variables:
  BYTEBASE_URL: "@@BYTEBASE_URL@@"
  BYTEBASE_WORKLOAD_IDENTITY: @@WORKLOAD_IDENTITY@@
  BYTEBASE_PROJECT: projects/@@PROJECT_ID@@
  BYTEBASE_TARGETS: @@TARGETS_VARIABLE@@
  BYTEBASE_FILE_PATTERN: "@@FILE_PATTERN@@"
  BYTEBASE_ACTION_IMAGE: @@ACTION_IMAGE@@

bytebase-sql-review:
  stage: test
  image: $BYTEBASE_ACTION_IMAGE
  id_tokens:
    BYTEBASE_ID_TOKEN:
      aud: ` + identityAudience + `
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event" && $CI_MERGE_REQUEST_TARGET_BRANCH_NAME == $CI_DEFAULT_BRANCH
      changes:
        - @@FILE_PATTERN@@
  script:
    - bytebase-action check --url=$BYTEBASE_URL --project=$BYTEBASE_PROJECT --targets="$BYTEBASE_TARGETS" --file-pattern=$BYTEBASE_FILE_PATTERN

bytebase-create-rollout:
  stage: deploy
  image: $BYTEBASE_ACTION_IMAGE
  id_tokens:
    BYTEBASE_ID_TOKEN:
      aud: ` + identityAudience + `
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
      changes:
        - @@FILE_PATTERN@@
  script:
    - bytebase-action rollout --url=$BYTEBASE_URL --project=$BYTEBASE_PROJECT --targets="$BYTEBASE_TARGETS" --file-pattern=$BYTEBASE_FILE_PATTERN --output=bytebase-metadata.json
  artifacts:
    paths: [bytebase-metadata.json]
`

const gitlabDeployJob = `
bytebase-deploy-to-@@ENVIRONMENT@@:
  stage: deploy
  image: $BYTEBASE_ACTION_IMAGE
  needs: [@@NEEDS@@]
  id_tokens:
    BYTEBASE_ID_TOKEN:
      aud: ` + identityAudience + `
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
      changes:
        - @@FILE_PATTERN@@
  environment: @@ENVIRONMENT@@
  script:
    - PLAN=$(jq -r .plan bytebase-metadata.json)
    - bytebase-action rollout --url=$BYTEBASE_URL --project=$BYTEBASE_PROJECT --target-stage=environments/@@ENVIRONMENT@@ --plan=$PLAN
`
