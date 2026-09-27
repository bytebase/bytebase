import { create } from "@bufbuild/protobuf";
import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { useCommonSearchScopeOptions } from "./useCommonSearchScopeOptions";

const mocks = vi.hoisted(() => ({
  fetchInstanceList: vi.fn(),
  hasWorkspacePermission: true,
  hasProjectPermission: true,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/EngineIcon", () => ({
  EngineIcon: () => null,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({ fetchInstanceList: mocks.fetchInstanceList }),
  },
}));

vi.mock("@/types", () => {
  return {
    isDefaultProject: (name: string) => name === "projects/default",
    isValidProjectName: (name: string) =>
      name.startsWith("projects/") && name !== "projects/-",
  };
});

vi.mock("@/utils", () => {
  return {
    extractEnvironmentResourceName: (name: string) =>
      name.replace(/^environments\//, ""),
    extractInstanceResourceName: (name: string) =>
      name.match(/(?:^|\/)instances\/([^/]+)/)?.[1] ?? "",
    getDefaultPagination: () => 1000,
    hasWorkspacePermissionV2: () => mocks.hasWorkspacePermission,
    hasProjectPermissionV2: () => mocks.hasProjectPermission,
    supportedEngineV1List: () => [],
  };
});

beforeEach(() => {
  vi.clearAllMocks();
  mocks.hasWorkspacePermission = true;
  mocks.hasProjectPermission = true;
  mocks.fetchInstanceList.mockImplementation(async ({ parent }) => ({
    instances: parent
      ? [
          {
            name: "projects/app/instances/project-prod",
            title: "Project production",
          },
        ]
      : [{ name: "instances/shared", title: "Shared" }],
    nextPageToken: "",
  }));
});

describe("useCommonSearchScopeOptions", () => {
  test("discovers workspace and project instances with canonical values", async () => {
    const project = create(ProjectSchema, { name: "projects/app" });
    const { result } = renderHook(() =>
      useCommonSearchScopeOptions(["instance"], project)
    );

    const options = await result.current[0].onSearch?.("prod");

    expect(mocks.fetchInstanceList).toHaveBeenCalledTimes(2);
    expect(mocks.fetchInstanceList).toHaveBeenCalledWith(
      expect.objectContaining({ parent: "projects/app" })
    );
    expect(options?.map((option) => option.value)).toEqual([
      "instances/shared",
      "projects/app/instances/project-prod",
    ]);
  });

  test("supports users who can list only project instances", async () => {
    mocks.hasWorkspacePermission = false;
    const project = create(ProjectSchema, { name: "projects/app" });
    const { result } = renderHook(() =>
      useCommonSearchScopeOptions(["instance"], project)
    );

    const options = await result.current[0].onSearch?.("");

    expect(mocks.fetchInstanceList).toHaveBeenCalledOnce();
    expect(mocks.fetchInstanceList).toHaveBeenCalledWith(
      expect.objectContaining({ parent: "projects/app" })
    );
    expect(options?.map((option) => option.value)).toEqual([
      "projects/app/instances/project-prod",
    ]);
  });
});
