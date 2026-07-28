import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
} from "@/app/router/handles";
import type { PlanDetailPageSnapshot } from "./types";

const mocks = vi.hoisted(() => ({ replace: vi.fn() }));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: { replace: mocks.replace },
}));

import { useCanonicalPlanDetailRoute } from "./useCanonicalPlanDetailRoute";

const rolloutName = "projects/p/plans/1/rollout";
const snapshot = (options?: {
  hasRollout?: boolean;
  isCreating?: boolean;
  pageKey?: string;
  ready?: boolean;
  specs?: string[];
  rollout?: boolean;
}): PlanDetailPageSnapshot =>
  ({
    projectId: "p",
    planId: "1",
    pageKey: options?.pageKey ?? "p/1",
    isCreating: options?.isCreating ?? false,
    ready: options?.ready ?? true,
    plan: {
      name: "projects/p/plans/1",
      hasRollout: options?.hasRollout ?? options?.rollout ?? true,
      specs: (options?.specs ?? ["spec-1"]).map((id) => ({ id })),
      planCheckRunStatusCount: {},
    },
    rollout:
      options?.rollout === false
        ? undefined
        : {
            name: rolloutName,
            stages: [
              {
                name: `${rolloutName}/stages/a`,
                tasks: [{ name: `${rolloutName}/stages/a/tasks/t1` }],
              },
              {
                name: `${rolloutName}/stages/b`,
                tasks: [{ name: `${rolloutName}/stages/b/tasks/t2` }],
              },
            ],
          },
  }) as unknown as PlanDetailPageSnapshot;

const renderCanonical = (
  overrides: Partial<Parameters<typeof useCanonicalPlanDetailRoute>[0]>
) =>
  renderHook(() =>
    useCanonicalPlanDetailRoute({
      projectId: "p",
      planId: "1",
      routeName: PROJECT_V1_ROUTE_PLAN_DETAIL,
      routeQuery: {},
      snapshot: snapshot(),
      isEditing: false,
      bypassLeaveGuardOnce: vi.fn(),
      ...overrides,
    })
  );

beforeEach(() => {
  mocks.replace.mockReset();
});

describe("useCanonicalPlanDetailRoute", () => {
  test("does not canonicalize a plan-creation resource URL", async () => {
    renderCanonical({
      planId: "create",
      routeName: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
      routeQuery: {
        template: "bb.plan.change-database",
        databaseList: "databases/db1",
      },
      specId: "placeholder",
      snapshot: snapshot({ isCreating: true, pageKey: "p/create" }),
    });

    await waitFor(() => expect(mocks.replace).not.toHaveBeenCalled());
  });

  test("does not validate a new plan route against the previous plan snapshot", async () => {
    renderCanonical({
      planId: "2",
      routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      stageId: "a",
      taskId: "t1",
      snapshot: snapshot(),
    });

    await waitFor(() => expect(mocks.replace).not.toHaveBeenCalled());
  });

  test("removes contradictory selection queries from a spec resource", async () => {
    renderCanonical({
      routeName: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
      routeQuery: { phase: "deploy", stageId: "a", line: "42" },
      routeHash: "#result-7",
      specId: "spec-1",
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
          params: { projectId: "p", planId: "1", specId: "spec-1" },
          query: { line: "42" },
          hash: "#result-7",
        },
        { preventScrollReset: true }
      )
    );
  });

  test("removes phase queries from the specs collection resource", async () => {
    renderCanonical({
      routeName: PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
      routeQuery: { phase: "review", foo: "bar" },
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
          params: { projectId: "p", planId: "1" },
          query: { foo: "bar" },
        },
        { preventScrollReset: true }
      )
    );
  });

  test("removes selectors from a task resource without losing task-run state", async () => {
    renderCanonical({
      routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      routeQuery: {
        phase: "review",
        specId: "spec-1",
        stageId: "stale",
        taskId: "stale",
        taskRunId: "run-7",
      },
      routeHash: "#log",
      stageId: "b",
      taskId: "t2",
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
          params: {
            projectId: "p",
            planId: "1",
            stageId: "b",
            taskId: "t2",
          },
          query: { taskRunId: "run-7" },
          hash: "#log",
        },
        { preventScrollReset: true }
      )
    );
  });

  test("accepts a valid legacy phase on the plan root", async () => {
    renderCanonical({
      routeQuery: { phase: "review" },
    });

    await waitFor(() => expect(mocks.replace).not.toHaveBeenCalled());
  });

  test("upgrades legacy stage and task queries to a task resource", async () => {
    renderCanonical({
      routeQuery: { phase: "deploy", stageId: "a", taskId: "t1" },
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
          params: {
            projectId: "p",
            planId: "1",
            stageId: "a",
            taskId: "t1",
          },
          query: {},
        },
        { preventScrollReset: true }
      )
    );
  });

  test("waits for validation and canonicalizes a legacy task once", async () => {
    const { rerender } = renderHook(
      ({ currentSnapshot }: { currentSnapshot: PlanDetailPageSnapshot }) =>
        useCanonicalPlanDetailRoute({
          projectId: "p",
          planId: "1",
          routeName: PROJECT_V1_ROUTE_PLAN_DETAIL,
          routeQuery: { stageId: "a", taskId: "t2" },
          snapshot: currentSnapshot,
          isEditing: false,
          bypassLeaveGuardOnce: vi.fn(),
        }),
      { initialProps: { currentSnapshot: snapshot({ ready: false }) } }
    );

    expect(mocks.replace).not.toHaveBeenCalled();

    rerender({ currentSnapshot: snapshot() });

    await waitFor(() => expect(mocks.replace).toHaveBeenCalledTimes(1));
    expect(mocks.replace).toHaveBeenCalledWith(
      {
        name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
        params: {
          projectId: "p",
          planId: "1",
          stageId: "b",
          taskId: "t2",
        },
        query: {},
      },
      { preventScrollReset: true }
    );
  });

  test("upgrades a legacy spec query to a spec resource", async () => {
    renderCanonical({
      routeQuery: { phase: "changes", specId: "spec-1", line: "8" },
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
          params: { projectId: "p", planId: "1", specId: "spec-1" },
          query: { line: "8" },
        },
        { preventScrollReset: true }
      )
    );
  });

  test("gives a legacy spec selector precedence over stage and task", async () => {
    renderCanonical({
      routeQuery: {
        specId: "spec-1",
        stageId: "a",
        taskId: "t1",
        line: "8",
      },
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
          params: { projectId: "p", planId: "1", specId: "spec-1" },
          query: { line: "8" },
        },
        { preventScrollReset: true }
      )
    );
  });

  test("uses a task's actual owner when a legacy task has no stage", async () => {
    renderCanonical({
      routeQuery: { taskId: "t2", taskRunId: "run-7" },
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
          params: {
            projectId: "p",
            planId: "1",
            stageId: "b",
            taskId: "t2",
          },
          query: { taskRunId: "run-7" },
        },
        { preventScrollReset: true }
      )
    );
  });

  test("falls back from an unknown legacy task to its valid stage", async () => {
    renderCanonical({
      routeQuery: { stageId: "a", taskId: "missing" },
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
          params: { projectId: "p", planId: "1", stageId: "a" },
          query: {},
        },
        { preventScrollReset: true }
      )
    );
  });

  test("keeps a committed rollout URL while rollout data is still pending", async () => {
    renderCanonical({
      routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      stageId: "a",
      taskId: "t1",
      snapshot: snapshot({ hasRollout: true, rollout: false }),
    });

    await waitFor(() => expect(mocks.replace).not.toHaveBeenCalled());
  });

  test("keeps legacy rollout selectors until an expected rollout is visible", async () => {
    renderCanonical({
      routeQuery: { stageId: "a", taskId: "t1" },
      snapshot: snapshot({ hasRollout: true, rollout: false }),
    });

    await waitFor(() => expect(mocks.replace).not.toHaveBeenCalled());
  });

  test("corrects a task path whose task belongs to another stage", async () => {
    renderCanonical({
      routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      routeHash: "#log",
      routeQuery: { taskRunId: "run-7" },
      stageId: "a",
      taskId: "t2",
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
          params: {
            projectId: "p",
            planId: "1",
            stageId: "b",
            taskId: "t2",
          },
          query: { taskRunId: "run-7" },
          hash: "#log",
        },
        { preventScrollReset: true }
      )
    );
  });

  test("falls back from an unknown task and stage to the rollout", async () => {
    renderCanonical({
      routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      stageId: "missing",
      taskId: "missing",
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
          params: { projectId: "p", planId: "1" },
        },
        { preventScrollReset: true }
      )
    );
  });

  test("falls back from an unknown task to its valid stage", async () => {
    renderCanonical({
      routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      stageId: "a",
      taskId: "missing",
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
          params: { projectId: "p", planId: "1", stageId: "a" },
        },
        { preventScrollReset: true }
      )
    );
  });

  test("falls back from an unknown stage to the rollout", async () => {
    renderCanonical({
      routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
      stageId: "missing",
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
          params: { projectId: "p", planId: "1" },
        },
        { preventScrollReset: true }
      )
    );
  });

  test("falls back from resources that do not exist yet to the plan root", async () => {
    renderCanonical({
      routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
      snapshot: snapshot({ rollout: false }),
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_DETAIL,
          params: { projectId: "p", planId: "1" },
          query: {},
        },
        { preventScrollReset: true }
      )
    );
  });

  test("falls back from an unknown spec to the plan root", async () => {
    renderCanonical({
      routeName: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
      specId: "missing",
      snapshot: snapshot({ specs: ["spec-1"] }),
    });

    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        {
          name: PROJECT_V1_ROUTE_PLAN_DETAIL,
          params: { projectId: "p", planId: "1" },
          query: {},
        },
        { preventScrollReset: true }
      )
    );
  });

  test("waits for the snapshot before rejecting an unknown spec path", async () => {
    const { rerender } = renderHook(
      ({ currentSnapshot }: { currentSnapshot: PlanDetailPageSnapshot }) =>
        useCanonicalPlanDetailRoute({
          projectId: "p",
          planId: "1",
          routeName: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
          routeQuery: {},
          specId: "missing",
          snapshot: currentSnapshot,
          isEditing: false,
          bypassLeaveGuardOnce: vi.fn(),
        }),
      { initialProps: { currentSnapshot: snapshot({ ready: false }) } }
    );

    expect(mocks.replace).not.toHaveBeenCalled();

    rerender({
      currentSnapshot: snapshot({ ready: true, specs: ["spec-1"] }),
    });

    await waitFor(() => expect(mocks.replace).toHaveBeenCalledTimes(1));
  });

  test("bypasses the leave guard exactly once for a canonical replacement", async () => {
    const bypassLeaveGuardOnce = vi.fn();
    renderCanonical({
      isEditing: true,
      bypassLeaveGuardOnce,
      routeName: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
      routeQuery: { phase: "deploy" },
      specId: "spec-1",
    });

    await waitFor(() => expect(mocks.replace).toHaveBeenCalledTimes(1));
    expect(bypassLeaveGuardOnce).toHaveBeenCalledTimes(1);
    expect(bypassLeaveGuardOnce.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.replace.mock.invocationCallOrder[0]
    );
  });
});
