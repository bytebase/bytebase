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
  switchWorkspace: vi.fn(),
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
  useWorkspaceList: () => [
    {
      name: "workspaces/default",
      title: "Default Workspace",
    },
  ],
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
