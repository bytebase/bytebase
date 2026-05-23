import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  subscriptionState: {
    currentPlan: 3,
    expireAt: "",
    hasUnifiedInstanceLicense: true,
    instanceCountLimit: 10,
    instanceLicenseCount: 10,
    isExpired: false,
    isFreePlan: false,
    isSelfHostLicense: true,
    isTrialing: false,
    showTrial: false,
    trialingDays: 14,
    uploadLicense: vi.fn(),
    userCountLimit: 20,
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/react/components/InstanceAssignmentSheet", () => ({
  InstanceAssignmentSheet: () => null,
}));

vi.mock("@/react/components/ui/badge", () => ({
  Badge: ({ children }: { children: React.ReactNode }) => (
    <span>{children}</span>
  ),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => (
    <input {...props} />
  ),
}));

vi.mock("@/react/components/ui/textarea", () => ({
  Textarea: (props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) => (
    <textarea {...props} />
  ),
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useNotify: () => vi.fn(),
  useServerState: () => ({
    activatedInstanceCount: 2,
    isSaaSMode: false,
    totalInstanceCount: 3,
    userCountInIam: 4,
    workspaceResourceName: "workspaces/workspace-id",
  }),
  useSubscriptionState: () => mocks.subscriptionState,
}));

vi.mock("@/types", () => ({
  ENTERPRISE_INQUIRE_LINK: "https://example.com",
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: () => true,
}));

vi.mock("./PurchaseSection", () => ({
  PurchaseSection: () => null,
}));

let SubscriptionPage: typeof import("./SubscriptionPage").SubscriptionPage;

describe("SubscriptionPage", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(async () => {
    vi.clearAllMocks();
    window.open = vi.fn();
    mocks.subscriptionState.currentPlan = PlanType.ENTERPRISE;
    mocks.subscriptionState.expireAt = "";
    mocks.subscriptionState.hasUnifiedInstanceLicense = true;
    mocks.subscriptionState.instanceCountLimit = 10;
    mocks.subscriptionState.instanceLicenseCount = 10;
    mocks.subscriptionState.isExpired = false;
    mocks.subscriptionState.isFreePlan = false;
    mocks.subscriptionState.isSelfHostLicense = true;
    mocks.subscriptionState.isTrialing = false;
    mocks.subscriptionState.showTrial = false;
    mocks.subscriptionState.trialingDays = 14;
    mocks.subscriptionState.uploadLicense = vi.fn();
    mocks.subscriptionState.userCountLimit = 20;
    ({ SubscriptionPage } = await import("./SubscriptionPage"));
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);
  });

  test("displays unified instance usage as current count over limit", () => {
    act(() => {
      root.render(<SubscriptionPage />);
    });

    expect(
      container.textContent?.includes(
        "subscription.instance-assignment.used-and-total-instance"
      )
    ).toBe(true);
    expect(container.textContent?.includes("3/10")).toBe(true);

    act(() => root.unmount());
    container.remove();
  });

  test("requesting enterprise opens inquiry page without an opener", () => {
    mocks.subscriptionState.showTrial = true;
    act(() => {
      root.render(<SubscriptionPage />);
    });

    const buttons = Array.from(container.querySelectorAll("button"));
    const upgradeButton = buttons.find((button) =>
      button.textContent?.includes("subscription.enterprise-free-trial")
    );
    expect(upgradeButton).not.toBeUndefined();

    act(() => {
      upgradeButton?.click();
    });

    expect(window.open).toHaveBeenCalledWith(
      "https://example.com",
      "_blank",
      "noopener,noreferrer"
    );

    act(() => root.unmount());
    container.remove();
  });
});
