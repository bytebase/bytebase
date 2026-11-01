// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_CreateDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  RolloutSchema,
  StageSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { routeAfterRolloutCreated } from "./rollout";

const changePlan = create(PlanSchema, {
  name: "projects/p1/plans/677",
  specs: [
    create(Plan_SpecSchema, {
      config: {
        case: "changeDatabaseConfig",
        value: create(Plan_ChangeDatabaseConfigSchema, {
          targets: ["instances/i1/databases/db1"],
        }),
      },
    }),
  ],
});

const createDatabasePlan = create(PlanSchema, {
  name: "projects/p1/plans/677",
  specs: [
    create(Plan_SpecSchema, {
      config: {
        case: "createDatabaseConfig",
        value: create(Plan_CreateDatabaseConfigSchema, {
          target: "instances/i1",
          database: "db1",
        }),
      },
    }),
  ],
});

const rollout = create(RolloutSchema, {
  name: "projects/p1/plans/677/rollout",
  stages: [
    create(StageSchema, { name: "projects/p1/plans/677/rollout/stages/prod" }),
  ],
});

describe("routeAfterRolloutCreated", () => {
  test("a change plan moves on to its rollout on Plan Detail", () => {
    expect(routeAfterRolloutCreated(changePlan, rollout)).toBeDefined();
  });

  test("a create-database plan stays on Issue Detail", () => {
    expect(
      routeAfterRolloutCreated(createDatabasePlan, rollout)
    ).toBeUndefined();
  });

  test("a rollout without stages has nowhere to go", () => {
    expect(
      routeAfterRolloutCreated(
        changePlan,
        create(RolloutSchema, { name: rollout.name })
      )
    ).toBeUndefined();
  });
});
