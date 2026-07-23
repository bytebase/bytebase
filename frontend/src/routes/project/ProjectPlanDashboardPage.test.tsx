import { act, createElement, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { ProjectPlanDashboardPage } from "./ProjectPlanDashboardPage";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const plan = {
  name: "projects/foo/plans/1",
  title: "Plan 1",
  state: State.ACTIVE,
  creator: "users/creator@example.com",
  issue: "",
  hasRollout: false,
  specs: [],
  rolloutStageSummaries: [],
} as unknown as Plan;

const mocks = vi.hoisted(() => ({
  batchGetOrFetchUsers: vi.fn(async () => []),
  listUsers: vi.fn(async () => ({ users: [] })),
  listScrollRestorationKey: undefined as string | undefined,
  markListScrollRestorationEntry: vi.fn(),
  routerPush: vi.fn(),
  usePagedData: vi.fn(),
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/AdvancedSearch", () => ({
  AdvancedSearch: () => createElement("div"),
}));

vi.mock("@/components/DatabaseGroupTable", () => ({
  DatabaseGroupTable: () => null,
}));

vi.mock("@/components/database", () => ({
  DatabaseTable: () => null,
}));

vi.mock("@/components/HumanizeTs", () => ({
  HumanizeTs: () => createElement("span"),
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactNode }) => children,
  usePermissionCheck: () => [true, false],
}));

vi.mock("@/components/ProjectPageLayout", () => ({
  ProjectPageContent: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  ProjectPageFooter: ({ children }: { children: ReactNode }) =>
    createElement("footer", {}, children),
  ProjectPageInfo: () => null,
  ProjectPageLayout: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
}));

vi.mock("@/components/TaskStatusIcon", () => ({
  TaskStatusIcon: () => null,
}));

vi.mock("@/components/ui/badge", () => ({
  Badge: ({ children }: { children: ReactNode }) =>
    createElement("span", {}, children),
}));

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: { children: ReactNode }) =>
    createElement("button", props, children),
}));

vi.mock("@/components/ui/search-input", () => ({
  SearchInput: () => createElement("input"),
}));

vi.mock("@/components/ui/sheet", () => ({
  Sheet: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetBody: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetContent: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetFooter: ({ children }: { children: ReactNode }) =>
    createElement("footer", {}, children),
  SheetHeader: ({ children }: { children: ReactNode }) =>
    createElement("header", {}, children),
  SheetTitle: ({ children }: { children: ReactNode }) =>
    createElement("h2", {}, children),
}));

vi.mock("@/components/ui/table", () => ({
  Table: ({ children, ...props }: { children: ReactNode }) =>
    createElement("table", props, children),
  TableBody: ({ children }: { children: ReactNode }) =>
    createElement("tbody", {}, children),
  TableCell: ({ children }: { children?: ReactNode }) =>
    createElement("td", {}, children),
  TableHead: ({ children }: { children?: ReactNode }) =>
    createElement("th", {}, children),
  TableHeader: ({ children }: { children: ReactNode }) =>
    createElement("thead", {}, children),
  TableRow: ({ children, ...props }: { children?: ReactNode }) =>
    createElement("tr", props, children),
}));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => ({
    email: "me@example.com",
    name: "users/me@example.com",
  }),
}));

vi.mock("@/hooks/useColumnWidths", () => ({
  useColumnWidths: (columns: { defaultWidth: number }[]) => ({
    widths: columns.map((column) => column.defaultWidth),
    totalWidth: columns.reduce((sum, column) => sum + column.defaultWidth, 0),
    onResizeStart: vi.fn(),
    setWidths: vi.fn(),
  }),
}));

vi.mock("@/hooks/useMediaQuery", () => ({
  useMediaQuery: () => false,
}));

vi.mock("@/hooks/usePagedData", () => ({
  PagedTableFooter: () => null,
  usePagedData: mocks.usePagedData,
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: (name: string) => ({ name }),
}));

vi.mock("@/app/router", () => ({
  router: {
    push: mocks.routerPush,
    resolve: () => ({ fullPath: "/projects/foo/plans/1" }),
  },
}));

vi.mock("@/app/router/NavigationScrollRestoration", () => ({
  markListScrollRestorationEntry: mocks.markListScrollRestorationEntry,
  useListScrollRestorationKey: () => mocks.listScrollRestorationKey,
  useListScrollRestorationLoadMore: () => undefined,
}));

vi.mock("@/stores/app", () => {
  const storeState = {
    projectsByName: {},
    listUsers: mocks.listUsers,
    batchGetOrFetchUsers: mocks.batchGetOrFetchUsers,
    environmentList: [],
    getUserByIdentifier: () => undefined,
  };
  const useAppStore = (selector: (state: typeof storeState) => unknown) =>
    selector(storeState);
  return {
    useAppStore: Object.assign(useAppStore, {
      getState: () => ({
        batchGetOrFetchDatabases: vi.fn(async () => []),
        listPlans: vi.fn(async () => ({ plans: [], nextPageToken: "" })),
      }),
    }),
  };
});

describe("ProjectPlanDashboardPage scroll restoration", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    mocks.usePagedData.mockReset();
    mocks.listScrollRestorationKey = undefined;
    mocks.usePagedData.mockReturnValue({
      dataList: [plan],
      hasMore: true,
      isFetchingMore: false,
      isLoading: false,
      loadMore: vi.fn(),
      onPageSizeChange: vi.fn(),
      pageSize: 50,
      pageSizeOptions: [50, 100],
      refresh: vi.fn(),
      removeCache: vi.fn(),
      updateCache: vi.fn(),
    });
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.clearAllMocks();
  });

  test("does not restore paged data for a generic POP navigation", () => {
    act(() => {
      root.render(
        <MemoryRouter
          initialEntries={[
            {
              pathname: "/projects/foo/plans",
              search: "?q=state%3ADELETED",
              key: "plans-entry",
            },
          ]}
        >
          <ProjectPlanDashboardPage projectId="foo" />
        </MemoryRouter>
      );
    });

    const options = mocks.usePagedData.mock.lastCall?.[0] as
      | { cacheKey?: string; cacheRestoreToken?: string }
      | undefined;
    expect(options?.cacheRestoreToken).toBeUndefined();
    expect(options?.cacheKey).toContain("project-plans");
    expect(options?.cacheKey).toContain("state:DELETED");
  });

  test("uses the item-click restoration key for paged data", () => {
    mocks.listScrollRestorationKey = "plans-entry";
    act(() => {
      root.render(
        <MemoryRouter>
          <ProjectPlanDashboardPage projectId="foo" />
        </MemoryRouter>
      );
    });

    const options = mocks.usePagedData.mock.lastCall?.[0] as
      | { cacheRestoreToken?: string }
      | undefined;
    expect(options?.cacheRestoreToken).toBe("plans-entry");
  });

  test("marks the list entry before opening a Plan row", () => {
    act(() => {
      root.render(
        <MemoryRouter>
          <ProjectPlanDashboardPage projectId="foo" />
        </MemoryRouter>
      );
    });

    const row = container.querySelector<HTMLElement>(
      '[data-testid="plan-list-item"]'
    );
    expect(row).not.toBeNull();
    act(() => row?.click());

    expect(mocks.markListScrollRestorationEntry).toHaveBeenCalledOnce();
    expect(mocks.routerPush).toHaveBeenCalledWith("/projects/foo/plans/1");
    expect(
      mocks.markListScrollRestorationEntry.mock.invocationCallOrder[0]
    ).toBeLessThan(mocks.routerPush.mock.invocationCallOrder[0]);
  });

  test("exposes every Plan row as a semantic restoration anchor", () => {
    act(() => {
      root.render(
        <MemoryRouter>
          <ProjectPlanDashboardPage projectId="foo" />
        </MemoryRouter>
      );
    });

    expect(
      container.querySelector(
        '[data-scroll-restoration-anchor="projects/foo/plans/1"]'
      )
    ).not.toBeNull();
  });
});
