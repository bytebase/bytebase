import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getPlan: vi.fn(),
  getPlanCheckRun: vi.fn(),
  page: {} as Record<string, unknown>,
  patchState: vi.fn(),
  pushNotification: vi.fn(),
  refreshState: vi.fn(),
  runPlanChecks: vi.fn(),
  setIsRunningChecks: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/connect", () => ({
  planServiceClientConnect: {
    getPlan: mocks.getPlan,
    getPlanCheckRun: mocks.getPlanCheckRun,
    runPlanChecks: mocks.runPlanChecks,
  },
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => ({ email: "me@example.com" }),
}));

vi.mock("@/react/hooks/useProjectByName", () => ({
  useProjectByName: () => ({ name: "projects/foo" }),
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: (selector: (state: { projectsByName: object }) => unknown) =>
    selector({ projectsByName: {} }),
}));

vi.mock("@/store", () => ({
  projectNamePrefix: "projects/",
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/store/modules/v1/common", () => ({
  extractUserEmail: (name: string) => name.replace(/^users\//, ""),
}));

vi.mock("@/utils", () => ({
  hasProjectPermissionV2: () => true,
}));

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));

import { usePlanCheckActions } from "./usePlanCheckActions";

describe("usePlanCheckActions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    Object.assign(mocks.page, {
      isRunningChecks: false,
      patchState: mocks.patchState,
      plan: {
        creator: "users/me@example.com",
        hasRollout: false,
        name: "projects/foo/plans/1",
      },
      projectId: "foo",
      readonly: false,
      refreshState: mocks.refreshState,
      setIsRunningChecks: mocks.setIsRunningChecks,
    });
    mocks.getPlan.mockResolvedValue({ name: "projects/foo/plans/1" });
    mocks.getPlanCheckRun.mockResolvedValue({
      name: "projects/foo/plans/1/planCheckRun",
    });
    mocks.refreshState.mockResolvedValue(undefined);
    mocks.runPlanChecks.mockResolvedValue({});
  });

  test("refreshes the full page snapshot after starting checks", async () => {
    const { result } = renderHook(() => usePlanCheckActions());

    await act(async () => {
      await result.current.runChecks();
    });

    expect(mocks.runPlanChecks).toHaveBeenCalledTimes(1);
    expect(mocks.refreshState).toHaveBeenCalledTimes(1);
    expect(mocks.getPlan).not.toHaveBeenCalled();
    expect(mocks.getPlanCheckRun).not.toHaveBeenCalled();
    expect(mocks.patchState).not.toHaveBeenCalled();
    expect(mocks.setIsRunningChecks.mock.calls.map(([value]) => value)).toEqual([
      true,
      false,
    ]);
  });

  test("does not report a successful run as failed when refresh fails", async () => {
    mocks.refreshState.mockRejectedValue(new Error("refresh failed"));
    const { result } = renderHook(() => usePlanCheckActions());

    await act(async () => {
      await result.current.runChecks();
    });

    expect(mocks.runPlanChecks).toHaveBeenCalledTimes(1);
    expect(mocks.refreshState).toHaveBeenCalledTimes(1);
    expect(mocks.pushNotification).toHaveBeenCalledTimes(1);
    expect(mocks.pushNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        style: "SUCCESS",
        title: "plan.checks.started",
      })
    );
    expect(mocks.setIsRunningChecks.mock.calls.map(([value]) => value)).toEqual([
      true,
      false,
    ]);
  });

  test("reports a run failure without refreshing", async () => {
    mocks.runPlanChecks.mockRejectedValue(new Error("run failed"));
    const { result } = renderHook(() => usePlanCheckActions());

    await act(async () => {
      await result.current.runChecks();
    });

    expect(mocks.refreshState).not.toHaveBeenCalled();
    expect(mocks.pushNotification).toHaveBeenCalledTimes(1);
    expect(mocks.pushNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        style: "CRITICAL",
        title: "plan.checks.failed-to-run",
      })
    );
    expect(mocks.setIsRunningChecks.mock.calls.map(([value]) => value)).toEqual([
      true,
      false,
    ]);
  });

  test("refreshes the full page snapshot when opening check results", async () => {
    const { result } = renderHook(() => usePlanCheckActions());

    await act(async () => {
      await result.current.refreshChecks();
    });

    expect(mocks.refreshState).toHaveBeenCalledTimes(1);
    expect(mocks.getPlan).not.toHaveBeenCalled();
    expect(mocks.getPlanCheckRun).not.toHaveBeenCalled();
    expect(mocks.patchState).not.toHaveBeenCalled();
  });
});
