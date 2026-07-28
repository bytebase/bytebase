import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { createPlanSkeleton } from "./createPlan";

const project = create(ProjectSchema, { name: "projects/p" });

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
});
