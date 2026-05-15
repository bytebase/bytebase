import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorTab } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  isConnectedSQLEditorTab: vi.fn<(tab: SQLEditorTab) => boolean>(),
  getConnectionForSQLEditorTab:
    vi.fn<
      (tab: SQLEditorTab) => { instance: { engine: Engine } | undefined }
    >(),
}));

vi.mock("@/utils", () => ({
  isConnectedSQLEditorTab: mocks.isConnectedSQLEditorTab,
  getConnectionForSQLEditorTab: mocks.getConnectionForSQLEditorTab,
}));

vi.mock("@/react/components/instance/constants", () => ({
  EngineIconPath: {
    [Engine.POSTGRES]: "/icons/postgres.svg",
  } as Record<string, string>,
}));

let SheetConnectionIcon: typeof import("./SheetConnectionIcon").SheetConnectionIcon;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

const dummyTab = { id: "t1" } as unknown as SQLEditorTab;

beforeEach(async () => {
  vi.clearAllMocks();
  ({ SheetConnectionIcon } = await import("./SheetConnectionIcon"));
});

describe("SheetConnectionIcon", () => {
  test("renders engine icon when connected", () => {
    mocks.isConnectedSQLEditorTab.mockReturnValue(true);
    mocks.getConnectionForSQLEditorTab.mockReturnValue({
      instance: { engine: Engine.POSTGRES },
    });

    const { container, render, unmount } = renderIntoContainer(
      <SheetConnectionIcon tab={dummyTab} />
    );
    render();

    const img = container.querySelector("img");
    expect(img).not.toBeNull();
    expect(img?.getAttribute("src")).toBe("/icons/postgres.svg");

    unmount();
  });

  test("renders unlink icon when disconnected", () => {
    mocks.isConnectedSQLEditorTab.mockReturnValue(false);
    mocks.getConnectionForSQLEditorTab.mockReturnValue({ instance: undefined });

    const { container, render, unmount } = renderIntoContainer(
      <SheetConnectionIcon tab={dummyTab} />
    );
    render();

    expect(container.querySelector("img")).toBeNull();
    expect(container.querySelector("svg")).not.toBeNull();

    unmount();
  });

  test("renders unlink icon when instance engine has no icon path", () => {
    mocks.isConnectedSQLEditorTab.mockReturnValue(true);
    mocks.getConnectionForSQLEditorTab.mockReturnValue({
      instance: { engine: Engine.ENGINE_UNSPECIFIED },
    });

    const { container, render, unmount } = renderIntoContainer(
      <SheetConnectionIcon tab={dummyTab} />
    );
    render();

    expect(container.querySelector("img")).toBeNull();
    expect(container.querySelector("svg")).not.toBeNull();

    unmount();
  });
});
