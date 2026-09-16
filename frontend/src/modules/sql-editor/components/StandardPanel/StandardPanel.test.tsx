import type { ReactElement, RefObject } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

type PanelHandle = { collapse: () => void; expand: () => void };

const mocks = vi.hoisted(() => ({
  collapse: vi.fn(),
  expand: vi.fn(),
  // The editor pane's `onResize`, captured so a test can replay the layout
  // report the group sends on mount.
  onEditorResize: undefined as
    | ((size: { asPercentage: number }) => void)
    | undefined,
  state: {
    showAIPanel: false,
    resultPanelSize: 0.4,
    resultPanelMaximized: false,
    disconnected: false,
    setResultPanelMaximized: vi.fn(),
    setResultPanelMounted: vi.fn(),
    handleResultPanelResize: vi.fn(),
    handleEditorPanelResize: vi.fn(),
    setAsidePanelTab: vi.fn(),
    setShowConnectionPanel: vi.fn(),
  },
}));

vi.mock("react-resizable-panels", () => ({
  Group: ({ children }: { children: ReactElement | ReactElement[] }) => (
    <div>{children}</div>
  ),
  Panel: ({
    children,
    panelRef,
    onResize,
  }: {
    children?: ReactElement | ReactElement[];
    panelRef?: RefObject<PanelHandle | null>;
    onResize?: (size: { asPercentage: number }) => void;
  }) => {
    if (panelRef) {
      panelRef.current = { collapse: mocks.collapse, expand: mocks.expand };
      mocks.onEditorResize = onResize;
    }
    return <div>{children}</div>;
  },
  Separator: () => <div />,
}));

vi.mock("@/modules/ai/components", () => ({
  AIChatToSQL: () => <div />,
  AIContextProvider: ({ children }: { children: ReactElement }) => children,
}));

vi.mock("@/modules/schema-editor/resize", () => ({
  resizeHandleClass: () => "resize-handle",
}));

vi.mock("@/modules/sql-editor/components/ResultPanel/ResultPanel", () => ({
  ResultPanel: () => <div>result-panel</div>,
}));

vi.mock("./EditorMain", () => ({
  EditorMain: () => <div>editor-main</div>,
}));

vi.mock("@/modules/sql-editor/hooks/useSQLEditorState", () => ({
  useConnectionOfCurrentSQLEditorTab: () => ({ instance: {} }),
}));

vi.mock("@/modules/sql-editor/store", () => ({
  MINIMUM_RESULT_PANEL_SIZE: 0.2,
  MAXIMUM_RESULT_PANEL_SIZE: 0.8,
  selectEditorPanelSize: () => ({ size: 1, min: 0.5, max: 1 }),
  useSQLEditorStore: Object.assign(
    (selector: (state: typeof mocks.state) => unknown) =>
      selector(mocks.state),
    { getState: () => mocks.state }
  ),
}));

vi.mock("@/modules/sql-editor/store/tab", () => ({
  useCurrentSQLEditorTab: () => ({ id: "tab-1", mode: "SAVED_QUERY" }),
  useIsDisconnected: () => mocks.state.disconnected,
}));

vi.mock("@/utils", () => ({
  instanceV1HasReadonlyMode: () => true,
}));

import { StandardPanel } from "./StandardPanel";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot> | undefined;

const render = () => {
  container = document.createElement("div");
  document.body.append(container);
  const mounted = createRoot(container);
  root = mounted;
  act(() => {
    mounted.render(<StandardPanel />);
  });
};

beforeEach(() => {
  document.body.innerHTML = "";
  mocks.collapse.mockClear();
  mocks.expand.mockClear();
  mocks.onEditorResize = undefined;
  mocks.state.showAIPanel = false;
  mocks.state.resultPanelSize = 0.4;
  mocks.state.resultPanelMaximized = false;
  mocks.state.disconnected = false;
  mocks.state.setResultPanelMaximized.mockClear();
  mocks.state.setResultPanelMounted.mockClear();
  mocks.state.handleResultPanelResize.mockClear();
});

afterEach(() => {
  if (!root) return;
  const mounted = root;
  root = undefined;
  act(() => {
    mounted.unmount();
  });
});

describe("StandardPanel result pane", () => {
  test("collapses the editor while the result pane is maximized", () => {
    mocks.state.resultPanelMaximized = true;
    render();

    expect(mocks.collapse).toHaveBeenCalled();
    expect(mocks.expand).not.toHaveBeenCalled();
  });

  test("expands the editor while the result pane is docked", () => {
    render();

    expect(mocks.expand).toHaveBeenCalled();
    expect(mocks.collapse).not.toHaveBeenCalled();
  });

  // The group reports its default layout once on mount, before the collapse
  // lands. Reading "maximized" back out of that report cancelled it on every
  // remount — and the panel remounts on each tab switch.
  test("keeps a maximized pane through the group's first layout report", () => {
    mocks.state.resultPanelMaximized = true;
    render();

    act(() => {
      mocks.onEditorResize?.({ asPercentage: 60 });
    });

    expect(mocks.state.setResultPanelMaximized).not.toHaveBeenCalled();
  });

  test("reports the result pane's share of a resize", () => {
    render();

    act(() => {
      mocks.onEditorResize?.({ asPercentage: 75 });
    });

    expect(mocks.state.handleResultPanelResize).toHaveBeenCalledWith(0.25);
  });

  test("ignores a resize that reports no usable size", () => {
    render();

    act(() => {
      mocks.onEditorResize?.({ asPercentage: Number.NaN });
    });

    expect(mocks.state.handleResultPanelResize).not.toHaveBeenCalled();
  });

  test("tells the shell a result pane is on screen", () => {
    render();

    expect(mocks.state.setResultPanelMounted).toHaveBeenLastCalledWith(true);
  });

  // No result pane means no restore control, so the shell has to stop lending
  // the sidebar's width to a maximized one.
  test("tells the shell the result pane is gone once disconnected", () => {
    mocks.state.disconnected = true;
    render();

    expect(mocks.state.setResultPanelMounted).toHaveBeenLastCalledWith(false);
  });
});
