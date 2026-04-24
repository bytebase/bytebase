import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  logout: vi.fn(),
  uploadLicense: vi.fn(),
  emitStorageChangedEvent: vi.fn(),
  push: vi.fn(),
  resolve: vi.fn(({ name }: { name: string }) => ({ fullPath: `/${name}` })),
  currentRoute: {
    value: {
      name: "sql-editor.home",
    },
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) =>
      ({
        "common.language": "Language",
        "common.license": "License",
        "quick-start.self": "Quick Start",
        "common.logout": "Logout",
        "settings.general.workspace.default-landing-page.go-to-workspace":
          "Go to workspace",
        "settings.general.workspace.default-landing-page.go-to-sql-editor":
          "Go to SQL Editor",
        "subscription.plan.free.title": "Free",
        "subscription.plan.team.title": "Team",
        "subscription.plan.enterprise.title": "Enterprise",
      })[key] ?? key,
  }),
}));

vi.mock("@/plugins/i18n", () => ({
  default: {
    global: {
      locale: {
        value: "en-US",
      },
    },
  },
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: (getter: () => unknown) => getter(),
}));

vi.mock("@/react/components/UserAvatar", () => ({
  UserAvatar: () => <div data-testid="user-avatar" />,
}));

vi.mock("@/react/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: ReactElement[] }) => (
    <div>{children}</div>
  ),
  DropdownMenuTrigger: ({ children }: { children: ReactElement }) => (
    <div>{children}</div>
  ),
  DropdownMenuContent: ({ children }: { children: ReactElement[] }) => (
    <div>{children}</div>
  ),
  DropdownMenuItem: ({
    children,
    onClick,
  }: {
    children: ReactElement | string;
    onClick?: () => void;
  }) => <button onClick={onClick}>{children}</button>,
  DropdownMenuSeparator: () => <div />,
  DropdownMenuSubmenu: ({ children }: { children: ReactElement[] }) => (
    <div>{children}</div>
  ),
  DropdownMenuSubmenuTrigger: ({
    children,
  }: {
    children: ReactElement | ReactElement[] | string;
  }) => <div>{children}</div>,
  DropdownMenuSubmenuContent: ({ children }: { children: ReactElement[] }) => (
    <div>{children}</div>
  ),
}));

vi.mock("@/react/components/header/VersionMenuItem", () => ({
  VersionMenuItem: () => <div data-testid="version-item" />,
}));

vi.mock("./common", () => ({
  HEADER_LANGUAGE_OPTIONS: [{ label: "English", value: "en-US" }],
  resetQuickstartProgress: vi.fn(),
  setAppLocale: mocks.emitStorageChangedEvent,
}));

vi.mock("@/router", () => ({
  router: {
    currentRoute: mocks.currentRoute,
    push: mocks.push,
    resolve: mocks.resolve,
  },
}));

vi.mock("@/router/dashboard/workspaceRoutes", () => ({
  WORKSPACE_ROUTE_LANDING: "workspace.landing",
  WORKSPACE_ROUTE_MY_ISSUES: "workspace.my-issues",
}));

vi.mock("@/router/dashboard/workspaceSetting", () => ({
  SETTING_ROUTE_PROFILE: "setting.profile",
}));

vi.mock("@/router/sqlEditor", () => ({
  SQL_EDITOR_HOME_MODULE: "sql-editor.home",
}));

vi.mock("@/store", () => ({
  useActuatorV1Store: () => ({
    quickStartEnabled: true,
  }),
  useAuthStore: () => ({
    logout: mocks.logout,
  }),
  useCurrentUserV1: () => ({
    value: {
      title: "Alice",
      email: "alice@example.com",
    },
  }),
  useSubscriptionV1Store: () => ({
    currentPlan: 0,
    uploadLicense: mocks.uploadLicense,
  }),
  useUIStateStore: () => ({
    saveIntroStateByKey: vi.fn(),
  }),
  useWorkspaceV1Store: () => ({
    currentWorkspace: {
      logo: "",
    },
  }),
}));

vi.mock("@/utils", () => ({
  isDev: () => false,
  isSQLEditorRoute: () => true,
}));

let ProfileMenuTrigger: typeof import("./ProfileMenuTrigger").ProfileMenuTrigger;

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
  window.open = vi.fn();
  ({ ProfileMenuTrigger } = await import("./ProfileMenuTrigger"));
});

describe("ProfileMenuTrigger", () => {
  test("supports locale changes, workspace toggle, and logout", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ProfileMenuTrigger size="medium" link />
    );

    render();

    const englishButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("English")
    );
    act(() => {
      englishButton?.click();
    });
    expect(mocks.emitStorageChangedEvent).toHaveBeenCalled();

    const workspaceButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("Go to workspace"));
    act(() => {
      workspaceButton?.click();
    });
    expect(window.open).toHaveBeenCalledWith(
      `/${WORKSPACE_ROUTE_LANDING}`,
      "_blank",
      "noopener,noreferrer"
    );

    const logoutButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Logout")
    );
    act(() => {
      logoutButton?.click();
    });
    expect(mocks.logout).toHaveBeenCalled();

    unmount();
  });
});
