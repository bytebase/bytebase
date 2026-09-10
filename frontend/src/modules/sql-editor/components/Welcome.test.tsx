import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  routerPush: vi.fn(),
  navigatePush: vi.fn(),
  onChangeProject: vi.fn(),
  fetchDatabases: vi.fn(),
  projectName: "" as string,
  projectData: { name: "projects/test" } as { name: string },
  projects: [] as { name: string }[],
  hasDefaultProject: false,
  themeDark: false,
  deniedPermissions: new Set<string>(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactElement }) => children,
  usePermissionCheck: (permissions: readonly string[]) => [
    !permissions.some((permission) => mocks.deniedPermissions.has(permission)),
    "missing permission",
  ],
}));

vi.mock("@/components/BytebaseLogo", () => ({
  BytebaseLogo: ({
    builtinTheme,
    className,
  }: {
    builtinTheme?: string;
    className?: string;
  }) => (
    <div
      data-testid="welcome-logo"
      data-builtin-theme={builtinTheme}
      className={className}
    />
  ),
}));

vi.mock("@/hooks/useAppState", () => ({
  useProjectList: (
    _query: string,
    { excludeDefault = true }: { excludeDefault?: boolean } = {}
  ) => ({
    projects: [
      ...(mocks.hasDefaultProject && !excludeDefault
        ? [{ name: "projects/default" }]
        : []),
      ...mocks.projects,
    ],
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useAppProject", () => ({
  useAppProject: () => (mocks.projectName ? mocks.projectData : undefined),
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: { push: mocks.routerPush },
  SQL_EDITOR_PROJECT_MODULE: "sql-editor.project",
  useNavigate: () => ({ push: mocks.navigatePush }),
}));

vi.mock("@/app/router/handles", () => ({
  PROJECT_V1_ROUTE_DASHBOARD: "workspace.project",
  PROJECT_V1_ROUTE_INSTANCE_CREATE: "workspace.project.instance.create",
}));

vi.mock("@/utils/v1/project", () => ({
  extractProjectResourceName: (name: string) => name.replace("projects/", ""),
}));

vi.mock("@/lib/productIntro", () => ({
  CREATE_PROJECT_PRODUCT_INTRO: "create-project",
  PRODUCT_INTRO_QUERY_KEY: "product-intro",
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  useSQLEditorEditorState: (
    selector: (state: { project: string }) => unknown
  ) => selector({ project: mocks.projectName }),
}));

vi.mock("@/modules/sql-editor/store", () => ({
  useSQLEditorStore: (
    selector: (state: { maybeSwitchProject: typeof mocks.onChangeProject }) => unknown
  ) => selector({ maybeSwitchProject: mocks.onChangeProject }),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({ fetchDatabases: mocks.fetchDatabases }),
  },
}));

vi.mock("@/components/header/ProjectSwitchPanel", () => ({
  ProjectSwitchPanel: ({
    onSelectProject,
  }: {
    onSelectProject: (project: { name: string }) => void;
  }) => (
    <button
      data-testid="choose-project"
      onClick={() => onSelectProject({ name: "projects/selected" })}
    >
      Choose project
    </button>
  ),
}));

vi.mock("@/components/ui/popover", () => ({
  Popover: ({ children }: { children: ReactElement }) => <>{children}</>,
  PopoverContent: ({ children }: { children: ReactElement }) => <>{children}</>,
  PopoverTrigger: ({ render }: { render: ReactElement }) => render,
}));

vi.mock("@/assets/logo-full.svg", () => ({
  default: "/assets/logo-full.svg",
}));

vi.mock("@/modules/sql-editor/components/theme/SQLEditorThemeScope", () => ({
  useSQLEditorTheme: () => ({
    id: mocks.themeDark ? "dark" : "light",
  }),
}));

vi.mock("@/modules/sql-editor/components/theme/derive", () => ({
  isDarkTheme: () => mocks.themeDark,
}));

let Welcome: typeof import("./Welcome").Welcome;

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

const flushEffects = async () => {
  await act(async () => {
    await Promise.resolve();
  });
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.projectName = "";
  mocks.projects = [];
  mocks.hasDefaultProject = false;
  mocks.themeDark = false;
  mocks.deniedPermissions.clear();
  mocks.fetchDatabases.mockResolvedValue({ databases: [] });
  ({ Welcome } = await import("./Welcome"));
});

describe("Welcome", () => {
  test("switches to a selected accessible project", async () => {
    mocks.projects = [{ name: "projects/selected" }];
    mocks.onChangeProject.mockResolvedValue("projects/selected");
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();

    expect(
      container.querySelector('[data-testid="select-project"]')
    ).not.toBeNull();
    expect(
      container.querySelector('[data-testid="create-project"]')
    ).toBeNull();

    act(() => {
      (
        container.querySelector(
          '[data-testid="choose-project"]'
        ) as HTMLButtonElement
      ).click();
    });
    expect(mocks.onChangeProject).toHaveBeenCalledWith("projects/selected");
    await flushEffects();
    expect(mocks.navigatePush).toHaveBeenCalledWith({
      name: "sql-editor.project",
      params: { project: "selected" },
    });
    unmount();
  });

  test("offers project creation when no accessible project exists", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();

    act(() => {
      (
        container.querySelector(
          '[data-testid="create-project"]'
        ) as HTMLButtonElement
      ).click();
    });
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.project",
      query: { "product-intro": "create-project" },
    });
    unmount();
  });

  test("treats the default project as no project", () => {
    mocks.hasDefaultProject = true;
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();

    expect(
      container.querySelector('[data-testid="select-project"]')
    ).toBeNull();
    expect(
      container.querySelector('[data-testid="create-project"]')
    ).not.toBeNull();
    unmount();
  });

  test("keeps project creation visible but disabled without permission", () => {
    mocks.deniedPermissions.add("bb.projects.create");
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();

    expect(
      container.querySelector('[data-testid="create-project"]')
    ).toBeDisabled();
    unmount();
  });

  test("creates an instance in the selected project when it has no databases", async () => {
    mocks.projectName = "projects/test";
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    await flushEffects();

    const addInstance = container.querySelector(
      '[data-testid="create-project-instance"]'
    ) as HTMLButtonElement;
    expect(addInstance).not.toBeNull();
    expect(container.querySelector('[data-testid="connect-database"]')).toBeNull();
    act(() => addInstance.click());
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.project.instance.create",
      params: { projectId: "test" },
    });
    unmount();
  });

  test("keeps instance creation visible but disabled without project permission", async () => {
    mocks.projectName = "projects/test";
    mocks.deniedPermissions.add("bb.instances.create");
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    await flushEffects();

    expect(
      container.querySelector('[data-testid="create-project-instance"]')
    ).toBeDisabled();
    unmount();
  });

  test("keeps the database action visible but disabled without query permission", async () => {
    mocks.projectName = "projects/test";
    mocks.fetchDatabases.mockResolvedValue({
      databases: [{ name: "databases/db" }],
    });
    mocks.deniedPermissions.add("bb.sql.select");
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    await flushEffects();

    expect(
      container.querySelector('[data-testid="connect-database"]')
    ).toBeDisabled();
    unmount();
  });

  test("opens the connection panel when a database is available", async () => {
    mocks.projectName = "projects/test";
    mocks.fetchDatabases.mockResolvedValue({
      databases: [{ name: "databases/db" }],
    });
    const onChangeConnection = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={onChangeConnection} />
    );
    render();
    await flushEffects();

    act(() => {
      (
        container.querySelector(
          '[data-testid="connect-database"]'
        ) as HTMLButtonElement
      ).click();
    });
    expect(onChangeConnection).toHaveBeenCalledTimes(1);
    unmount();
  });

  test("passes dark SQL Editor theme to the logo", () => {
    mocks.themeDark = true;
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    expect(
      container.querySelector('[data-testid="welcome-logo"]')?.getAttribute(
        "data-builtin-theme"
      )
    ).toBe("dark");
    unmount();
  });
});
