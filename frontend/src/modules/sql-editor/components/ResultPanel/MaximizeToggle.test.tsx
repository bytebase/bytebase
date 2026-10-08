import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  state: { resultPanelMaximized: false },
  setResultPanelMaximized: vi.fn((value: boolean) => {
    mocks.state.resultPanelMaximized = value;
  }),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/modules/sql-editor/store", () => {
  const state = {
    get resultPanelMaximized() {
      return mocks.state.resultPanelMaximized;
    },
    setResultPanelMaximized: mocks.setResultPanelMaximized,
  };
  return {
    useSQLEditorStore: Object.assign(
      (selector: (s: typeof state) => unknown) => selector(state),
      { getState: () => state }
    ),
  };
});

// The shared Tooltip renders a floating popup through a portal, which this
// test doesn't need; only the trigger it wraps matters here.
vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactElement }) => children,
}));

import { MaximizeToggle } from "./MaximizeToggle";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot> | undefined;

const render = () => {
  container = document.createElement("div");
  document.body.append(container);
  const mounted = createRoot(container);
  root = mounted;
  act(() => {
    mounted.render(<MaximizeToggle />);
  });
};

const button = () => container.querySelector("button") as HTMLButtonElement;

const pressEscape = () => {
  act(() => {
    window.dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true })
    );
  });
};

beforeEach(() => {
  document.body.innerHTML = "";
  mocks.state.resultPanelMaximized = false;
  mocks.setResultPanelMaximized.mockClear();
});

// Unmount between tests: a mounted toggle keeps its window keydown listener,
// and a leftover one would answer the next test's Escape.
afterEach(() => {
  if (!root) return;
  const mounted = root;
  root = undefined;
  act(() => {
    mounted.unmount();
  });
});

describe("MaximizeToggle", () => {
  test("offers Maximize while the pane is docked", () => {
    render();
    expect(button().getAttribute("aria-label")).toBe("common.maximize");
    expect(button().getAttribute("aria-pressed")).toBe("false");

    act(() => {
      button().click();
    });
    expect(mocks.setResultPanelMaximized).toHaveBeenCalledWith(true);
  });

  test("offers Restore while the pane is maximized", () => {
    mocks.state.resultPanelMaximized = true;
    render();
    expect(button().getAttribute("aria-label")).toBe("common.restore");
    expect(button().getAttribute("aria-pressed")).toBe("true");

    act(() => {
      button().click();
    });
    expect(mocks.setResultPanelMaximized).toHaveBeenCalledWith(false);
  });

  test("Escape restores a maximized pane", () => {
    mocks.state.resultPanelMaximized = true;
    render();
    pressEscape();
    expect(mocks.setResultPanelMaximized).toHaveBeenCalledWith(false);
  });

  test("Escape does nothing while the pane is docked", () => {
    render();
    pressEscape();
    expect(mocks.setResultPanelMaximized).not.toHaveBeenCalled();
  });

  test("Escape leaves the pane alone while a dialog is open", () => {
    mocks.state.resultPanelMaximized = true;
    render();
    const dialog = document.createElement("div");
    dialog.setAttribute("role", "dialog");
    document.body.append(dialog);

    pressEscape();
    expect(mocks.setResultPanelMaximized).not.toHaveBeenCalled();
  });

  test("Escape restores past a dialog kept mounted while closed", () => {
    mocks.state.resultPanelMaximized = true;
    render();
    for (const closedAttribute of ["hidden", "data-closed"]) {
      const closed = document.createElement("div");
      closed.setAttribute("role", "dialog");
      closed.setAttribute(closedAttribute, "");
      document.body.append(closed);
    }

    pressEscape();
    expect(mocks.setResultPanelMaximized).toHaveBeenCalledWith(false);
  });
});
