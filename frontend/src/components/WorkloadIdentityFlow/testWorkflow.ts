// The one-off workflows the Done step offers, each of which exchanges a CI
// token for a Bytebase access token and prints the result. A refusal must fail
// the job and print the server's reason, which plain `curl -sS` does not do:
// it exits 0 on HTTP 401.

export const GITHUB_TEST_WORKFLOW_PATH =
  ".github/workflows/bytebase-wif-test.yml";
export const GITLAB_TEST_JOB_NAME = "bytebase-wif-test";

export interface TestWorkflowParams {
  externalUrl: string;
  email: string;
  audience: string;
}

const exchangeUrl = (externalUrl: string) =>
  `${externalUrl.replace(/\/+$/, "")}/v1/auth:exchangeToken`;

// Runs on a push of its own file, on any branch, so it needs no pull request
// to a protected default branch and stays quiet on other pushes.
// workflow_dispatch is not used: it only works from the default branch.
export const buildGitHubTestWorkflow = ({
  externalUrl,
  email,
  audience,
}: TestWorkflowParams) => `name: bytebase-wif-test
on:
  push:
    paths:
      - ${GITHUB_TEST_WORKFLOW_PATH}
permissions:
  id-token: write
  contents: read
jobs:
  sign-in:
    runs-on: ubuntu-latest
    steps:
      - name: Fetch GitHub OIDC token
        run: |
          OIDC_TOKEN=$(curl -sS --fail -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \\
            "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=${encodeURIComponent(audience)}" | jq -r .value)
          echo "::add-mask::$OIDC_TOKEN"
          echo "OIDC_TOKEN=$OIDC_TOKEN" >> $GITHUB_ENV
      - name: Exchange for a Bytebase access token
        run: |
          RESPONSE=$(curl -sS --fail-with-body ${exchangeUrl(externalUrl)} \\
            -H "Content-Type: application/json" \\
            -d '{"token":"'"$OIDC_TOKEN"'","email":"${email}"}') \\
            || { echo "Sign-in refused: $RESPONSE"; exit 1; }
          echo "$RESPONSE" | jq '.accessToken = "<redacted>"'
`;

export const buildGitLabTestJob = ({
  externalUrl,
  email,
  audience,
}: TestWorkflowParams) => `${GITLAB_TEST_JOB_NAME}:
  image: alpine:3.20
  when: manual
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: ${audience}
  script:
    - apk add --no-cache curl jq
    - |
      RESPONSE=$(curl -sS --fail-with-body ${exchangeUrl(externalUrl)} \\
        -H "Content-Type: application/json" \\
        -d "{\\"token\\":\\"$GITLAB_OIDC_TOKEN\\",\\"email\\":\\"${email}\\"}") \\
        || { echo "Sign-in refused: $RESPONSE"; exit 1; }
    - echo "$RESPONSE" | jq '.accessToken = "<redacted>"'
`;
