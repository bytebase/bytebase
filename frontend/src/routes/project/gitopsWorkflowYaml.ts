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

export interface YamlParams {
  branch: string;
  runsOn?: string;
  bytebaseUrl: string;
  actionImage: string;
  workloadIdentityEmail: string;
  projectId: string;
  targetsPlaceholder: string;
}

const exchangeTokenStep = (indent: string) =>
  `${indent}- name: Exchange token
${indent}  id: bytebase-auth
${indent}  run: |
${indent}    OIDC_TOKEN=$(curl -s -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \\
${indent}      "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=${GENERATED_WORKFLOW_AUDIENCE}" | jq -r '.value')
${indent}    ACCESS_TOKEN=$(curl -s -X POST "$BYTEBASE_URL/v1/auth:exchangeToken" \\
${indent}      -H "Content-Type: application/json" \\
${indent}      -d "{\\"token\\":\\"$OIDC_TOKEN\\",\\"email\\":\\"$BYTEBASE_WORKLOAD_IDENTITY\\"}" \\
${indent}      | jq -r '.accessToken')
${indent}    echo "access-token=$ACCESS_TOKEN" >> $GITHUB_OUTPUT`;

const accessTokenFlag =
  "--access-token=${{ steps.bytebase-auth.outputs.access-token }}";

export function generateGithubSqlReviewYaml(p: YamlParams): string {
  return `name: SQL Review
on:
  pull_request:
    branches: ["${p.branch}"]
    paths: ["migrations/*.sql"]
jobs:
  sql-review:
    permissions:
      id-token: write
      pull-requests: write
    runs-on: ${p.runsOn}
    container:
      image: ${p.actionImage}
    env:
      BYTEBASE_URL: ${p.bytebaseUrl}
      BYTEBASE_WORKLOAD_IDENTITY: ${p.workloadIdentityEmail}
    steps:
      - uses: actions/checkout@v4
${exchangeTokenStep("      ")}
      - name: SQL Review
        env:
          GITHUB_TOKEN: \${{ secrets.GITHUB_TOKEN }}
        run: |
          bytebase-action check \\
            --url=$BYTEBASE_URL \\
            ${accessTokenFlag} \\
            --project=projects/${p.projectId} \\
            --targets=${p.targetsPlaceholder} \\
            --file-pattern=migrations/*.sql`;
}

export function generateGithubReleaseYaml(p: YamlParams): string {
  return `name: Rollout
on:
  push:
    branches: ["${p.branch}"]
    paths: ["migrations/*.sql"]
env:
  BYTEBASE_URL: ${p.bytebaseUrl}
  BYTEBASE_WORKLOAD_IDENTITY: ${p.workloadIdentityEmail}
  BYTEBASE_PROJECT: projects/${p.projectId}
jobs:
  build:
    runs-on: ${p.runsOn}
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: echo "Building..."
  create-rollout:
    needs: build
    permissions:
      id-token: write
    runs-on: ${p.runsOn}
    container:
      image: ${p.actionImage}
    outputs:
      bytebase-plan: \${{ steps.set-output.outputs.plan }}
    steps:
      - uses: actions/checkout@v4
${exchangeTokenStep("      ")}
      - name: Create rollout
        run: |
          bytebase-action rollout \\
            --url=$BYTEBASE_URL \\
            ${accessTokenFlag} \\
            --project=$BYTEBASE_PROJECT \\
            --targets=${p.targetsPlaceholder} \\
            --file-pattern=migrations/*.sql \\
            --output=\${{ runner.temp }}/bytebase-metadata.json
      - name: Set output
        id: set-output
        run: |
          PLAN=$(jq -r .plan \${{ runner.temp }}/bytebase-metadata.json)
          echo "plan=$PLAN" >> $GITHUB_OUTPUT
  deploy-to-test:
    needs: create-rollout
    permissions:
      id-token: write
    runs-on: ${p.runsOn}
    environment: test
    container:
      image: ${p.actionImage}
    steps:
      - uses: actions/checkout@v4
${exchangeTokenStep("      ")}
      - name: Deploy to test
        run: |
          bytebase-action rollout \\
            --url=$BYTEBASE_URL \\
            ${accessTokenFlag} \\
            --project=$BYTEBASE_PROJECT \\
            --target-stage=environments/test \\
            --plan=\${{ needs.create-rollout.outputs.bytebase-plan }}
  deploy-to-prod:
    needs: [deploy-to-test, create-rollout]
    permissions:
      id-token: write
    runs-on: ${p.runsOn}
    environment: prod
    container:
      image: ${p.actionImage}
    steps:
      - uses: actions/checkout@v4
${exchangeTokenStep("      ")}
      - name: Deploy to prod
        run: |
          bytebase-action rollout \\
            --url=$BYTEBASE_URL \\
            ${accessTokenFlag} \\
            --project=$BYTEBASE_PROJECT \\
            --target-stage=environments/prod \\
            --plan=\${{ needs.create-rollout.outputs.bytebase-plan }}`;
}

const gitlabExchangeScript = `    - |
      ACCESS_TOKEN=$(curl -s -X POST "$BYTEBASE_URL/v1/auth:exchangeToken" \\
        -H "Content-Type: application/json" \\
        -d "{\\"token\\":\\"$GITLAB_OIDC_TOKEN\\",\\"email\\":\\"$BYTEBASE_WORKLOAD_IDENTITY\\"}" \\
        | jq -r '.accessToken')
      export BYTEBASE_ACCESS_TOKEN=$ACCESS_TOKEN`;

export function generateGitlabCiYaml(p: YamlParams): string {
  return `stages:
  - sql-review
  - create-rollout
  - deploy-to-test
  - deploy-to-prod

variables:
  BYTEBASE_URL: ${p.bytebaseUrl}
  BYTEBASE_WORKLOAD_IDENTITY: ${p.workloadIdentityEmail}
  BYTEBASE_PROJECT: projects/${p.projectId}
  BYTEBASE_TARGETS: ${p.targetsPlaceholder}
  FILE_PATTERN: "migrations/*.sql"

sql-review:
  stage: sql-review
  image: ${p.actionImage}
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: ${GENERATED_WORKFLOW_AUDIENCE}
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
  script:
${gitlabExchangeScript}
    - bytebase-action check --url=$BYTEBASE_URL --access-token=$BYTEBASE_ACCESS_TOKEN --project=$BYTEBASE_PROJECT --targets=$BYTEBASE_TARGETS --file-pattern=$FILE_PATTERN

create-rollout:
  stage: create-rollout
  image: ${p.actionImage}
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: ${GENERATED_WORKFLOW_AUDIENCE}
  rules:
    - if: $CI_COMMIT_BRANCH == "${p.branch}"
  script:
${gitlabExchangeScript}
    - bytebase-action rollout --url=$BYTEBASE_URL --access-token=$BYTEBASE_ACCESS_TOKEN --project=$BYTEBASE_PROJECT --targets=$BYTEBASE_TARGETS --file-pattern=$FILE_PATTERN --output=bytebase-metadata.json
  artifacts:
    paths: [bytebase-metadata.json]

deploy-to-test:
  stage: deploy-to-test
  image: ${p.actionImage}
  needs: [create-rollout]
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: ${GENERATED_WORKFLOW_AUDIENCE}
  rules:
    - if: $CI_COMMIT_BRANCH == "${p.branch}"
  environment: test
  script:
${gitlabExchangeScript}
    - PLAN=$(jq -r .plan bytebase-metadata.json)
    - bytebase-action rollout --url=$BYTEBASE_URL --access-token=$BYTEBASE_ACCESS_TOKEN --project=$BYTEBASE_PROJECT --target-stage=environments/test --plan=$PLAN

deploy-to-prod:
  stage: deploy-to-prod
  image: ${p.actionImage}
  needs: [create-rollout, deploy-to-test]
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: ${GENERATED_WORKFLOW_AUDIENCE}
  rules:
    - if: $CI_COMMIT_BRANCH == "${p.branch}"
  environment: prod
  when: manual
  script:
${gitlabExchangeScript}
    - PLAN=$(jq -r .plan bytebase-metadata.json)
    - bytebase-action rollout --url=$BYTEBASE_URL --access-token=$BYTEBASE_ACCESS_TOKEN --project=$BYTEBASE_PROJECT --target-stage=environments/prod --plan=$PLAN`;
}
