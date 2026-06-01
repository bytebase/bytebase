import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  // useAppProject (app store) return value.
  projectData: { name: "projects/proj1", title: "Project 1" } as {
    name: string;
    title: string;
  },
  // Zustand editor store project name.
  project: "projects/proj1" as string,
  // New zustand store state + setter.
  state: {
    highlightAccessGrantName: undefined as string | undefined,
  },
  setHighlightAccessGrantName: vi.fn(),
  searchMyAccessGrants: vi.fn(),
  fetchDatabases: vi
    .fn()
    .mockResolvedValue({ databases: [], nextPageToken: "" }),
  getOrFetchDatabaseByName: vi.fn().mockResolvedValue({}),
  fetchIssueByName: vi.fn().mockResolvedValue({}),
  hasFeature: vi.fn(() => true),
  sqlEditorEventsEmit: vi.fn().mockResolvedValue(undefined),
  getDefaultPagination: vi.fn(() => 20),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useAppProject", () => ({
  useAppProject: () => mocks.projectData,
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      searchMyAccessGrants: mocks.searchMyAccessGrants,
      fetchDatabases: mocks.fetchDatabases,
      getOrFetchDatabaseByName: mocks.getOrFetchDatabaseByName,
      fetchIssueByName: mocks.fetchIssueByName,
      hasFeature: mocks.hasFeature,
    }),
}));

// Zustand editor store — active project read.
vi.mock("@/react/stores/sqlEditor/editor", () => ({
  useSQLEditorEditorState: (selector: (s: { project: string }) => unknown) =>
    selector({ project: mocks.project }),
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  useSQLEditorStore: Object.assign(
    (
      selector: (s: {
        highlightAccessGrantName: string | undefined;
        setHighlightAccessGrantName: (v: string | undefined) => void;
      }) => unknown
    ) =>
      selector({
        highlightAccessGrantName: mocks.state.highlightAccessGrantName,
        setHighlightAccessGrantName: mocks.setHighlightAccessGrantName,
      }),
    {
      getState: () => ({
        highlightAccessGrantName: mocks.state.highlightAccessGrantName,
        setHighlightAccessGrantName: mocks.setHighlightAccessGrantName,
      }),
    }
  ),
}));

vi.mock("@/utils", () => ({
  getDefaultPagination: mocks.getDefaultPagination,
}));

vi.mock("@/views/sql-editor/events", () => ({
  sqlEditorEvents: {
    emit: mocks.sqlEditorEventsEmit,
  },
}));

vi.mock("@/types/proto-es/v1/access_grant_service_pb", () => ({
  AccessGrant_Status: {
    ACTIVE: 2,
    PENDING: 1,
    REVOKED: 3,
    0: "UNSPECIFIED",
    1: "PENDING",
    2: "ACTIVE",
    3: "REVOKED",
  },
}));

vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanFeature: { FEATURE_JIT: 5 },
}));

vi.mock("@/react/hooks/useSQLEditorBridge", () => ({
  useSQLEditorFeature: () => mocks.hasFeature(),
}));

vi.mock("@/react/components/AdvancedSearch", () => ({
  AdvancedSearch: ({
    placeholder,
    onParamsChange,
  }: {
    placeholder?: string;
    onParamsChange: (p: unknown) => void;
  }) => (
    <input
      data-testid="advanced-search"
      placeholder={placeholder}
      onChange={() => onParamsChange({ query: "", scopes: [] })}
    />
  ),
}));

vi.mock("@/react/components/FeatureBadge", () => ({
  FeatureBadge: () => <span data-testid="feature-badge" />,
}));

vi.mock("@/react/components/PermissionGuard", () => ({
  PermissionGuard: ({
    children,
  }: {
    children: (props: { disabled: boolean }) => React.ReactNode;
  }) => <>{children({ disabled: false })}</>,
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
    ...props
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
    [key: string]: unknown;
  }) => (
    <button disabled={disabled} onClick={onClick} {...props}>
      {children}
    </button>
  ),
}));

vi.mock("./AccessGrantItem", () => ({
  AccessGrantItem: ({
    grant,
    onRun,
    onRequest,
  }: {
    grant: { name: string; query: string };
    onRun: (g: unknown) => void;
    onRequest: (g: unknown) => void;
  }) => (
    <div data-testid="access-grant-item" data-grant-name={grant.name}>
      <span>{grant.query}</span>
      <button data-run-btn onClick={() => onRun(grant)}>
        Run
      </button>
      <button data-request-btn onClick={() => onRequest(grant)}>
        Re-request
      </button>
    </div>
  ),
}));

vi.mock("./AccessGrantRequestDrawer", () => ({
  AccessGrantRequestDrawer: ({ onClose }: { onClose: () => void }) => (
    <div data-testid="access-grant-request-drawer">
      <button data-close-btn onClick={onClose}>
        Close
      </button>
    </div>
  ),
}));

type GrantLike = {
  name: string;
  query: string;
  targets: string[];
  unmask: boolean;
  issue: string;
  status: number;
  expiration: { case: string };
};

const makeGrant = (name: string): GrantLike => ({
  name,
  query: `SELECT * FROM ${name}`,
  targets: ["instances/inst1/databases/db1"],
  unmask: false,
  issue: "",
  status: 2,
  expiration: { case: "none" },
});

// Stub ResizeObserver — not provided by jsdom
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

let AccessPane: typeof import("./AccessPane").AccessPane;

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

const setupDefaultMocks = () => {
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });

  mocks.projectData = { name: "projects/proj1", title: "Project 1" };

  mocks.project = "projects/proj1";

  mocks.state.highlightAccessGrantName = undefined;

  mocks.searchMyAccessGrants.mockResolvedValue({
    accessGrants: [],
    nextPageToken: "",
  });

  mocks.fetchIssueByName.mockResolvedValue({});

  mocks.fetchDatabases.mockResolvedValue({ databases: [], nextPageToken: "" });
  mocks.getOrFetchDatabaseByName.mockResolvedValue({});

  mocks.hasFeature.mockReturnValue(true);
  mocks.getDefaultPagination.mockReturnValue(20);
};

beforeEach(async () => {
  vi.clearAllMocks();
  setupDefaultMocks();

  ({ AccessPane } = await import("./AccessPane"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("AccessPane", () => {
  test("empty state — shows no-access-requests text when no grants and not loading", async () => {
    const { container, render, unmount } = renderIntoContainer(<AccessPane />);
    render();

    // Wait for effects to settle
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    expect(container.textContent).toContain("sql-editor.no-access-requests");
    unmount();
  });

  test("loading state — shows spinner while loading", async () => {
    let resolveSearch: (val: unknown) => void = () => {};
    mocks.searchMyAccessGrants.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveSearch = resolve;
        })
    );

    const { container, render, unmount } = renderIntoContainer(<AccessPane />);
    render();

    // Should have spinner (Loader2 renders as svg with animate-spin)
    // Check loading indicator is present while fetch is pending
    const spinner = container.querySelector(".animate-spin");
    expect(spinner).not.toBeNull();

    // Resolve to avoid memory leaks
    await act(async () => {
      resolveSearch({ accessGrants: [], nextPageToken: "" });
      await new Promise((r) => setTimeout(r, 0));
    });

    unmount();
  });

  test("renders grants list when populated", async () => {
    const grants = [makeGrant("grant1"), makeGrant("grant2")];
    mocks.searchMyAccessGrants.mockResolvedValue({
      accessGrants: grants,
      nextPageToken: "",
    });

    const { container, render, unmount } = renderIntoContainer(<AccessPane />);
    render();

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    const items = container.querySelectorAll(
      "[data-testid='access-grant-item']"
    );
    expect(items.length).toBe(2);
    unmount();
  });

  test("click Request Access → drawer opens", async () => {
    const { container, render, unmount } = renderIntoContainer(<AccessPane />);
    render();

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    // No drawer initially
    expect(
      container.querySelector("[data-testid='access-grant-request-drawer']")
    ).toBeNull();

    // Find and click the "request access" button
    const buttons = container.querySelectorAll("button");
    let requestBtn: HTMLButtonElement | null = null;
    for (const btn of Array.from(buttons)) {
      if (btn.textContent?.includes("sql-editor.request-access")) {
        requestBtn = btn;
        break;
      }
    }
    expect(requestBtn).not.toBeNull();

    await act(async () => {
      requestBtn!.click();
    });

    expect(
      container.querySelector("[data-testid='access-grant-request-drawer']")
    ).not.toBeNull();

    unmount();
  });

  test("click Run on a grant → emits execute-sql event", async () => {
    const grant = makeGrant("grant1");
    mocks.searchMyAccessGrants.mockResolvedValue({
      accessGrants: [grant],
      nextPageToken: "",
    });

    const { container, render, unmount } = renderIntoContainer(<AccessPane />);
    render();

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    const runBtn = container.querySelector(
      "[data-run-btn]"
    ) as HTMLButtonElement;
    expect(runBtn).not.toBeNull();

    await act(async () => {
      runBtn.click();
    });

    expect(mocks.sqlEditorEventsEmit).toHaveBeenCalledWith(
      "execute-sql",
      expect.objectContaining({
        statement: grant.query,
      })
    );
    unmount();
  });
});
