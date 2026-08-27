import { describe, expect, test } from "vitest";
import { parseStringToResource } from "./databaseResource";

describe("parseStringToResource", () => {
  test.each([
    [
      "instances/prod/databases/app/schemas/public/tables/users/columns/id",
      "instances/prod/databases/app",
    ],
    [
      "projects/acme/instances/prod/databases/app/schemas/public/tables/users/columns/id",
      "projects/acme/instances/prod/databases/app",
    ],
  ])("preserves the canonical database parent in %s", (name, database) => {
    expect(parseStringToResource(name)).toEqual({
      instanceResourceId: "prod",
      databaseResourceId: "app",
      databaseFullName: database,
      schema: "public",
      table: "users",
      columns: ["id"],
    });
  });

  test("rejects malformed project database names", () => {
    expect(
      parseStringToResource("projects/acme/databases/app")
    ).toBeUndefined();
  });
});
