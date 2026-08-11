import { describe, expect, test } from "vitest";
import { PROJECT_V1_ROUTE_DATABASE_DETAIL } from "@/app/router/handles";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { autoDatabaseRoute } from "./auto-route";

describe("autoDatabaseRoute", () => {
  test("preserves the canonical project instance parent", () => {
    const database = {
      name: "projects/proj1/instances/inst1/databases/db1",
      project: "projects/proj1",
    } as Database;

    expect(autoDatabaseRoute(database)).toEqual({
      name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
      params: {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db1",
      },
      query: {
        parent: "projects/proj1/instances/inst1",
      },
    });
  });
});
