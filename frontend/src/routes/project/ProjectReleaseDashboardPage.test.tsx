import { act, createElement, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { State, VCSType } from "@/types/proto-es/v1/common_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Release } from "@/types/proto-es/v1/release_service_pb";
import { ProjectReleaseDashboardPage } from "./ProjectReleaseDashboardPage";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const gitOpsPlan = {
  name: "projects/foo/plans/7",
  title: "Release r1 to all environments",
  state: State.ACTIVE,
  creator: "users/ci@example.com",
  issue: "",
  hasRollout: true,
  specs: [
    {
      config: {
        case: "changeDatabaseConfig",
        value: { release: "projects/foo/releases/release_20261022-RC01" },
      },
    },
  ],
  rolloutStageSummaries: [],
} as unknown as Plan;

const release = {
  name: "projects/foo/releases/release_20261022-RC01",
  state: State.ACTIVE,
  vcsSource: {
    vcsType: VCSType.GITHUB,
    url: "https://github.com/acme/app/commit/0123456789abcdef",
  },
} as unknown as Release;

const mocks = vi.hoisted(() => ({
  fetchRelease: vi.fn(),
  listPlans: vi.fn(),
  probeListPlans: vi.fn(),
  canSetUpGitOps: true,
  routerPush: vi.fn(),
  usePagedData: vi.fn(),
}));

const store = vi.hoisted(() => ({
  current: undefined as
    | import("zustand").UseBoundStore<
        import("zustand").StoreApi<Record<string, unknown>>
      >
    | undefined,
}));

const pagedData = (dataList: Plan[]) => ({
  dataList,
  hasMore: false,
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

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/api", () => ({
  planServiceClientConnect: { listPlans: mocks.probeListPlans },
}));

vi.mock("@/app/router", () => ({
  router: {
    push: mocks.routerPush,
    resolve: ({
      name,
      params,
    }: {
      name: string;
      params: Record<string, string>;
    }) => ({ fullPath: `/${name}/${params.planId ?? ""}` }),
  },
}));

vi.mock("@/app/router/NavigationScrollRestoration", () => ({
  markListScrollRestorationEntry: vi.fn(),
  useListScrollRestorationKey: () => undefined,
  useListScrollRestorationLoadMore: () => undefined,
}));

vi.mock("@/components/AdvancedSearch", () => ({
  AdvancedSearch: () => createElement("div"),
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactNode }) => children,
  usePermissionCheck: () => [mocks.canSetUpGitOps, undefined],
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

vi.mock("@/components/TimeRangePicker", () => ({
  TimeRangePicker: () => null,
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

vi.mock("@/stores/app", async () => {
  const { create: createStore } = await import("zustand");
  store.current = createStore<Record<string, unknown>>(() => ({}));
  return { useAppStore: store.current };
});

const resetStore = () => {
  store.current?.setState(
    {
      releasesByName: {},
      environmentList: [],
      fetchRelease: mocks.fetchRelease,
      getEnvironmentByName: () => undefined,
      getUserByIdentifier: () => undefined,
      listPlans: mocks.listPlans,
    },
    true
  );
};

describe("ProjectReleaseDashboardPage", () => {
  let container: HTMLDivElement;
  let root: Root;

  const render = async () => {
    await act(async () => {
      root.render(
        <MemoryRouter initialEntries={["/projects/foo/releases?category=hotfix"]}>
          <ProjectReleaseDashboardPage projectId="foo" />
        </MemoryRouter>
      );
    });
  };

  const text = () => container.textContent ?? "";

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    resetStore();
    mocks.fetchRelease.mockImplementation(async (name: string) => {
      store.current?.setState((state) => ({
        releasesByName: {
          ...(state.releasesByName as Record<string, Release>),
          [name]: release,
        },
      }));
      return release;
    });
    mocks.listPlans.mockResolvedValue({ plans: [], nextPageToken: "" });
    mocks.probeListPlans.mockResolvedValue({ plans: [] });
    mocks.usePagedData.mockReturnValue(pagedData([gitOpsPlan]));
    mocks.canSetUpGitOps = true;
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.clearAllMocks();
  });

  test("lists GitOps Plans, ignoring the old category parameter", async () => {
    await render();

    const options = mocks.usePagedData.mock.lastCall?.[0] as {
      fetchList: (params: { pageSize: number; pageToken: string }) => unknown;
    };
    await options.fetchList({ pageSize: 50, pageToken: "" });

    expect(mocks.listPlans).toHaveBeenCalledWith(
      expect.objectContaining({
        find: {
          project: "projects/foo",
          query: "",
          hasRelease: true,
          specType: "change_database_config",
          state: "ACTIVE",
          creator: "",
          release: undefined,
          createdTsAfter: undefined,
          createdTsBefore: undefined,
          taskStatuses: undefined,
        },
      })
    );
  });

  test("renders the release ID, source commit, stages, and created columns", async () => {
    await render();

    const headers = Array.from(container.querySelectorAll("th")).map(
      (th) => th.textContent
    );
    expect(headers).toEqual([
      "release.self",
      "release.source",
      "rollout.stage.self",
      "common.created-at",
    ]);
    expect(text()).toContain("release_20261022-RC01");
    expect(text()).not.toContain(gitOpsPlan.title);
    expect(mocks.fetchRelease).toHaveBeenCalledOnce();
    const commit = container.querySelector<HTMLAnchorElement>(
      'a[href="https://github.com/acme/app/commit/0123456789abcdef"]'
    );
    expect(commit?.textContent).toBe("0123456");
  });

  test("shows - as the source when the release cannot be read", async () => {
    mocks.fetchRelease.mockRejectedValue(new Error("not found"));

    await render();

    expect(container.querySelector("a")).toBeNull();
    const sourceCell = container.querySelectorAll("td")[1];
    expect(sourceCell.textContent).toBe("-");
  });

  test("opens a Plan's rollout from its row", async () => {
    await render();

    act(() =>
      container
        .querySelector<HTMLElement>('[data-testid="plan-list-item"]')
        ?.click()
    );

    expect(mocks.routerPush).toHaveBeenCalledWith(
      "/workspace.project.plan.detail.rollout/7"
    );
  });

  test("offers GitOps setup at the top right once the project has a GitOps Plan", async () => {
    await render();

    expect(
      container.querySelector('[data-testid="release-gitops-setup"]')
    ).not.toBeNull();
  });

  test("shows the setup empty state when the project has no GitOps Plan", async () => {
    mocks.usePagedData.mockReturnValue(pagedData([]));

    await render();

    expect(
      container.querySelector('[data-testid="release-setup-empty-state"]')
    ).not.toBeNull();
    expect(text()).toContain("release.empty.title");
    expect(
      container.querySelector('[data-testid="release-gitops-setup"]')
    ).toBeNull();
    expect(mocks.probeListPlans).toHaveBeenCalledWith(
      expect.objectContaining({
        parent: "projects/foo",
        filter: 'has_release == true && spec_type == "change_database_config"',
        pageSize: 1,
      })
    );
  });

  test("shows No data when filters hide the project's GitOps Plans", async () => {
    mocks.usePagedData.mockReturnValue(pagedData([]));
    mocks.probeListPlans.mockResolvedValue({ plans: [gitOpsPlan] });

    await render();

    expect(text()).toContain("common.no-data");
    expect(text()).not.toContain("release.empty.title");
    expect(
      container.querySelector('[data-testid="release-gitops-setup"]')
    ).not.toBeNull();
  });

  test.each([
    { name: "the toolbar", plans: [gitOpsPlan], label: "release.gitops-setup" },
    { name: "the empty state", plans: [], label: "release.empty.set-up" },
  ])("disables GitOps setup in $name without setup permission", async ({ plans, label }) => {
    mocks.canSetUpGitOps = false;
    mocks.usePagedData.mockReturnValue(pagedData(plans));

    await render();

    const button = Array.from(container.querySelectorAll("button")).find(
      (element) => element.textContent?.includes(label)
    );
    expect(button?.disabled).toBe(true);
  });
});
