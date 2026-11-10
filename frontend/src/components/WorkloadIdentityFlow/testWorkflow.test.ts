import { load } from "js-yaml";
import { describe, expect, test } from "vitest";
import {
  buildGitHubTestWorkflow,
  buildGitLabTestJob,
  GITHUB_TEST_WORKFLOW_PATH,
  GITLAB_TEST_JOB_NAME,
} from "./testWorkflow";

const params = {
  externalUrl: "https://bytebase.example.com/",
  email: "deploy@acme-deploy.workload.bytebase.com",
  audience: "bytebase",
};

describe("buildGitHubTestWorkflow", () => {
  test("is a workflow that runs on a push of its own file", () => {
    const workflow = load(buildGitHubTestWorkflow(params)) as {
      on: { push: { paths: string[] } };
      permissions: Record<string, string>;
      jobs: { "sign-in": { steps: { run: string }[] } };
    };
    expect(workflow.on.push.paths).toEqual([GITHUB_TEST_WORKFLOW_PATH]);
    expect(workflow.permissions["id-token"]).toBe("write");

    const [fetchToken, exchange] = workflow.jobs["sign-in"].steps;
    expect(fetchToken.run).toContain("&audience=bytebase");
    expect(exchange.run).toContain(
      "--fail-with-body https://bytebase.example.com/v1/auth:exchangeToken"
    );
    expect(exchange.run).toContain(`"email":"${params.email}"`);
    expect(exchange.run).toContain("Sign-in refused");
  });
});

describe("buildGitLabTestJob", () => {
  test("is a manual job that requests the identity's audience", () => {
    const job = load(buildGitLabTestJob(params)) as Record<
      string,
      {
        image: string;
        when: string;
        id_tokens: { GITLAB_OIDC_TOKEN: { aud: string } };
        script: string[];
      }
    >;
    const testJob = job[GITLAB_TEST_JOB_NAME];
    expect(testJob.when).toBe("manual");
    expect(testJob.image).toBe("alpine:3.20");
    expect(testJob.id_tokens.GITLAB_OIDC_TOKEN.aud).toBe("bytebase");

    const exchange = testJob.script[1];
    expect(exchange).toContain(
      "--fail-with-body https://bytebase.example.com/v1/auth:exchangeToken"
    );
    expect(exchange).toContain(
      `{\\"token\\":\\"$GITLAB_OIDC_TOKEN\\",\\"email\\":\\"${params.email}\\"}`
    );
    expect(exchange).toContain("Sign-in refused");
    expect(testJob.script[2]).toBe(
      `echo "$RESPONSE" | jq '.accessToken = "<redacted>"'`
    );
  });
});
