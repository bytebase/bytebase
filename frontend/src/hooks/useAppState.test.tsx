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
      serverInfo: { saas: false },
    }),
}));

import { useRecentProjects } from "./useAppState";

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
  });

  test("does not expose the unknown project placeholder for uncached recent projects", async () => {
    localStorage.setItem(recentProjectsKey, JSON.stringify(["projects/app"]));

    const { result } = renderHook(() => useRecentProjects());

    await waitFor(() => {
      expect(mocks.batchFetchProjects).toHaveBeenCalledWith(["projects/app"]);
    });

    expect(result.current.projects).toEqual([]);
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
