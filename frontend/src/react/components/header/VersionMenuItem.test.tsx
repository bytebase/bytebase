import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { SETTING_ROUTE_WORKSPACE_SUBSCRIPTION } from "@/react/router";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { STORAGE_KEY_RELEASE } from "@/utils/storage-keys";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  canManageSettings: true,
  serverInfo: {
    version: "1.0.0",
    gitCommit: "backend123",
    saas: false,
    demo: false,
  },
  subscription: {
    plan: 1,
  },
  push: vi.fn(),
  closeMenu: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, string>) =>
      ({
        "common.demo-mode": "Demo Mode",
        "subscription.plan.free.title": "Free",
        "subscription.plan.team.title": "Team",
        "subscription.plan.enterprise.title": "Enterprise",
        "remind.release.new-version-available-with-tag": `New version ${params?.tag}`,
        "common.dismiss": "Dismiss",
        "common.learn-more": "Learn more",
      })[key] ?? key,
  }),
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useServerInfo: () => mocks.serverInfo,
  useSubscription: () => ({
    subscription: mocks.subscription,
  }),
  useWorkspacePermission: () => mocks.canManageSettings,
}));

vi.mock("@/react/router", () => ({
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION: "setting.workspace.subscription",
  useNavigate: () => ({
    push: mocks.push,
  }),
}));

vi.mock("@/react/components/ui/badge", () => ({
  Badge: ({ children }: { children: ReactElement | string }) => (
    <span>{children}</span>
  ),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
  }: {
    children: ReactElement | string;
    onClick?: () => void;
  }) => <button onClick={onClick}>{children}</button>,
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({
    children,
    open,
  }: {
    children: ReactElement | ReactElement[];
    open: boolean;
  }) => (open ? <div>{children}</div> : null),
  DialogContent: ({
    children,
  }: {
    children: ReactElement | ReactElement[];
  }) => <div>{children}</div>,
  DialogDescription: ({ children }: { children: ReactElement | string }) => (
    <div>{children}</div>
  ),
  DialogTitle: ({ children }: { children: ReactElement | string }) => (
    <h2>{children}</h2>
  ),
}));

let VersionMenuItem: typeof import("./VersionMenuItem").VersionMenuItem;

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
  localStorage.clear();
  mocks.canManageSettings = true;
  mocks.serverInfo = {
    version: "1.0.0",
    gitCommit: "backend123",
    saas: false,
    demo: false,
  };
  mocks.subscription = {
    plan: PlanType.TEAM,
  };
  window.open = vi.fn();
  ({ VersionMenuItem } = await import("./VersionMenuItem"));
});

describe("VersionMenuItem", () => {
  test("links subscription when the user can manage settings", () => {
    const { container, render, unmount } = renderIntoContainer(
      <VersionMenuItem onCloseMenu={mocks.closeMenu} />
    );
    render();

    const planButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Team")
    );
    expect(planButton).not.toBeUndefined();
    act(() => {
      planButton?.click();
    });

    expect(mocks.push).toHaveBeenCalledWith({
      name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
    });
    expect(mocks.closeMenu).toHaveBeenCalled();
    unmount();
  });

  test("opens the release dialog and learn-more link for new releases", () => {
    localStorage.setItem(
      STORAGE_KEY_RELEASE,
      JSON.stringify({
        latest: {
          tag_name: "2.0.0",
          html_url: "https://example.com/release",
        },
        ignoreRemindModalTillNextRelease: false,
        nextCheckTs: 0,
      })
    );
    const { container, render, unmount } = renderIntoContainer(
      <VersionMenuItem onCloseMenu={mocks.closeMenu} />
    );
    render();

    const versionButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("v1.0.0")
    );
    act(() => {
      versionButton?.click();
    });

    expect(container.textContent).toContain("New version 2.0.0");
    expect(mocks.closeMenu).toHaveBeenCalled();

    const learnMoreButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("Learn more"));
    act(() => {
      learnMoreButton?.click();
    });

    expect(window.open).toHaveBeenCalledWith(
      "https://example.com/release",
      "_blank"
    );
    unmount();
  });

  test("hides version info in SaaS/cloud mode", () => {
    mocks.serverInfo = {
      version: "1.0.0",
      gitCommit: "backend123",
      saas: true,
      demo: false,
    };
    const { container, render, unmount } = renderIntoContainer(
      <VersionMenuItem onCloseMenu={mocks.closeMenu} />
    );
    render();

    expect(container.textContent).not.toContain("v1.0.0");
    expect(container.textContent).not.toContain("BE Git hash");
    expect(container.textContent).not.toContain("FE Git hash");
    // The plan label should still be visible.
    expect(container.textContent).toContain("Team");
    unmount();
  });
});
