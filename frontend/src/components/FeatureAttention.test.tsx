import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import dynamicEn from "@/locales/dynamic/en-US.json";
import { ENTERPRISE_INQUIRE_LINK } from "@/types/common";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
  useSubscriptionState: vi.fn(),
  useServerState: vi.fn(),
  useAppStore: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(() => true),
  autoSubscriptionRoute: vi.fn(() => "/subscription"),
  routerPush: vi.fn(),
}));

let FeatureAttention: typeof import("./FeatureAttention").FeatureAttention;

const translateFromDynamicEn = (key: string) => {
  const segments = key.startsWith("dynamic.")
    ? key.slice("dynamic.".length).split(".")
    : key.split(".");
  const value = segments.reduce<unknown>((node, segment) => {
    if (!node || typeof node !== "object") {
      return undefined;
    }
    return (node as Record<string, unknown>)[segment];
  }, dynamicEn);
  return typeof value === "string" ? value : key;
};

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/hooks/useAppState", () => ({
  useSubscriptionState: mocks.useSubscriptionState,
  useServerState: mocks.useServerState,
}));

vi.mock("@/components/InstanceAssignmentSheet", () => ({
  InstanceAssignmentSheet: () => null,
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    push: mocks.routerPush,
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: mocks.useAppStore,
}));

vi.mock("@/types", () => ({
  ENTERPRISE_INQUIRE_LINK,
  instanceLimitFeature: new Set<PlanFeature>([
    PlanFeature.FEATURE_DATA_MASKING,
  ]),
}));

vi.mock("@/utils", () => ({
  autoSubscriptionRoute: mocks.autoSubscriptionRoute,
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
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
      }),
  };
};

beforeEach(async () => {
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.useSubscriptionState.mockReset();
  mocks.useSubscriptionState.mockReturnValue({
    isTrialing: false,
    trialingDays: 14,
  });
  mocks.useServerState.mockReset();
  mocks.useServerState.mockReturnValue({
    totalInstanceCount: 0,
    activatedInstanceCount: 0,
  });
  mocks.useAppStore.mockReset();
  mocks.useAppStore.mockImplementation(
    (selector: (state: Record<string, unknown>) => unknown) =>
      selector({
        hasInstanceFeature: () => false,
        instanceMissingLicense: () => false,
        hasUnifiedInstanceLicense: () => false,
        getMinimumRequiredPlan: () => PlanType.TEAM,
        hasFeature: () => false,
      })
  );
  mocks.hasWorkspacePermissionV2.mockReset();
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);
  mocks.autoSubscriptionRoute.mockReset();
  mocks.autoSubscriptionRoute.mockReturnValue("/subscription");
  mocks.routerPush.mockReset();
  ({ FeatureAttention } = await import("./FeatureAttention"));
});

describe("FeatureAttention", () => {
  test("renders the JIT feature title and description from dynamic i18n", () => {
    mocks.useTranslation.mockReturnValue({
      t: (key: string) => translateFromDynamicEn(key),
    });
    const { container, render, unmount } = renderIntoContainer(
      <FeatureAttention feature={PlanFeature.FEATURE_JIT} />
    );

    render();

    expect(container.textContent).toContain("Just-In-Time access");
    expect(container.textContent).toContain(
      "Allow users to request temporary database access with approval."
    );
    expect(container.textContent).not.toContain(
      "dynamic.subscription.features.FEATURE_JIT"
    );

    unmount();
  });

  test("renders warning state as an alert and opens the inquiry link", () => {
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    const { container, render, unmount } = renderIntoContainer(
      <FeatureAttention feature={PlanFeature.FEATURE_AUDIT_LOG} />
    );

    render();

    const alert = container.querySelector('[role="alert"]');
    expect(alert).toBeTruthy();
    expect(alert?.querySelectorAll("svg")).toHaveLength(1);

    const actionButton = [...container.querySelectorAll("button")].find(
      (button) =>
        button.textContent?.includes("subscription.request-n-days-trial")
    );
    expect(actionButton).toBeTruthy();

    act(() => {
      actionButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });

    expect(openSpy).toHaveBeenCalledWith(ENTERPRISE_INQUIRE_LINK, "_blank");

    openSpy.mockRestore();
    unmount();
  });

  test("does not show assignment attention in unified instance license mode", () => {
    mocks.useSubscriptionState.mockReturnValue({
      isTrialing: false,
      trialingDays: 14,
    });
    mocks.useServerState.mockReturnValue({
      totalInstanceCount: 2,
      activatedInstanceCount: 1,
    });
    mocks.useAppStore.mockImplementation(
      (selector: (state: Record<string, unknown>) => unknown) =>
        selector({
          hasInstanceFeature: () => true,
          instanceMissingLicense: () => false,
          hasUnifiedInstanceLicense: () => true,
          getMinimumRequiredPlan: () => PlanType.TEAM,
          hasFeature: () => true,
        })
    );
    const { container, render, unmount } = renderIntoContainer(
      <FeatureAttention feature={PlanFeature.FEATURE_DATA_MASKING} />
    );

    render();

    expect(container.querySelector('[role="alert"]')).toBeNull();
    expect(
      [...container.querySelectorAll("button")].some((button) =>
        button.textContent?.includes(
          "subscription.instance-assignment.assign-license"
        )
      )
    ).toBe(false);

    unmount();
  });
});
