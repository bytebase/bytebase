import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, test } from "vitest";
import {
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
} from "@/app/router/handles";
import { PlanDetailStoreProvider } from "../../shared/stores/PlanDetailStoreProvider";
import { usePlanDetailStore } from "../../shared/stores/usePlanDetailStore";
import { useRouteSelection } from "./useRouteSelection";

const wrapper = ({ children }: { children: ReactNode }) => (
  <PlanDetailStoreProvider>{children}</PlanDetailStoreProvider>
);

type RouteSelectionParams = Parameters<typeof useRouteSelection>[0];

const renderSelection = (initialProps: RouteSelectionParams) =>
  renderHook(
    (params: RouteSelectionParams) => {
      const route = useRouteSelection(params);
      const routePhase = usePlanDetailStore((state) => state.routePhase);
      const selectedSpecId = usePlanDetailStore(
        (state) => state.selectedSpecId
      );
      const selectedStageId = usePlanDetailStore(
        (state) => state.selectedStageId
      );
      const selectedTaskName = usePlanDetailStore(
        (state) => state.selectedTaskName
      );
      return {
        route,
        store: {
          routePhase,
          selectedSpecId,
          selectedStageId,
          selectedTaskName,
        },
      };
    },
    { initialProps, wrapper }
  );

describe("useRouteSelection", () => {
  test.each([
    {
      expected: {
        phase: "changes",
        selectionKey: "spec:",
        specId: undefined,
        stageId: undefined,
        taskId: undefined,
      },
      name: "spec collection",
      params: {
        routeName: PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
        routeQuery: {},
      },
    },
    {
      expected: {
        phase: "changes",
        selectionKey: "spec:spec-2",
        specId: "spec-2",
        stageId: undefined,
        taskId: undefined,
      },
      name: "specific spec",
      params: {
        routeName: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
        routeQuery: {},
        specId: "spec-2",
      },
    },
    {
      expected: {
        phase: "deploy",
        selectionKey: "rollout",
        specId: undefined,
        stageId: undefined,
        taskId: undefined,
      },
      name: "rollout",
      params: {
        routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
        routeQuery: {},
      },
    },
    {
      expected: {
        phase: "deploy",
        selectionKey: "stage:prod",
        specId: undefined,
        stageId: "prod",
        taskId: undefined,
      },
      name: "specific stage",
      params: {
        routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
        routeQuery: {},
        stageId: "prod",
      },
    },
    {
      expected: {
        phase: "deploy",
        selectionKey: "task:42",
        specId: undefined,
        stageId: "prod",
        taskId: "42",
      },
      name: "specific task",
      params: {
        routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
        routeQuery: {},
        stageId: "prod",
        taskId: "42",
      },
    },
  ] as const)(
    "derives the owning phase and identity for $name",
    async ({ expected, params }) => {
      const { result } = renderSelection(params);

      expect(result.current.route).toEqual(expected);
      await waitFor(() =>
        expect(result.current.store).toEqual({
          routePhase: expected.phase,
          selectedSpecId: expected.specId,
          selectedStageId: expected.stageId,
          selectedTaskName: expected.taskId,
        })
      );
    }
  );

  test.each(["changes", "review", "deploy"] as const)(
    "accepts the legacy %s phase only on the plan root",
    async (phase) => {
      const { result } = renderSelection({
        routeName: PROJECT_V1_ROUTE_PLAN_DETAIL,
        routeQuery: { phase },
      });

      expect(result.current.route).toEqual({
        phase,
        selectionKey: `plan:${phase}`,
        specId: undefined,
        stageId: undefined,
        taskId: undefined,
      });
      await waitFor(() =>
        expect(result.current.store.routePhase).toBe(phase)
      );
    }
  );

  test("ignores contradictory legacy selectors on a resource path", async () => {
    const { result } = renderSelection({
      routeName: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
      routeQuery: {
        phase: "deploy",
        specId: "query-spec",
        stageId: "prod",
        taskId: "42",
      },
      specId: "path-spec",
      stageId: "ignored-stage",
      taskId: "ignored-task",
    });

    expect(result.current.route).toEqual({
      phase: "changes",
      selectionKey: "spec:path-spec",
      specId: "path-spec",
      stageId: undefined,
      taskId: undefined,
    });
    await waitFor(() =>
      expect(result.current.store).toEqual({
        routePhase: "changes",
        selectedSpecId: "path-spec",
        selectedStageId: undefined,
        selectedTaskName: undefined,
      })
    );
  });

  test.each([
    {
      expected: {
        phase: "changes",
        selectionKey: "spec:spec-1",
        specId: "spec-1",
        stageId: undefined,
        taskId: undefined,
      },
      name: "spec wins over rollout selectors",
      routeQuery: {
        phase: "review",
        specId: "spec-1",
        stageId: "prod",
        taskId: "42",
      },
    },
    {
      expected: {
        phase: "deploy",
        selectionKey: "task:42",
        specId: undefined,
        stageId: "prod",
        taskId: "42",
      },
      name: "task wins over stage and phase",
      routeQuery: {
        phase: "review",
        stageId: "prod",
        taskId: "42",
      },
    },
    {
      expected: {
        phase: "deploy",
        selectionKey: "task:42",
        specId: undefined,
        stageId: undefined,
        taskId: "42",
      },
      name: "task remains identifiable without a legacy stage",
      routeQuery: { phase: "review", taskId: "42" },
    },
    {
      expected: {
        phase: "deploy",
        selectionKey: "stage:prod",
        specId: undefined,
        stageId: "prod",
        taskId: undefined,
      },
      name: "stage wins over phase",
      routeQuery: { phase: "review", stageId: "prod" },
    },
  ] as const)(
    "applies legacy selector precedence when $name",
    async ({ expected, routeQuery }) => {
      const { result } = renderSelection({
        routeName: PROJECT_V1_ROUTE_PLAN_DETAIL,
        routeQuery,
      });

      expect(result.current.route).toEqual(expected);
      await waitFor(() =>
        expect(result.current.store).toEqual({
          routePhase: expected.phase,
          selectedSpecId: expected.specId,
          selectedStageId: expected.stageId,
          selectedTaskName: expected.taskId,
        })
      );
    }
  );

  test("uses the task identity while canonicalizing its owning stage", async () => {
    const { result, rerender } = renderSelection({
      routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      routeQuery: {},
      stageId: "stale",
      taskId: "42",
    });

    expect(result.current.route.selectionKey).toBe("task:42");

    rerender({
      routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      routeQuery: {},
      stageId: "prod",
      taskId: "42",
    });

    expect(result.current.route.selectionKey).toBe("task:42");
    await waitFor(() =>
      expect(result.current.store).toEqual({
        routePhase: "deploy",
        selectedSpecId: undefined,
        selectedStageId: "prod",
        selectedTaskName: "42",
      })
    );
  });

  test("clears resource selection when returning to the plain plan root", async () => {
    const { result, rerender } = renderSelection({
      routeName: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      routeQuery: {},
      stageId: "prod",
      taskId: "42",
    });
    await waitFor(() =>
      expect(result.current.store.selectedTaskName).toBe("42")
    );

    rerender({
      routeName: PROJECT_V1_ROUTE_PLAN_DETAIL,
      routeQuery: { activity: "comment-7" },
    });

    expect(result.current.route).toEqual({
      phase: undefined,
      selectionKey: "plan:",
      specId: undefined,
      stageId: undefined,
      taskId: undefined,
    });
    await waitFor(() =>
      expect(result.current.store).toEqual({
        routePhase: undefined,
        selectedSpecId: undefined,
        selectedStageId: undefined,
        selectedTaskName: undefined,
      })
    );
  });
});
