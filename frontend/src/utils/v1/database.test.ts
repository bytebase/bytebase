import { describe, expect, test } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { databaseV1UrlWithSuffix } from "./database";

describe("databaseV1UrlWithSuffix", () => {
  const database = {
    name: "projects/proj1/instances/inst1/databases/db1",
    project: "projects/proj1",
  } as Database;

  test("places descendant paths before the parent query", () => {
    expect(databaseV1UrlWithSuffix(database, "/revisions/7")).toBe(
      "/projects/proj1/instances/inst1/databases/db1/revisions/7?parent=projects%2Fproj1%2Finstances%2Finst1"
    );
  });
});
