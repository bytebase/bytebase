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
  routerPush: vi.fn(),
  usePagedData: vi.fn(),
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/components/AdvancedSearch", () => ({
  AdvancedSearch: () => createElement("div"),
}));

vi.mock("@/react/components/DatabaseGroupTable", () => ({
  DatabaseGroupTable: () => null,
}));

vi.mock("@/react/components/database", () => ({
  DatabaseTable: () => null,
}));

vi.mock("@/react/components/HumanizeTs", () => ({
  HumanizeTs: () => createElement("span"),
}));

vi.mock("@/react/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactNode }) => children,
  usePermissionCheck: () => [true, false],
}));

vi.mock("@/react/components/ProjectPageLayout", () => ({
  ProjectPageContent: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  ProjectPageFooter: ({ children }: { children: ReactNode }) =>
    createElement("footer", {}, children),
  ProjectPageInfo: () => null,
  ProjectPageLayout: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
}));

vi.mock("@/react/components/TaskStatusIcon", () => ({
  TaskStatusIcon: () => null,
}));

vi.mock("@/react/components/ui/badge", () => ({
  Badge: ({ children }: { children: ReactNode }) =>
    createElement("span", {}, children),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({ children, ...props }: { children: ReactNode }) =>
    createElement("button", props, children),
}));

vi.mock("@/react/components/ui/search-input", () => ({
  SearchInput: () => createElement("input"),
}));

vi.mock("@/react/components/ui/sheet", () => ({
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

vi.mock("@/react/components/ui/table", () => ({
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

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => ({
    email: "me@example.com",
    name: "users/me@example.com",
  }),
}));

vi.mock("@/react/hooks/useColumnWidths", () => ({
  useColumnWidths: (columns: { defaultWidth: number }[]) => ({
    widths: columns.map((column) => column.defaultWidth),
    totalWidth: columns.reduce((sum, column) => sum + column.defaultWidth, 0),
    onResizeStart: vi.fn(),
    setWidths: vi.fn(),
  }),
}));

vi.mock("@/react/hooks/useMediaQuery", () => ({
  useMediaQuery: () => false,
}));

vi.mock("@/react/hooks/usePagedData", () => ({
  PagedTableFooter: () => null,
  usePagedData: mocks.usePagedData,
}));

vi.mock("@/react/hooks/useProjectByName", () => ({
  useProjectByName: (name: string) => ({ name }),
}));

vi.mock("@/react/router", () => ({
  router: {
    push: mocks.routerPush,
    resolve: () => ({ fullPath: "/projects/foo/plans/1" }),
  },
}));

vi.mock("@/react/router/NavigationScrollRestoration", () => ({
  useScrollRestorationLoadMore: () => undefined,
}));

vi.mock("@/react/stores/app", () => {
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

  test("uses the generic POP cache contract for the URL-owned view", () => {
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
    expect(options?.cacheRestoreToken).toBe("plans-entry");
    expect(options?.cacheKey).toContain("project-plans");
    expect(options?.cacheKey).toContain("state:DELETED");
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
