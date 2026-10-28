import { describe, expect, test } from "vitest";
import {
  bytebaseActionImage,
  generateGithubReleaseYaml,
  generateGithubSqlReviewYaml,
  generateGitlabCiYaml,
  type YamlParams,
} from "./gitopsWorkflowYaml";

const params: YamlParams = {
  branch: "main",
  runsOn: "ubuntu-latest",
  bytebaseUrl: "https://bytebase.example.com",
  actionImage: "bytebase/bytebase-action:cloud",
  workloadIdentityEmail: "ci@example.workload.bytebase.com",
  projectId: "example",
  targetsPlaceholder: "instances/example/databases/example",
};

// Top-level GitLab job names map to their indented bodies.
const gitlabJobs = (yaml: string): Record<string, string> => {
  const jobs: Record<string, string> = {};
  for (const block of yaml.split(/\n(?=\S)/)) {
    const [header, ...body] = block.split("\n");
    jobs[header.replace(/:$/, "")] = body.join("\n");
  }
  return jobs;
};

describe("bytebaseActionImage", () => {
  test("uses the cloud image for a Cloud server", () => {
    expect(bytebaseActionImage("cloud-20261023")).toBe(
      "bytebase/bytebase-action:cloud"
    );
  });

  test("pins the self-hosted server version", () => {
    expect(bytebaseActionImage("3.24.0")).toBe(
      "bytebase/bytebase-action:3.24.0"
    );
  });

  test("falls back to the untagged image for an unparsable version", () => {
    expect(bytebaseActionImage("development")).toBe("bytebase/bytebase-action");
    expect(bytebaseActionImage("")).toBe("bytebase/bytebase-action");
  });
});

describe("generated workflows", () => {
  test.each([
    ["GitHub SQL review", generateGithubSqlReviewYaml, 1],
    ["GitHub release", generateGithubReleaseYaml, 3],
    ["GitLab CI", generateGitlabCiYaml, 4],
  ])("%s uses the action image for every job", (_, generate, jobCount) => {
    const images = generate(params).match(/image: .+/g);
    expect(images).toEqual(
      Array(jobCount).fill("image: bytebase/bytebase-action:cloud")
    );
  });

  test("GitLab deploy jobs receive the rollout metadata artifact", () => {
    const jobs = gitlabJobs(generateGitlabCiYaml(params));
    expect(jobs["create-rollout"]).toContain(
      "artifacts:\n    paths: [bytebase-metadata.json]"
    );
    expect(jobs["deploy-to-test"]).toContain("needs: [create-rollout]");
    expect(jobs["deploy-to-prod"]).toContain(
      "needs: [create-rollout, deploy-to-test]"
    );
  });
});
