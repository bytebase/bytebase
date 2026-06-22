import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  // Zustand editor store project read.
  project: "projects/proj1" as string,
  // Current tab connection database used to derive default targets.
  currentTabDatabase: "instances/inst1/databases/db1" as string | undefined,
  // New zustand setters.
  setAsidePanelTab: vi.fn(),
  setHighlightAccessGrantName: vi.fn(),
  pushNotification: vi.fn(),
  fetchDatabases: vi
    .fn()
    .mockResolvedValue({ databases: [], nextPageToken: "" }),
  createAccessGrant: vi.fn(),
  routerResolve: vi.fn(() => ({ fullPath: "/projects/proj1/issues/123" })),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => ({
    name: "users/me",
    email: "me@example.com",
  }),
}));

vi.mock("@/react/stores/app", () => {
  // `notify` reuses the `pushNotification` vi.fn so the existing test
  // assertions on `mocks.pushNotification` keep working after the migration
  // from the Pinia helper to the app-store notification slice.
  const state = () => ({
    fetchDatabases: mocks.fetchDatabases,
    notify: mocks.pushNotification,
    // Consumed by useWorkspaceSQLEditorTheme (via useActiveSQLEditorTheme,
    // which drives the drawer's Monaco theme). Empty profile resolves to the
    // default theme.
    getWorkspaceProfile: () => ({}),
  });
  return {
    useAppStore: Object.assign(
      (selector: (s: ReturnType<typeof state>) => unknown) => selector(state()),
      { getState: state }
    ),
  };
});

// Zustand editor store — active project read.
vi.mock("@/react/stores/sqlEditor/editor", () => ({
  useSQLEditorEditorState: (selector: (s: { project: string }) => unknown) =>
    selector({ project: mocks.project }),
}));

// Zustand tab store — imperative getter to derive default targets, plus the
// reactive hook consumed by useActiveSQLEditorTheme (drives the drawer's Monaco
// theme). The tab has no mode, so the active theme resolves to the default.
vi.mock("@/react/stores/sqlEditor/tab", () => ({
  getSQLEditorTabsState: () => ({
    currentTabId: "tab1",
    tabsById: new Map([
      ["tab1", { connection: { database: mocks.currentTabDatabase } }],
    ]),
  }),
  useSQLEditorTabState: (
    selector: (s: {
      currentTabId: string;
      tabsById: Map<string, { mode?: string }>;
    }) => unknown
  ) =>
    selector({
      currentTabId: "tab1",
      tabsById: new Map([["tab1", {}]]),
    }),
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  useSQLEditorStore: (
    selector: (s: {
      setAsidePanelTab: (tab: string) => void;
      setHighlightAccessGrantName: (v: string | undefined) => void;
    }) => unknown
  ) =>
    selector({
      setAsidePanelTab: mocks.setAsidePanelTab,
      setHighlightAccessGrantName: mocks.setHighlightAccessGrantName,
    }),
}));

vi.mock("@/connect", () => ({
  accessGrantServiceClientConnect: {
    createAccessGrant: mocks.createAccessGrant,
  },
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: {
    resolve: mocks.routerResolve,
  },
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: vi.fn((name: string) => {
    const parts = name.split("/");
    return {
      databaseName: parts[parts.length - 1] ?? name,
      instance: parts[1] ?? "",
    };
  }),
  extractIssueUID: vi.fn((name: string) => name.split("/").pop() ?? "123"),
  extractProjectResourceName: vi.fn((name: string) => {
    const match = name.match(/projects\/(.+?)(?:\/|$)/);
    return match?.[1] ?? "";
  }),
}));

vi.mock("@bufbuild/protobuf", () => ({
  create: vi.fn((_schema: unknown, data: unknown) => data),
}));

vi.mock("@bufbuild/protobuf/wkt", () => ({
  DurationSchema: {},
  TimestampSchema: {},
}));

vi.mock("@/types/proto-es/v1/access_grant_service_pb", () => ({
  AccessGrant_Status: { PENDING: 1, ACTIVE: 2 },
  AccessGrantSchema: {},
  CreateAccessGrantRequestSchema: {},
}));

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({
    children,
    open,
    onOpenChange,
  }: {
    children: React.ReactNode;
    open: boolean;
    onOpenChange: (next: boolean) => void;
  }) => (
    <div
      data-testid="sheet"
      data-open={open}
      onClick={() => onOpenChange(false)}
    >
      {children}
    </div>
  ),
  SheetContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="sheet-content">{children}</div>
  ),
  SheetHeader: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="sheet-header">{children}</div>
  ),
  SheetTitle: ({ children }: { children: React.ReactNode }) => (
    <h2 data-testid="sheet-title">{children}</h2>
  ),
  SheetBody: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="sheet-body">{children}</div>
  ),
  SheetFooter: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="sheet-footer">{children}</div>
  ),
}));

vi.mock("@/react/components/ui/alert", () => ({
  Alert: ({
    children,
    title,
    description,
  }: {
    children?: React.ReactNode;
    title?: React.ReactNode;
    description?: React.ReactNode;
  }) => (
    <div data-testid="alert">
      {title}
      {description}
      {children}
    </div>
  ),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
    "data-submit-btn": submitBtn,
    ...props
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
    "data-submit-btn"?: boolean;
    [key: string]: unknown;
  }) => (
    <button
      data-submit-btn={submitBtn ? "" : undefined}
      disabled={disabled}
      onClick={onClick}
      {...props}
    >
      {children}
    </button>
  ),
}));

vi.mock("@/react/components/ui/combobox", () => ({
  Combobox: ({
    value,
    onChange,
    options,
    multiple,
  }: {
    value: string | string[];
    onChange: (val: string | string[]) => void;
    options: { value: string; label: string }[];
    multiple?: boolean;
  }) => (
    <select
      data-testid={multiple ? "multi-combobox" : "combobox"}
      multiple={multiple}
      value={multiple ? (value as string[]) : (value as string)}
      onChange={(e) => {
        if (multiple) {
          const selected = Array.from(e.target.selectedOptions).map(
            (o) => o.value
          );
          onChange(selected);
        } else {
          onChange(e.target.value);
        }
      }}
    >
      {options.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  ),
}));

vi.mock("@/react/components/ui/expiration-picker", () => ({
  ExpirationPicker: ({
    value,
    onChange,
  }: {
    value?: string;
    onChange: (val: string | undefined) => void;
  }) => (
    <input
      data-testid="expiration-picker"
      type="datetime-local"
      value={value ?? ""}
      onChange={(e) => onChange(e.target.value || undefined)}
    />
  ),
}));

vi.mock("@/react/components/ui/textarea", () => ({
  Textarea: ({
    value,
    onChange,
    ...props
  }: {
    value: string;
    onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => void;
    [key: string]: unknown;
  }) => (
    <textarea
      data-testid="textarea"
      value={value}
      onChange={onChange}
      {...props}
    />
  ),
}));

vi.mock("@/react/components/monaco/MonacoEditor", () => ({
  MonacoEditor: ({
    content,
    onChange,
  }: {
    content: string;
    onChange?: (val: string) => void;
  }) => (
    <textarea
      data-testid="monaco-editor"
      value={content}
      onChange={(e) => onChange?.(e.target.value)}
    />
  ),
}));

let AccessGrantRequestDrawer: typeof import("./AccessGrantRequestDrawer").AccessGrantRequestDrawer;

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

const setupMocks = () => {
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });

  mocks.project = "projects/proj1";
  mocks.currentTabDatabase = "instances/inst1/databases/db1";

  mocks.fetchDatabases.mockResolvedValue({ databases: [], nextPageToken: "" });
};

beforeEach(async () => {
  vi.clearAllMocks();
  setupMocks();

  // Mock window.open
  Object.defineProperty(window, "open", {
    value: vi.fn(),
    writable: true,
  });

  ({ AccessGrantRequestDrawer } = await import("./AccessGrantRequestDrawer"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("AccessGrantRequestDrawer", () => {
  test("renders with pre-filled targets, query, unmask when passed as props", () => {
    const onClose = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantRequestDrawer
        targets={["instances/inst1/databases/mydb"]}
        query="SELECT id FROM orders"
        unmask={true}
        onClose={onClose}
      />
    );
    render();

    const monacoEditor = container.querySelector(
      "[data-testid='monaco-editor']"
    ) as HTMLTextAreaElement;
    expect(monacoEditor).not.toBeNull();
    expect(monacoEditor.value).toBe("SELECT id FROM orders");

    const checkbox = container.querySelector(
      "input[type='checkbox']"
    ) as HTMLInputElement;
    expect(checkbox).not.toBeNull();
    expect(checkbox.checked).toBe(true);

    unmount();
  });

  test("Submit button disabled when required fields missing", () => {
    const onClose = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantRequestDrawer onClose={onClose} />
    );
    render();

    // No targets, no query, no reason — submit should be disabled
    const submitBtn = container.querySelector(
      "[data-submit-btn]"
    ) as HTMLButtonElement;
    expect(submitBtn).not.toBeNull();
    expect(submitBtn.disabled).toBe(true);

    unmount();
  });

  test("Submit calls createAccessGrant with correct payload shape", async () => {
    const mockResponse = {
      status: 2, // ACTIVE
      issue: "",
      name: "projects/proj1/accessGrants/grant-new",
    };
    mocks.createAccessGrant.mockResolvedValue(mockResponse);

    const onClose = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantRequestDrawer
        targets={["instances/inst1/databases/db1"]}
        query="SELECT * FROM t"
        onClose={onClose}
      />
    );
    render();

    // Fill in reason
    const textarea = container.querySelector(
      "[data-testid='textarea']"
    ) as HTMLTextAreaElement;
    await act(async () => {
      Object.defineProperty(textarea, "value", {
        writable: true,
        value: "test reason",
      });
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
      const changeEvent = new Event("change", { bubbles: true });
      Object.defineProperty(changeEvent, "target", {
        writable: false,
        value: textarea,
      });
      textarea.dispatchEvent(changeEvent);
    });

    // Submit button should now be enabled (targets + query + reason filled)
    const submitBtn = container.querySelector(
      "[data-submit-btn]"
    ) as HTMLButtonElement;
    expect(submitBtn).not.toBeNull();

    // Manually trigger submit
    await act(async () => {
      submitBtn.click();
    });

    // createAccessGrant should have been called
    expect(mocks.createAccessGrant).toHaveBeenCalled();
    unmount();
  });

  test("On success ACTIVE without issue → sets asidePanelTab and highlightAccessGrantName", async () => {
    const mockResponse = {
      status: 2, // ACTIVE
      issue: "",
      name: "projects/proj1/accessGrants/grant-xyz",
    };
    mocks.createAccessGrant.mockResolvedValue(mockResponse);

    const onClose = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantRequestDrawer
        targets={["instances/inst1/databases/db1"]}
        query="SELECT * FROM t"
        onClose={onClose}
      />
    );
    render();

    // Fill reason
    const textarea = container.querySelector(
      "[data-testid='textarea']"
    ) as HTMLTextAreaElement;
    await act(async () => {
      Object.defineProperty(textarea, "value", {
        writable: true,
        value: "my reason",
      });
      const changeEvent = new Event("change", { bubbles: true });
      Object.defineProperty(changeEvent, "target", {
        writable: false,
        value: textarea,
      });
      textarea.dispatchEvent(changeEvent);
    });

    const submitBtn = container.querySelector(
      "[data-submit-btn]"
    ) as HTMLButtonElement;
    await act(async () => {
      submitBtn.click();
    });

    // After submit: check pushNotification was called and zustand setters were invoked
    expect(mocks.pushNotification).toHaveBeenCalledWith(
      expect.objectContaining({ style: "SUCCESS" })
    );
    expect(mocks.setAsidePanelTab).toHaveBeenCalledWith("ACCESS");
    expect(mocks.setHighlightAccessGrantName).toHaveBeenCalledWith(
      "projects/proj1/accessGrants/grant-xyz"
    );
    expect(onClose).toHaveBeenCalled();
    unmount();
  });
});
