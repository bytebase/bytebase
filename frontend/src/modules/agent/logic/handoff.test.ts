import { describe, expect, test } from "vitest";
import {
  DATABASE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_INSTANCE_CREATE,
  PROJECT_V1_ROUTE_INSTANCES,
  PROJECT_V1_ROUTE_ISSUES,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
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

  test("requires a failed Task Run in the observed Rollout", () => {
    expect(
      collectAgentHandoffs(
        "get_issue_overview",
        JSON.stringify({
          project,
          plan: { status: "available", name: "projects/sample/plans/1" },
          rollout: {
            status: "available",
            name: "projects/sample/plans/1/rollout",
          },
          taskRuns: {
            status: "available",
            taskRuns: [
              {
                name: "projects/other/plans/1/rollout/stages/1/tasks/1/taskRuns/1",
                status: "FAILED",
              },
            ],
          },
        }),
        { projectName: project }
      )
    ).toEqual([]);
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

  test("prefers an eligible database transfer for an empty project", () => {
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
    ).toEqual([{ kind: "transfer-project-databases", project }]);
  });

  test.each([
    ["empty", "empty", "connect-project-instance"],
    ["present", "empty", "create-project-database"],
    ["unknown", "empty", undefined],
    ["empty", "unknown", "connect-project-instance"],
  ] as const)(
    "chooses setup from instance %s and transfer %s",
    (instance, transferableDatabase, expectedKind) => {
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
      expect(result).toEqual(
        expectedKind ? [{ kind: expectedKind, project }] : []
      );
    }
  );

  test("does not infer setup without complete empty-project evidence", () => {
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
      { ...overview, databases: { ...overview.databases, partial: true } },
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
          plan: { status: "available", name: "projects/sample/plans/1" },
          rollout: { status: "unavailable" },
          taskRuns: {
            status: "available",
            taskRuns: [{ name: "taskRuns/1", status: "FAILED" }],
          },
        }),
        { projectName: project }
      )
    ).toEqual([]);
  });

  test("collects a Plan rollout only for an observed failed Task Run", () => {
    expect(
      collectAgentHandoffs(
        "get_issue_overview",
        JSON.stringify({
          project,
          plan: { status: "available", name: "projects/sample/plans/1" },
          rollout: {
            status: "available",
            name: "projects/sample/plans/1/rollout",
          },
          taskRuns: {
            status: "available",
            taskRuns: [
              {
                name: "projects/sample/plans/1/rollout/stages/1/tasks/1/taskRuns/1",
                status: "FAILED",
              },
            ],
          },
        }),
        { projectName: project }
      )
    ).toEqual([{ kind: "view-failed-tasks", plan: "projects/sample/plans/1" }]);
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
        kind: "review-database-changes",
        project: "projects/-",
      })
    ).toBeUndefined();
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
