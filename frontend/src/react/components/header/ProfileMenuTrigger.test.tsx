import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { WORKSPACE_ROUTE_LANDING } from "@/react/router";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";

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
    name: "sql-editor.home",
    fullPath: "/sql-editor",
    params: {},
    query: {},
  },
  resetQuickstart: vi.fn(),
  hideQuickStart: false,
  isDev: false,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    i18n: {
      language: "en-US",
    },
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
  setAppLocale: mocks.emitStorageChangedEvent,
}));

vi.mock("@/react/router", () => ({
  useCurrentRoute: () => mocks.currentRoute,
  useNavigate: () => ({
    push: mocks.push,
    resolve: mocks.resolve,
  }),
  isSqlEditorRouteName: (name?: string) => name?.startsWith("sql-editor"),
  AUTH_SIGNIN_MODULE: "auth.signin",
  WORKSPACE_ROUTE_LANDING: "workspace.landing",
  SETTING_ROUTE_PROFILE: "setting.profile",
  SQL_EDITOR_HOME_MODULE: "sql-editor.home",
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => ({
    title: "Alice",
    email: "alice@example.com",
  }),
  useServerInfo: () => ({
    enableSample: true,
    activatedUserCount: 1,
  }),
  useSubscription: () => ({
    subscription: { plan: PlanType.FREE },
    uploadLicense: mocks.uploadLicense,
  }),
  useWorkspace: () => ({
    logo: "",
  }),
  useAppFeature: () => mocks.hideQuickStart,
  useQuickstartReset: () => mocks.resetQuickstart,
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      logout: mocks.logout,
    }),
  },
}));

vi.mock("@/utils/util", () => ({
  isDev: () => mocks.isDev,
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
  mocks.hideQuickStart = false;
  mocks.isDev = false;
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

  test("hides quick start when the app feature disables it", () => {
    mocks.hideQuickStart = true;
    const { container, render, unmount } = renderIntoContainer(
      <ProfileMenuTrigger size="medium" link />
    );

    render();

    expect(container.textContent).not.toContain("Quick Start");
    unmount();
  });

  test("uploads a development license from the dev license menu", () => {
    mocks.isDev = true;
    const { container, render, unmount } = renderIntoContainer(
      <ProfileMenuTrigger size="medium" link />
    );

    render();

    const teamButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Team")
    );
    expect(teamButton).not.toBeUndefined();
    act(() => {
      teamButton?.click();
    });

    expect(mocks.uploadLicense).toHaveBeenCalledTimes(1);
    unmount();
  });
});
