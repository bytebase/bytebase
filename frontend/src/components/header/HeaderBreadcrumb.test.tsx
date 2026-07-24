import type { MouseEvent, ReactElement, ReactNode } from "react";
import { cloneElement, createContext, useContext } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { PROJECT_V1_ROUTE_ISSUES } from "@/app/router/handles";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  currentRoute: {
    name: "workspace.project.database",
    fullPath: "/projects/recent-project/databases",
    params: {
      projectId: "recent-project",
    },
    query: {},
  },
  record: vi.fn(),
  resolve: vi.fn(({ name }: { name: string }) => ({
    fullPath: `/${name}`,
  })),
  projectSwitchPanelProps: undefined as
    | { excludeDefaultProject?: boolean }
    | undefined,
  onSelectWorkspace: vi.fn(),
  switchWorkspace: vi.fn(),
  workspaceList: [
    {
      name: "workspaces/default",
      title: "Default Workspace",
    },
  ],
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string) =>
      ({
        "project.select": "Select project",
        "subscription.plan.free.title": "Free",
      })[key] ?? key,
  }),
}));

vi.mock("@/app/router", () => ({
  useCurrentRoute: () => mocks.currentRoute,
  useNavigate: () => ({
    resolve: mocks.resolve,
  }),
  WORKSPACE_ROUTE_LANDING: "workspace.landing",
}));

vi.mock("@/hooks/useAppState", () => ({
  useProject: () => ({
    name: "projects/recent-project",
    title: "Recent Project",
  }),
  useRecentVisit: () => ({
    record: mocks.record,
  }),
  useSubscription: () => ({
    subscription: undefined,
  }),
  useSwitchWorkspace: () => mocks.switchWorkspace,
  useWorkspace: () => ({
    name: "workspaces/default",
    title: "Default Workspace",
  }),
  useWorkspaceList: () => mocks.workspaceList,
}));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({
    to,
    children,
    className,
    onClick,
  }: {
    to: { name?: string; params?: Record<string, string | undefined> };
    children: ReactNode;
    className?: string;
    onClick?: () => void;
  }) => (
    <a
      className={className}
      data-route-name={to.name}
      data-project-id={to.params?.projectId}
      href={`/${to.name ?? ""}`}
      onClick={(event: MouseEvent<HTMLAnchorElement>) => {
        event.preventDefault();
        onClick?.();
      }}
    >
      {children}
    </a>
  ),
}));

type PopoverContextValue = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

const PopoverContext = createContext<PopoverContextValue | undefined>(
  undefined
);

vi.mock("@/components/ui/popover", () => ({
  Popover: ({
    open,
    onOpenChange,
    children,
  }: PopoverContextValue & { children: ReactNode }) => (
    <PopoverContext.Provider value={{ open, onOpenChange }}>
      {children}
    </PopoverContext.Provider>
  ),
  PopoverContent: ({ children }: { children: ReactNode }) => {
    const context = useContext(PopoverContext);
    return context?.open ? <div>{children}</div> : null;
  },
  PopoverTrigger: ({
    render,
    children,
  }: {
    render: ReactElement<{ onClick?: () => void; children?: ReactNode }>;
    children: ReactNode;
  }) => {
    const context = useContext(PopoverContext);
    return cloneElement(render, {
      onClick: () => context?.onOpenChange(!context.open),
      children,
    });
  },
}));

vi.mock("@/components/header/ProjectSwitchPanel", () => ({
  ProjectSwitchPanel: (props: { excludeDefaultProject?: boolean }) => {
    mocks.projectSwitchPanelProps = props;
    return <div data-testid="project-switch-panel" />;
  },
}));

vi.mock("@/components/header/ProjectCreateDialog", () => ({
  ProjectCreateDialog: () => null,
}));

let HeaderBreadcrumb: typeof import("./HeaderBreadcrumb").HeaderBreadcrumb;

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
  mocks.projectSwitchPanelProps = undefined;
  mocks.workspaceList = [
    {
      name: "workspaces/default",
      title: "Default Workspace",
    },
  ];
  ({ HeaderBreadcrumb } = await import("./HeaderBreadcrumb"));
});

describe("HeaderBreadcrumb", () => {
  test("opens project switcher only from the chevron and links title to project issues", () => {
    const { container, render, unmount } = renderIntoContainer(
      <HeaderBreadcrumb />
    );

    render();

    const projectLink = container.querySelector<HTMLAnchorElement>(
      `a[data-route-name="${PROJECT_V1_ROUTE_ISSUES}"]`
    );
    expect(projectLink).not.toBeNull();
    expect(projectLink?.textContent).toContain("Recent Project");
    expect(projectLink?.dataset.projectId).toBe("recent-project");

    act(() => {
      projectLink?.click();
    });
    expect(
      container.querySelector('[data-testid="project-switch-panel"]')
    ).toBeNull();

    const projectSwitchButton = container.querySelector<HTMLButtonElement>(
      "button"
    );
    act(() => {
      projectSwitchButton?.click();
    });
    expect(
      container.querySelector('[data-testid="project-switch-panel"]')
    ).not.toBeNull();

    unmount();
  });

  test("uses custom project selection when clicking the project title", () => {
    const onSelectProject = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <HeaderBreadcrumb onSelectProject={onSelectProject} />
    );

    render();

    expect(
      container.querySelector(`a[data-route-name="${PROJECT_V1_ROUTE_ISSUES}"]`)
    ).toBeNull();
    const projectButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Recent Project")
    );
    expect(projectButton).not.toBeUndefined();

    act(() => {
      projectButton?.click();
    });

    expect(onSelectProject).toHaveBeenCalledTimes(1);
    expect(onSelectProject).toHaveBeenCalledWith(
      { name: "projects/recent-project", title: "Recent Project" },
      expect.objectContaining({ ctrlKey: false, metaKey: false })
    );
    expect(mocks.record).not.toHaveBeenCalled();

    unmount();
  });

  test("uses custom workspace selection when clicking the workspace title", () => {
    const { container, render, unmount } = renderIntoContainer(
      <HeaderBreadcrumb onSelectWorkspace={mocks.onSelectWorkspace} />
    );

    render();

    expect(
      container.querySelector('a[data-route-name="workspace.landing"]')
    ).toBeNull();
    const workspaceButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("Default Workspace"));
    expect(workspaceButton).not.toBeUndefined();

    act(() => {
      workspaceButton?.click();
    });

    expect(mocks.onSelectWorkspace).toHaveBeenCalledWith(
      "workspaces/default",
      expect.objectContaining({ ctrlKey: false, metaKey: false })
    );
    expect(mocks.record).not.toHaveBeenCalled();

    unmount();
  });

  test("uses custom workspace selection when switching workspace", () => {
    mocks.workspaceList = [
      {
        name: "workspaces/default",
        title: "Default Workspace",
      },
      {
        name: "workspaces/other",
        title: "Other Workspace",
      },
    ];
    const onBeforeSwitchWorkspace = vi.fn(() => true);
    const { container, render, unmount } = renderIntoContainer(
      <HeaderBreadcrumb
        onBeforeSwitchWorkspace={onBeforeSwitchWorkspace}
        onSelectWorkspace={mocks.onSelectWorkspace}
      />
    );

    render();

    const workspaceSwitchButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent === "");
    act(() => {
      workspaceSwitchButton?.click();
    });
    const otherWorkspaceButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("Other Workspace"));
    act(() => {
      otherWorkspaceButton?.click();
    });

    expect(onBeforeSwitchWorkspace).toHaveBeenCalledTimes(1);
    expect(mocks.onSelectWorkspace).toHaveBeenCalledWith(
      "workspaces/other",
      expect.objectContaining({ ctrlKey: false, metaKey: false })
    );
    expect(mocks.switchWorkspace).not.toHaveBeenCalled();

    unmount();
  });

  test("checks before switching workspace", () => {
    mocks.workspaceList = [
      {
        name: "workspaces/default",
        title: "Default Workspace",
      },
      {
        name: "workspaces/other",
        title: "Other Workspace",
      },
    ];
    const onBeforeSwitchWorkspace = vi.fn(() => false);
    const { container, render, unmount } = renderIntoContainer(
      <HeaderBreadcrumb onBeforeSwitchWorkspace={onBeforeSwitchWorkspace} />
    );

    render();

    const workspaceSwitchButton = container.querySelector<HTMLButtonElement>(
      "button"
    );
    act(() => {
      workspaceSwitchButton?.click();
    });
    const otherWorkspaceButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("Other Workspace"));
    act(() => {
      otherWorkspaceButton?.click();
    });

    expect(onBeforeSwitchWorkspace).toHaveBeenCalledTimes(1);
    expect(mocks.switchWorkspace).not.toHaveBeenCalled();

    unmount();
  });

  test("passes default-project visibility to the project switcher", () => {
    const { container, render, unmount } = renderIntoContainer(
      <HeaderBreadcrumb projectSwitchExcludeDefaultProject={false} />
    );

    render();

    const projectSwitchButton = container.querySelector<HTMLButtonElement>(
      "button"
    );
    act(() => {
      projectSwitchButton?.click();
    });
    expect(mocks.projectSwitchPanelProps?.excludeDefaultProject).toBe(false);

    unmount();
  });
});
