import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import dayjs from "dayjs";
import { createElement, type ReactNode } from "react";
import { MemoryRouter, useLocation } from "react-router";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { useAppStore } from "@/stores/app";
import { ProjectIssueDashboardPage } from "./ProjectIssueDashboardPage";

const mocks = vi.hoisted(() => ({
  usePagedData: vi.fn(),
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/app/router/NavigationScrollRestoration", () => ({
  markListScrollRestorationEntry: vi.fn(),
  useListScrollRestorationKey: () => undefined,
  useListScrollRestorationLoadMore: vi.fn(),
}));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => ({ email: "me@example.com" }),
}));

vi.mock("@/hooks/useMediaQuery", () => ({
  useMediaQuery: () => false,
}));

vi.mock("@/hooks/usePagedData", () => ({
  PagedTableFooter: () => null,
  usePagedData: mocks.usePagedData,
}));

vi.mock("@/components/IssueTable", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/components/IssueTable")>()),
  useIssueSearchScopeOptions: () => [],
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

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}
globalThis.ResizeObserver =
  ResizeObserverMock as unknown as typeof ResizeObserver;

const listIssues = vi.fn(async () => ({ issues: [], nextPageToken: "" }));

// The `q` value as stored in the URL, where scope values are percent-encoded.
let q = "";
function LocationProbe() {
  const { search } = useLocation();
  q = new URLSearchParams(search).get("q") ?? "";
  return null;
}

const renderPage = (url = "/projects/foo/issues") =>
  render(
    <MemoryRouter initialEntries={[url]}>
      <ProjectIssueDashboardPage projectId="foo" />
      <LocationProbe />
    </MemoryRouter>
  );

const lastIssueFind = async () => {
  const { fetchList } = mocks.usePagedData.mock.lastCall?.[0] as {
    fetchList: (params: { pageSize: number; pageToken: string }) => unknown;
  };
  listIssues.mockClear();
  await fetchList({ pageSize: 50, pageToken: "" });
  return (listIssues.mock.lastCall as unknown as [{ find: unknown }])[0].find;
};

const timeRangeTrigger = () =>
  screen.getByRole("button", { name: /^common\.created/ });

const pickDay = (day: number) => {
  const grids = screen.getAllByRole("grid");
  fireEvent.click(
    within(grids[grids.length - 1]).getByRole("button", {
      name: new RegExp(`\\b${day}(st|nd|rd|th)\\b`),
    })
  );
};

const createdScopeChip = () =>
  document.querySelector('[data-search-scope-id="created"]');

beforeEach(() => {
  q = "";
  listIssues.mockClear();
  mocks.usePagedData.mockReset();
  mocks.usePagedData.mockReturnValue({
    dataList: [],
    hasMore: false,
    isFetchingMore: false,
    isLoading: false,
    loadMore: vi.fn(),
    onPageSizeChange: vi.fn(),
    pageSize: 50,
    pageSizeOptions: [50],
    refresh: vi.fn(),
  });
  useAppStore.setState({
    listIssues,
    batchGetOrFetchUsers: vi.fn(async () => []),
  } as unknown as Partial<ReturnType<typeof useAppStore.getState>>);
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("ProjectIssueDashboardPage time range", () => {
  test("applies a picked range to the URL and the issue filter", async () => {
    renderPage();

    fireEvent.click(timeRangeTrigger());
    pickDay(15);
    pickDay(20);
    fireEvent.click(screen.getByRole("button", { name: "common.apply" }));

    const from = dayjs().date(15).startOf("day").valueOf();
    const to = dayjs().date(20).endOf("day").valueOf();
    await waitFor(() => expect(q).toContain(`created:${from}%2C${to}`));
    expect(q).not.toContain("project:");
    expect(createdScopeChip()).toBeNull();
    expect(await lastIssueFind()).toMatchObject({
      project: "projects/foo",
      createdTsAfter: from,
      createdTsBefore: to,
    });
  });

  test("restores the range from the URL and keeps it across preset tabs", async () => {
    renderPage("/projects/foo/issues?q=status%3AOPEN%20created%3A1000%2C2000");

    expect(createdScopeChip()).toBeNull();
    expect(await lastIssueFind()).toMatchObject({
      createdTsAfter: 1000,
      createdTsBefore: 2000,
    });

    fireEvent.click(screen.getByRole("tab", { name: "issue.table.closed" }));
    await waitFor(() => expect(q).toContain("status:DONE"));
    expect(q).toContain("created:1000%2C2000");
  });

  test("clears the range", async () => {
    renderPage("/projects/foo/issues?q=status%3AOPEN%20created%3A1000%2C2000");

    fireEvent.click(timeRangeTrigger());
    await act(async () => {
      fireEvent.click(
        screen.getByRole("button", { name: "time-range.all-time" })
      );
    });

    await waitFor(() => expect(q).not.toContain("created:"));
    expect(q).toContain("status:OPEN");
    expect(await lastIssueFind()).toMatchObject({
      createdTsAfter: undefined,
      createdTsBefore: undefined,
    });
  });
});
