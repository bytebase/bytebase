import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
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
  useVueState: vi.fn((getter: () => unknown) => getter()),
  useSubscriptionV1Store: vi.fn(),
  useActuatorV1Store: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(() => true),
  autoSubscriptionRoute: vi.fn(() => "/subscription"),
  routerPush: vi.fn(),
}));

let FeatureAttention: typeof import("./FeatureAttention").FeatureAttention;

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/react/components/InstanceAssignmentBridge", () => ({
  InstanceAssignmentBridge: () => null,
}));

vi.mock("@/router", () => ({
  router: {
    push: mocks.routerPush,
  },
}));

vi.mock("@/store", () => ({
  useSubscriptionV1Store: mocks.useSubscriptionV1Store,
  useActuatorV1Store: mocks.useActuatorV1Store,
}));

vi.mock("@/types", () => ({
  ENTERPRISE_INQUIRE_LINK,
  instanceLimitFeature: new Set<PlanFeature>(),
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
  mocks.useVueState.mockReset();
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());
  mocks.useSubscriptionV1Store.mockReset();
  mocks.useSubscriptionV1Store.mockReturnValue({
    hasInstanceFeature: vi.fn(() => false),
    instanceMissingLicense: vi.fn(() => false),
    isTrialing: false,
    trialingDays: 14,
    getMinimumRequiredPlan: vi.fn(() => PlanType.TEAM),
  });
  mocks.useActuatorV1Store.mockReset();
  mocks.useActuatorV1Store.mockReturnValue({
    totalInstanceCount: 0,
    activatedInstanceCount: 0,
  });
  mocks.hasWorkspacePermissionV2.mockReset();
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);
  mocks.autoSubscriptionRoute.mockReset();
  mocks.autoSubscriptionRoute.mockReturnValue("/subscription");
  mocks.routerPush.mockReset();
  ({ FeatureAttention } = await import("./FeatureAttention"));
});

describe("FeatureAttention", () => {
  test("renders warning state as an alert and opens the inquiry link", () => {
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    const { container, render, unmount } = renderIntoContainer(
      <FeatureAttention feature={PlanFeature.FEATURE_AUDIT_LOG} />
    );

    render();

    const alert = container.querySelector('[role="alert"]');
    expect(alert).toBeTruthy();

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
});
