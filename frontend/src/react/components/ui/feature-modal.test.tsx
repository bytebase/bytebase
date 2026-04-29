import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useSubscriptionState: vi.fn(),
  useAppStore: vi.fn(),
  instanceMissingLicense: false,
  requiredPlan: 1, // TEAM
  showTrial: false,
  trialingDays: 14,
  hasWorkspacePermissionV2: vi.fn().mockReturnValue(true),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useSubscriptionState: mocks.useSubscriptionState,
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: mocks.useAppStore,
}));

vi.mock("@/router", () => ({
  router: { push: vi.fn() },
}));

vi.mock("@/router/dashboard/workspaceRoutes", () => ({
  INSTANCE_ROUTE_DASHBOARD: "workspace.instance",
}));

vi.mock("@/types", () => ({
  ENTERPRISE_INQUIRE_LINK: "https://enterprise",
}));

vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanFeature: {
    FEATURE_UNSPECIFIED: 0,
    FEATURE_BATCH_QUERY: 1,
    FEATURE_DATABASE_GROUPS: 2,
    // Support reverse lookup: PlanFeature[feature] must return the name.
    0: "FEATURE_UNSPECIFIED",
    1: "FEATURE_BATCH_QUERY",
    2: "FEATURE_DATABASE_GROUPS",
  },
  PlanType: {
    FREE: 0,
    TEAM: 1,
    ENTERPRISE: 3,
  },
}));

vi.mock("@/utils", () => ({
  autoSubscriptionRoute: () => ({ name: "subscription" }),
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({
    open,
    children,
    onOpenChange,
  }: {
    open: boolean;
    children: React.ReactNode;
    onOpenChange: (v: boolean) => void;
  }) => (
    <div data-testid="dialog" data-open={String(open)}>
      {open ? children : null}
      <button
        data-testid="dialog-close"
        onClick={() => onOpenChange(false)}
        type="button"
      />
    </div>
  ),
  DialogContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dialog-content">{children}</div>
  ),
  DialogTitle: ({ children }: { children: React.ReactNode }) => (
    <h2 data-testid="dialog-title">{children}</h2>
  ),
  DialogClose: ({
    children,
    "aria-label": ariaLabel,
  }: {
    children: React.ReactNode;
    "aria-label"?: string;
  }) => (
    <button data-testid="dialog-close-x" aria-label={ariaLabel} type="button">
      {children}
    </button>
  ),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
  }) => (
    <button data-testid="button" onClick={onClick} type="button">
      {children}
    </button>
  ),
}));

let FeatureModal: typeof import("./feature-modal").FeatureModal;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.instanceMissingLicense = false;
  mocks.requiredPlan = 1;
  mocks.showTrial = false;
  mocks.trialingDays = 14;
  mocks.useSubscriptionState.mockReturnValue({
    showTrial: mocks.showTrial,
    trialingDays: mocks.trialingDays,
  });
  mocks.useAppStore.mockImplementation(
    (selector: (state: Record<string, unknown>) => unknown) =>
      selector({
        instanceMissingLicense: () => mocks.instanceMissingLicense,
        getMinimumRequiredPlan: () => mocks.requiredPlan,
      })
  );
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);
  ({ FeatureModal } = await import("./feature-modal"));
});

describe("FeatureModal", () => {
  test("renders nothing when feature is undefined", () => {
    const { container, render, unmount } = renderIntoContainer(
      <FeatureModal open feature={undefined} onOpenChange={() => {}} />
    );
    render();
    const title = container.querySelector("[data-testid='dialog-title']");
    expect(title).toBeNull();
    unmount();
  });

  test("renders the feature title from the dynamic namespace when open", () => {
    const { container, render, unmount } = renderIntoContainer(
      <FeatureModal open feature={1} onOpenChange={() => {}} />
    );
    render();
    const dialogTitle = container.querySelector("[data-testid='dialog-title']");
    // Dialog header shows the static "Disabled Feature" label.
    expect(dialogTitle?.textContent).toBe("subscription.disabled-feature");
    // Body heading shows the dynamic feature title.
    const heading = container.querySelector("h3");
    expect(heading?.textContent).toBe(
      "dynamic.subscription.features.FEATURE_BATCH_QUERY.title"
    );
    unmount();
  });

  test("renders OK-only button when the user lacks settings permission", () => {
    mocks.hasWorkspacePermissionV2.mockReturnValue(false);
    const onOpenChange = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <FeatureModal open feature={1} onOpenChange={onOpenChange} />
    );
    render();
    const buttons = container.querySelectorAll("[data-testid='button']");
    expect(buttons).toHaveLength(1);
    expect(buttons[0].textContent).toBe("common.ok");
    act(() => {
      (buttons[0] as HTMLButtonElement).dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(onOpenChange).toHaveBeenCalledWith(false);
    unmount();
  });

  test("renders 'assign license' CTA when the instance is missing a license", () => {
    mocks.instanceMissingLicense = true;
    const { container, render, unmount } = renderIntoContainer(
      <FeatureModal open feature={1} onOpenChange={() => {}} />
    );
    render();
    const buttons = container.querySelectorAll("[data-testid='button']");
    expect(buttons[0].textContent).toBe(
      "subscription.instance-assignment.assign-license"
    );
    unmount();
  });
});
