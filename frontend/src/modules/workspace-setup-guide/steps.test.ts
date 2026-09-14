// @vitest-environment node
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

const getStepDefinition = (id: GuideStepId) => {
  const definition = GUIDE_STEP_REGISTRY.find((step) => step.id === id);
  if (!definition) throw new Error(`Missing step definition: ${id}`);
  return definition;
};

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
    expect(GUIDE_STEP_REGISTRY.map(({ id }) => id)).toEqual([
      "create-project",
      "connect-instance",
      "explore-database",
      "query-data",
      "create-database-change",
      "add-member",
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
    ["add-member", { hasOtherWorkspaceMember: true }],
  ] as const)("uses observable evidence for %s", (stepId, completed) => {
    expect(getStepDefinition(stepId).isComplete(createContext(completed))).toBe(
      true
    );
  });

  test("keeps database exploration incomplete without a concrete target", () => {
    expect(
      getStepDefinition("explore-database").isComplete(
        createContext({ hasExploredDatabase: true })
      )
    ).toBe(false);
  });

  test.each([createContext(), createContext({ projectName: "projects/app" })])(
    "always opens the workspace project list with the create highlight",
    (context) => {
      expect(
        getStepDefinition("create-project").resolveActions(context)
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
        getStepDefinition("connect-instance").resolveActions(context)
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
      getStepDefinition("connect-instance").resolveActions(createContext())
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
      getStepDefinition("explore-database").resolveActions(context)
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
        getStepDefinition("query-data").resolveActions(
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
        getStepDefinition("create-database-change").resolveActions(
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

  test("opens Users when self-host has no other human user", () => {
    const context = createContext();

    expect(getStepDefinition("add-member").resolveActions(context)).toEqual({
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
  });

  test.each([
    createContext({ isSaaS: true }),
    createContext({ hasOtherHumanUser: true }),
  ])("opens Members when the teammate can be granted access", (context) => {
    expect(getStepDefinition("add-member").resolveActions(context)).toEqual({
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
  });

  test.each([
    ["create-project", PROJECT_V1_ROUTE_DASHBOARD, true],
    ["connect-instance", PROJECT_V1_ROUTE_INSTANCES, true],
    ["explore-database", "workspace.project.database.detail", true],
    ["query-data", "sql-editor.database", true],
    ["create-database-change", PROJECT_V1_ROUTE_PLAN_DETAIL, true],
    ["add-member", WORKSPACE_ROUTE_USERS, true],
    ["add-member", WORKSPACE_ROUTE_MEMBERS, true],
  ] as const)("matches %s on %s", (stepId, name, expected) => {
    expect(
      getStepDefinition(stepId as GuideStepId).matchesRoute({
        name,
        params: {},
      })
    ).toBe(expected);
  });
});
