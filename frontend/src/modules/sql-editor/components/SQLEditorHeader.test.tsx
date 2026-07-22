import type { ReactElement, ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { WORKSPACE_ROUTE_LANDING } from "@/app/router";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  breadcrumbProps: undefined as
    | {
        projectId?: string;
        currentProjectName?: string;
        projectSwitchExcludeDefaultProject?: boolean;
        onSelectProject?: (
          project: { name: string },
          event: { ctrlKey?: boolean; metaKey?: boolean }
        ) => void;
      }
    | undefined,
  maybeSwitchProject: vi.fn().mockResolvedValue(undefined),
  record: vi.fn(),
  setRecentProject: vi.fn(),
  loadWorkspacePermissionState: vi.fn().mockResolvedValue(undefined),
  hasProjectPermission: vi.fn(),
  allowAccessDefaultProject: true,
  defaultProjectName: "projects/default",
  themeDark: true,
  resolve: vi.fn(
    ({
      params,
    }: {
      name: string;
      params?: { project?: string };
    }) => ({
      fullPath: params?.project
        ? `/sql-editor/projects/${params.project}`
        : "/sql-editor",
    })
  ),
  project: "projects/recent-project",
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) =>
      ({
        "sql-editor.self": "SQL Editor",
      })[key] ?? key,
  }),
}));

vi.mock("@/app/router", () => ({
  SQL_EDITOR_PROJECT_MODULE: "sql-editor.project",
  WORKSPACE_ROUTE_LANDING: "workspace.landing",
  useNavigate: () => ({
    resolve: mocks.resolve,
  }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useRecentVisit: () => ({
    record: mocks.record,
  }),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (
    selector: (state: {
      serverInfo: { defaultProject: string };
      setRecentProject: typeof mocks.setRecentProject;
      loadWorkspacePermissionState: typeof mocks.loadWorkspacePermissionState;
      hasProjectPermission: typeof mocks.hasProjectPermission;
    }) => unknown
  ) =>
    selector({
      serverInfo: { defaultProject: mocks.defaultProjectName },
      setRecentProject: mocks.setRecentProject,
      loadWorkspacePermissionState: mocks.loadWorkspacePermissionState,
      hasProjectPermission: mocks.hasProjectPermission,
    }),
}));

vi.mock("@/types/v1/project", () => ({
  defaultProject: (name: string) => ({
    name,
    title: "Default project",
  }),
}));

vi.mock("@/modules/sql-editor/store", () => ({
  useSQLEditorStore: (
    selector: (state: {
      maybeSwitchProject: typeof mocks.maybeSwitchProject;
    }) => unknown
  ) =>
    selector({
      maybeSwitchProject: mocks.maybeSwitchProject,
    }),
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  useSQLEditorEditorState: (selector: (state: { project: string }) => unknown) =>
    selector({ project: mocks.project }),
}));

vi.mock("@/modules/sql-editor/components/theme/SQLEditorThemeScope", () => ({
  useSQLEditorTheme: () => ({
    id: mocks.themeDark ? "dark" : "light",
  }),
}));

vi.mock("@/modules/sql-editor/components/theme/derive", () => ({
  isDarkTheme: () => mocks.themeDark,
}));

vi.mock("@/components/BytebaseLogo", () => ({
  BytebaseLogo: ({
    redirect,
    builtinTheme,
  }: {
    redirect?: string;
    builtinTheme?: string;
    className?: string;
  }) => (
    <div
      data-testid="sql-editor-logo"
      data-redirect={redirect}
      data-builtin-theme={builtinTheme}
    />
  ),
}));

vi.mock("@/components/header/HeaderBreadcrumb", () => ({
  HeaderBreadcrumb: (props: NonNullable<typeof mocks.breadcrumbProps>) => {
    mocks.breadcrumbProps = props;
    return (
      <div
        data-testid="sql-editor-breadcrumb"
        data-project-id={props.projectId}
        data-current-project-name={props.currentProjectName}
      />
    );
  },
}));

vi.mock("@/components/header/ProfileMenuTrigger", () => ({
  ProfileMenuTrigger: ({
    children,
  }: {
    children?: ReactNode;
    size?: string;
    link?: boolean;
  }) => <div data-testid="sql-editor-profile">{children}</div>,
}));

let SQLEditorHeader: typeof import("./SQLEditorHeader").SQLEditorHeader;

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

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.breadcrumbProps = undefined;
  mocks.project = "projects/recent-project";
  mocks.allowAccessDefaultProject = true;
  mocks.defaultProjectName = "projects/default";
  mocks.hasProjectPermission.mockImplementation(
    () => mocks.allowAccessDefaultProject
  );
  mocks.loadWorkspacePermissionState.mockResolvedValue(undefined);
  mocks.themeDark = true;
  window.open = vi.fn();
  ({ SQLEditorHeader } = await import("./SQLEditorHeader"));
});

describe("SQLEditorHeader", () => {
  test("renders logo, breadcrumb, and profile dropdown", () => {
    const { container, render, unmount } = renderIntoContainer(
      <SQLEditorHeader />
    );
    render();

    const logo = container.querySelector("[data-testid='sql-editor-logo']");
    expect(logo?.getAttribute("data-redirect")).toBe(WORKSPACE_ROUTE_LANDING);
    expect(logo?.getAttribute("data-builtin-theme")).toBe("dark");
    expect(
      container.querySelector("[data-testid='sql-editor-breadcrumb']")
    ).not.toBeNull();
    expect(
      container.querySelector("[data-testid='sql-editor-profile']")
    ).not.toBeNull();
    expect(mocks.breadcrumbProps?.projectId).toBe("recent-project");
    expect(mocks.breadcrumbProps?.currentProjectName).toBe(
      "projects/recent-project"
    );
    expect(mocks.breadcrumbProps?.projectSwitchExcludeDefaultProject).toBe(
      false
    );
    expect(mocks.loadWorkspacePermissionState).toHaveBeenCalled();

    unmount();
  });

  test("excludes the default project when SQL Editor cannot access it", () => {
    mocks.allowAccessDefaultProject = false;
    const { render, unmount } = renderIntoContainer(<SQLEditorHeader />);
    render();

    expect(mocks.breadcrumbProps?.projectSwitchExcludeDefaultProject).toBe(true);

    unmount();
  });

  test("switches the SQL Editor project from the breadcrumb switcher", () => {
    const { render, unmount } = renderIntoContainer(<SQLEditorHeader />);
    render();

    act(() => {
      mocks.breadcrumbProps?.onSelectProject?.(
        { name: "projects/other-project" },
        { ctrlKey: false, metaKey: false }
      );
    });

    expect(mocks.resolve).toHaveBeenCalledWith({
      name: "sql-editor.project",
      params: { project: "other-project" },
    });
    expect(mocks.record).toHaveBeenCalledWith(
      "/sql-editor/projects/other-project"
    );
    expect(mocks.setRecentProject).toHaveBeenCalledWith(
      "projects/other-project"
    );
    expect(mocks.maybeSwitchProject).toHaveBeenCalledWith(
      "projects/other-project"
    );
    expect(window.open).not.toHaveBeenCalled();

    unmount();
  });

  test("records recent project when opening the SQL Editor project in a new tab", () => {
    const { render, unmount } = renderIntoContainer(<SQLEditorHeader />);
    render();

    act(() => {
      mocks.breadcrumbProps?.onSelectProject?.(
        { name: "projects/other-project" },
        { ctrlKey: true, metaKey: false }
      );
    });

    expect(mocks.record).toHaveBeenCalledWith(
      "/sql-editor/projects/other-project"
    );
    expect(mocks.setRecentProject).toHaveBeenCalledWith(
      "projects/other-project"
    );
    expect(window.open).toHaveBeenCalledWith(
      "/sql-editor/projects/other-project",
      "_blank"
    );
    expect(mocks.maybeSwitchProject).not.toHaveBeenCalled();

    unmount();
  });
});
