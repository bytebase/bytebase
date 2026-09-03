import { describe, expect, test } from "vitest";
import {
  PROJECT_V1_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_INSTANCES,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  WORKSPACE_ROUTE_MEMBERS,
  WORKSPACE_ROUTE_USERS,
} from "@/app/router/handles";
import {
  CREATE_INSTANCE_PRODUCT_INTRO,
  CREATE_PROJECT_PRODUCT_INTRO,
  CREATE_USER_PRODUCT_INTRO,
  GRANT_ACCESS_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
  PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
} from "@/lib/productIntro";
import { GUIDE_STEP_REGISTRY } from "./steps";
import type { GuideContext, GuideStepId } from "./types";

const createContext = (
  overrides: Partial<GuideContext> = {}
): GuideContext => ({
  hasProject: false,
  hasInstance: false,
  hasExploredDatabase: false,
  hasRunStatement: false,
  hasCreatedChangeIssue: false,
  isSaaS: false,
  hasOtherHumanUser: false,
  hasOtherWorkspaceMember: false,
  projectName: "",
  instanceName: "",
  databaseProjectName: "",
  databaseName: "",
  route: { name: "workspace.home", params: {} },
  ...overrides,
});

describe("GUIDE_STEP_REGISTRY", () => {
  test("contains only reusable resource and customer-action definitions", () => {
    expect(Object.keys(GUIDE_STEP_REGISTRY)).toEqual([
      "create-project",
      "connect-instance",
      "explore-database",
      "query-data",
      "create-database-change",
      "create-user",
      "grant-access",
    ]);
  });

  test.each([
    ["create-project", { hasProject: true }],
    ["connect-instance", { hasInstance: true }],
    [
      "explore-database",
      {
        hasExploredDatabase: true,
        databaseProjectName: "projects/app",
        databaseName: "instances/sample/databases/employee",
      },
    ],
    ["query-data", { hasRunStatement: true }],
    ["create-database-change", { hasCreatedChangeIssue: true }],
    ["create-user", { hasOtherWorkspaceMember: true }],
    ["grant-access", { hasOtherWorkspaceMember: true }],
  ] as const)("uses observable evidence for %s", (stepId, completed) => {
    expect(
      GUIDE_STEP_REGISTRY[stepId].isComplete(createContext(completed))
    ).toBe(true);
  });

  test("keeps database exploration incomplete without a concrete target", () => {
    expect(
      GUIDE_STEP_REGISTRY["explore-database"].isComplete(
        createContext({ hasExploredDatabase: true })
      )
    ).toBe(false);
  });

  test.each([createContext(), createContext({ projectName: "projects/app" })])(
    "always opens the workspace project list with the create highlight",
    (context) => {
      expect(
        GUIDE_STEP_REGISTRY["create-project"].resolveActions(context)
      ).toEqual({
        select: {
          type: "navigate",
          target: {
            name: PROJECT_V1_ROUTE_DASHBOARD,
            query: {
              [PRODUCT_INTRO_QUERY_KEY]: CREATE_PROJECT_PRODUCT_INTRO,
            },
          },
        },
      });
    }
  );

  test.each([
    createContext({ projectName: "projects/app" }),
    createContext({
      projectName: "projects/app",
      instanceName: "instances/prod",
    }),
  ])(
    "always opens the project instance list with the create highlight",
    (context) => {
      expect(
        GUIDE_STEP_REGISTRY["connect-instance"].resolveActions(context)
      ).toEqual({
        select: {
          type: "navigate",
          target: {
            name: PROJECT_V1_ROUTE_INSTANCES,
            params: { projectId: "app" },
            query: {
              [PRODUCT_INTRO_QUERY_KEY]: CREATE_INSTANCE_PRODUCT_INTRO,
            },
          },
        },
      });
    }
  );

  test("keeps the connect action fixed while its project dependency is blocked", () => {
    expect(
      GUIDE_STEP_REGISTRY["connect-instance"].resolveActions(createContext())
    ).toEqual({
      select: {
        type: "navigate",
        target: {
          name: PROJECT_V1_ROUTE_INSTANCES,
          params: { projectId: "" },
          query: {
            [PRODUCT_INTRO_QUERY_KEY]: CREATE_INSTANCE_PRODUCT_INTRO,
          },
        },
      },
    });
  });

  test.each([
    createContext({ projectName: "projects/app" }),
    createContext({
      projectName: "projects/app",
      databaseProjectName: "projects/other",
      databaseName: "instances/sample/databases/employee",
    }),
  ])("always opens the project database page with its highlight", (context) => {
    expect(
      GUIDE_STEP_REGISTRY["explore-database"].resolveActions(context)
    ).toEqual({
      select: {
        type: "navigate",
        target: {
          name: PROJECT_V1_ROUTE_DATABASES,
          params: { projectId: "app" },
          query: {
            [PRODUCT_INTRO_QUERY_KEY]: PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
          },
        },
      },
    });
  });

  test.each([false, true])(
    "always queries the discovered database when completion is %s",
    (hasRunStatement) => {
      expect(
        GUIDE_STEP_REGISTRY["query-data"].resolveActions(
          createContext({
            hasRunStatement,
            databaseName: "instances/sample/databases/employee",
            databaseProjectName: "projects/app",
          })
        )
      ).toEqual({
        primary: {
          type: "open-sql-editor",
          database: {
            name: "instances/sample/databases/employee",
            project: "projects/app",
          },
        },
      });
    }
  );

  test.each([false, true])(
    "always starts a database change when completion is %s",
    (hasCreatedChangeIssue) => {
      expect(
        GUIDE_STEP_REGISTRY["create-database-change"].resolveActions(
          createContext({
            hasCreatedChangeIssue,
            databaseName: "instances/sample/databases/employee",
            databaseProjectName: "projects/app",
          })
        )
      ).toEqual({
        select: {
          type: "create-change",
          project: "projects/app",
          database: "instances/sample/databases/employee",
        },
      });
    }
  );

  test.each([createContext(), createContext({ isSaaS: true })])(
    "always opens Users for the self-host teammate step",
    (context) => {
      expect(
        GUIDE_STEP_REGISTRY["create-user"].resolveActions(context)
      ).toEqual({
        select: {
          type: "navigate",
          target: {
            name: WORKSPACE_ROUTE_USERS,
            query: {
              [PRODUCT_INTRO_QUERY_KEY]: CREATE_USER_PRODUCT_INTRO,
            },
          },
        },
      });
    }
  );

  test.each([createContext(), createContext({ hasOtherHumanUser: true })])(
    "always opens Members for the SaaS teammate step",
    (context) => {
      expect(
        GUIDE_STEP_REGISTRY["grant-access"].resolveActions(context)
      ).toEqual({
        select: {
          type: "navigate",
          target: {
            name: WORKSPACE_ROUTE_MEMBERS,
            query: {
              [PRODUCT_INTRO_QUERY_KEY]: GRANT_ACCESS_PRODUCT_INTRO,
            },
          },
        },
      });
    }
  );

  test.each([
    ["create-project", PROJECT_V1_ROUTE_DASHBOARD, true],
    ["connect-instance", PROJECT_V1_ROUTE_INSTANCES, true],
    ["explore-database", "workspace.project.database.detail", true],
    ["query-data", "sql-editor.database", true],
    ["create-database-change", PROJECT_V1_ROUTE_PLAN_DETAIL, true],
    ["create-user", WORKSPACE_ROUTE_USERS, true],
    ["grant-access", WORKSPACE_ROUTE_MEMBERS, true],
  ] as const)("matches %s on %s", (stepId, name, expected) => {
    expect(
      GUIDE_STEP_REGISTRY[stepId as GuideStepId].matchesRoute({
        name,
        params: {},
      })
    ).toBe(expected);
  });
});
