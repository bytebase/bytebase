import { describe, expect, test } from "vitest";
import {
  DATABASE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_INSTANCE_CREATE,
  PROJECT_V1_ROUTE_INSTANCES,
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_ISSUES,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
  WORKSPACE_ROUTE_MY_ISSUES,
} from "@/app/router/handles";
import {
  PREPARE_DATABASE_PRODUCT_INTRO,
  PREPARE_DATABASE_TRANSFER_TIP,
  PRODUCT_INTRO_QUERY_KEY,
  PRODUCT_INTRO_TIP_QUERY_KEY,
} from "@/lib/productIntro";
import { buildSearchParamsBySearchText } from "@/utils";
import {
  acceptAgentHandoff,
  collectAgentHandoffs,
  resolveAgentHandoffRoute,
} from "./handoff";

const project = "projects/sample";
const workspace = "workspaces/example";

describe("collectAgentHandoffs", () => {
  test("collects an unnumbered workspace review queue from a partial result", () => {
    expect(
      collectAgentHandoffs(
        "get_workspace_attention",
        JSON.stringify({
          issues: [
            {
              name: "projects/sample/issues/1",
              title: "Review change",
              relation: "review",
            },
          ],
          partial: true,
        }),
        { workspaceName: workspace }
      )
    ).toEqual([{ kind: "review-workspace-issues", workspace }]);
  });

  test("does not offer a queue from an unavailable or empty review facet", () => {
    expect(
      collectAgentHandoffs(
        "get_workspace_overview",
        JSON.stringify({
          workspace,
          attention: { status: "unavailable" },
        }),
        { workspaceName: workspace }
      )
    ).toEqual([]);
    expect(
      collectAgentHandoffs(
        "get_workspace_attention",
        JSON.stringify({ issues: [], partial: true }),
        { workspaceName: workspace }
      )
    ).toEqual([]);
  });

  test("requires a canonical in-scope Issue for queue evidence", () => {
    expect(
      collectAgentHandoffs(
        "get_project_open_issues",
        JSON.stringify({
          project,
          issues: [
            { name: "projects/other/issues/2", relation: "review" },
            { name: "not-an-issue", relation: "review" },
          ],
        }),
        { projectName: project }
      )
    ).toEqual([]);
    expect(
      collectAgentHandoffs(
        "get_workspace_attention",
        JSON.stringify({
          issues: [{ name: "not-an-issue", relation: "review" }],
        }),
        { workspaceName: workspace }
      )
    ).toEqual([]);
  });

  test("opens an available exact Issue and its verified linked Plan", () => {
    expect(
      collectAgentHandoffs(
        "get_issue_overview",
        JSON.stringify({
          project,
          issue: { status: "available", name: "projects/sample/issues/9" },
          plan: {
            status: "available",
            name: "projects/sample/plans/1",
            reviewOnPlan: true,
          },
        }),
        { projectName: project }
      )
    ).toEqual([
      { kind: "open-issue", issue: "projects/sample/issues/9" },
      { kind: "review-issue-plan", plan: "projects/sample/plans/1" },
    ]);
  });

  test("keeps the Issue link when the linked Plan redirects to Issue detail", () => {
    expect(
      collectAgentHandoffs(
        "get_issue_overview",
        JSON.stringify({
          project,
          issue: { status: "available", name: "projects/sample/issues/9" },
          plan: {
            status: "available",
            name: "projects/sample/plans/1",
            reviewOnPlan: false,
          },
          rollout: {
            status: "available",
            name: "projects/sample/plans/1/rollout",
            failedTasks: ["projects/sample/plans/1/rollout/stages/-/tasks/t1"],
          },
        }),
        { projectName: project }
      )
    ).toEqual([{ kind: "open-issue", issue: "projects/sample/issues/9" }]);
  });

  test("does not offer exact links from unavailable or out-of-scope facets", () => {
    for (const payload of [
      {
        project,
        issue: { status: "unavailable", name: "projects/sample/issues/9" },
      },
      {
        project,
        issue: { status: "available", name: "projects/other/issues/9" },
      },
      {
        project,
        issue: { status: "available", name: "projects/sample/issues/-" },
      },
    ]) {
      expect(
        collectAgentHandoffs("get_issue_overview", JSON.stringify(payload), {
          projectName: project,
        })
      ).toEqual([]);
    }
  });

  test("does not offer an unrelated or malformed linked Plan", () => {
    for (const planName of [
      "projects/other/plans/1",
      "projects/sample/plans/-",
      "projects/sample/plans/1/rollout",
    ]) {
      expect(
        collectAgentHandoffs(
          "get_issue_overview",
          JSON.stringify({
            project,
            issue: { status: "available", name: "projects/sample/issues/9" },
            plan: { status: "available", name: planName },
          }),
          { projectName: project }
        )
      ).toEqual([{ kind: "open-issue", issue: "projects/sample/issues/9" }]);
    }
  });

  test("collects project review and database-change queues from fresh results", () => {
    expect(
      collectAgentHandoffs(
        "get_project_overview",
        JSON.stringify({
          project,
          personalIssues: {
            status: "available",
            issues: [
              {
                name: "projects/sample/issues/1",
                relation: "review",
              },
            ],
          },
          openDatabaseChanges: {
            status: "available",
            issues: [{ name: "projects/sample/issues/2" }],
          },
          databases: { status: "available", databases: [] },
        }),
        { projectName: project }
      )
    ).toEqual([
      { kind: "review-project-issues", project },
      { kind: "review-database-changes", project },
    ]);
  });

  test("offers transfer and connect when both are eligible for an empty project", () => {
    expect(
      collectAgentHandoffs(
        "get_project_overview",
        JSON.stringify({
          project,
          databases: { status: "available", partial: false, databases: [] },
          setup: {
            isDefault: false,
            instance: "empty",
            transferableDatabase: "present",
            canConnectInstance: true,
            canTransferDatabases: true,
            canCreateDatabase: true,
          },
        }),
        { projectName: project }
      )
    ).toEqual([
      { kind: "transfer-project-databases", project },
      { kind: "connect-project-instance", project },
    ]);
  });

  test("offers both transfer and create when both setup flows are available", () => {
    expect(
      collectAgentHandoffs(
        "get_project_overview",
        JSON.stringify({
          project,
          databases: { status: "available", partial: false, databases: [] },
          setup: {
            isDefault: false,
            instance: "present",
            transferableDatabase: "present",
            canConnectInstance: true,
            canTransferDatabases: true,
            canCreateDatabase: true,
          },
        }),
        { projectName: project }
      )
    ).toEqual([
      { kind: "transfer-project-databases", project },
      { kind: "create-project-database", project },
      { kind: "connect-project-instance", project },
    ]);
  });

  test("offers eligible setup flows for an established project", () => {
    expect(
      collectAgentHandoffs(
        "get_project_overview",
        JSON.stringify({
          project,
          databases: {
            status: "available",
            partial: true,
            databases: [{ name: "instances/shared/databases/orders" }],
          },
          setup: {
            isDefault: false,
            instance: "present",
            transferableDatabase: "present",
            canConnectInstance: true,
            canTransferDatabases: true,
            canCreateDatabase: true,
          },
        }),
        { projectName: project }
      )
    ).toEqual([
      { kind: "transfer-project-databases", project },
      { kind: "create-project-database", project },
      { kind: "connect-project-instance", project },
    ]);
  });

  test.each([
    ["empty", "empty", ["connect-project-instance"]],
    [
      "present",
      "empty",
      ["create-project-database", "connect-project-instance"],
    ],
    ["unknown", "empty", ["connect-project-instance"]],
    ["empty", "unknown", ["connect-project-instance"]],
  ] as const)(
    "chooses setup from instance %s and transfer %s",
    (instance, transferableDatabase, expectedKinds) => {
      const result = collectAgentHandoffs(
        "get_project_overview",
        JSON.stringify({
          project,
          databases: { status: "available", partial: false, databases: [] },
          setup: {
            isDefault: false,
            instance,
            transferableDatabase,
            canConnectInstance: true,
            canTransferDatabases: true,
            canCreateDatabase: true,
          },
        }),
        { projectName: project }
      );
      expect(result).toEqual(expectedKinds.map((kind) => ({ kind, project })));
    }
  );

  test("does not infer setup from invalid or inaccessible state", () => {
    const overview = {
      project,
      databases: { status: "available", partial: false, databases: [] },
      setup: {
        isDefault: false,
        instance: "empty",
        transferableDatabase: "empty",
        canConnectInstance: true,
        canTransferDatabases: true,
        canCreateDatabase: true,
      },
    };
    for (const payload of [
      { ...overview, databases: { status: "unavailable" } },
      { ...overview, setup: undefined },
      { ...overview, setup: { ...overview.setup, isDefault: true } },
      { ...overview, setup: { ...overview.setup, canConnectInstance: false } },
    ]) {
      expect(
        collectAgentHandoffs("get_project_overview", JSON.stringify(payload), {
          projectName: project,
        })
      ).toEqual([]);
    }
  });

  test("rejects a mismatched project and failed task evidence without a rollout", () => {
    expect(
      collectAgentHandoffs(
        "get_project_overview",
        JSON.stringify({ project: "projects/other" }),
        { projectName: project }
      )
    ).toEqual([]);
    expect(
      collectAgentHandoffs(
        "get_issue_overview",
        JSON.stringify({
          project,
          issue: { status: "available", name: "projects/sample/issues/9" },
          plan: {
            status: "available",
            name: "projects/sample/plans/1",
            reviewOnPlan: true,
          },
          rollout: { status: "unavailable" },
          taskRuns: {
            status: "available",
            taskRuns: [{ name: "taskRuns/1", status: "FAILED" }],
          },
        }),
        { projectName: project }
      )
    ).toEqual([
      { kind: "open-issue", issue: "projects/sample/issues/9" },
      { kind: "review-issue-plan", plan: "projects/sample/plans/1" },
    ]);
  });

  test("offers an exact failed Task and broad Rollout only from current task status", () => {
    expect(
      collectAgentHandoffs(
        "get_issue_overview",
        JSON.stringify({
          project,
          issue: { status: "available", name: "projects/sample/issues/9" },
          plan: {
            status: "available",
            name: "projects/sample/plans/1",
            reviewOnPlan: true,
          },
          rollout: {
            status: "available",
            name: "projects/sample/plans/1/rollout",
            failedTasks: [
              "projects/sample/plans/1/rollout/stages/prod/tasks/t1",
            ],
          },
          taskRuns: {
            status: "available",
            taskRuns: [
              {
                name: "projects/sample/plans/1/rollout/stages/prod/tasks/t1/taskRuns/run-1",
                status: "FAILED",
              },
            ],
          },
        }),
        { projectName: project }
      )
    ).toEqual([
      { kind: "open-issue", issue: "projects/sample/issues/9" },
      { kind: "review-issue-plan", plan: "projects/sample/plans/1" },
      {
        kind: "view-failed-task",
        task: "projects/sample/plans/1/rollout/stages/prod/tasks/t1",
      },
      { kind: "view-failed-tasks", plan: "projects/sample/plans/1" },
    ]);
  });

  test("historical failed Task Runs do not produce failed-task links", () => {
    expect(
      collectAgentHandoffs(
        "get_issue_overview",
        JSON.stringify({
          project,
          issue: { status: "available", name: "projects/sample/issues/9" },
          plan: {
            status: "available",
            name: "projects/sample/plans/1",
            reviewOnPlan: true,
          },
          rollout: {
            status: "available",
            name: "projects/sample/plans/1/rollout",
            failedTasks: [],
          },
          taskRuns: {
            status: "available",
            taskRuns: [
              {
                name: "projects/sample/plans/1/rollout/stages/prod/tasks/t1/taskRuns/old",
                status: "FAILED",
              },
            ],
          },
        }),
        { projectName: project }
      )
    ).toEqual([
      { kind: "open-issue", issue: "projects/sample/issues/9" },
      { kind: "review-issue-plan", plan: "projects/sample/plans/1" },
    ]);
  });

  test("accepts a canonical failed Task under the default Rollout stage", () => {
    expect(
      collectAgentHandoffs(
        "get_issue_overview",
        JSON.stringify({
          project,
          issue: { status: "available", name: "projects/sample/issues/9" },
          plan: {
            status: "available",
            name: "projects/sample/plans/1",
            reviewOnPlan: true,
          },
          rollout: {
            status: "available",
            name: "projects/sample/plans/1/rollout",
            failedTasks: ["projects/sample/plans/1/rollout/stages/-/tasks/t1"],
          },
        }),
        { projectName: project }
      )
    ).toContainEqual({
      kind: "view-failed-task",
      task: "projects/sample/plans/1/rollout/stages/-/tasks/t1",
    });
  });

  test("ignores failed-task names outside the verified Rollout", () => {
    const malformed = [
      "projects/other/plans/1/rollout/stages/prod/tasks/t1",
      "projects/sample/plans/other/rollout/stages/prod/tasks/t1",
      "projects/sample/plans/1/rollout/stages/prod/tasks/-",
      "projects/sample/plans/1/rollout/stages/prod/tasks/t1/taskRuns/r1",
    ];
    expect(
      collectAgentHandoffs(
        "get_issue_overview",
        JSON.stringify({
          project,
          issue: { status: "available", name: "projects/sample/issues/9" },
          plan: {
            status: "available",
            name: "projects/sample/plans/1",
            reviewOnPlan: true,
          },
          rollout: {
            status: "available",
            name: "projects/sample/plans/1/rollout",
            failedTasks: malformed,
          },
        }),
        { projectName: project }
      )
    ).toEqual([
      { kind: "open-issue", issue: "projects/sample/issues/9" },
      { kind: "review-issue-plan", plan: "projects/sample/plans/1" },
    ]);
  });
});

describe("acceptAgentHandoff", () => {
  test("accepts only an exact currently observed kind and target", () => {
    const observed = [
      { kind: "review-database-changes" as const, project },
      { kind: "review-database-changes" as const, project },
    ];
    expect(
      acceptAgentHandoff({ kind: "review-database-changes", project }, observed)
    ).toEqual(observed[0]);
    expect(
      acceptAgentHandoff(
        { kind: "review-database-changes", project: "projects/other" },
        observed
      )
    ).toBeUndefined();
    expect(
      acceptAgentHandoff(
        {
          kind: "review-database-changes",
          project,
          href: "https://example.com",
        },
        observed
      )
    ).toBeUndefined();
  });

  test("does not accept unobserved exact Issue, Plan, or Task links", () => {
    const issue = "projects/sample/issues/9";
    const plan = "projects/sample/plans/7";
    const task = `${plan}/rollout/stages/-/tasks/t1`;
    const observed = [
      { kind: "open-issue" as const, issue },
      { kind: "review-issue-plan" as const, plan },
      { kind: "view-failed-task" as const, task },
    ];
    for (const target of observed) {
      expect(acceptAgentHandoff(target, observed)).toEqual(target);
    }
    expect(
      acceptAgentHandoff(
        { kind: "view-failed-task", task: `${plan}/rollout/stages/-/tasks/t2` },
        observed
      )
    ).toBeUndefined();
  });
});

describe("resolveAgentHandoffRoute", () => {
  test("maps only canonical targets to registered routes", () => {
    expect(
      resolveAgentHandoffRoute({ kind: "review-workspace-issues", workspace })
    ).toEqual({ name: WORKSPACE_ROUTE_MY_ISSUES });
    expect(
      resolveAgentHandoffRoute({ kind: "open-project-instances", project })
    ).toEqual({
      name: PROJECT_V1_ROUTE_INSTANCES,
      params: { projectId: "sample" },
    });
    expect(
      resolveAgentHandoffRoute({ kind: "connect-project-instance", project })
    ).toEqual({
      name: PROJECT_V1_ROUTE_INSTANCE_CREATE,
      params: { projectId: "sample" },
    });
    expect(
      resolveAgentHandoffRoute({ kind: "create-project-database", project })
    ).toEqual({
      name: PROJECT_V1_ROUTE_DATABASES,
      params: { projectId: "sample" },
      query: { createDatabase: "1" },
    });
    expect(
      resolveAgentHandoffRoute({ kind: "transfer-project-databases", project })
    ).toEqual({
      name: DATABASE_ROUTE_DASHBOARD,
      query: {
        [PRODUCT_INTRO_QUERY_KEY]: PREPARE_DATABASE_PRODUCT_INTRO,
        [PRODUCT_INTRO_TIP_QUERY_KEY]: PREPARE_DATABASE_TRANSFER_TIP,
        targetProject: project,
      },
    });
    expect(
      resolveAgentHandoffRoute({
        kind: "view-failed-tasks",
        plan: "projects/sample/plans/7",
      })
    ).toEqual({
      name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
      params: { projectId: "sample", planId: "7" },
    });
    expect(
      resolveAgentHandoffRoute({
        kind: "open-issue",
        issue: "projects/sample/issues/9",
      })
    ).toEqual({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: { projectId: "sample", issueId: "9" },
    });
    expect(
      resolveAgentHandoffRoute({
        kind: "review-issue-plan",
        plan: "projects/sample/plans/7",
      })
    ).toEqual({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: { projectId: "sample", planId: "7" },
      query: { phase: "review" },
    });
    expect(
      resolveAgentHandoffRoute({
        kind: "view-failed-task",
        task: "projects/sample/plans/7/rollout/stages/-/tasks/t1",
      })
    ).toEqual({
      name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      params: {
        projectId: "sample",
        planId: "7",
        stageId: "-",
        taskId: "t1",
      },
    });
    expect(
      resolveAgentHandoffRoute({
        kind: "review-database-changes",
        project: "projects/-",
      })
    ).toBeUndefined();
    for (const invalid of [
      { kind: "open-issue", issue: "projects/-/issues/9" },
      { kind: "review-issue-plan", plan: "projects/sample/plans/-" },
      {
        kind: "view-failed-task",
        task: "projects/sample/plans/7/rollout/stages/prod/tasks/-",
      },
    ]) {
      expect(resolveAgentHandoffRoute(invalid)).toBeUndefined();
    }
  });

  test("uses the Issue page's q filter for open Database Changes", () => {
    const target = resolveAgentHandoffRoute({
      kind: "review-database-changes",
      project,
    });
    expect(target).toMatchObject({
      name: PROJECT_V1_ROUTE_ISSUES,
      params: { projectId: "sample" },
    });
    if (!target || typeof target === "string" || !("query" in target)) {
      throw new Error("Missing query");
    }
    const params = buildSearchParamsBySearchText(String(target.query?.q));
    expect(params.scopes).toEqual(
      expect.arrayContaining([
        { id: "status", value: "OPEN" },
        { id: "issue-type", value: "DATABASE_CHANGE" },
      ])
    );
  });
});
