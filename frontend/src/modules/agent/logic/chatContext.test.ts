import { describe, expect, test } from "vitest";
import type { ReactRoute } from "@/app/router";
import {
  PROJECT_V1_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_DETAIL,
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_ISSUES,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
  PROJECT_V1_ROUTE_PLANS,
  PROJECT_V1_ROUTE_SETTINGS,
} from "@/app/router/handles";
import { resolveAgentPageResource, resolveChatContext } from "./chatContext";

const routeWithParams = (params: ReactRoute["params"]) =>
  ({ params }) as ReactRoute;
const detailRoute = (
  name: string,
  params: ReactRoute["params"],
  query: ReactRoute["query"] = {}
): ReactRoute => ({
  name,
  params,
  query,
  fullPath: "",
  hash: "",
  requiredPermissions: [],
  overrideDocumentTitle: false,
});

describe("resolveChatContext", () => {
  test("uses the project resource on project routes", () => {
    expect(
      resolveChatContext(
        routeWithParams({ projectId: "orders" }),
        "workspaces/demo"
      )
    ).toBe("projects/orders");
    expect(
      resolveChatContext(
        routeWithParams({ project: "orders" }),
        "workspaces/demo"
      )
    ).toBe("projects/orders");
  });

  test("uses the active workspace on other routes", () => {
    expect(resolveChatContext(routeWithParams({}), "workspaces/demo")).toBe(
      "workspaces/demo"
    );
  });

  test("does not invent a context before the workspace is known", () => {
    expect(resolveChatContext(routeWithParams({}), "")).toBeUndefined();
  });
});

describe("resolveAgentPageResource", () => {
  test.each([
    PROJECT_V1_ROUTE_DETAIL,
    PROJECT_V1_ROUTE_DATABASES,
    PROJECT_V1_ROUTE_ISSUES,
    PROJECT_V1_ROUTE_PLANS,
    PROJECT_V1_ROUTE_SETTINGS,
  ])("uses the project start on %s", (name) => {
    expect(
      resolveAgentPageResource(detailRoute(name, { projectId: "orders" }))
    ).toEqual({
      kind: "project",
      name: "projects/orders",
      project: "projects/orders",
    });
  });

  test("does not treat workspace or unrelated routes as project pages", () => {
    expect(
      resolveAgentPageResource(detailRoute(PROJECT_V1_ROUTE_DASHBOARD, {}))
    ).toBeUndefined();
    expect(
      resolveAgentPageResource(
        detailRoute("sql-editor.project", { projectId: "orders" })
      )
    ).toBeUndefined();
  });

  test("uses the issue's project", () => {
    expect(
      resolveAgentPageResource(
        detailRoute(PROJECT_V1_ROUTE_ISSUE_DETAIL, {
          projectId: "orders",
          issueId: "42",
        })
      )
    ).toEqual({
      kind: "issue",
      name: "projects/orders/issues/42",
      project: "projects/orders",
    });
  });

  test.each([
    PROJECT_V1_ROUTE_PLAN_DETAIL,
    PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
    PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
    PROJECT_V1_ROUTE_PLAN_ROLLOUT,
    PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
    PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
  ])("uses the Plan on %s", (name) => {
    expect(
      resolveAgentPageResource(
        detailRoute(name, { projectId: "orders", planId: "42" })
      )
    ).toEqual({
      kind: "plan",
      name: "projects/orders/plans/42",
      project: "projects/orders",
    });
  });

  test.each(["create", "CREATE", "-"])(
    "does not reference a Plan for the %s route parameter",
    (planId) => {
      expect(
        resolveAgentPageResource(
          detailRoute(PROJECT_V1_ROUTE_PLAN_DETAIL, {
            projectId: "orders",
            planId,
          })
        )
      ).toEqual({
        kind: "project",
        name: "projects/orders",
        project: "projects/orders",
      });
    }
  );

  test("uses the canonical database parent when present", () => {
    expect(
      resolveAgentPageResource(
        detailRoute(
          PROJECT_V1_ROUTE_DATABASE_DETAIL,
          {
            projectId: "orders",
            instanceId: "primary",
            databaseName: "app",
          },
          { parent: "projects/orders/instances/primary" }
        )
      )
    ).toEqual({
      kind: "database",
      project: "projects/orders",
      names: ["projects/orders/instances/primary/databases/app"],
    });
  });

  test.each([
    PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
    PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
  ])("keeps database-specific context on %s", (name) => {
    expect(
      resolveAgentPageResource(
        detailRoute(
          name,
          {
            projectId: "orders",
            instanceId: "primary",
            databaseName: "app",
          },
          { parent: "projects/orders/instances/primary" }
        )
      )
    ).toEqual({
      kind: "database",
      project: "projects/orders",
      names: ["projects/orders/instances/primary/databases/app"],
    });
  });

  test("falls back to both database scopes without trusting an unrelated parent", () => {
    expect(
      resolveAgentPageResource(
        detailRoute(
          PROJECT_V1_ROUTE_DATABASE_DETAIL,
          {
            projectId: "orders",
            instanceId: "primary",
            databaseName: "app",
          },
          { parent: "projects/other/instances/primary" }
        )
      )
    ).toEqual({
      kind: "database",
      project: "projects/orders",
      names: [
        "projects/orders/instances/primary/databases/app",
        "instances/primary/databases/app",
      ],
    });
  });
});
