import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  databaseRefValue: {
    name: "instances/i/databases/db",
    successfulSyncTime: undefined,
  },
  syncDatabase: vi.fn().mockResolvedValue(undefined),
  getOrFetchDatabaseMetadata: vi.fn().mockResolvedValue(undefined),
  isValidDatabaseName: vi.fn(() => true),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (k: string) => k,
    i18n: { language: "en-US" },
  }),
  Trans: ({ i18nKey }: { i18nKey: string }) => <span>{i18nKey}</span>,
}));

vi.mock("@/react/components/HumanizeTs", () => ({
  HumanizeTs: () => <span data-testid="humanize" />,
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({
    children,
    content,
  }: {
    children: React.ReactNode;
    content: React.ReactNode;
  }) => (
    <div data-testid="tooltip-wrap">
      <div data-testid="tooltip-content">{content}</div>
      {children}
    </div>
  ),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
  }) => (
    <button
      type="button"
      data-testid="sync-button"
      onClick={onClick}
      disabled={disabled}
    >
      {children}
    </button>
  ),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: (getter: () => unknown) => getter(),
}));

vi.mock("@/store", () => ({
  useDatabaseV1Store: () => ({ syncDatabase: mocks.syncDatabase }),
  useDBSchemaV1Store: () => ({
    getOrFetchDatabaseMetadata: mocks.getOrFetchDatabaseMetadata,
  }),
}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useConnectionOfCurrentSQLEditorTab: () => ({
    database: { value: mocks.databaseRefValue },
  }),
}));

vi.mock("@/types", () => ({
  getDateForPbTimestampProtoEs: () => undefined,
  isValidDatabaseName: mocks.isValidDatabaseName,
}));

let SyncSchemaButton: typeof import("./SyncSchemaButton").SyncSchemaButton;

const renderInto = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  return {
    container,
    render: () => act(() => root.render(element)),
    unmount: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.isValidDatabaseName.mockReturnValue(true);
  ({ SyncSchemaButton } = await import("./SyncSchemaButton"));
});

describe("SyncSchemaButton", () => {
  test("disables the button when there's no valid database name", () => {
    mocks.isValidDatabaseName.mockReturnValue(false);
    const { container, render, unmount } = renderInto(<SyncSchemaButton />);
    render();
    const btn = container.querySelector(
      "[data-testid='sync-button']"
    ) as HTMLButtonElement;
    expect(btn).not.toBeNull();
    expect(btn.disabled).toBe(true);
    // Disabled state skips the tooltip wrapper.
    expect(container.querySelector("[data-testid='tooltip-wrap']")).toBeNull();
    unmount();
  });

  test("clicking the button calls syncDatabase + getOrFetchDatabaseMetadata", async () => {
    const { container, render, unmount } = renderInto(<SyncSchemaButton />);
    render();
    const btn = container.querySelector(
      "[data-testid='sync-button']"
    ) as HTMLButtonElement;
    await act(async () => {
      btn.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(mocks.syncDatabase).toHaveBeenCalledWith(
      "instances/i/databases/db",
      true
    );
    expect(mocks.getOrFetchDatabaseMetadata).toHaveBeenCalledWith({
      database: "instances/i/databases/db",
      skipCache: true,
    });
    unmount();
  });

  test("renders a tooltip with the click-to-sync hint when enabled", () => {
    const { container, render, unmount } = renderInto(<SyncSchemaButton />);
    render();
    expect(
      container.querySelector("[data-testid='tooltip-wrap']")
    ).not.toBeNull();
    expect(container.textContent).toContain("sql-editor.click-to-sync-now");
    unmount();
  });
});
