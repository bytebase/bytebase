import { describe, expect, test } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName } from "./database";

describe("extractDatabaseResourceName", () => {
  const database = {
    name: "projects/proj1/instances/inst1/databases/db1",
    project: "projects/proj1",
  } as Database;

  test("extracts the canonical database parent with the resource parts", () => {
    expect(extractDatabaseResourceName(database.name)).toMatchObject({
      parent: "projects/proj1/instances/inst1",
      database: "projects/proj1/instances/inst1/databases/db1",
      instanceName: "inst1",
      databaseName: "db1",
    });
  });

  test("extracts a workspace instance parent", () => {
    expect(
      extractDatabaseResourceName("instances/inst1/databases/db1")
    ).toMatchObject({
      parent: "instances/inst1",
      database: "instances/inst1/databases/db1",
      instanceName: "inst1",
      databaseName: "db1",
    });
  });
});
