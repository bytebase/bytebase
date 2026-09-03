import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { WORKSPACE_ROUTE_LANDING } from "@/app/router";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  logout: vi.fn(),
  uploadLicense: vi.fn(),
  emitStorageChangedEvent: vi.fn(),
  push: vi.fn(),
  resolve: vi.fn(({ name }: { name: string }) => ({
    fullPath: `/${name}`,
    href: `/${name}`,
  })),
  currentRoute: {
    name: "sql-editor.home",
    fullPath: "/sql-editor",
    params: {},
    query: {},
  },
  resumeQuickstart: vi.fn(),
  captureMetric: vi.fn(),
  scenarioId: "query-data" as string | undefined,
  workspaceUsage: undefined as string | undefined,
  introState: {} as Record<string, boolean>,
  hideQuickStart: false,
  canReadSetupResources: true,
  userCountInIam: 1,
  isDev: false,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: {
    init: vi.fn(),
    type: "3rdParty",
  },
  useTranslation: () => ({
    i18n: {
      language: "en-US",
    },
    t: (key: string) =>
      ({
        "common.language": "Language",
        "common.license": "License",
        "workspace-setup-guide.getting-started": "Getting started",
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

vi.mock("@/components/UserAvatar", () => ({
  UserAvatar: () => <div data-testid="user-avatar" />,
}));

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: ReactElement[] }) => (
    <div>{children}</div>
  ),
  DropdownMenuTrigger: ({ children }: { children: ReactElement }) => (
    <div>{children}</div>
  ),
  DropdownMenuContent: ({
    children,
    className,
  }: {
    children: ReactElement[];
    className?: string;
  }) => (
    <div data-testid="profile-menu-content" className={className}>
      {children}
    </div>
  ),
  DropdownMenuItem: ({
    children,
    onClick,
    render,
  }: {
    children?: ReactElement | string;
    onClick?: () => void;
    render?: ReactElement;
  }) => render ?? <button onClick={onClick}>{children}</button>,
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

vi.mock("@/components/header/VersionMenuItem", () => ({
  VersionMenuItem: () => <div data-testid="version-item" />,
}));

vi.mock("./common", () => ({
  HEADER_LANGUAGE_OPTIONS: [{ label: "English", value: "en-US" }],
  setAppLocale: mocks.emitStorageChangedEvent,
}));

vi.mock("@/app/router", () => ({
  router: {
    push: mocks.push,
    resolve: mocks.resolve,
  },
  useCurrentRoute: () => mocks.currentRoute,
  useNavigate: () => ({
    push: mocks.push,
    resolve: mocks.resolve,
  }),
  isSqlEditorRouteName: (name?: string) => name?.startsWith("sql-editor"),
  AUTH_SIGNIN_MODULE: "auth.signin",
  WORKSPACE_ROUTE_LANDING: "workspace.landing",
  ACCOUNT_ROUTE: "account",
  SQL_EDITOR_DATABASE_MODULE: "sql-editor.database",
  SQL_EDITOR_HOME_MODULE: "sql-editor.home",
  SQL_EDITOR_PROJECT_MODULE: "sql-editor.project",
}));

vi.mock("@/hooks/useAppState", () => ({
  useOptionalCurrentUser: () => ({
    title: "Alice",
    email: "alice@example.com",
  }),
  useSubscription: () => ({
    subscription: { plan: PlanType.FREE },
    uploadLicense: mocks.uploadLicense,
  }),
  useWorkspace: () => ({
    logo: "",
  }),
  useAppFeature: () => mocks.hideQuickStart,
  useWorkspaceSetupGuideResume: () => mocks.resumeQuickstart,
  useIntroStateByKey: (key: string) => mocks.introState[key] ?? false,
}));

vi.mock("@/app/analytics/provider", () => ({
  behaviorAnalytics: {
    captureMetric: mocks.captureMetric,
  },
}));

vi.mock("@/modules/workspace-setup-guide/selection", () => ({
  readGuideWorkspaceUsage: () => mocks.workspaceUsage,
  readSelectedGuideScenarioId: () => mocks.scenarioId,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: unknown) => unknown) =>
      selector({
        workspaceSetupGuideEnabled: (allowMultipleMembers = false) =>
          !mocks.hideQuickStart &&
          mocks.canReadSetupResources &&
          (mocks.userCountInIam === 1 || allowMultipleMembers),
      }),
    {
      getState: () => ({
        logout: mocks.logout,
      }),
    }
  ),
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
  mocks.canReadSetupResources = true;
  mocks.userCountInIam = 1;
  mocks.isDev = false;
  mocks.scenarioId = "query-data";
  mocks.workspaceUsage = undefined;
  mocks.introState = {};
  mocks.currentRoute.name = "sql-editor.home";
  window.open = vi.fn();
  ({ ProfileMenuTrigger } = await import("./ProfileMenuTrigger"));
});

describe("ProfileMenuTrigger", () => {
  test("expands the profile menu content instead of scrolling it", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ProfileMenuTrigger size="medium" link />
    );

    render();

    const content = container.querySelector(
      "[data-testid='profile-menu-content']"
    );
    expect(content?.className).toContain("max-h-none");
    expect(content?.className).toContain("overflow-visible");

    unmount();
  });

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

  test("renders the shared SQL Editor link outside SQL Editor", () => {
    mocks.currentRoute.name = "workspace.landing";
    const { container, render, unmount } = renderIntoContainer(
      <ProfileMenuTrigger size="medium" link />
    );

    render();

    const sqlEditorLink = Array.from(container.querySelectorAll("a")).find(
      (link) => link.textContent?.includes("Go to SQL Editor")
    );
    expect(sqlEditorLink).not.toBeUndefined();
    expect(sqlEditorLink).toHaveAttribute("target", "_blank");

    unmount();
  });

  test("hides getting started when the app feature disables it", () => {
    mocks.hideQuickStart = true;
    const { container, render, unmount } = renderIntoContainer(
      <ProfileMenuTrigger size="medium" link />
    );

    render();

    expect(container.textContent).not.toContain("Getting started");
    unmount();
  });

  test("resumes the selected guide", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ProfileMenuTrigger size="medium" link />
    );

    render();

    const gettingStartedButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent === "Getting started");
    expect(gettingStartedButton).not.toBeUndefined();
    act(() => gettingStartedButton?.click());
    expect(mocks.resumeQuickstart).toHaveBeenCalledTimes(1);
    expect(mocks.captureMetric).not.toHaveBeenCalled();
    unmount();
  });

  test("resumes the generic setup journey without scenario analytics", () => {
    mocks.scenarioId = undefined;
    const { container, render, unmount } = renderIntoContainer(
      <ProfileMenuTrigger size="medium" link />
    );

    render();

    const gettingStartedButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent === "Getting started");
    act(() => gettingStartedButton?.click());
    expect(mocks.resumeQuickstart).toHaveBeenCalledTimes(1);
    expect(mocks.captureMetric).not.toHaveBeenCalled();
    unmount();
  });

  test.each([0, 2])("hides getting started for IAM user count %s", (count) => {
    mocks.userCountInIam = count;
    const { container, render, unmount } = renderIntoContainer(
      <ProfileMenuTrigger size="medium" link />
    );

    render();

    expect(container.textContent).not.toContain("Getting started");
    unmount();
  });

  test("keeps the original admin's unfinished team guide resumable", () => {
    mocks.userCountInIam = 2;
    mocks.workspaceUsage = "team";
    const { container, render, unmount } = renderIntoContainer(
      <ProfileMenuTrigger size="medium" link />
    );

    render();

    expect(container.textContent).toContain("Getting started");
    unmount();
  });

  test("ends the multi-member exception after completion acknowledgment", () => {
    mocks.userCountInIam = 2;
    mocks.workspaceUsage = "team";
    mocks.introState["workspace-setup-guide.completed.query-data"] = true;
    const { container, render, unmount } = renderIntoContainer(
      <ProfileMenuTrigger size="medium" link />
    );

    render();

    expect(container.textContent).not.toContain("Getting started");
    unmount();
  });

  test("does not give an invited member the local team exception", () => {
    mocks.userCountInIam = 2;
    mocks.workspaceUsage = undefined;
    const { container, render, unmount } = renderIntoContainer(
      <ProfileMenuTrigger size="medium" link />
    );

    render();

    expect(container.textContent).not.toContain("Getting started");
    unmount();
  });

  test("hides getting started without setup resource permissions", () => {
    mocks.canReadSetupResources = false;
    const { container, render, unmount } = renderIntoContainer(
      <ProfileMenuTrigger size="medium" link />
    );

    render();

    expect(container.textContent).not.toContain("Getting started");
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
