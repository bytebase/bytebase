import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useSQLEditorTabStore: vi.fn(),
  // Legacy Pinia useSQLEditorStore (editor.ts).
  useSQLEditorPiniaStore: vi.fn(),
  useConnectionOfCurrentSQLEditorTab: vi.fn(),
  // New zustand store setter.
  setShowConnectionPanel: vi.fn(),
  isValidInstanceName: vi.fn(),
  isValidDatabaseName: vi.fn(),
  extractDatabaseResourceName: vi.fn(),
  getDatabaseEnvironment: vi.fn(),
  getInstanceResource: vi.fn(),
  EngineIconPath: {} as Record<string, string>,
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
  useSQLEditorStore: mocks.useSQLEditorPiniaStore,
  useConnectionOfCurrentSQLEditorTab: mocks.useConnectionOfCurrentSQLEditorTab,
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  useSQLEditorStore: (
    selector: (s: { setShowConnectionPanel: (v: boolean) => void }) => unknown
  ) =>
    selector({
      setShowConnectionPanel: mocks.setShowConnectionPanel,
    }),
}));

vi.mock("@/types", () => ({
  isValidInstanceName: mocks.isValidInstanceName,
  isValidDatabaseName: mocks.isValidDatabaseName,
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: mocks.extractDatabaseResourceName,
  getDatabaseEnvironment: mocks.getDatabaseEnvironment,
  getInstanceResource: mocks.getInstanceResource,
}));

vi.mock("@/components/InstanceForm/constants", () => ({
  EngineIconPath: mocks.EngineIconPath,
}));

vi.mock("@/react/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environmentName }: { environmentName: string }) => (
    <span data-testid="env-label">{environmentName}</span>
  ),
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

let DatabaseChooser: typeof import("./DatabaseChooser").DatabaseChooser;

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

const mockDatabase = { name: "instances/inst1/databases/mydb", engine: 0 };
const mockInstance = {
  name: "instances/inst1",
  title: "My Instance",
  engine: 0,
};
const mockEnvironment = { name: "environments/prod" };

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useConnectionOfCurrentSQLEditorTab.mockReturnValue({
    database: { value: mockDatabase },
    instance: { value: mockInstance },
  });
  mocks.useSQLEditorTabStore.mockReturnValue({
    currentTab: { id: "tab1" },
    isInBatchMode: false,
  });
  mocks.useSQLEditorPiniaStore.mockReturnValue({ projectContextReady: true });
  mocks.isValidInstanceName.mockReturnValue(true);
  mocks.isValidDatabaseName.mockReturnValue(true);
  mocks.getInstanceResource.mockReturnValue(mockInstance);
  mocks.getDatabaseEnvironment.mockReturnValue(mockEnvironment);
  mocks.extractDatabaseResourceName.mockReturnValue({ databaseName: "mydb" });
  mocks.useVueState.mockImplementation((getter) => getter());
  ({ DatabaseChooser } = await import("./DatabaseChooser"));
});

describe("DatabaseChooser", () => {
  test("renders placeholder when no valid connection", () => {
    mocks.isValidInstanceName.mockReturnValue(false);
    mocks.isValidDatabaseName.mockReturnValue(false);
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseChooser />
    );
    render();
    expect(container.textContent).toContain(
      "sql-editor.select-a-database-to-start"
    );
    unmount();
  });

  test("renders breadcrumb with env label and instance title and database name when connected", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseChooser />
    );
    render();
    expect(container.querySelector("[data-testid='env-label']")).not.toBeNull();
    expect(container.textContent).toContain("My Instance");
    expect(container.textContent).toContain("mydb");
    unmount();
  });

  test("button is disabled when disabled prop is true", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseChooser disabled={true} />
    );
    render();
    const button = container.querySelector("button");
    expect(button?.hasAttribute("disabled")).toBe(true);
    unmount();
  });

  test("button is disabled when projectContextReady is false", () => {
    mocks.useSQLEditorPiniaStore.mockReturnValue({
      projectContextReady: false,
    });
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseChooser />
    );
    render();
    const button = container.querySelector("button");
    expect(button?.hasAttribute("disabled")).toBe(true);
    unmount();
  });

  test("click invokes setShowConnectionPanel(true)", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseChooser />
    );
    render();
    act(() => {
      container.querySelector("button")?.click();
    });
    expect(mocks.setShowConnectionPanel).toHaveBeenCalledWith(true);
    unmount();
  });
});
