import type { TFunction } from "i18next";
import { describe, expect, test } from "vitest";
import type { ExemptionGrant } from "@/lib/sensitive-data/types";
import {
  generateMemberScopeSummary,
  resourceCoversDatabase,
} from "./maskingExemption";

const t = ((key: string, options?: Record<string, unknown>) => {
  if (key === "database.all") return "All databases";
  if (key === "cel.condition.self") return "Condition";
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

const grants = (
  count: number,
  databaseResources: ExemptionGrant["databaseResources"]
) =>
  Array.from({ length: count }, () => ({
    databaseResources,
  })) as ExemptionGrant[];

describe("generateMemberScopeSummary", () => {
  test("uses the active translator for the summary", () => {
    expect(generateMemberScopeSummary(t, grants(3, []), ["bytebase_210"])).toBe(
      "3 exemptions · bytebase_210"
    );
  });

  test("shows all databases when any grant has global scope", () => {
    expect(
      generateMemberScopeSummary(t, grants(3, []), ["", "bytebase_210"])
    ).toBe("3 exemptions · All databases");
  });

  test("shows a condition when any grant cannot be summarized", () => {
    expect(
      generateMemberScopeSummary(t, grants(1, undefined), ["bytebase_210"])
    ).toBe("1 exemption · Condition");
  });
});

describe("resourceCoversDatabase", () => {
  test("matches a parsed database scope on a project instance", () => {
    expect(
      resourceCoversDatabase(
        {
          databaseFullName: "instances/prod/databases/hr",
          instanceResourceId: "prod",
          databaseResourceId: "hr",
        },
        "projects/payroll/instances/prod/databases/hr"
      )
    ).toBe(true);
  });

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
