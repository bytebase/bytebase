import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { PurchaseSection } from "./PurchaseSection";

const mocks = vi.hoisted(() => ({
  isTrialing: false,
  isExpired: false,
  allowManage: true,
  fetchPaymentInfo: vi.fn(),
  fetchPurchasePlans: vi.fn(),
  refreshSubscription: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/hooks/useAppState", () => ({
  useSubscriptionState: () => ({
    currentPlan: PlanType.ENTERPRISE,
    isFreePlan: false,
    isExpired: mocks.isExpired,
    isTrialing: mocks.isTrialing,
    subscription: { seats: 20 },
  }),
}));
vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: unknown) => unknown) =>
      selector({ ...mocks, paymentInfo: null, purchasePlans: [] }),
    { getState: () => mocks }
  ),
}));
vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: () => mocks.allowManage,
}));
vi.mock("./CancelSubscriptionDialog", () => ({
  CancelSubscriptionDialog: () => null,
}));

beforeEach(() => {
  vi.clearAllMocks();
  mocks.isTrialing = false;
  mocks.isExpired = false;
  mocks.allowManage = true;
});
afterEach(cleanup);

test("does not offer cancellation or fetch payment info for a license trial", () => {
  mocks.isTrialing = true;
  render(<PurchaseSection onRequireEnterprise={vi.fn()} />);
  expect(
    screen.queryByRole("button", { name: "subscription.purchase.cancel" })
  ).toBeNull();
  expect(mocks.fetchPaymentInfo).not.toHaveBeenCalled();
});

test("keeps cancellation available for an active paid subscription", () => {
  render(<PurchaseSection onRequireEnterprise={vi.fn()} />);
  expect(
    screen.getByRole("button", { name: "subscription.purchase.cancel" })
  ).toBeTruthy();
  expect(mocks.fetchPaymentInfo).toHaveBeenCalledTimes(1);
});

test("hides cancellation when the paid subscription has expired", () => {
  mocks.isExpired = true;
  render(<PurchaseSection onRequireEnterprise={vi.fn()} />);
  expect(
    screen.queryByRole("button", { name: "subscription.purchase.cancel" })
  ).toBeNull();
});
