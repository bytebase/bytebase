import { load } from "js-yaml";
import { describe, expect, test } from "vitest";
import cases from "./gitops-workflow-cases/cases.json";
import {
  bytebaseActionImage,
  EXTERNAL_URL_PLACEHOLDER,
  GITLAB_INCLUDE,
  generateWorkflowFiles,
  isValidMigrationFolder,
  type WorkflowParams,
} from "./gitopsWorkflowYaml";

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

const workflow: WorkflowParams = {
  provider: "github",
  bytebaseUrl: "https://bytebase.example.com",
  actionImage: "bytebase/bytebase-action:3.14.0",
  workloadIdentityEmail: "deploy-ci@shop.workload.bytebase.com",
  projectId: "shop",
  targets: [
    "instances/pg-test/databases/shop_test",
    "instances/pg-prod/databases/shop_prod",
  ],
  environments: ["test", "staging", "prod"],
  folder: "migrations",
};

type Job = {
  if?: string;
  needs?: string | string[];
  environment?: string;
  image?: string;
  stage?: string;
  rules?: { if: string; changes: string[] }[];
  container?: { image: string };
  env?: Record<string, string>;
  permissions?: Record<string, string>;
  id_tokens?: Record<string, { aud: string }>;
  steps?: { uses?: string; run?: string }[];
  script?: string[];
};
type Workflow = {
  on: Record<string, { paths: string[] }>;
  env?: Record<string, string>;
  variables?: Record<string, string>;
  jobs: Record<string, Job>;
};

const parse = (content: string) => load(content) as Workflow;
// GitLab jobs are top-level keys next to `variables`.
const parsedGitlabJobs = (content: string) => {
  const { variables: _, ...jobs } = load(content) as Record<string, Job>;
  return jobs;
};

describe("generateWorkflowFiles", () => {
  test("GitHub gives a review file and a deploy file", () => {
    const files = generateWorkflowFiles(workflow);
    expect(files.map((f) => f.path)).toEqual([
      ".github/workflows/bytebase-review.yml",
      ".github/workflows/bytebase-deploy.yml",
    ]);
    for (const file of files) {
      expect(() => parse(file.content)).not.toThrow();
      expect(file.content).toContain("https://bytebase.example.com");
      expect(file.content).toContain("deploy-ci@shop.workload.bytebase.com");
      expect(file.content).toContain("bytebase/bytebase-action:3.14.0");
      expect(file.content).not.toContain("secrets.BYTEBASE");
    }
  });

  test("GitHub review runs on pull requests into the default branch", () => {
    const [review] = generateWorkflowFiles(workflow);
    const parsed = parse(review.content);
    expect(parsed.on.pull_request.paths).toEqual(["migrations/*.sql"]);
    expect(parsed.jobs["sql-review"].if).toBe(
      "github.base_ref == github.event.repository.default_branch"
    );
    const run = parsed.jobs["sql-review"].steps?.at(-1)?.run ?? "";
    expect(run).toContain("--project=projects/shop");
    expect(run).toContain(
      "--targets=instances/pg-test/databases/shop_test,instances/pg-prod/databases/shop_prod"
    );
    expect(run).toContain("--file-pattern=migrations/*.sql");
  });

  test("GitHub deploy has one job per environment, in deploy order", () => {
    const [, deploy] = generateWorkflowFiles(workflow);
    const parsed = parse(deploy.content);
    expect(Object.keys(parsed.jobs)).toEqual([
      "create-rollout",
      "deploy-to-test",
      "deploy-to-staging",
      "deploy-to-prod",
    ]);
    expect(parsed.jobs["create-rollout"].if).toBe(
      "github.ref_name == github.event.repository.default_branch"
    );
    expect(parsed.jobs["deploy-to-test"].needs).toEqual(["create-rollout"]);
    expect(parsed.jobs["deploy-to-staging"].needs).toEqual([
      "create-rollout",
      "deploy-to-test",
    ]);
    expect(parsed.jobs["deploy-to-prod"].needs).toEqual([
      "create-rollout",
      "deploy-to-staging",
    ]);
    expect(parsed.jobs["deploy-to-prod"].environment).toBe("prod");
    expect(deploy.content).toContain("--target-stage=environments/staging");
  });

  test("a database group is one target", () => {
    const [review] = generateWorkflowFiles({
      ...workflow,
      targets: ["projects/shop/databaseGroups/all-shops"],
    });
    expect(review.content).toContain(
      "--targets=projects/shop/databaseGroups/all-shops \\"
    );
  });

  test("a database name the shell would split is quoted on GitHub", () => {
    const files = generateWorkflowFiles({
      ...workflow,
      targets: [
        "instances/sql-prod/databases/Shop Orders",
        "instances/sql-prod/databases/o'brien",
      ],
    });
    for (const file of files) {
      expect(file.content).toContain(
        "--targets='instances/sql-prod/databases/Shop Orders,instances/sql-prod/databases/o'\\''brien' \\"
      );
    }
  });

  test("a database name GitLab would split or expand stays whole", () => {
    const targets = [
      "instances/sql-prod/databases/Shop Orders",
      "instances/sql-prod/databases/cost$center",
    ];
    const [file] = generateWorkflowFiles({
      ...workflow,
      provider: "gitlab",
      targets,
    });
    const parsed = load(file.content) as { variables: Record<string, string> };
    expect(parsed.variables.BYTEBASE_TARGETS).toBe(
      targets.join(",").replace("$", "$$$$")
    );
    const jobs = parsedGitlabJobs(file.content);
    for (const name of ["bytebase-sql-review", "bytebase-create-rollout"]) {
      expect(jobs[name].script?.join("\n")).toContain(
        '--targets="$BYTEBASE_TARGETS"'
      );
    }
  });

  test("GitLab gives one file with every job", () => {
    const files = generateWorkflowFiles({ ...workflow, provider: "gitlab" });
    expect(files.map((f) => f.path)).toEqual(["bytebase.gitlab-ci.yml"]);
    const parsed = load(files[0].content) as {
      variables: Record<string, string>;
    };
    expect(parsed.variables).toMatchObject({
      BYTEBASE_URL: "https://bytebase.example.com",
      BYTEBASE_PROJECT: "projects/shop",
      BYTEBASE_FILE_PATTERN: "migrations/*.sql",
      BYTEBASE_ACTION_IMAGE: "bytebase/bytebase-action:3.14.0",
    });
    const jobs = parsedGitlabJobs(files[0].content);
    expect(Object.keys(jobs)).toEqual([
      "bytebase-sql-review",
      "bytebase-create-rollout",
      "bytebase-deploy-to-test",
      "bytebase-deploy-to-staging",
      "bytebase-deploy-to-prod",
    ]);
    expect(jobs["bytebase-deploy-to-staging"].needs).toEqual([
      "bytebase-create-rollout",
      "bytebase-deploy-to-test",
    ]);
    // A deploy job must exist whenever the job it needs does.
    expect(jobs["bytebase-deploy-to-prod"].rules).toEqual(
      jobs["bytebase-create-rollout"].rules
    );
    expect(jobs["bytebase-sql-review"].rules?.[0].changes).toEqual([
      "migrations/*.sql",
    ]);
    expect(GITLAB_INCLUDE).toBe("include:\n  - local: bytebase.gitlab-ci.yml");
  });

  test("the folder sets the triggers and the file pattern", () => {
    const files = generateWorkflowFiles({ ...workflow, folder: "db/changes" });
    expect(parse(files[1].content).on.push.paths).toEqual(["db/changes/*.sql"]);
    expect(files[1].content).toContain("--file-pattern=db/changes/*.sql");
    expect(files[1].content).not.toContain("migrations/");
  });

  test("the URL placeholder still parses", () => {
    const [review] = generateWorkflowFiles({
      ...workflow,
      bytebaseUrl: EXTERNAL_URL_PLACEHOLDER,
    });
    const parsed = parse(review.content) as unknown as {
      jobs: Record<string, { env: Record<string, string> }>;
    };
    expect(parsed.jobs["sql-review"].env.BYTEBASE_URL).toBe("{EXTERNAL_URL}");
  });
});

describe("isValidMigrationFolder", () => {
  test.each([
    ["migrations", true],
    ["db/migrations", true],
    ["db/v1.2_changes", true],
    ["", false],
    ["/migrations", false],
    ["migrations/", false],
    ["db/../migrations", false],
    [".", false],
    ["my migrations", false],
    ["migrations/*", false],
  ])("%s is %s", (folder, valid) => {
    expect(isValidMigrationFolder(folder)).toBe(valid);
  });
});

describe("sign-in in the generated files", () => {
  const email = "deploy-ci@shop.workload.bytebase.com";

  // bytebase-action signs in by itself, so the files hold no exchange script.
  test.each([
    ["GitHub Actions", "github"],
    ["GitLab CI", "gitlab"],
  ] as const)(
    "%s: no job has a sign-in step or passes a token",
    (_, provider) => {
      const yaml = generateWorkflowFiles({ ...workflow, provider })
        .map((file) => file.content)
        .join("\n");
      expect(yaml).not.toContain("exchangeToken");
      expect(yaml).not.toContain("access-token");
      expect(yaml).not.toContain("ACCESS_TOKEN");
      expect(yaml).not.toContain("curl");
    }
  );

  test("every GitHub job that checks out the repository may read it", () => {
    const jobs = generateWorkflowFiles(workflow).flatMap((file) =>
      Object.values(parse(file.content).jobs)
    );
    expect(jobs).toHaveLength(5);
    for (const job of jobs) {
      expect(
        job.steps?.some((step) => step.uses === "actions/checkout@v4")
      ).toBe(true);
      expect(job.permissions?.contents).toBe("read");
    }
  });

  test("every GitHub job names the identity and may request an ID token", () => {
    const jobs = generateWorkflowFiles(workflow).flatMap((file) => {
      const parsed = parse(file.content);
      return Object.values(parsed.jobs).map((job) => ({
        job,
        identity:
          job.env?.BYTEBASE_WORKLOAD_IDENTITY ??
          parsed.env?.BYTEBASE_WORKLOAD_IDENTITY,
      }));
    });
    expect(jobs).toHaveLength(5);
    for (const { job, identity } of jobs) {
      expect(identity).toBe(email);
      expect(job.permissions?.["id-token"]).toBe("write");
    }
  });

  test("every GitLab job hands its ID token to the action", () => {
    const [file] = generateWorkflowFiles({ ...workflow, provider: "gitlab" });
    const parsed = load(file.content) as { variables: Record<string, string> };
    expect(parsed.variables.BYTEBASE_WORKLOAD_IDENTITY).toBe(email);
    const jobs = Object.values(parsedGitlabJobs(file.content));
    expect(jobs).toHaveLength(5);
    for (const job of jobs) {
      expect(job.id_tokens).toEqual({ BYTEBASE_ID_TOKEN: { aud: "bytebase" } });
    }
  });
});

// The Go generator behind the get_gitops_workflow MCP tool renders the same
// cases against the same expected files (backend/api/mcp/gitops_workflow_test.go).
describe("the cases shared with the MCP tool", () => {
  test.each(cases)("$name", async ({ name, params }) => {
    const rendered = generateWorkflowFiles(params as WorkflowParams)
      .map((file) => `=== ${file.path} ===\n${file.content}`)
      .join("");
    await expect(rendered).toMatchFileSnapshot(
      `./gitops-workflow-cases/${name}.txt`
    );
  });
});
