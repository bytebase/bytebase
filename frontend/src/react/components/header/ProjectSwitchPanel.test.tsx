import type { ChangeEvent, InputHTMLAttributes, ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { PROJECT_V1_ROUTE_DETAIL } from "@/react/router";

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

const otherRecentProject = {
  name: "projects/other-recent",
  title: "Other Project",
  labels: {},
};

const mocks = vi.hoisted(() => ({
  record: vi.fn(),
  push: vi.fn(),
  resolve: vi.fn(
    ({ name, params }: { name: string; params?: { projectId: string } }) => ({
      fullPath: params?.projectId
        ? `/projects/${params.projectId}`
        : `/${name}`,
    })
  ),
  currentRoute: {
    name: "workspace.project.detail",
    fullPath: "/projects/recent-project",
    params: {
      projectId: "recent-project",
    },
    query: {},
  },
  requestCreate: vi.fn(),
  close: vi.fn(),
  recentProjects: [] as Array<typeof recentProject>,
  allProjects: [] as Array<typeof allProject>,
  hasMore: false,
  loadMore: vi.fn(),
  onPageSizeChange: vi.fn(),
  inputOnChange: undefined as
    | ((event: ChangeEvent<HTMLInputElement>) => void)
    | undefined,
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
        "quick-action.new-project": "New Project",
      })[key] ?? key,
  }),
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useRecentVisit: () => ({
    record: mocks.record,
  }),
  useProject: () => recentProject,
  useRecentProjects: () => ({
    projects: mocks.recentProjects,
  }),
  useProjectList: () => ({
    projects: mocks.allProjects,
    isLoading: false,
    isFetchingMore: false,
    hasMore: mocks.hasMore,
    loadMore: mocks.loadMore,
    pageSize: 50,
    pageSizeOptions: [50],
    onPageSizeChange: mocks.onPageSizeChange,
  }),
  useWorkspacePermission: () => true,
  projectMatchesKeyword: (project: typeof recentProject, keyword: string) =>
    project.title.toLowerCase().includes(keyword.toLowerCase()) ||
    project.name.toLowerCase().includes(keyword.toLowerCase()),
}));

vi.mock("@/react/router", () => ({
  useCurrentRoute: () => mocks.currentRoute,
  useNavigate: () => ({
    resolve: mocks.resolve,
    push: mocks.push,
  }),
  PROJECT_V1_ROUTE_DETAIL: "workspace.project.detail",
  WORKSPACE_ROUTE_LANDING: "workspace.landing",
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: InputHTMLAttributes<HTMLInputElement>) => {
    mocks.inputOnChange = props.onChange;
    return <input {...props} />;
  },
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
  mocks.recentProjects = [recentProject];
  mocks.allProjects = [allProject];
  mocks.hasMore = false;
  mocks.inputOnChange = undefined;
  window.open = vi.fn();
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

    const createButton = container.querySelector<HTMLButtonElement>(
      'button[aria-label="New Project"]'
    );
    expect(createButton).not.toBeUndefined();
    act(() => {
      createButton?.click();
    });
    expect(mocks.requestCreate).toHaveBeenCalled();
    unmount();
  });

  test("loads more projects from the all tab", () => {
    mocks.hasMore = true;
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSwitchPanel
        onClose={mocks.close}
        onRequestCreate={mocks.requestCreate}
      />
    );

    render();

    const loadMoreButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("Load more"));
    expect(loadMoreButton).not.toBeUndefined();
    act(() => {
      loadMoreButton?.click();
    });

    expect(mocks.loadMore).toHaveBeenCalledTimes(1);
    unmount();
  });

  test("filters recent projects by keyword", () => {
    mocks.recentProjects = [recentProject, otherRecentProject];
    mocks.allProjects = [];
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSwitchPanel
        onClose={mocks.close}
        onRequestCreate={mocks.requestCreate}
      />
    );

    render();

    act(() => {
      mocks.inputOnChange?.({
        target: { value: "recent project" },
      } as ChangeEvent<HTMLInputElement>);
    });

    expect(container.textContent).toContain("Recent Project");
    expect(container.textContent).not.toContain("Other Project");
    unmount();
  });

  test("closes after selecting the current project", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSwitchPanel
        onClose={mocks.close}
        onRequestCreate={mocks.requestCreate}
      />
    );

    render();

    const currentProjectRow = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("Recent Project"));
    act(() => {
      currentProjectRow?.click();
    });

    expect(mocks.close).toHaveBeenCalledTimes(1);
    unmount();
  });

  test("opens project selection in a new window for ctrl-click", () => {
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
    act(() => {
      row?.dispatchEvent(
        new MouseEvent("click", {
          bubbles: true,
          cancelable: true,
          ctrlKey: true,
        })
      );
    });

    expect(window.open).toHaveBeenCalledWith(
      "/projects/recent-project",
      "_blank"
    );
    expect(mocks.push).not.toHaveBeenCalled();
    expect(mocks.close).toHaveBeenCalled();
    unmount();
  });
});
