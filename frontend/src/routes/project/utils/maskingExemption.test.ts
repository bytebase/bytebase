import type { TFunction } from "i18next";
import { describe, expect, test } from "vitest";
import {
  generateMemberScopeSummary,
  resourceCoversDatabase,
} from "./maskingExemption";

const t = ((key: string, options?: Record<string, unknown>) => {
  if (key === "database.all") return "All databases";
  if (key === "project.masking-exemption.n-exemptions") {
    const count = options?.count;
    return `${count} exemption${count === 1 ? "" : "s"}`;
  }
  if (key === "project.masking-exemption.n-databases") {
    const count = options?.count;
    return `${count} database${count === 1 ? "" : "s"}`;
  }
  return key;
}) as TFunction;

describe("generateMemberScopeSummary", () => {
  test("uses the active translator for the summary", () => {
    expect(generateMemberScopeSummary(t, 3, ["bytebase_210"])).toBe(
      "3 exemptions · bytebase_210"
    );
  });

  test("shows all databases when any grant has global scope", () => {
    expect(generateMemberScopeSummary(t, 3, ["", "bytebase_210"])).toBe(
      "3 exemptions · All databases"
    );
  });
});

describe("resourceCoversDatabase", () => {
  test("matches global and instance-scoped resources", () => {
    expect(
      resourceCoversDatabase(
        { databaseFullName: "", columns: ["email"] },
        "instances/prod/databases/hr"
      )
    ).toBe(true);
    expect(
      resourceCoversDatabase(
        { databaseFullName: "", instanceResourceId: "prod" },
        "instances/prod/databases/hr"
      )
    ).toBe(true);
    expect(
      resourceCoversDatabase(
        { databaseFullName: "", instanceResourceId: "prod" },
        "instances/dev/databases/hr"
      )
    ).toBe(false);
  });
});
