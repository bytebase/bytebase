import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test } from "vitest";
import { saveInitialSQL } from "@/lib/plan/initialSQLStorage";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { getSheetStatement } from "@/utils";
import { createPlanSkeleton } from "./createPlan";
import { getLocalSheetByName } from "./localSheet";

const project = create(ProjectSchema, { name: "projects/p" });

beforeEach(() => {
  localStorage.clear();
});

describe("createPlanSkeleton", () => {
  test("creates a change-database spec without a template query", async () => {
    const plan = await createPlanSkeleton(project, {
      databaseList: "projects/p/databases/db",
    });

    expect(plan.specs).toHaveLength(1);
    expect(plan.specs[0].config).toEqual({
      case: "changeDatabaseConfig",
      value: expect.objectContaining({
        targets: ["projects/p/databases/db"],
      }),
    });
  });

  test("ignores the legacy template query", async () => {
    const plan = await createPlanSkeleton(project, {
      template: "bb.plan.change-database",
      databaseList: "projects/p/databases/db",
    });

    expect(plan.specs[0].config.case).toBe("changeDatabaseConfig");
  });

  test("consumes initial SQL transferred through local storage", async () => {
    const sqlStorageKey = saveInitialSQL("ALTER TABLE users ADD active bool;");

    const plan = await createPlanSkeleton(project, {
      databaseList: "projects/p/databases/db",
      sqlStorageKey,
    });

    const spec = plan.specs[0];
    expect(spec.config.case).toBe("changeDatabaseConfig");
    if (spec.config.case !== "changeDatabaseConfig") return;
    expect(
      getSheetStatement(getLocalSheetByName(spec.config.value.sheet))
    ).toBe("ALTER TABLE users ADD active bool;");
    expect(localStorage.getItem(sqlStorageKey)).toBeNull();
  });

  test("consumes per-database SQL transferred through local storage", async () => {
    const sqlMapStorageKey = saveInitialSQL({
      "projects/p/databases/one": "ALTER TABLE one ADD active bool;",
      "projects/p/databases/two": "ALTER TABLE two ADD active bool;",
    });

    const plan = await createPlanSkeleton(project, {
      databaseList: "projects/p/databases/one,projects/p/databases/two",
      sqlMapStorageKey,
    });

    expect(
      plan.specs.map((spec) => {
        if (spec.config.case !== "changeDatabaseConfig") return "";
        return getSheetStatement(getLocalSheetByName(spec.config.value.sheet));
      })
    ).toEqual([
      "ALTER TABLE one ADD active bool;",
      "ALTER TABLE two ADD active bool;",
    ]);
    expect(localStorage.getItem(sqlMapStorageKey)).toBeNull();
  });
});
