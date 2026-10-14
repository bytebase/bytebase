import { render } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { AUTH_SETUP_MODULE, AUTH_SIGNIN_MODULE } from "@/app/router/handles";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { SplashLayout, SplashPanel } from "./SplashLayout";

const storeState = vi.hoisted(() => ({
  currentPlan: 0,
  isTrialing: false,
  routeName: "",
}));

// The routed page, which may put a panel of its own in the left column.
vi.mock("react-router", () => ({
  Outlet: () => (
    <div data-testid="auth-page">
      <SplashPanel>
        <p>the page's own panel</p>
      </SplashPanel>
    </div>
  ),
  useMatches: () => [{ handle: { name: storeState.routeName } }],
}));

vi.mock("@/stores/app", () => {
  type MockState = {
    currentPlan: () => number;
    isTrialing: () => boolean;
  };
  const state = {
    currentPlan: () => storeState.currentPlan,
    isTrialing: () => storeState.isTrialing,
  } satisfies MockState;
  const useAppStore = (selector: (state: MockState) => unknown) =>
    selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});

describe("SplashLayout", () => {
  beforeEach(() => {
    storeState.currentPlan = PlanType.FREE;
    storeState.isTrialing = false;
    storeState.routeName = AUTH_SIGNIN_MODULE;
  });

  test("keeps the mounted auth layout stable when subscription loads during login", () => {
    const { container, rerender } = render(<SplashLayout />);

    expect(container.querySelector("img")).toBeInTheDocument();

    storeState.currentPlan = PlanType.ENTERPRISE;
    rerender(<SplashLayout />);

    expect(container.querySelector("img")).toBeInTheDocument();
  });

  test("keeps the illustration on sign-in, whatever panel the page offers", () => {
    const { container, queryByText } = render(<SplashLayout />);

    expect(container.querySelector("img")).toBeInTheDocument();
    expect(queryByText("the page's own panel")).not.toBeInTheDocument();
  });

  test.each([
    { plan: "the free plan", currentPlan: PlanType.FREE },
    { plan: "a paid plan", currentPlan: PlanType.ENTERPRISE },
  ])("gives the left column to the setup wizard's panel on $plan", ({
    currentPlan,
  }) => {
    storeState.routeName = AUTH_SETUP_MODULE;
    storeState.currentPlan = currentPlan;
    const { container, getByText, getByTestId } = render(<SplashLayout />);

    const panel = getByText("the page's own panel");
    const [leftColumn, rightColumn] = container.firstElementChild?.children ?? [];
    expect(container.querySelector("img")).not.toBeInTheDocument();
    expect(leftColumn).toContainElement(panel);
    expect(rightColumn).toContainElement(getByTestId("auth-page"));
    expect(rightColumn).not.toContainElement(panel);
  });
});
