import semver from "semver";
import { GENERATED_WORKFLOW_AUDIENCE } from "@/utils/workloadIdentity";

// bytebase-action exits when its version is incompatible with the server's: a
// Cloud server needs the :cloud image, and a self-hosted server rejects a newer
// minor release, which the untagged `latest` image can be.
export function bytebaseActionImage(serverVersion: string): string {
  if (serverVersion.startsWith("cloud")) {
    return "bytebase/bytebase-action:cloud";
  }
  const version = semver.valid(serverVersion);
  return version
    ? `bytebase/bytebase-action:${version}`
    : "bytebase/bytebase-action";
}

// The folder the generated workflows watch for migration files, unless the
// owner changes it.
export const DEFAULT_MIGRATION_FOLDER = "migrations";

// Written in place of the Bytebase URL while the workspace External URL is not
// set. The page shows it marked and does not offer the files for copying.
export const EXTERNAL_URL_PLACEHOLDER = "{EXTERNAL_URL}";

// A relative folder path: segments of letters, digits, ".", "_" and "-",
// joined by "/", with no "." or ".." segment.
export function isValidMigrationFolder(folder: string): boolean {
  if (!/^[A-Za-z0-9._-]+(?:\/[A-Za-z0-9._-]+)*$/.test(folder)) return false;
  return folder.split("/").every((part) => part !== "." && part !== "..");
}

export type WorkflowProvider = "github" | "gitlab";

export interface WorkflowParams {
  provider: WorkflowProvider;
  // The workspace External URL, or EXTERNAL_URL_PLACEHOLDER.
  bytebaseUrl: string;
  actionImage: string;
  workloadIdentityEmail: string;
  projectId: string;
  // One database group, or one or more databases, as resource names.
  targets: string[];
  // Environment IDs in deploy order, one deploy job each.
  environments: string[];
  folder: string;
}

export interface WorkflowFile {
  path: string;
  content: string;
}

export const GITLAB_WORKFLOW_PATH = "bytebase.gitlab-ci.yml";

// The line the owner adds to the project's own .gitlab-ci.yml.
export const GITLAB_INCLUDE = `include:
  - local: ${GITLAB_WORKFLOW_PATH}`;

// Every file the owner adds for the given git provider, in the order shown.
//
// No job has a sign-in step: bytebase-action signs in as
// BYTEBASE_WORKLOAD_IDENTITY with the job's own ID token. On GitHub it requests
// the token itself, which needs `id-token: write`. On GitLab the job hands it
// over as BYTEBASE_ID_TOKEN.
export function generateWorkflowFiles(p: WorkflowParams): WorkflowFile[] {
  if (p.provider === "gitlab") {
    return [{ path: GITLAB_WORKFLOW_PATH, content: gitlabWorkflow(p) }];
  }
  return [
    {
      path: ".github/workflows/bytebase-review.yml",
      content: githubReviewWorkflow(p),
    },
    {
      path: ".github/workflows/bytebase-deploy.yml",
      content: githubDeployWorkflow(p),
    },
  ];
}

const filePattern = (p: WorkflowParams) => `${p.folder}/*.sql`;
const targetList = (p: WorkflowParams) => p.targets.join(",");

function githubReviewWorkflow(p: WorkflowParams): string {
  return `name: Bytebase SQL review
on:
  pull_request:
    paths: ["${filePattern(p)}"]
jobs:
  sql-review:
    # Reviews pull requests into the default branch, the branch that deploys.
    if: github.base_ref == github.event.repository.default_branch
    permissions:
      id-token: write
      pull-requests: write
    runs-on: ubuntu-latest
    container:
      image: ${p.actionImage}
    env:
      BYTEBASE_URL: "${p.bytebaseUrl}"
      BYTEBASE_WORKLOAD_IDENTITY: ${p.workloadIdentityEmail}
    steps:
      - uses: actions/checkout@v4
      - name: SQL review
        env:
          GITHUB_TOKEN: \${{ secrets.GITHUB_TOKEN }}
        run: |
          bytebase-action check \\
            --url=$BYTEBASE_URL \\
            --project=projects/${p.projectId} \\
            --targets=${targetList(p)} \\
            --file-pattern=${filePattern(p)}
`;
}

const githubDeployJob = (
  p: WorkflowParams,
  environment: string,
  previous: string[]
) => `  deploy-to-${environment}:
    needs: [${previous.join(", ")}]
    permissions:
      id-token: write
    runs-on: ubuntu-latest
    environment: ${environment}
    container:
      image: ${p.actionImage}
    steps:
      - uses: actions/checkout@v4
      - name: Deploy to ${environment}
        run: |
          bytebase-action rollout \\
            --url=$BYTEBASE_URL \\
            --project=$BYTEBASE_PROJECT \\
            --target-stage=environments/${environment} \\
            --plan=\${{ needs.create-rollout.outputs.bytebase-plan }}
`;

function githubDeployWorkflow(p: WorkflowParams): string {
  const deployJobs = p.environments.map((environment, i) =>
    githubDeployJob(
      p,
      environment,
      i === 0
        ? ["create-rollout"]
        : ["create-rollout", `deploy-to-${p.environments[i - 1]}`]
    )
  );
  return `name: Bytebase deploy
on:
  push:
    paths: ["${filePattern(p)}"]
env:
  BYTEBASE_URL: "${p.bytebaseUrl}"
  BYTEBASE_WORKLOAD_IDENTITY: ${p.workloadIdentityEmail}
  BYTEBASE_PROJECT: projects/${p.projectId}
jobs:
  create-rollout:
    # Deploys what is merged into the default branch.
    if: github.ref_name == github.event.repository.default_branch
    permissions:
      id-token: write
    runs-on: ubuntu-latest
    container:
      image: ${p.actionImage}
    outputs:
      bytebase-plan: \${{ steps.set-output.outputs.plan }}
    steps:
      - uses: actions/checkout@v4
      - name: Create rollout
        run: |
          bytebase-action rollout \\
            --url=$BYTEBASE_URL \\
            --project=$BYTEBASE_PROJECT \\
            --targets=${targetList(p)} \\
            --file-pattern=${filePattern(p)} \\
            --output=\${{ runner.temp }}/bytebase-metadata.json
      - name: Set output
        id: set-output
        run: |
          PLAN=$(jq -r .plan \${{ runner.temp }}/bytebase-metadata.json)
          echo "plan=$PLAN" >> $GITHUB_OUTPUT
${deployJobs.join("")}`;
}

const gitlabRules = (p: WorkflowParams, condition: string) => `  rules:
    - if: ${condition}
      changes:
        - ${filePattern(p)}`;

const gitlabDeployJob = (
  p: WorkflowParams,
  environment: string,
  previous: string[]
) => `
bytebase-deploy-to-${environment}:
  stage: deploy
  image: $BYTEBASE_ACTION_IMAGE
  needs: [${previous.join(", ")}]
  id_tokens:
    BYTEBASE_ID_TOKEN:
      aud: ${GENERATED_WORKFLOW_AUDIENCE}
${gitlabRules(p, "$CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH")}
  environment: ${environment}
  script:
    - PLAN=$(jq -r .plan bytebase-metadata.json)
    - bytebase-action rollout --url=$BYTEBASE_URL --project=$BYTEBASE_PROJECT --target-stage=environments/${environment} --plan=$PLAN
`;

function gitlabWorkflow(p: WorkflowParams): string {
  const deployJobs = p.environments.map((environment, i) =>
    gitlabDeployJob(
      p,
      environment,
      i === 0
        ? ["bytebase-create-rollout"]
        : [
            "bytebase-create-rollout",
            `bytebase-deploy-to-${p.environments[i - 1]}`,
          ]
    )
  );
  return `# Bytebase review and deploy jobs. They run in the default stages test and deploy.
variables:
  BYTEBASE_URL: "${p.bytebaseUrl}"
  BYTEBASE_WORKLOAD_IDENTITY: ${p.workloadIdentityEmail}
  BYTEBASE_PROJECT: projects/${p.projectId}
  BYTEBASE_TARGETS: ${targetList(p)}
  BYTEBASE_FILE_PATTERN: "${filePattern(p)}"
  BYTEBASE_ACTION_IMAGE: ${p.actionImage}

bytebase-sql-review:
  stage: test
  image: $BYTEBASE_ACTION_IMAGE
  id_tokens:
    BYTEBASE_ID_TOKEN:
      aud: ${GENERATED_WORKFLOW_AUDIENCE}
${gitlabRules(p, '$CI_PIPELINE_SOURCE == "merge_request_event" && $CI_MERGE_REQUEST_TARGET_BRANCH_NAME == $CI_DEFAULT_BRANCH')}
  script:
    - bytebase-action check --url=$BYTEBASE_URL --project=$BYTEBASE_PROJECT --targets=$BYTEBASE_TARGETS --file-pattern=$BYTEBASE_FILE_PATTERN

bytebase-create-rollout:
  stage: deploy
  image: $BYTEBASE_ACTION_IMAGE
  id_tokens:
    BYTEBASE_ID_TOKEN:
      aud: ${GENERATED_WORKFLOW_AUDIENCE}
${gitlabRules(p, "$CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH")}
  script:
    - bytebase-action rollout --url=$BYTEBASE_URL --project=$BYTEBASE_PROJECT --targets=$BYTEBASE_TARGETS --file-pattern=$BYTEBASE_FILE_PATTERN --output=bytebase-metadata.json
  artifacts:
    paths: [bytebase-metadata.json]
${deployJobs.join("")}`;
}
