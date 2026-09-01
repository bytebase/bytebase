import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  DATABASE_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  SQL_EDITOR_DATABASE_MODULE,
} from "@/app/router/handles";
import { sqlEditorEvents } from "@/modules/sql-editor/model/events";
import { State } from "@/types/proto-es/v1/common_pb";
import type { GuideRoute } from "./types";

const mocks = vi.hoisted(() => ({
  projectsByName: {} as Record<string, unknown>,
  instancesByName: {} as Record<string, unknown>,
  databasesByName: {} as Record<string, unknown>,
  fetchProjectList: vi.fn(),
  fetchInstanceList: vi.fn(),
  fetchDatabases: vi.fn(),
  introState: {} as Record<string, boolean>,
  saveIntroStateByKey: vi.fn(),
  searchQueryHistories: vi.fn(),
  workspaceResourceName: "workspaces/default",
  defaultProject: "projects/default",
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
    workspaceResourceName: () => mocks.workspaceResourceName,
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
        getIntroStateByKey: (key: string) => mocks.introState[key] ?? false,
        saveIntroStateByKey: mocks.saveIntroStateByKey,
      }),
    }
  );
  return { useAppStore };
});

vi.mock("@/api", () => ({
  queryHistoryServiceClientConnect: {
    searchQueryHistories: mocks.searchQueryHistories,
  },
}));

import { useGuideContext } from "./useGuideContext";

const databaseExploredKey = "workspace-setup-guide.database-explored";
const queryExecutedKey = "workspace-setup-guide.query-executed";
const route: GuideRoute = { name: "workspace.home", params: {} };

const renderGuideContext = (
  props: {
    enabled: boolean;
    dismissed: boolean;
    route: GuideRoute;
  } = { enabled: true, dismissed: false, route }
) => renderHook((nextProps) => useGuideContext(nextProps), { initialProps: props });

describe("useGuideContext", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.introState = {};
    mocks.projectsByName = {};
    mocks.instancesByName = {};
    mocks.databasesByName = {};
    mocks.defaultProject = "projects/default";
    mocks.workspaceResourceName = "workspaces/default";
    mocks.fetchProjectList.mockResolvedValue({ projects: [], nextPageToken: "" });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({ databases: [], nextPageToken: "" });
    mocks.saveIntroStateByKey.mockImplementation(({ key, newState }) => {
      mocks.introState[key] = newState;
    });
  });

  test("reports an empty workspace after the initial scan", async () => {
    const { result } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.context).toMatchObject({
      hasProject: false,
      hasInstance: false,
      hasExploredDatabase: false,
      hasFirstQuery: false,
      projectName: "",
      databaseProjectName: "",
      databaseName: "",
    });
  });

  test("recognizes project, project-owned instance, and project database", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/app" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockImplementation(async ({ parent } = {}) => ({
      instances:
        parent === "projects/app"
          ? [{ name: "projects/app/instances/sample" }]
          : [],
      nextPageToken: "",
    }));
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "projects/app/instances/sample/databases/employee",
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
      projectName: "projects/app",
      databaseProjectName: "projects/app",
      databaseName: "projects/app/instances/sample/databases/employee",
    });
  });

  test("keeps the setup project when the first database belongs elsewhere", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/sample/databases/employee",
          project: "projects/default",
        },
      ],
      nextPageToken: "",
    });
    const { result } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.context).toMatchObject({
      projectName: "projects/project-a",
      databaseProjectName: "projects/default",
      databaseName: "instances/sample/databases/employee",
    });
  });

  test("counts workspace-owned instances as project and instance readiness", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-sample" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/sample" }],
      nextPageToken: "",
    });
    const { result } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.context).toMatchObject({
      hasProject: true,
      hasInstance: true,
    });
  });

  test("counts project-owned instances as project and instance readiness", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/app" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockImplementation(async ({ parent } = {}) => ({
      instances:
        parent === "projects/app"
          ? [{ name: "projects/app/instances/sample" }]
          : [],
      nextPageToken: "",
    }));
    const { result } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.context).toMatchObject({
      hasProject: true,
      hasInstance: true,
    });
  });

  test("uses only the first resource page", async () => {
    mocks.introState[databaseExploredKey] = true;
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-sample" }],
      nextPageToken: "project-page-2",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/sample" }],
      nextPageToken: "instance-page-2",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/sample/databases/employee",
          project: "projects/project-sample",
        },
      ],
      nextPageToken: "database-page-2",
    });
    const { result } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.context).toMatchObject({
      hasExploredDatabase: true,
      databaseName: "instances/sample/databases/employee",
    });
    expect(mocks.fetchProjectList).toHaveBeenCalledTimes(1);
    expect(mocks.fetchInstanceList).toHaveBeenCalledTimes(2);
    expect(mocks.fetchDatabases).toHaveBeenCalledTimes(1);
    expect(mocks.fetchProjectList).toHaveBeenCalledWith({
      pageSize: 1,
      silent: true,
      filter: { excludeDefault: true, state: State.ACTIVE },
    });
    expect(mocks.fetchInstanceList).toHaveBeenCalledWith({
      parent: "projects/project-sample",
      pageSize: 1,
      silent: true,
      filter: { state: State.ACTIVE },
    });
    expect(mocks.fetchDatabases).toHaveBeenCalledWith({
      parent: "workspaces/default",
      pageSize: 1,
      silent: true,
      filter: { project: "projects/project-sample" },
    });
  });

  test.each([
    {
      name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
      params: {
        projectId: "project-sample",
        instanceId: "sample-one",
        databaseName: "employee",
      },
    },
    {
      name: INSTANCE_ROUTE_DATABASE_DETAIL,
      params: { instanceId: "sample-one", databaseName: "employee" },
    },
    {
      name: SQL_EDITOR_DATABASE_MODULE,
      params: {
        project: "project-sample",
        instance: "sample-one",
        database: "employee",
      },
    },
  ])("marks a concrete database route as explored: $name", async (route) => {
    renderGuideContext({ enabled: true, dismissed: false, route });

    await waitFor(() =>
      expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
        key: databaseExploredKey,
        newState: true,
      })
    );
  });

  test.each([
    {
      name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
      params: { projectId: "project-sample", instanceId: "sample-one" },
    },
    {
      name: INSTANCE_ROUTE_DATABASE_DETAIL,
      params: { instanceId: "sample-one" },
    },
    {
      name: SQL_EDITOR_DATABASE_MODULE,
      params: { project: "project-sample", instance: "sample-one" },
    },
  ])("requires every database parameter for $name", async (route) => {
    const { result } = renderGuideContext({
      enabled: true,
      dismissed: false,
      route,
    });

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(mocks.saveIntroStateByKey).not.toHaveBeenCalledWith({
      key: databaseExploredKey,
      newState: true,
    });
    expect(result.current.context.hasExploredDatabase).toBe(false);
  });

  test("does not count the workspace database list as exploration", async () => {
    const { result } = renderGuideContext({
      enabled: true,
      dismissed: false,
      route: { name: DATABASE_ROUTE_DASHBOARD, params: {} },
    });

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(mocks.saveIntroStateByKey).not.toHaveBeenCalledWith({
      key: databaseExploredKey,
      newState: true,
    });
    expect(result.current.context.hasExploredDatabase).toBe(false);
  });

  test("keeps database exploration complete after it is persisted", async () => {
    mocks.introState[databaseExploredKey] = true;
    const { result } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.context.hasExploredDatabase).toBe(true);
  });

  test("marks query execution and retains its exact target", async () => {
    const { result } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));
    await act(async () => {
      await sqlEditorEvents.emit("query-executed", {
        database: "projects/app/instances/sample/databases/employee",
        project: "projects/app",
      });
    });

    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: databaseExploredKey,
      newState: true,
    });
    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: queryExecutedKey,
      newState: true,
    });
    expect(result.current.context).toMatchObject({
      hasExploredDatabase: true,
      hasFirstQuery: true,
      databaseProjectName: "projects/app",
      databaseName: "projects/app/instances/sample/databases/employee",
    });
  });

  test("keeps query progress and its event target while resource scans finish", async () => {
    let resolveFirstScan: ((value: { projects: unknown[]; nextPageToken: string }) => void) | undefined;
    let resolveSecondScan: ((value: { projects: unknown[]; nextPageToken: string }) => void) | undefined;
    mocks.fetchProjectList
      .mockImplementationOnce(
        () => new Promise((resolve) => { resolveFirstScan = resolve; })
      )
      .mockImplementationOnce(
        () => new Promise((resolve) => { resolveSecondScan = resolve; })
      );
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/discovered-instance" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/discovered-instance/databases/discovered-db",
          project: "projects/discovered-project",
        },
      ],
      nextPageToken: "",
    });
    const { result } = renderGuideContext();

    await waitFor(() => expect(mocks.fetchProjectList).toHaveBeenCalledTimes(1));
    await act(async () => {
      await sqlEditorEvents.emit("query-executed", {
        database: "instances/event-instance/databases/event-db",
        project: "projects/event-project",
      });
    });
    await waitFor(() => expect(mocks.fetchProjectList).toHaveBeenCalledTimes(2));

    await act(async () => {
      resolveFirstScan?.({
        projects: [{ name: "projects/discovered-project" }],
        nextPageToken: "",
      });
    });
    await waitFor(() =>
      expect(result.current.context).toMatchObject({
        hasExploredDatabase: true,
        hasFirstQuery: true,
        databaseProjectName: "projects/event-project",
        databaseName: "instances/event-instance/databases/event-db",
      })
    );

    await act(async () => {
      resolveSecondScan?.({
        projects: [{ name: "projects/discovered-project" }],
        nextPageToken: "",
      });
    });
    await waitFor(() =>
      expect(result.current.context).toMatchObject({
        hasFirstQuery: true,
        databaseProjectName: "projects/event-project",
        databaseName: "instances/event-instance/databases/event-db",
      })
    );
  });

  test("does not reconstruct query completion from query history", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/app" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        { name: "instances/prod/databases/main", project: "projects/app" },
      ],
      nextPageToken: "",
    });
    const { result } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.context.hasFirstQuery).toBe(false);
    expect(mocks.searchQueryHistories).not.toHaveBeenCalled();
  });

  test("does not search query history for guide progress", async () => {
    const { result } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(mocks.searchQueryHistories).not.toHaveBeenCalled();
  });

  test("skips first query check before a project database exists", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    const { result } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.context).toMatchObject({
      databaseName: "",
      hasFirstQuery: false,
    });
    expect(mocks.searchQueryHistories).not.toHaveBeenCalled();
  });

  test("refreshes when a new instance is added to the app store", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    const { result, rerender } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));
    mocks.instancesByName = { "instances/instance-a": {} };
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    rerender({ enabled: true, dismissed: false, route });

    await waitFor(() => expect(result.current.context.hasInstance).toBe(true));
  });

  test("refreshes when a database is added to the app store", async () => {
    mocks.introState[databaseExploredKey] = true;
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    const { result, rerender } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));
    mocks.databasesByName = { "instances/instance-a/databases/db-a": {} };
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });
    rerender({ enabled: true, dismissed: false, route });

    await waitFor(() =>
      expect(result.current.context.databaseName).toBe(
        "instances/instance-a/databases/db-a"
      )
    );
  });

  test("refreshes when the route changes after setup progress changes elsewhere", async () => {
    mocks.introState[databaseExploredKey] = true;
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    const { result, rerender } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });
    rerender({
      enabled: true,
      dismissed: false,
      route: { name: "workspace.member", params: {} },
    });

    await waitFor(() =>
      expect(result.current.context.databaseName).toBe(
        "instances/instance-a/databases/db-a"
      )
    );
  });

  test("keeps the guide visible while progress is refreshing", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    const { result, rerender } = renderGuideContext();

    await waitFor(() => expect(result.current.loading).toBe(false));
    mocks.instancesByName = { "instances/instance-a": {} };
    mocks.fetchProjectList.mockReturnValue(new Promise(() => undefined));
    rerender({
      enabled: true,
      dismissed: false,
      route: { name: "workspace.instance.detail", params: {} },
    });

    await waitFor(() => expect(mocks.fetchProjectList).toHaveBeenCalledTimes(2));
    expect(result.current.loading).toBe(false);
    expect(result.current.context.hasProject).toBe(true);
  });

  test("keeps the latest facts when a resource refresh fails", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    const { result, rerender } = renderGuideContext();

    await waitFor(() => expect(result.current.context.hasProject).toBe(true));
    mocks.instancesByName = { "instances/instance-a": {} };
    mocks.fetchProjectList.mockRejectedValueOnce(new Error("permission denied"));
    rerender({ enabled: true, dismissed: false, route });

    await waitFor(() => expect(mocks.fetchProjectList).toHaveBeenCalledTimes(2));
    expect(result.current.context.hasProject).toBe(true);
  });

  test.each([
    { enabled: false, dismissed: false },
    { enabled: true, dismissed: true },
  ])("resets facts when disabled or dismissed", async ({ enabled, dismissed }) => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    const { result, rerender } = renderGuideContext();

    await waitFor(() => expect(result.current.context.hasProject).toBe(true));
    rerender({ enabled, dismissed, route });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.context).toMatchObject({
      hasProject: false,
      hasInstance: false,
      hasExploredDatabase: false,
      hasFirstQuery: false,
      projectName: "",
      databaseProjectName: "",
      databaseName: "",
    });
  });
});
