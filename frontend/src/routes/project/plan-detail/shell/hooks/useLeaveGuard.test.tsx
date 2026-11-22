import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, expect, test, vi } from "vitest";
import { PlanDetailStoreProvider } from "../../shared/stores/PlanDetailStoreProvider";
import { usePlanDetailStoreApi } from "../../shared/stores/usePlanDetailStore";
import { useLeaveGuard } from "./useLeaveGuard";

const mocks = vi.hoisted(() => ({ replace: vi.fn() }));

vi.mock("@/app/router", () => ({
  router: { beforeEach: () => () => {}, replace: mocks.replace },
}));

const wrapper = ({ children }: { children: ReactNode }) => (
  <PlanDetailStoreProvider>{children}</PlanDetailStoreProvider>
);

const renderGuard = () =>
  renderHook(
    () => ({ guard: useLeaveGuard(), store: usePlanDetailStoreApi() }),
    { wrapper }
  );

// Editing one change, the reviewer asks for a thread on another; the guard
// holds that navigation for confirmation.
const holdThreadNavigation = (
  store: ReturnType<typeof usePlanDetailStoreApi>
) =>
  act(() => {
    const state = store.getState();
    state.setEditing("statement:spec-1", true);
    state.requestThreadFocus({
      commentName: "projects/p/issues/1/issueComments/7",
      specId: "spec-2",
      lineNumber: 4,
    });
    state.setPendingLeaveTarget("/projects/p/plans/1/specs/spec-2");
    state.setPendingLeaveConfirm(true);
  });

beforeEach(() => vi.clearAllMocks());

test("cancelling the leave drops the thread focus waiting on it", () => {
  const { result } = renderGuard();
  holdThreadNavigation(result.current.store);

  act(() => result.current.guard.resolveLeaveConfirm(false));

  expect(result.current.store.getState().threadFocus).toBeUndefined();
  expect(mocks.replace).not.toHaveBeenCalled();
});

test("confirming the leave keeps the thread focus for the destination", () => {
  const { result } = renderGuard();
  holdThreadNavigation(result.current.store);

  act(() => result.current.guard.resolveLeaveConfirm(true));

  expect(result.current.store.getState().threadFocus?.specId).toBe("spec-2");
  expect(mocks.replace).toHaveBeenCalledWith(
    "/projects/p/plans/1/specs/spec-2"
  );
});
