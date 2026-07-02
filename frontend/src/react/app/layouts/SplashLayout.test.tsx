import { render } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { AUTH_SIGNIN_MODULE } from "@/react/router/handles";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { SplashLayout } from "./SplashLayout";

const storeState = vi.hoisted(() => ({
  currentPlan: 0,
  isTrialing: false,
}));

vi.mock("react-router-dom", () => ({
  Outlet: () => <div data-testid="auth-page" />,
  useMatches: () => [{ handle: { name: AUTH_SIGNIN_MODULE } }],
}));

vi.mock("@/react/stores/app", () => {
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
  });

  test("keeps the mounted auth layout stable when subscription loads during login", () => {
    const { container, rerender } = render(<SplashLayout />);

    expect(container.querySelector("img")).toBeInTheDocument();

    storeState.currentPlan = PlanType.ENTERPRISE;
    rerender(<SplashLayout />);

    expect(container.querySelector("img")).toBeInTheDocument();
  });
});
