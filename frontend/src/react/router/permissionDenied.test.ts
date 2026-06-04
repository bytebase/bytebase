import { describe, expect, test } from "vitest";
import type { Permission } from "@/types";
import { buildPermissionDeniedRouteQuery } from "./permissionDenied";

describe("permission denied route query", () => {
  test("falls back to current route permissions", () => {
    expect(
      buildPermissionDeniedRouteQuery({
        route: {
          fullPath: "/projects/prod/issues/101",
          requiredPermissions: ["bb.issues.get"],
        },
      })
    ).toEqual({
      from: "/projects/prod/issues/101",
      api: "",
      permissions: "bb.issues.get",
      resources: "",
    });
  });

  test("prefers explicit API, permissions, and resources", () => {
    expect(
      buildPermissionDeniedRouteQuery({
        route: {
          fullPath: "/databases",
          requiredPermissions: ["bb.databases.list"] as Permission[],
        },
        api: "/bytebase.v1.DatabaseService/ListDatabases",
        permissions: ["bb.instances.get"],
        resources: ["instances/prod"],
      })
    ).toEqual({
      from: "/databases",
      api: "/bytebase.v1.DatabaseService/ListDatabases",
      permissions: "bb.instances.get",
      resources: "instances/prod",
    });
  });
});
