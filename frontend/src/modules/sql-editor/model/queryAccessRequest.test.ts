import { describe, expect, test } from "vitest";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  getQueryAccessRequest,
  isQueryAccessGrantPermission,
} from "./queryAccessRequest";

describe("getQueryAccessRequest", () => {
  test.each([
    ["select", ["bb.sql.select"]],
    ["info", ["bb.sql.info"]],
    ["select and info", ["bb.sql.select", "bb.sql.info"]],
    ["empty permissions", []],
  ])(
    "prefers Access Grant for %s when both workflows are enabled",
    (_, permissions) => {
      expect(
        getQueryAccessRequest(
          { allowJustInTimeAccess: true, allowRequestRole: true },
          permissions
        )
      ).toEqual({
        kind: "accessGrant",
        requiredPermission: "bb.accessGrants.create",
        requiredFeature: PlanFeature.FEATURE_JIT,
      });
    }
  );

  test("offers Access Grant when role requests are disabled", () => {
    expect(
      getQueryAccessRequest(
        { allowJustInTimeAccess: true, allowRequestRole: false },
        ["bb.sql.select"]
      )
    ).toMatchObject({ kind: "accessGrant" });
  });

  test.each([
    ["data write", ["bb.sql.dml"]],
    ["mixed read and data write", ["bb.sql.select", "bb.sql.dml"]],
    ["schema write", ["bb.sql.ddl"]],
    ["database metadata", ["bb.databases.get"]],
  ])(
    "uses Request Role for %s when both workflows are enabled",
    (_, permissions) => {
      expect(
        getQueryAccessRequest(
          { allowJustInTimeAccess: true, allowRequestRole: true },
          permissions
        )
      ).toEqual({
        kind: "requestRole",
        requiredPermission: "bb.issues.create",
        requiredFeature: PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW,
      });
    }
  );

  test("uses Request Role for reads when JIT is disabled", () => {
    expect(
      getQueryAccessRequest(
        { allowJustInTimeAccess: false, allowRequestRole: true },
        ["bb.sql.select"]
      )
    ).toMatchObject({ kind: "requestRole" });
  });

  test.each([
    [undefined, ["bb.sql.select"]],
    [
      { allowJustInTimeAccess: false, allowRequestRole: false },
      ["bb.sql.select"],
    ],
    [{ allowJustInTimeAccess: true, allowRequestRole: false }, ["bb.sql.dml"]],
  ])("does not offer an unavailable workflow", (project, permissions) => {
    expect(getQueryAccessRequest(project, permissions)).toBeUndefined();
  });
});

describe("isQueryAccessGrantPermission", () => {
  test.each([
    ["bb.sql.select", true],
    ["bb.sql.info", true],
    ["bb.sql.dml", false],
    ["bb.sql.ddl", false],
    ["bb.sql.explain", false],
    ["bb.databases.get", false],
    ["", false],
  ])("classifies %s", (permission, expected) => {
    expect(isQueryAccessGrantPermission(permission)).toBe(expected);
  });
});
