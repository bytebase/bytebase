import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  SQL_EDITOR_DATABASE_MODULE,
  WORKSPACE_ROUTE_MY_ISSUES,
} from "@/react/router";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  record: vi.fn(),
  toggleAgent: vi.fn(),
  resolve: vi.fn(({ name }: { name: string }) => ({
    href: `/${name}`,
    fullPath: `/${name}`,
  })),
  push: vi.fn(),
  currentPlan: 0,
  currentRoute: {
    name: "workspace.project.database.detail",
    fullPath: "/projects/sample-project",
    params: {
      projectId: "sample-project",
      instanceId: "prod",
      databaseName: "db",
    },
    query: {},
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) =>
      ({
        "agent.self": "Agent",
        "common.want-help": "Want help",
        "sql-editor.self": "SQL Editor",
        "issue.my-issues": "My Issues",
      })[key] ?? key,
  }),
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useRecentVisit: () => ({
    record: mocks.record,
  }),
  useSubscription: () => ({
    subscription: {
      plan: mocks.currentPlan,
    },
  }),
}));

vi.mock("@/react/components/BytebaseLogo", () => ({
  BytebaseLogo: ({ redirect }: { redirect?: string }) => (
    <div data-testid="bytebase-logo" data-redirect={redirect} />
  ),
}));

vi.mock("@/react/components/header/HeaderBreadcrumb", () => ({
  HeaderBreadcrumb: () => <div data-testid="header-breadcrumb" />,
}));

vi.mock("@/react/components/header/ProfileMenuTrigger", () => ({
  ProfileMenuTrigger: () => <div data-testid="profile-menu-trigger" />,
}));

vi.mock("@/react/router", () => ({
  useCurrentRoute: () => mocks.currentRoute,
  useNavigate: () => ({
    resolve: mocks.resolve,
    push: mocks.push,
  }),
  WORKSPACE_ROUTE_LANDING: "workspace.landing",
  WORKSPACE_ROUTE_MY_ISSUES: "workspace.my-issues",
  SQL_EDITOR_HOME_MODULE: "sql-editor.home",
  SQL_EDITOR_PROJECT_MODULE: "sql-editor.project",
  SQL_EDITOR_DATABASE_MODULE: "sql-editor.database",
}));

vi.mock("@/utils/storage-keys", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
  };
});

vi.mock("@/react/plugins/agent/store/agent", () => ({
  useAgentStore: {
    getState: () => ({
      toggle: mocks.toggleAgent,
    }),
  },
}));

let DashboardHeader: typeof import("./DashboardHeader").DashboardHeader;

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
  Object.defineProperty(window, "innerWidth", {
    configurable: true,
    writable: true,
    value: 1024,
  });
  window.open = vi.fn();
  mocks.currentPlan = PlanType.FREE;
  mocks.currentRoute = {
    name: "workspace.project.database.detail",
    fullPath: "/projects/sample-project",
    params: {
      projectId: "sample-project",
      instanceId: "prod",
      databaseName: "db",
    },
    query: {},
  };
  ({ DashboardHeader } = await import("./DashboardHeader"));
});

describe("DashboardHeader", () => {
  test("renders logo, toggle, and shared entrypoints", () => {
    const onOpenMobileSidebar = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <DashboardHeader
        showLogo
        showMobileSidebarToggle
        onOpenMobileSidebar={onOpenMobileSidebar}
      />
    );

    render();

    expect(
      container.querySelector('[data-testid="bytebase-logo"]')
    ).not.toBeNull();
    expect(
      container.querySelector('[data-testid="header-breadcrumb"]')
    ).not.toBeNull();
    expect(
      container.querySelector('[data-testid="profile-menu-trigger"]')
    ).not.toBeNull();

    const toggleButton = container.querySelector(
      ".md\\:hidden"
    ) as HTMLButtonElement | null;
    expect(toggleButton).not.toBeNull();
    act(() => {
      toggleButton?.click();
    });
    expect(onOpenMobileSidebar).toHaveBeenCalledTimes(1);

    unmount();
  });

  test("routes sql editor and my issues actions correctly", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DashboardHeader showLogo={false} />
    );

    render();

    const sqlEditorButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("SQL Editor"));
    expect(sqlEditorButton).not.toBeUndefined();
    act(() => {
      sqlEditorButton?.click();
    });
    expect(window.open).toHaveBeenCalledWith(
      `/${SQL_EDITOR_DATABASE_MODULE}`,
      "_blank",
      "noopener,noreferrer"
    );

    const myIssuesButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("My Issues"));
    expect(myIssuesButton).not.toBeUndefined();
    act(() => {
      myIssuesButton?.click();
    });
    expect(mocks.record).toHaveBeenCalledWith(`/${WORKSPACE_ROUTE_MY_ISSUES}`);
    expect(mocks.push).toHaveBeenCalledWith({
      name: WORKSPACE_ROUTE_MY_ISSUES,
    });
    expect(localStorage.getItem("bb.components.MY_ISSUES.id")).not.toBeNull();

    unmount();
  });
});
