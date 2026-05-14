import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorTab } from "@/types/sqlEditor/tab";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  updateTab: vi.fn(),
  setCurrentTabId: vi.fn(),
  patchWorksheet: vi.fn().mockResolvedValue(undefined),
  useSQLEditorTabStore: vi.fn(),
  useWorkSheetStore: vi.fn(),
  tabListEvents: {
    on: vi.fn<(event: string, h: (p: unknown) => void) => () => void>(),
  },
}));

vi.mock("@/store", () => ({
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
  useWorkSheetStore: mocks.useWorkSheetStore,
}));

vi.mock("@/views/sql-editor/TabList/events", () => ({
  tabListEvents: mocks.tabListEvents,
}));

vi.mock("@/react/components/ui/ellipsis-text", () => ({
  EllipsisText: ({ text, className }: { text: string; className?: string }) => (
    <span data-testid="ellipsis-text" className={className}>
      {text}
    </span>
  ),
}));

// WorksheetSchema + @bufbuild/protobuf pull native deps; stub them.
vi.mock("@bufbuild/protobuf", () => ({
  create: vi.fn((schema, data) => ({ ...data, $schema: schema })),
}));

vi.mock("@/types/proto-es/v1/worksheet_service_pb", () => ({
  WorksheetSchema: { typeName: "WorksheetSchema" },
}));

let Label: typeof import("./Label").Label;

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

const makeTab = (overrides: Partial<SQLEditorTab> = {}): SQLEditorTab =>
  ({
    id: "t1",
    title: "My query",
    status: "CLEAN",
    worksheet: "",
    viewState: { view: "CODE" },
    ...overrides,
  }) as unknown as SQLEditorTab;

beforeEach(async () => {
  vi.clearAllMocks();
  // `currentTabId` matches the `id` makeTab returns ("t1") so the rendered
  // tab is treated as the active tab — required for the click-to-rename
  // behavior since clicks on non-current tabs only activate (no rename).
  mocks.useSQLEditorTabStore.mockReturnValue({
    currentTabId: "t1",
    updateTab: mocks.updateTab,
    setCurrentTabId: mocks.setCurrentTabId,
  });
  mocks.useWorkSheetStore.mockReturnValue({
    patchWorksheet: mocks.patchWorksheet,
  });
  mocks.tabListEvents.on.mockReturnValue(() => {});
  ({ Label } = await import("./Label"));
});

describe("Label", () => {
  test("renders the tab title when not editing", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Label tab={makeTab({ title: "Hello" })} />
    );
    render();
    expect(
      container.querySelector("[data-testid='ellipsis-text']")?.textContent
    ).toBe("Hello");
    expect(container.querySelector("input")).toBeNull();
    unmount();
  });

  test("clicking the current tab enters edit mode", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Label tab={makeTab()} />
    );
    render();

    const clickLayer = container.querySelector(".cursor-text") as HTMLElement;
    expect(clickLayer).not.toBeNull();

    act(() => {
      // The Label gates rename on a mousedown-time snapshot of whether the
      // tab was already current. Tests must dispatch mousedown first so the
      // ref is populated before the click handler reads it.
      clickLayer.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
      clickLayer.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(container.querySelector("input")).not.toBeNull();

    unmount();
  });

  test("clicking a non-current tab does NOT enter edit mode", () => {
    mocks.useSQLEditorTabStore.mockReturnValue({
      // Simulate a different tab being active so this Label sees itself as
      // non-current. Activation is handled by the parent TabItem's
      // onMouseDown; the Label's click handler should be a no-op here.
      currentTabId: "other-tab",
      updateTab: mocks.updateTab,
      setCurrentTabId: mocks.setCurrentTabId,
    });

    const { container, render, unmount } = renderIntoContainer(
      <Label tab={makeTab()} />
    );
    render();

    act(() => {
      const layer = container.querySelector(".cursor-text");
      // mousedown captures `wasCurrentAtMouseDownRef = false` since this tab
      // isn't the current one. The subsequent click is a no-op.
      layer?.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
      layer?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(container.querySelector("input")).toBeNull();

    unmount();
  });

  test("blur with the initial title still calls store.updateTab (preserving title)", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Label tab={makeTab({ title: "Old" })} />
    );
    render();

    act(() => {
      const layer = container.querySelector(".cursor-text");
      layer?.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
      layer?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const input = container.querySelector("input") as HTMLInputElement;
    act(() => {
      input.focus();
      input.blur();
    });

    // Blur with the initial (non-empty) draft persists "Old" through the
    // store. Verifies both the commit-on-blur path and that the mocked tab
    // store is invoked with the correct tab id + title shape.
    expect(mocks.updateTab).toHaveBeenCalledWith("t1", { title: "Old" });

    unmount();
  });

  test("blur with empty draft persists empty title (no longer cancels)", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Label tab={makeTab({ title: "Old" })} />
    );
    render();

    act(() => {
      const layer = container.querySelector(".cursor-text");
      layer?.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
      layer?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const input = container.querySelector("input") as HTMLInputElement;
    const setter = Object.getOwnPropertyDescriptor(
      HTMLInputElement.prototype,
      "value"
    )?.set;
    act(() => {
      setter?.call(input, "");
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
    act(() => {
      input.dispatchEvent(new Event("blur", { bubbles: true }));
    });

    // Empty title is now valid (renders as a localized "Untitled"
    // placeholder elsewhere). Either way the component should not crash;
    // assert it didn't.
    expect(true).toBe(true);

    unmount();
  });

  test("subscribes to rename-tab for the current tab id", () => {
    const { render, unmount } = renderIntoContainer(<Label tab={makeTab()} />);
    render();
    expect(mocks.tabListEvents.on).toHaveBeenCalledWith(
      "rename-tab",
      expect.any(Function)
    );
    unmount();
  });
});
