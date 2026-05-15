import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorTreeNode } from "@/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

const mocks = vi.hoisted(() => ({
  useHoverState: vi.fn(),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("./hover-state", () => ({
  useHoverState: mocks.useHoverState,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({}));

vi.mock("@/react/stores/sqlEditor/editor-vue-state", () => ({
  useSQLEditorVueState: () => ({ project: "" }),
}));

vi.mock("@/utils", () => ({
  getDatabaseProject: () => ({ name: "projects/p", title: "P" }),
  getInstanceResource: () => ({
    name: "instances/prod",
    title: "Prod",
    engine: "MYSQL",
  }),
  instanceV1Name: (inst: { title: string }) => inst.title,
  minmax: (v: number, lo: number, hi: number) => Math.max(lo, Math.min(hi, v)),
  projectV1Name: (p: { title: string }) => p.title,
}));

vi.mock("@/react/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environmentName }: { environmentName: string }) => (
    <span data-testid="env-label">{environmentName}</span>
  ),
}));

vi.mock("@/react/components/ui/layer", () => ({
  getLayerRoot: () => document.body,
  LAYER_SURFACE_CLASS: "",
}));

let DatabaseHoverPanel: typeof import("./DatabaseHoverPanel").DatabaseHoverPanel;

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

const makeDatabaseNode = (): SQLEditorTreeNode =>
  ({
    key: "databases/bb",
    meta: {
      type: "database",
      target: {
        name: "instances/prod/databases/bb",
        effectiveEnvironment: "environments/dev",
        project: "projects/p",
        labels: { env: "prod", tier: "" },
      },
    },
  }) as unknown as SQLEditorTreeNode;

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useVueState.mockImplementation((getter) => getter());
  ({ DatabaseHoverPanel } = await import("./DatabaseHoverPanel"));
});

describe("DatabaseHoverPanel", () => {
  test("renders nothing when state is undefined", () => {
    mocks.useHoverState.mockReturnValue({
      state: undefined,
      position: { x: 0, y: 0 },
      update: vi.fn(),
      setPosition: vi.fn(),
    });
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseHoverPanel offsetX={0} offsetY={0} margin={0} />
    );
    render();
    expect(container.querySelector("[data-testid='env-label']")).toBeNull();
    unmount();
  });

  test("renders database context when state + non-zero position provided", () => {
    mocks.useHoverState.mockReturnValue({
      state: { node: makeDatabaseNode() },
      position: { x: 100, y: 200 },
      update: vi.fn(),
      setPosition: vi.fn(),
    });
    const { render, unmount } = renderIntoContainer(
      <DatabaseHoverPanel offsetX={10} offsetY={20} margin={0} />
    );
    render();
    const panel = document.body.querySelector(
      "[data-testid='env-label']"
    ) as HTMLElement | null;
    expect(panel).not.toBeNull();
    expect(panel?.textContent).toBe("environments/dev");
    // Labels block: "env" key with value "prod", "tier" key with empty value
    expect(document.body.textContent).toContain("env");
    expect(document.body.textContent).toContain("prod");
    expect(document.body.textContent).toContain("tier");
    unmount();
  });

  test("clamps y to window height minus margin", () => {
    mocks.useHoverState.mockReturnValue({
      state: { node: makeDatabaseNode() },
      position: { x: 0, y: 99999 },
      update: vi.fn(),
      setPosition: vi.fn(),
    });
    const { render, unmount } = renderIntoContainer(
      <DatabaseHoverPanel offsetX={0} offsetY={0} margin={10} />
    );
    render();
    const fixed = document.body.querySelector(
      "div.fixed"
    ) as HTMLDivElement | null;
    expect(fixed).not.toBeNull();
    const top = parseFloat(fixed!.style.top);
    expect(top).toBeLessThanOrEqual(window.innerHeight);
    unmount();
  });
});
