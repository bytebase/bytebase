import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useSQLEditorStore: vi.fn(),
  currentAsidePanelTab: "WORKSHEET" as string,
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  useSQLEditorStore: mocks.useSQLEditorStore,
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

let TabItem: typeof import("./TabItem").TabItem;

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
  mocks.currentAsidePanelTab = "WORKSHEET";
  mocks.useSQLEditorStore.mockImplementation(
    (selector: (s: { asidePanelTab: string }) => unknown) =>
      selector({ asidePanelTab: mocks.currentAsidePanelTab })
  );
  ({ TabItem } = await import("./TabItem"));
});

describe("TabItem", () => {
  test("renders label for WORKSHEET tab", () => {
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="WORKSHEET" onClick={() => {}} />
    );
    render();
    expect(container.textContent).toContain("worksheet.self");
    expect(container.querySelector("button")).not.toBeNull();
    unmount();
  });

  test("renders label for SCHEMA tab", () => {
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="SCHEMA" onClick={() => {}} />
    );
    render();
    expect(container.textContent).toContain("common.schema");
    unmount();
  });

  test("renders label for HISTORY tab", () => {
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="HISTORY" onClick={() => {}} />
    );
    render();
    expect(container.textContent).toContain("common.history");
    unmount();
  });

  test("renders label for ACCESS tab", () => {
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="ACCESS" onClick={() => {}} />
    );
    render();
    expect(container.textContent).toContain("sql-editor.jit");
    unmount();
  });

  test("applies active class when asidePanelTab matches", () => {
    mocks.currentAsidePanelTab = "SCHEMA";
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="SCHEMA" onClick={() => {}} />
    );
    render();
    const button = container.querySelector("button");
    expect(button?.className).toContain("bg-accent/10");
    expect(button?.className).toContain("text-accent");
    unmount();
  });

  test("does NOT apply active class when asidePanelTab differs", () => {
    mocks.currentAsidePanelTab = "WORKSHEET";
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="SCHEMA" onClick={() => {}} />
    );
    render();
    const button = container.querySelector("button");
    expect(button?.className).not.toContain("bg-accent/10");
    unmount();
  });

  test("calls onClick when button is clicked", () => {
    const handler = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <TabItem tab="SCHEMA" onClick={handler} />
    );
    render();
    act(() => {
      container.querySelector("button")?.click();
    });
    expect(handler).toHaveBeenCalledTimes(1);
    unmount();
  });
});
