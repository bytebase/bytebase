import { act, createElement, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { ProjectReleaseDashboardPage } from "./ProjectReleaseDashboardPage";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  listReleaseCategories: vi.fn(async () => ({ categories: [] })),
  listReleasesByProject: vi.fn(async () => ({
    nextPageToken: "",
    releases: [],
  })),
  usePagedData: vi.fn(),
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/api", () => ({
  releaseServiceClientConnect: {
    listReleaseCategories: mocks.listReleaseCategories,
  },
}));

vi.mock("@/app/router", () => ({
  router: { push: vi.fn() },
}));

vi.mock("@/app/router/NavigationScrollRestoration", () => ({
  markListScrollRestorationEntry: vi.fn(),
  useListScrollRestorationLoadMore: () => undefined,
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

vi.mock("@/hooks/usePagedData", () => ({
  PagedTableFooter: () => null,
  usePagedData: mocks.usePagedData,
}));

vi.mock("@/hooks/useURLSearchParam", () => ({
  useURLSearchParam: () => ["", vi.fn()],
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (
    selector: (state: { listReleasesByProject: typeof mocks.listReleasesByProject }) => unknown
  ) => selector({ listReleasesByProject: mocks.listReleasesByProject }),
}));

const pagedData = {
  dataList: [],
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
};

describe("ProjectReleaseDashboardPage", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    mocks.usePagedData.mockReturnValue(pagedData);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.clearAllMocks();
  });

  test("hides the category filter when the project has no release categories", async () => {
    await act(async () => {
      root.render(<ProjectReleaseDashboardPage projectId="foo" />);
      await Promise.resolve();
    });

    expect(container.querySelector('[role="combobox"]')).toBeNull();
  });
});
