import type { ReactElement, ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  Announcement_AlertLevel,
  type WorkspaceProfileSetting,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  getMinimumRequiredPlan: vi.fn(),
  isDev: vi.fn(),
  navigatePush: vi.fn(),
  t: vi.fn(),
  urlfy: vi.fn((value: string) => `https://${value}`),
  useAppStore: vi.fn(),
  usePlanFeature: vi.fn(),
  useServerState: vi.fn(),
  useSubscriptionState: vi.fn(),
  useWorkspacePermission: vi.fn(),
  useWorkspaceProfile: vi.fn(),
}));

const translations: Record<string, string> = {
  "banner.cloud": "Cloud",
  "banner.deploy": "Self Host",
  "banner.external-url": "Bytebase has not configured --external-url",
  "banner.license-expired": "{{plan}} expired on {{expireAt}}",
  "banner.license-expires": "{{plan}} expires in {{days}} days on {{expireAt}}",
  "banner.request-demo": "Book a demo",
  "banner.trial-expires":
    "{{plan}} trial expires in {{days}} days on {{expireAt}}",
  "common.configure-now": "Configure now",
  "common.dismiss": "Dismiss",
  "subscription.overuse-modal.description": "Overuse {{plan}}",
  "subscription.overuse-warning": "{{neededPlan}} on {{currentPlan}}",
  "subscription.plan.enterprise.title": "Enterprise",
  "subscription.plan-features": "{{plan}} Features",
  "subscription.plan.free.title": "Free",
  "subscription.plan.team.title": "Pro",
  "subscription.purchase.subscribe": "Subscribe now",
  "subscription.purchase.update": "Update Subscription",
  "subscription.upgrade": "Upgrade",
  "subscription.upgrade-now": "Upgrade Now",
};

function interpolate(key: string, values?: Record<string, unknown>) {
  let result = translations[key] ?? key;
  for (const [name, value] of Object.entries(values ?? {})) {
    result = result.replaceAll(`{{${name}}}`, String(value));
  }
  return result;
}

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: mocks.t,
  }),
  Trans: ({
    i18nKey,
    values,
  }: {
    i18nKey: string;
    values?: Record<string, unknown>;
    components?: Record<string, ReactNode>;
  }) => <span>{interpolate(i18nKey, values)}</span>,
}));

vi.mock("@/react/hooks/useAppState", () => ({
  usePlanFeature: mocks.usePlanFeature,
  useServerState: mocks.useServerState,
  useSubscriptionState: mocks.useSubscriptionState,
  useWorkspacePermission: mocks.useWorkspacePermission,
  useWorkspaceProfile: mocks.useWorkspaceProfile,
}));

vi.mock("@/react/router", () => ({
  SETTING_ROUTE_WORKSPACE_GENERAL: "setting.workspace.general",
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION: "setting.workspace.subscription",
  useNavigate: () => ({
    push: mocks.navigatePush,
  }),
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: mocks.useAppStore,
}));

vi.mock("@/utils", () => ({
  isDev: mocks.isDev,
  urlfy: mocks.urlfy,
}));

let BannersWrapper: typeof import("./BannersWrapper").BannersWrapper;

const defaultSubscriptionState = {
  currentPlan: PlanType.FREE,
  daysBeforeExpire: -1,
  expireAt: "",
  isExpired: false,
  isTrialing: false,
};

function renderIntoContainer(element: ReactElement) {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  act(() => {
    root.render(element);
  });
  return {
    container,
    unmount: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
}

beforeEach(async () => {
  mocks.getMinimumRequiredPlan.mockReset();
  mocks.getMinimumRequiredPlan.mockReturnValue(PlanType.FREE);
  mocks.isDev.mockReset();
  mocks.isDev.mockReturnValue(false);
  mocks.navigatePush.mockReset();
  mocks.t.mockReset();
  mocks.t.mockImplementation(interpolate);
  mocks.urlfy.mockClear();
  mocks.usePlanFeature.mockReset();
  mocks.usePlanFeature.mockReturnValue(false);
  mocks.useServerState.mockReset();
  mocks.useServerState.mockReturnValue({
    needConfigureExternalUrl: false,
    serverInfo: undefined,
  });
  mocks.useSubscriptionState.mockReset();
  mocks.useSubscriptionState.mockReturnValue(defaultSubscriptionState);
  mocks.useWorkspacePermission.mockReset();
  mocks.useWorkspacePermission.mockReturnValue(false);
  mocks.useWorkspaceProfile.mockReset();
  mocks.useWorkspaceProfile.mockReturnValue(undefined);
  mocks.useAppStore.mockReset();
  mocks.useAppStore.mockImplementation((selector) =>
    selector({
      getMinimumRequiredPlan: mocks.getMinimumRequiredPlan,
    })
  );
  ({ BannersWrapper } = await import("./BannersWrapper"));
});

describe("BannersWrapper", () => {
  test("renders and dismisses demo banner", () => {
    mocks.useServerState.mockReturnValue({
      needConfigureExternalUrl: false,
      serverInfo: { demo: true },
    });
    const { container, unmount } = renderIntoContainer(<BannersWrapper />);

    expect(container.textContent).toContain("Book a demo");

    const dismissButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "Dismiss"
    );
    act(() => {
      dismissButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(container.textContent).not.toContain("Book a demo");
    unmount();
  });

  test("renders trialing subscription banner with subscription action", () => {
    mocks.useSubscriptionState.mockReturnValue({
      ...defaultSubscriptionState,
      currentPlan: PlanType.TEAM,
      daysBeforeExpire: 3,
      expireAt: "2030-01-01",
      isTrialing: true,
    });
    const { container, unmount } = renderIntoContainer(<BannersWrapper />);

    expect(container.textContent).toContain(
      "Pro trial expires in 3 days on 2030-01-01"
    );
    expect(container.textContent).toContain("Subscribe now");
    unmount();
  });

  test("renders external URL banner with configure action for permitted users", () => {
    mocks.useServerState.mockReturnValue({
      needConfigureExternalUrl: true,
      serverInfo: undefined,
    });
    mocks.useWorkspacePermission.mockReturnValue(true);
    const { container, unmount } = renderIntoContainer(<BannersWrapper />);

    expect(container.textContent).toContain(
      "Bytebase has not configured --external-url"
    );

    const configureButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("Configure now"));
    act(() => {
      configureButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });

    expect(mocks.navigatePush).toHaveBeenCalledWith({
      name: "setting.workspace.general",
    });
    unmount();
  });

  test("hides external URL banner in dev mode", () => {
    mocks.isDev.mockReturnValue(true);
    mocks.useServerState.mockReturnValue({
      needConfigureExternalUrl: true,
      serverInfo: undefined,
    });
    const { container, unmount } = renderIntoContainer(<BannersWrapper />);

    expect(container.textContent).not.toContain(
      "Bytebase has not configured --external-url"
    );
    unmount();
  });

  test("renders announcement only when feature is available", () => {
    mocks.usePlanFeature.mockReturnValue(true);
    mocks.useWorkspaceProfile.mockReturnValue({
      announcement: {
        level: Announcement_AlertLevel.INFO,
        link: "example.com/path",
        text: "Maintenance window",
      },
    } as WorkspaceProfileSetting);
    const { container, unmount } = renderIntoContainer(<BannersWrapper />);

    const link = container.querySelector("a");
    expect(container.textContent).toContain("Maintenance window");
    expect(link?.getAttribute("href")).toBe("https://example.com/path");
    unmount();
  });

  test("renders upgrade banner for unlicensed features above current plan", () => {
    mocks.useServerState.mockReturnValue({
      needConfigureExternalUrl: false,
      serverInfo: { unlicensedFeatures: ["FEATURE_BATCH_QUERY"] },
    });
    mocks.getMinimumRequiredPlan.mockReturnValue(PlanType.TEAM);
    const { container, unmount } = renderIntoContainer(<BannersWrapper />);

    expect(container.textContent).toContain("Pro Features on Free");
    expect(container.textContent).toContain("Upgrade");
    unmount();
  });
});
