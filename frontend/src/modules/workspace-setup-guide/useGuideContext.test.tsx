import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  SQL_EDITOR_DATABASE_MODULE,
} from "@/app/router/handles";
import { planEvents } from "@/lib/plan/events";
import { sqlEditorEvents } from "@/modules/sql-editor/model/events";
import { GUIDE_PROGRESS_KEYS } from "./progress";
import type {
  GuideRoute,
  GuideScenarioId,
  GuideWorkspaceUsage,
} from "./types";

const mocks = vi.hoisted(() => ({
  projectsByName: {} as Record<string, unknown>,
  instancesByName: {} as Record<string, unknown>,
  databasesByName: {} as Record<string, unknown>,
  usersByName: {} as Record<string, unknown>,
  fetchProjectList: vi.fn(),
  fetchInstanceList: vi.fn(),
  fetchDatabases: vi.fn(),
  listUsers: vi.fn(),
  introState: {} as Record<string, boolean>,
  saveIntroStateByKey: vi.fn(),
  captureMetric: vi.fn(),
  workspaceResourceName: "workspaces/default",
  defaultProject: "projects/default",
  currentUserName: "users/ed@example.com",
  isSaaS: false,
  workspacePolicy: {
    bindings: [
      {
        role: "roles/workspaceAdmin",
        members: ["users/ed@example.com"],
      },
    ],
  },
}));

vi.mock("@/app/analytics/provider", () => ({
  behaviorAnalytics: { captureMetric: mocks.captureMetric },
}));

vi.mock("@/hooks/useAppState", () => ({
  useIntroStateByKey: (key: string) => mocks.introState[key] ?? false,
}));

vi.mock("@/stores/app", () => {
  const state = () => ({
    serverInfo: { defaultProject: mocks.defaultProject },
    projectsByName: mocks.projectsByName,
    instancesByName: mocks.instancesByName,
    databasesByName: mocks.databasesByName,
    usersByName: mocks.usersByName,
    workspaceResourceName: () => mocks.workspaceResourceName,
    currentUserName: mocks.currentUserName,
    workspacePolicy: mocks.workspacePolicy,
    isSaaSMode: () => mocks.isSaaS,
  });
  const useAppStore = Object.assign(
    (selector: (value: ReturnType<typeof state>) => unknown) =>
      selector(state()),
    {
      getState: () => ({
        ...state(),
        fetchProjectList: mocks.fetchProjectList,
        fetchInstanceList: mocks.fetchInstanceList,
        fetchDatabases: mocks.fetchDatabases,
        listUsers: mocks.listUsers,
        getIntroStateByKey: (key: string) => mocks.introState[key] ?? false,
        saveIntroStateByKey: mocks.saveIntroStateByKey,
      }),
    }
  );
  return { useAppStore };
});

import {
  hasOtherHumanWorkspaceMember,
  useGuideContext,
} from "./useGuideContext";

const home: GuideRoute = { name: "workspace.home", params: {} };

const renderGuideContext = (
  props: {
    enabled: boolean;
    dismissed: boolean;
    route: GuideRoute;
    scenarioId?: GuideScenarioId;
    workspaceUsage?: GuideWorkspaceUsage;
  } = {
    enabled: true,
    dismissed: false,
    route: home,
  }
) => renderHook((value) => useGuideContext(value), { initialProps: props });

describe("useGuideContext", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.introState = {};
    mocks.projectsByName = {};
    mocks.instancesByName = {};
    mocks.databasesByName = {};
    mocks.usersByName = {};
    mocks.fetchProjectList.mockResolvedValue({ projects: [], nextPageToken: "" });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({ databases: [], nextPageToken: "" });
    mocks.listUsers.mockResolvedValue({ users: [], nextPageToken: "" });
    mocks.currentUserName = "users/ed@example.com";
    mocks.isSaaS = false;
    mocks.workspacePolicy = {
      bindings: [
        {
          role: "roles/workspaceAdmin",
          members: ["users/ed@example.com"],
        },
      ],
    };
    mocks.saveIntroStateByKey.mockImplementation(({ key, newState }) => {
      mocks.introState[key] = newState;
    });
  });

  test("separates discovered resources from learning evidence", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/app" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/sample" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/sample/databases/employee",
          project: "projects/app",
        },
      ],
      nextPageToken: "",
    });
    const { result } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.context).toMatchObject({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: false,
      hasCreatedChangeIssue: false,
      projectName: "projects/app",
      instanceName: "instances/sample",
      databaseName: "instances/sample/databases/employee",
    });
    expect(mocks.captureMetric).not.toHaveBeenCalled();
  });

  test.each([
    "user:teammate@example.com",
    "users/teammate@example.com",
  ])("accepts another explicit human principal: %s", (member) => {
    expect(
      hasOtherHumanWorkspaceMember(
        { bindings: [{ role: "roles/workspaceMember", members: [member] }] },
        "users/ed@example.com"
      )
    ).toBe(true);
  });

  test.each([
    "user:ed@example.com",
    "users/ed@example.com",
    "allUsers",
    "user:allUsers",
    "users/allUsers",
    "group:developers@example.com",
    "serviceAccount:bot@example.com",
    "workloadIdentity:github@example.com",
  ])("rejects a non-teammate principal: %s", (member) => {
    expect(
      hasOtherHumanWorkspaceMember(
        { bindings: [{ role: "roles/workspaceMember", members: [member] }] },
        "users/ed@example.com"
      )
    ).toBe(false);
  });

  test("lists active users only for a self-host team journey", async () => {
    mocks.listUsers.mockResolvedValue({
      users: [
        { name: "users/ed@example.com" },
        { name: "users/teammate@example.com" },
      ],
      nextPageToken: "",
    });
    const { result } = renderGuideContext({
      enabled: true,
      dismissed: false,
      route: home,
      scenarioId: "query-data",
      workspaceUsage: "team",
    });

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(mocks.listUsers).toHaveBeenCalledWith({
      pageSize: 100,
      filter: { state: 1 },
    });
    expect(result.current.context).toMatchObject({
      isSaaS: false,
      hasOtherHumanUser: true,
      hasOtherWorkspaceMember: false,
    });
  });

  test.each([
    { isSaaS: true, workspaceUsage: "team" as const },
    { isSaaS: false, workspaceUsage: "solo" as const },
    { isSaaS: false, workspaceUsage: undefined },
  ])("does not list users outside a self-host team journey: %j", async (input) => {
    mocks.isSaaS = input.isSaaS;
    const { result } = renderGuideContext({
      enabled: true,
      dismissed: false,
      route: home,
      workspaceUsage: input.workspaceUsage,
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(mocks.listUsers).not.toHaveBeenCalled();
  });

  test("uses workspace IAM as teammate completion authority", async () => {
    mocks.workspacePolicy = {
      bindings: [
        {
          role: "roles/workspaceMember",
          members: ["user:invited@example.com"],
        },
      ],
    };
    const { result } = renderGuideContext({
      enabled: true,
      dismissed: false,
      route: home,
      workspaceUsage: "team",
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.context.hasOtherWorkspaceMember).toBe(true);
    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: GUIDE_PROGRESS_KEYS.teammateAdded,
      newState: true,
    });
    expect(mocks.captureMetric).not.toHaveBeenCalled();
  });

  test.each([
    {
      route: {
        name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
        params: {
          projectId: "app",
          instanceId: "sample",
          databaseName: "employee",
        },
      },
    },
    {
      route: {
        name: SQL_EDITOR_DATABASE_MODULE,
        params: { project: "app", instance: "sample", database: "employee" },
      },
    },
  ])("records concrete database route evidence", async ({ route }) => {
    renderGuideContext({
      enabled: true,
      dismissed: false,
      route,
    });

    await waitFor(() =>
      expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
        key: GUIDE_PROGRESS_KEYS.databaseExplored,
        newState: true,
      })
    );
    expect(mocks.captureMetric).not.toHaveBeenCalled();
  });

  test("records a populated project database page as exploration", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/app" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/sample" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/sample/databases/employee",
          project: "projects/app",
        },
      ],
      nextPageToken: "",
    });
    const { result } = renderGuideContext({
      enabled: true,
      dismissed: false,
      route: {
        name: PROJECT_V1_ROUTE_DATABASES,
        params: { projectId: "app" },
      },
    });

    await waitFor(() =>
      expect(result.current.context.hasExploredDatabase).toBe(true)
    );
    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: GUIDE_PROGRESS_KEYS.databaseExplored,
      newState: true,
    });
  });

  test("does not record an empty project database page as exploration", async () => {
    const { result } = renderGuideContext({
      enabled: true,
      dismissed: false,
      route: {
        name: PROJECT_V1_ROUTE_DATABASES,
        params: { projectId: "app" },
      },
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.context.hasExploredDatabase).toBe(false);
    expect(mocks.saveIntroStateByKey).not.toHaveBeenCalledWith({
      key: GUIDE_PROGRESS_KEYS.databaseExplored,
      newState: true,
    });
  });

  test("completes Query Data after any statement execution", async () => {
    const { result } = renderGuideContext({
      enabled: true,
      dismissed: false,
      route: home,
      scenarioId: "query-data",
    });
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await sqlEditorEvents.emit("query-executed", {
        database: "instances/sample/databases/employee",
        project: "projects/app",
      });
    });

    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: GUIDE_PROGRESS_KEYS.statementRun,
      newState: true,
    });
    expect(result.current.context).toMatchObject({
      hasRunStatement: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });
    expect(mocks.captureMetric).not.toHaveBeenCalled();
  });

  test("records a newly created database-change issue", async () => {
    const { result } = renderGuideContext({
      enabled: true,
      dismissed: false,
      route: home,
      scenarioId: "create-database-change",
    });
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await planEvents.emit("database-change-issue-created", {
        issue: "projects/app/issues/1",
        project: "projects/app",
      });
    });

    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: GUIDE_PROGRESS_KEYS.changeIssueCreated,
      newState: true,
    });
    expect(result.current.context.hasCreatedChangeIssue).toBe(true);
    expect(mocks.captureMetric).not.toHaveBeenCalled();
  });

  test.each([undefined, "query-data" as const])(
    "does not record a change issue for scenario %s",
    async (scenarioId) => {
      renderGuideContext({
        enabled: true,
        dismissed: false,
        route: home,
        scenarioId,
      });

      await act(async () => {
        await planEvents.emit("database-change-issue-created", {
          issue: "projects/app/issues/1",
          project: "projects/app",
        });
      });

      expect(mocks.saveIntroStateByKey).not.toHaveBeenCalledWith({
        key: GUIDE_PROGRESS_KEYS.changeIssueCreated,
        newState: true,
      });
    }
  );

  test("keeps self-host user evidence false when listing users fails", async () => {
    mocks.listUsers.mockRejectedValue(new Error("unavailable"));
    const { result } = renderGuideContext({
      enabled: true,
      dismissed: false,
      route: home,
      workspaceUsage: "team",
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.context.hasOtherHumanUser).toBe(false);
  });

  test("ignores statement events without a database", async () => {
    const { result } = renderGuideContext();
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await sqlEditorEvents.emit("query-executed", {
        database: "",
        project: "projects/app",
      });
    });

    expect(result.current.context.hasRunStatement).toBe(false);
  });

  test.each([
    { enabled: false, dismissed: false },
    { enabled: true, dismissed: true },
  ])(
    "does not record while disabled or dismissed: $enabled/$dismissed",
    async (state) => {
      renderGuideContext({
        ...state,
        route: home,
        scenarioId: "query-data",
      });

      await act(async () => {
        await sqlEditorEvents.emit("query-executed", {
          database: "instances/sample/databases/employee",
          project: "projects/app",
        });
      });

      expect(mocks.saveIntroStateByKey).not.toHaveBeenCalled();
    }
  );
});
