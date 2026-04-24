import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const recentProject = {
  name: "projects/recent-project",
  title: "Recent Project",
  labels: {},
};

const allProject = {
  name: "projects/all-project",
  title: "All Project",
  labels: {},
};

const mocks = vi.hoisted(() => ({
  record: vi.fn(),
  push: vi.fn(),
  resolve: vi.fn(({ params }: { params: { projectId: string } }) => ({
    fullPath: `/projects/${params.projectId}`,
  })),
  currentRoute: {
    value: {
      params: {
        projectId: "recent-project",
      },
    },
  },
  requestCreate: vi.fn(),
  close: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) =>
      ({
        "common.recent": "Recent",
        "common.all": "All",
        "common.filter-by-name": "Filter by name",
        "common.back-to-workspace": "Back to workspace",
        "common.create": "Create",
        "common.no-data": "No data",
        "common.loading": "Loading",
        "common.rows-per-page": "Rows per page",
        "common.load-more": "Load more",
      })[key] ?? key,
  }),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: (getter: () => unknown) => getter(),
}));

vi.mock("@/components/Project/useRecentProjects", () => ({
  useRecentProjects: () => ({
    recentViewProjects: { value: [recentProject] },
  }),
}));

vi.mock("@/react/hooks/usePagedData", () => ({
  usePagedData: () => ({
    dataList: [allProject],
    isLoading: false,
    isFetchingMore: false,
    hasMore: false,
    loadMore: vi.fn(),
    pageSize: 50,
    pageSizeOptions: [50],
    onPageSizeChange: vi.fn(),
  }),
  PagedTableFooter: () => <div data-testid="paged-footer" />,
}));

vi.mock("@/router", () => ({
  router: {
    currentRoute: mocks.currentRoute,
    resolve: mocks.resolve,
    push: mocks.push,
  },
}));

vi.mock("@/router/useRecentVisit", () => ({
  useRecentVisit: () => ({
    record: mocks.record,
  }),
}));

vi.mock("@/store", () => ({
  useProjectV1Store: () => ({
    getProjectByName: (name: string) =>
      name === "projects/recent-project" ? recentProject : allProject,
  }),
}));

vi.mock("@/types", () => ({
  isDefaultProject: () => false,
  isValidProjectName: () => true,
}));

vi.mock("@/utils", () => ({
  filterProjectV1ListByKeyword: (
    list: (typeof recentProject)[],
    keyword: string
  ) =>
    list.filter((project) =>
      project.title.toLowerCase().includes(keyword.toLowerCase())
    ),
  hasWorkspacePermissionV2: () => true,
  isDev: () => false,
}));

vi.mock("@/react/components/ui/tabs", () => ({
  Tabs: ({ children }: { children: ReactElement[] }) => <div>{children}</div>,
  TabsList: ({ children }: { children: ReactElement[] }) => (
    <div>{children}</div>
  ),
  TabsTrigger: ({
    children,
    value,
    onClick,
  }: {
    children: ReactElement | string;
    value: string;
    onClick?: () => void;
  }) => (
    <button data-testid={`tab-${value}`} onClick={onClick}>
      {children}
    </button>
  ),
  TabsPanel: ({ children }: { children: ReactElement | ReactElement[] }) => (
    <div>{children}</div>
  ),
}));

let ProjectSwitchPanel: typeof import("./ProjectSwitchPanel").ProjectSwitchPanel;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  ({ ProjectSwitchPanel } = await import("./ProjectSwitchPanel"));
});

describe("ProjectSwitchPanel", () => {
  test("records and navigates when selecting a project", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSwitchPanel
        onClose={mocks.close}
        onRequestCreate={mocks.requestCreate}
      />
    );

    render();

    const row = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Recent Project")
    );
    expect(row).not.toBeUndefined();
    act(() => {
      row?.click();
    });

    expect(mocks.resolve).toHaveBeenCalledWith({
      name: PROJECT_V1_ROUTE_DETAIL,
      params: { projectId: "recent-project" },
    });
    expect(mocks.record).toHaveBeenCalledWith("/projects/recent-project");
    expect(mocks.push).toHaveBeenCalled();
    expect(mocks.close).toHaveBeenCalled();
    unmount();
  });

  test("opens the create flow from the header action", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSwitchPanel
        onClose={mocks.close}
        onRequestCreate={mocks.requestCreate}
      />
    );

    render();

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === ""
    );
    expect(createButton).not.toBeUndefined();
    act(() => {
      createButton?.click();
    });
    expect(mocks.requestCreate).toHaveBeenCalled();
    unmount();
  });
});
