import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import { DatabaseGroupTable } from "./DatabaseGroupTable";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  fetchDBGroupListByProjectName: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/stores/app", () => {
  const state = {
    fetchDBGroupListByProjectName: mocks.fetchDBGroupListByProjectName,
  };
  const useAppStore = () => undefined;
  useAppStore.getState = () => state;
  return { useAppStore };
});

describe("DatabaseGroupTable", () => {
  let root: Root | undefined;
  let container: HTMLDivElement | undefined;

  afterEach(() => {
    act(() => root?.unmount());
    root = undefined;
    container = undefined;
    document.body.innerHTML = "";
    mocks.fetchDBGroupListByProjectName.mockReset();
    vi.restoreAllMocks();
  });

  test("leaves the loading state when the fetch fails", async () => {
    const error = new Error("failed to load database groups");
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});
    mocks.fetchDBGroupListByProjectName.mockRejectedValueOnce(error);
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    await act(async () => {
      root!.render(
        <DatabaseGroupTable
          projectName="projects/p"
          view={DatabaseGroupView.BASIC}
        />
      );
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(consoleError).toHaveBeenCalledWith(error);
    expect(container.textContent).not.toContain("common.loading");
    expect(container.textContent).toContain("common.no-data");
  });
});
