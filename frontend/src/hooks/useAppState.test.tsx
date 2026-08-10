import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  storageKeyRecentProjects,
  workspaceCacheScope,
} from "@/utils/storage-keys";

const mocks = vi.hoisted(() => ({
  batchFetchProjects: vi.fn(),
  getProjectByName: vi.fn(),
  loadCurrentUser: vi.fn(),
  loadServerInfo: vi.fn(),
  projectsByName: {} as Record<string, Project>,
  searchProjects: vi.fn(),
}));

vi.mock("@/stores/app", () => ({
  getProjectResourceId: (name: string) => name.split("/").pop() ?? "",
  isConnectAlreadyExists: () => false,
  isDefaultProjectName: (name: string) => name === "projects/default",
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      appFeatures: {},
      batchFetchProjects: mocks.batchFetchProjects,
      currentUser: {
        email: "dev@example.com",
        workspace: "workspaces/default",
      },
      getProjectByName: mocks.getProjectByName,
      loadCurrentUser: mocks.loadCurrentUser,
      loadServerInfo: mocks.loadServerInfo,
      projectsByName: mocks.projectsByName,
      searchProjects: mocks.searchProjects,
      serverInfo: { saas: false },
    }),
}));

import { useProjectList, useRecentProjects } from "./useAppState";

const recentProjectsKey = storageKeyRecentProjects(
  workspaceCacheScope(false, "workspaces/default"),
  "dev@example.com"
);

describe("useRecentProjects", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    mocks.projectsByName = {};
    mocks.batchFetchProjects.mockResolvedValue([]);
    mocks.getProjectByName.mockImplementation((name: string) => ({
      name: name === "projects/default" ? name : "projects/-1",
    }));
    mocks.searchProjects.mockResolvedValue({
      projects: [],
      nextPageToken: "",
    });
  });

  test("fetches an already-settled project query immediately", () => {
    const { rerender } = renderHook(
      ({ query }: { query: string }) => useProjectList(query),
      { initialProps: { query: "alpha" } }
    );

    expect(mocks.searchProjects).toHaveBeenLastCalledWith({
      pageSize: 50,
      pageToken: "",
      query: "alpha",
      excludeDefault: true,
    });

    rerender({ query: "beta" });
    expect(mocks.searchProjects).toHaveBeenLastCalledWith({
      pageSize: 50,
      pageToken: "",
      query: "beta",
      excludeDefault: true,
    });
  });

  test("does not expose the unknown project placeholder for uncached recent projects", async () => {
    localStorage.setItem(recentProjectsKey, JSON.stringify(["projects/app"]));

    const { result } = renderHook(() => useRecentProjects());

    await waitFor(() => {
      expect(mocks.batchFetchProjects).toHaveBeenCalledWith(
        ["projects/app"],
        true
      );
    });

    expect(result.current.projects).toEqual([]);
  });

  test("loads recent projects silently and omits inaccessible cached projects", async () => {
    const accessibleProject = { name: "projects/app" } as Project;
    localStorage.setItem(
      recentProjectsKey,
      JSON.stringify(["projects/app", "projects/removed"])
    );
    mocks.getProjectByName.mockImplementation((name: string) =>
      name === accessibleProject.name
        ? accessibleProject
        : { name: "projects/-1" }
    );

    const { result } = renderHook(() => useRecentProjects());

    await waitFor(() => {
      expect(mocks.batchFetchProjects).toHaveBeenCalledWith(
        ["projects/app", "projects/removed"],
        true
      );
    });

    expect(result.current.projects).toEqual([accessibleProject]);
  });

  test("keeps the synthesized default project when requested", async () => {
    localStorage.setItem(
      recentProjectsKey,
      JSON.stringify(["projects/default"])
    );

    const { result } = renderHook(() =>
      useRecentProjects({ excludeDefault: false })
    );

    await waitFor(() => {
      expect(result.current.projects).toEqual([{ name: "projects/default" }]);
    });
  });
});
