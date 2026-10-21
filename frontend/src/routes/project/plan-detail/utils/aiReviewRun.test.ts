import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  DatabaseGroupSchema,
  DatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_CreateDatabaseConfigSchema,
  type Plan_Spec,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { hasReviewableSQL, loadedMatchedDatabaseCount } from "./aiReviewRun";

const GROUP = "projects/p/databaseGroups/g";
const DATABASE = "instances/i/databases/d";

const change = (targets: string[], release = ""): Plan_Spec["config"] => ({
  case: "changeDatabaseConfig",
  value: create(Plan_ChangeDatabaseConfigSchema, { targets, release }),
});

const plan = (...configs: Plan_Spec["config"][]) =>
  create(PlanSchema, {
    specs: configs.map((config, i) => ({ id: `${i}`, config })),
  });

const groupMatching = (databases: string[]) =>
  loadedMatchedDatabaseCount({
    dbGroupsByName: {
      [GROUP]: create(DatabaseGroupSchema, {
        name: GROUP,
        matchedDatabases: databases.map((name) => ({ name })),
      }),
    },
    dbGroupViewByName: { [GROUP]: DatabaseGroupView.FULL },
  });

const notLoaded = loadedMatchedDatabaseCount({
  dbGroupsByName: {},
  dbGroupViewByName: {},
});

describe("hasReviewableSQL", () => {
  test("a change spec with database targets is reviewable", () => {
    expect(hasReviewableSQL(plan(change([DATABASE])), notLoaded)).toBe(true);
  });

  test("create-database and release-backed specs are not", () => {
    const createDatabase: Plan_Spec["config"] = {
      case: "createDatabaseConfig",
      value: create(Plan_CreateDatabaseConfigSchema, { database: "d" }),
    };
    expect(
      hasReviewableSQL(
        plan(createDatabase, change([DATABASE], "releases/r")),
        notLoaded
      )
    ).toBe(false);
  });

  test("a group that matches no databases is not", () => {
    expect(hasReviewableSQL(plan(change([GROUP])), groupMatching([]))).toBe(
      false
    );
  });

  test("a group with matched databases is", () => {
    expect(
      hasReviewableSQL(plan(change([GROUP])), groupMatching([DATABASE]))
    ).toBe(true);
  });

  test("a group not loaded yet counts as reviewable", () => {
    expect(hasReviewableSQL(plan(change([GROUP])), notLoaded)).toBe(true);
  });

  test("a group loaded without the full view counts as reviewable", () => {
    const basicView = loadedMatchedDatabaseCount({
      dbGroupsByName: { [GROUP]: create(DatabaseGroupSchema, { name: GROUP }) },
      dbGroupViewByName: { [GROUP]: DatabaseGroupView.BASIC },
    });
    expect(hasReviewableSQL(plan(change([GROUP])), basicView)).toBe(true);
  });

  test("an empty group beside a spec with a database is reviewable", () => {
    expect(
      hasReviewableSQL(
        plan(change([GROUP]), change([DATABASE])),
        groupMatching([])
      )
    ).toBe(true);
  });
});
