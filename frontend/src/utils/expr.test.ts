import { describe, expect, test } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { getDatabaseFullNameOption, getDatabaseNameOptionConfig } from "./expr";

describe("getDatabaseNameOptionConfig", () => {
  test("uses the instance title and database name as the display label", () => {
    const database = {
      name: "instances/mysql-instance/databases/bytebase_210",
      instanceResource: { title: "MySQL instance" },
    } as Database;

    expect(getDatabaseFullNameOption(database)).toEqual({
      label: "MySQL instance (mysql-instance) / bytebase_210",
      value: database.name,
    });
  });

  test("falls back to parsed resource IDs when the database cannot be fetched", () => {
    const config = getDatabaseNameOptionConfig("projects/project-a");

    expect(
      config.fallback?.("instances/mysql-instance/databases/bytebase_210")
    ).toEqual({
      label: "mysql-instance / bytebase_210",
      value: "instances/mysql-instance/databases/bytebase_210",
    });
  });

  test("keeps an empty database condition blank", () => {
    const config = getDatabaseNameOptionConfig("projects/project-a");

    expect(config.fallback?.("")).toEqual({ label: "", value: "" });
  });
});
