import { describe, expect, test } from "vitest";
import {
  filterPlanTargets,
  getDatabaseGroupRouteParams,
  splitInlineDatabases,
} from "./targets";

describe("splitInlineDatabases", () => {
  test("splits database group children into inline and overflow lists", () => {
    expect(splitInlineDatabases(["a", "b", "c"], 2)).toEqual({
      extraDatabases: ["c"],
      inlineDatabases: ["a", "b"],
    });
  });

  test("filters database targets by display name and groups by resource name", () => {
    expect(
      filterPlanTargets({
        getDatabaseDisplayName: () => "orders",
        query: "ord",
        targets: [
          "projects/p/instances/i/databases/db1",
          "projects/p/databaseGroups/prod",
        ],
      })
    ).toEqual(["projects/p/instances/i/databases/db1"]);

    expect(
      filterPlanTargets({
        getDatabaseDisplayName: () => "orders",
        query: "prod",
        targets: [
          "projects/p/instances/i/databases/db1",
          "projects/p/databaseGroups/prod",
        ],
      })
    ).toEqual(["projects/p/databaseGroups/prod"]);
  });

  test("builds database group route params for project routes", () => {
    expect(
      getDatabaseGroupRouteParams({
        databaseGroupName: "prod",
        projectName: "projects/project-a",
      })
    ).toEqual({
      databaseGroupName: "prod",
      projectId: "project-a",
    });
  });
});
