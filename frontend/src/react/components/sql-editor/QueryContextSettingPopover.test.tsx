import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useSQLEditorTabStore: vi.fn(),
  useSQLEditorVueState: vi.fn(),
  useConnectionOfCurrentSQLEditorTab: vi.fn(),
  getInstanceResource: vi.fn(),
  readableDataSourceType: vi.fn((type: number) => `type-${type}`),
  orderBy: vi.fn((arr: unknown[]) => arr),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useConnectionOfCurrentSQLEditorTab: mocks.useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
}));

vi.mock("@/react/stores/sqlEditor/editor-vue-state", () => ({
  useSQLEditorVueState: mocks.useSQLEditorVueState,
}));

vi.mock("@/utils", () => ({
  getInstanceResource: mocks.getInstanceResource,
  readableDataSourceType: mocks.readableDataSourceType,
}));

vi.mock("lodash-es", () => ({
  orderBy: mocks.orderBy,
}));

// Inline Popover and Tooltip so the popover content renders synchronously
vi.mock("@/react/components/ui/popover", () => ({
  Popover: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="popover">{children}</div>
  ),
  PopoverTrigger: ({
    children,
    render: renderEl,
  }: {
    children?: React.ReactNode;
    render?: React.ReactElement;
  }) => (
    <div data-testid="popover-trigger">
      {renderEl ? renderEl : null}
      {children}
    </div>
  ),
  PopoverContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="popover-content">{children}</div>
  ),
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/react/components/MaxRowCountSelect", () => ({
  MaxRowCountSelect: ({
    value,
    onChange,
  }: {
    value: number;
    onChange: (n: number) => void;
  }) => (
    <div data-testid="max-row-count-select">
      <button type="button" onClick={() => onChange(500)} data-value={value}>
        row-count
      </button>
    </div>
  ),
}));

let QueryContextSettingPopover: typeof import("./QueryContextSettingPopover").QueryContextSettingPopover;

const mockInstance = {
  engine: 0, // Not REDIS (Engine.REDIS = 8)
  dataSources: [
    {
      id: "ds-admin",
      type: 0 /* ADMIN */,
      username: "admin-user",
      redisType: 0,
    },
    { id: "ds-ro", type: 1 /* READ_ONLY */, username: "ro-user", redisType: 0 },
  ],
};

const mockConnection = {
  database: "instances/inst1/databases/mydb",
  instance: "instances/inst1",
};

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

  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });
  mocks.getInstanceResource.mockReturnValue(mockInstance);
  mocks.orderBy.mockImplementation((arr: unknown[]) => arr);

  mocks.useSQLEditorTabStore.mockReturnValue({
    currentTab: { mode: "WORKSHEET" },
    updateCurrentTab: vi.fn(),
  });
  mocks.useSQLEditorVueState.mockReturnValue({
    resultRowsLimit: 1000,
    redisCommandOption: 0,
    queryDataPolicy: {
      allowAdminDataSource: true,
      maximumResultRows: Number.MAX_VALUE,
    },
  });
  mocks.useConnectionOfCurrentSQLEditorTab.mockReturnValue({
    database: { value: { name: "instances/inst1/databases/mydb" } },
    connection: { value: mockConnection },
  });

  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

  ({ QueryContextSettingPopover } = await import(
    "./QueryContextSettingPopover"
  ));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("QueryContextSettingPopover", () => {
  test("returns null when current tab mode is ADMIN", () => {
    mocks.useSQLEditorTabStore.mockReturnValue({
      currentTab: { mode: "ADMIN" },
      updateCurrentTab: vi.fn(),
    });
    const { container, render, unmount } = renderIntoContainer(
      <QueryContextSettingPopover />
    );
    render();
    expect(container.querySelector("[data-testid='popover']")).toBeNull();
    unmount();
  });

  test("renders trigger button with ChevronDown when mode is not ADMIN", () => {
    const { container, render, unmount } = renderIntoContainer(
      <QueryContextSettingPopover />
    );
    render();
    expect(container.querySelector("[data-testid='popover']")).not.toBeNull();
    // The SVG icon (ChevronDown) should be present somewhere in the trigger
    expect(container.querySelector("svg")).not.toBeNull();
    unmount();
  });

  test("renders data source options from instance.dataSources", () => {
    const { container, render, unmount } = renderIntoContainer(
      <QueryContextSettingPopover />
    );
    render();
    const content = container.querySelector("[data-testid='popover-content']");
    // Should have radio inputs for each data source + automatic
    const radios = content?.querySelectorAll("input[type='radio']") ?? [];
    // automatic + 2 data sources = 3
    expect(radios.length).toBeGreaterThanOrEqual(3);
    unmount();
  });

  test("clicking a data source label calls tabStore.updateCurrentTab with updated connection", () => {
    const updateCurrentTab = vi.fn();
    mocks.useSQLEditorTabStore.mockReturnValue({
      currentTab: { mode: "WORKSHEET" },
      updateCurrentTab,
    });
    const { container, render, unmount } = renderIntoContainer(
      <QueryContextSettingPopover />
    );
    render();
    const content = container.querySelector("[data-testid='popover-content']");
    const labels = Array.from(content?.querySelectorAll("label") ?? []);
    // Second label is first real data source (ds-admin), first is "Automatic"
    const dsLabel = labels[1] as HTMLLabelElement;
    act(() => {
      // React synthetic events fire from direct click; native click triggers the label
      // which causes the radio's onChange to fire via React's synthetic event delegation.
      dsLabel.click();
    });
    expect(updateCurrentTab).toHaveBeenCalled();
    unmount();
  });

  test("Redis sub-group does not render when engine is not REDIS", () => {
    const { container, render, unmount } = renderIntoContainer(
      <QueryContextSettingPopover />
    );
    render();
    const content = container.querySelector("[data-testid='popover-content']");
    const redisRadios =
      content?.querySelectorAll("input[name='redis-command']") ?? [];
    expect(redisRadios.length).toBe(0);
    unmount();
  });

  test("Redis sub-group renders when engine is REDIS", () => {
    // Engine.REDIS = 8 (from common_pb.d.ts)
    mocks.getInstanceResource.mockReturnValue({
      ...mockInstance,
      engine: 8,
    });
    const { container, render, unmount } = renderIntoContainer(
      <QueryContextSettingPopover />
    );
    render();
    const content = container.querySelector("[data-testid='popover-content']");
    const redisRadios =
      content?.querySelectorAll("input[name='redis-command']") ?? [];
    expect(redisRadios.length).toBe(2);
    unmount();
  });

  test("MaxRowCountSelect renders inside popover content", () => {
    const { container, render, unmount } = renderIntoContainer(
      <QueryContextSettingPopover />
    );
    render();
    const content = container.querySelector("[data-testid='popover-content']");
    expect(
      content?.querySelector("[data-testid='max-row-count-select']")
    ).not.toBeNull();
    unmount();
  });
});
