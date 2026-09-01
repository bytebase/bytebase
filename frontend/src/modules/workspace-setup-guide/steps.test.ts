import { describe, expect, test } from "vitest";
import {
  DATABASE_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_INSTANCES,
} from "@/app/router/handles";
import {
  CREATE_INSTANCE_PRODUCT_INTRO,
  CREATE_PROJECT_PRODUCT_INTRO,
  PREPARE_DATABASE_PRODUCT_INTRO,
  PREPARE_DATABASE_TRANSFER_TIP,
  PRODUCT_INTRO_QUERY_KEY,
  PRODUCT_INTRO_TIP_QUERY_KEY,
  PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
} from "@/lib/productIntro";
import { GUIDE_STEP_REGISTRY } from "./steps";
import type { GuideContext } from "./types";

const createContext = (
  overrides: Partial<GuideContext> = {}
): GuideContext => ({
  hasProject: false,
  hasInstance: false,
  hasExploredDatabase: false,
  hasFirstQuery: false,
  projectName: "",
  databaseProjectName: "",
  databaseName: "",
  route: { name: "workspace.home", params: {} },
  ...overrides,
});

describe("GUIDE_STEP_REGISTRY", () => {
  test("preserves legacy analytics and translation keys", () => {
    expect(
      Object.values(GUIDE_STEP_REGISTRY).map(
        ({ id, analyticsKey, labelKey, descriptionKey }) => ({
          id,
          analyticsKey,
          labelKey,
          descriptionKey,
        })
      )
    ).toEqual([
      {
        id: "create-project",
        analyticsKey: "hasProject",
        labelKey: "workspace-setup-guide.steps.project",
        descriptionKey: "workspace-setup-guide.descriptions.project",
      },
      {
        id: "connect-instance",
        analyticsKey: "hasInstance",
        labelKey: "workspace-setup-guide.steps.instance",
        descriptionKey: "workspace-setup-guide.descriptions.instance",
      },
      {
        id: "explore-database",
        analyticsKey: "hasExploredDatabase",
        labelKey: "workspace-setup-guide.steps.database",
        descriptionKey: "workspace-setup-guide.descriptions.database",
      },
      {
        id: "query-data",
        analyticsKey: "hasFirstQuery",
        labelKey: "workspace-setup-guide.steps.query",
        descriptionKey: "workspace-setup-guide.descriptions.sql-editor",
      },
    ]);
  });

  test("routes project creation with the current product intro", () => {
    expect(
      GUIDE_STEP_REGISTRY["create-project"].resolveActions(createContext())
    ).toEqual({
      select: {
        type: "navigate",
        target: {
          name: "workspace.project",
          query: { [PRODUCT_INTRO_QUERY_KEY]: CREATE_PROJECT_PRODUCT_INTRO },
        },
      },
    });
    expect(
      GUIDE_STEP_REGISTRY["create-project"].resolveActions(
        createContext({ hasProject: true })
      )
    ).toEqual({});
  });

  test("routes instance connection through the project when available", () => {
    expect(
      GUIDE_STEP_REGISTRY["connect-instance"].resolveActions(
        createContext({ projectName: "projects/app" })
      )
    ).toEqual({
      select: {
        type: "navigate",
        target: {
          name: PROJECT_V1_ROUTE_INSTANCES,
          params: { projectId: "app" },
          query: { [PRODUCT_INTRO_QUERY_KEY]: CREATE_INSTANCE_PRODUCT_INTRO },
        },
      },
    });
    expect(
      GUIDE_STEP_REGISTRY["connect-instance"].resolveActions(createContext())
    ).toEqual({
      select: {
        type: "navigate",
        target: {
          name: INSTANCE_ROUTE_DASHBOARD,
          query: { [PRODUCT_INTRO_QUERY_KEY]: CREATE_INSTANCE_PRODUCT_INTRO },
        },
      },
    });
  });

  test("routes database preparation with transfer guidance", () => {
    expect(
      GUIDE_STEP_REGISTRY["explore-database"].resolveActions(createContext())
    ).toEqual({
      select: {
        type: "navigate",
        target: {
          name: DATABASE_ROUTE_DASHBOARD,
          query: {
            [PRODUCT_INTRO_QUERY_KEY]: PREPARE_DATABASE_PRODUCT_INTRO,
            [PRODUCT_INTRO_TIP_QUERY_KEY]: PREPARE_DATABASE_TRANSFER_TIP,
          },
        },
      },
    });
  });

  test("routes an existing project database with synced guidance", () => {
    expect(
      GUIDE_STEP_REGISTRY["explore-database"].resolveActions(
        createContext({
          databaseProjectName: "projects/app",
          databaseName: "instances/sample/databases/employee",
        })
      )
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

  test("resolves the current query and change actions together", () => {
    expect(
      GUIDE_STEP_REGISTRY["query-data"].resolveActions(
        createContext({
          databaseProjectName: "projects/app",
          databaseName: "instances/sample/databases/employee",
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
      secondary: {
        type: "create-change",
        project: "projects/app",
        database: "instances/sample/databases/employee",
      },
    });
  });

  test("omits query actions without a complete database target", () => {
    expect(
      GUIDE_STEP_REGISTRY["query-data"].resolveActions(createContext())
    ).toEqual({});
  });

  test.each([
    ["create-project", { hasProject: true }],
    ["connect-instance", { hasInstance: true }],
    ["explore-database", { hasExploredDatabase: true }],
    ["query-data", { hasFirstQuery: true }],
  ] as const)("evaluates completion for %s", (stepId, completed) => {
    expect(
      GUIDE_STEP_REGISTRY[stepId].isComplete(createContext(completed))
    ).toBe(true);
    expect(GUIDE_STEP_REGISTRY[stepId].isComplete(createContext())).toBe(false);
  });

  test.each([
    ["create-project", "workspace.project", true],
    ["connect-instance", "workspace.instance.detail", true],
    ["connect-instance", "workspace.project.instance", true],
    ["explore-database", "workspace.database.detail", true],
    ["explore-database", "workspace.project.database", true],
    ["query-data", "sql-editor.database", true],
    ["query-data", "workspace.home", false],
  ] as const)("matches %s against %s", (stepId, name, expected) => {
    expect(GUIDE_STEP_REGISTRY[stepId].matchesRoute({ name, params: {} })).toBe(
      expected
    );
  });
});
