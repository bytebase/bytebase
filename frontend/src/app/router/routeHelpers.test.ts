import { describe, expect, test } from "vitest";
import {
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
} from "./handles";
import { resolvePath, setRouteNameIndex } from "./navigation";
import {
  buildPlanCreateRoute,
  buildPlanDeployRouteFromRolloutName,
  buildPlanRolloutRouteFromPlanName,
  buildStageRoute,
  buildTaskDetailRoute,
} from "./routeHelpers";
import { buildRouteNameIndex } from "./routes";

describe("plan detail resource route helpers", () => {
  test("builds plan creation on the plan root without a placeholder spec", () => {
    const query = {
      databaseList: "projects/p/databases/db",
    };

    expect(buildPlanCreateRoute("p", query)).toEqual({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: { projectId: "p", planId: "create" },
      query,
    });
  });

  test("maps rollout, stage, and task resource names to resource paths", () => {
    const rollout = "projects/p/plans/1/rollout";
    const stage = `${rollout}/stages/prod`;
    const task = `${stage}/tasks/42`;

    expect(buildPlanDeployRouteFromRolloutName(rollout)).toEqual({
      name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
      params: { projectId: "p", planId: "1" },
    });
    expect(buildPlanRolloutRouteFromPlanName("projects/p/plans/1")).toEqual({
      name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
      params: { projectId: "p", planId: "1" },
    });
    expect(buildStageRoute(stage)).toEqual({
      name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
      params: { projectId: "p", planId: "1", stageId: "prod" },
    });
    expect(buildTaskDetailRoute(task)).toEqual({
      name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      params: {
        projectId: "p",
        planId: "1",
        stageId: "prod",
        taskId: "42",
      },
    });
  });

  test("resolves the canonical resource URL hierarchy", () => {
    setRouteNameIndex(buildRouteNameIndex());
    const params = {
      projectId: "p",
      planId: "1",
      stageId: "prod",
      taskId: "42",
    };

    expect(resolvePath(PROJECT_V1_ROUTE_PLAN_ROLLOUT, { params })).toBe(
      "/projects/p/plans/1/rollout"
    );
    expect(resolvePath(PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE, { params })).toBe(
      "/projects/p/plans/1/rollout/stages/prod"
    );
    expect(resolvePath(PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK, { params })).toBe(
      "/projects/p/plans/1/rollout/stages/prod/tasks/42"
    );
  });

  test("keeps rollout resources in the plan route-name family", () => {
    expect(PROJECT_V1_ROUTE_PLAN_ROLLOUT).toBe(
      `${PROJECT_V1_ROUTE_PLAN_DETAIL}.rollout`
    );
    expect(PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE).toBe(
      `${PROJECT_V1_ROUTE_PLAN_DETAIL}.rollout.stage`
    );
    expect(PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK).toBe(
      `${PROJECT_V1_ROUTE_PLAN_DETAIL}.rollout.stage.task`
    );
  });
});
