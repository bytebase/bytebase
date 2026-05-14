import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  setShowConnectionPanel: vi.fn(),
  useSQLEditorStore: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  useSQLEditorStore: mocks.useSQLEditorStore,
}));

let ConnectionHolder: typeof import("./ConnectionHolder").ConnectionHolder;

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

beforeEach(async () => {
  vi.clearAllMocks();
  // The component selects `s.setShowConnectionPanel` from the store; the
  // mock invokes the supplied selector against a stub state.
  mocks.useSQLEditorStore.mockImplementation(
    (selector: (s: { setShowConnectionPanel: typeof vi.fn }) => unknown) =>
      selector({
        setShowConnectionPanel: mocks.setShowConnectionPanel,
      })
  );
  ({ ConnectionHolder } = await import("./ConnectionHolder"));
});

describe("ConnectionHolder", () => {
  test("renders the Connect-to-database label", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionHolder />
    );
    render();
    expect(container.textContent).toContain("sql-editor.connect-to-a-database");
    expect(container.querySelector("button")).not.toBeNull();
    unmount();
  });

  test("click invokes setShowConnectionPanel(true)", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionHolder />
    );
    render();
    const button = container.querySelector("button");
    act(() => {
      button?.click();
    });
    expect(mocks.setShowConnectionPanel).toHaveBeenCalledWith(true);
    unmount();
  });
});
