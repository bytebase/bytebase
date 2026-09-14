// @vitest-environment node
import { describe, expect, test } from "vitest";
import {
  PROJECT_V1_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
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
import { GUIDE_STEP_DEFINITIONS } from "./steps";
import type { GuideContext, GuideStepDefinition, GuideStepId } from "./types";

const GUIDE_STEP_BY_ID = Object.fromEntries(
  GUIDE_STEP_DEFINITIONS.map((definition) => [definition.id, definition])
) as Record<GuideStepId, GuideStepDefinition>;

const createContext = (
  overrides: Partial<GuideContext> = {}
): GuideContext => ({
  hasProject: false,
  hasInstance: false,
  hasExploredDatabase: false,
  hasRunStatement: false,
  hasCreatedChangeIssue: false,
  hasMarkedSensitiveData: false,
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

describe("GUIDE_STEP_DEFINITIONS", () => {
  test("contains only reusable resource and customer-action definitions", () => {
    expect(GUIDE_STEP_DEFINITIONS.map(({ id }) => id)).toEqual([
      "create-project",
      "connect-instance",
      "explore-database",
      "query-data",
      "create-database-change",
      "mark-sensitive-data",
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
    ["mark-sensitive-data", { hasMarkedSensitiveData: true }],
    ["add-member", { hasOtherWorkspaceMember: true }],
  ] as const)("uses observable evidence for %s", (stepId, completed) => {
    expect(GUIDE_STEP_BY_ID[stepId].isComplete(createContext(completed))).toBe(
      true
    );
  });

  test("keeps database exploration incomplete without a concrete target", () => {
    expect(
      GUIDE_STEP_BY_ID["explore-database"].isComplete(
        createContext({ hasExploredDatabase: true })
      )
    ).toBe(false);
  });

  test.each([createContext(), createContext({ projectName: "projects/app" })])(
    "always opens the workspace project list with the create highlight",
    (context) => {
      expect(
        GUIDE_STEP_BY_ID["create-project"].resolveActions(context)
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
        GUIDE_STEP_BY_ID["connect-instance"].resolveActions(context)
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
      GUIDE_STEP_BY_ID["connect-instance"].resolveActions(createContext())
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
      GUIDE_STEP_BY_ID["explore-database"].resolveActions(context)
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
        GUIDE_STEP_BY_ID["query-data"].resolveActions(
          createContext({
            hasRunStatement,
            databaseName: "instances/sample/databases/employee",
            databaseProjectName: "projects/app",
          })
        )
      ).toEqual({
        select: {
          type: "navigate",
          target: {
            name: "sql-editor.database",
            params: {
              project: "app",
              instance: "sample",
              database: "employee",
            },
          },
        },
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
    "always opens the discovered database catalog when sensitive data completion is %s",
    (hasMarkedSensitiveData) => {
      expect(
        GUIDE_STEP_BY_ID["mark-sensitive-data"].resolveActions(
          createContext({
            hasMarkedSensitiveData,
            databaseName: "instances/sample/databases/employee",
            databaseProjectName: "projects/app",
          })
        )
      ).toEqual({
        select: {
          type: "navigate",
          target: {
            name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
            params: {
              projectId: "app",
              instanceId: "sample",
              databaseName: "employee",
            },
            query: {
              parent: "instances/sample",
              intro: "mark-sensitive-data",
            },
            hash: "#catalog",
          },
        },
      });
    }
  );

  test.each([false, true])(
    "always starts a database change when completion is %s",
    (hasCreatedChangeIssue) => {
      expect(
        GUIDE_STEP_BY_ID["create-database-change"].resolveActions(
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

    expect(GUIDE_STEP_BY_ID["add-member"].resolveActions(context)).toEqual({
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
    expect(GUIDE_STEP_BY_ID["add-member"].resolveActions(context)).toEqual({
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
    ["mark-sensitive-data", PROJECT_V1_ROUTE_DATABASE_DETAIL, true],
    ["add-member", WORKSPACE_ROUTE_USERS, true],
    ["add-member", WORKSPACE_ROUTE_MEMBERS, true],
  ] as const)("matches %s on %s", (stepId, name, expected) => {
    expect(
      GUIDE_STEP_BY_ID[stepId as GuideStepId].matchesRoute({
        name,
        params: {},
      })
    ).toBe(expected);
  });
});
