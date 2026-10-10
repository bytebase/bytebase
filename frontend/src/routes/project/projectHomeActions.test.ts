import { describe, expect, test } from "vitest";
import type { Permission } from "@/types/iam";
import { storageKeyProjectHomeShortcuts } from "@/utils/storage-keys";
import {
  getProjectHomeActions,
  getProjectHomeAvailableActions,
} from "./projectHomeActions";
import {
  getVisibleProjectHomeShortcuts,
  readProjectHomeShortcuts,
  reorderProjectHomeShortcuts,
} from "./projectHomeShortcuts";

const actions = (
  permissions: Permission[],
  availability: "present" | "empty" | "unknown",
  isDefault = false
) =>
  getProjectHomeActions({
    availability,
    instanceAvailability: "empty",
    isDefault,
    accessGrantsAvailable: true,
    hasPermission: (permission) => permissions.includes(permission),
  });

describe("getProjectHomeActions", () => {
  test("offers creation and querying for a connected project without duplicating work queues", () => {
    expect(
      actions(
        [
          "bb.databases.list",
          "bb.plans.list",
          "bb.plans.create",
          "bb.projects.getIamPolicy",
          "bb.accessGrants.list",
        ],
        "present"
      )
    ).toEqual(["plans", "query"]);
  });

  test("keeps read-only plans without defaulting to Issues", () => {
    expect(
      actions(
        ["bb.databases.list", "bb.plans.list", "bb.issues.list"],
        "present"
      )
    ).toEqual(["plans"]);
  });

  test("links an empty project to project instances only when connection is allowed", () => {
    expect(
      actions(["bb.instances.list", "bb.instances.create"], "empty")
    ).toEqual(["instances"]);
    expect(actions(["bb.instances.list"], "empty")).toEqual(["instances"]);
    expect(
      actions(["bb.instances.list", "bb.instances.create"], "empty", true)
    ).toEqual([]);
  });

  test("suggests instance setup and membership for a brand-new project", () => {
    expect(
      actions(
        [
          "bb.instances.list",
          "bb.instances.create",
          "bb.projects.getIamPolicy",
          "bb.projects.setIamPolicy",
          "bb.issues.list",
          "bb.accessGrants.list",
        ],
        "empty"
      )
    ).toEqual(["instances", "members"]);
  });

  test("suggests database creation instead of connection when an instance exists", () => {
    expect(
      getProjectHomeActions({
        availability: "empty",
        instanceAvailability: "present",
        isDefault: false,
        accessGrantsAvailable: true,
        hasPermission: (permission) =>
          [
            "bb.databases.list",
            "bb.instances.list",
            "bb.issues.create",
            "bb.plans.create",
            "bb.sheets.create",
            "bb.projects.getIamPolicy",
            "bb.projects.setIamPolicy",
          ].includes(permission),
      })
    ).toEqual(["createDatabase", "members"]);
  });

  test("uses neutral instance browsing when database availability is unknown", () => {
    expect(
      actions(["bb.instances.list", "bb.instances.create"], "unknown")
    ).toEqual(["instances"]);
  });

  test("stays useful without any optional permission", () => {
    expect(actions(["bb.projects.get"], "unknown")).toEqual([]);
  });

  test("offers database browsing to a read-only member without other workflows", () => {
    expect(actions(["bb.databases.list"], "present")).toEqual(["databases"]);
    expect(actions(["bb.databases.list"], "present", true)).toEqual([
      "databases",
    ]);
  });

  test("does not suggest membership management without write permission", () => {
    expect(actions(["bb.projects.getIamPolicy"], "empty")).toEqual([]);
  });

  test("omits access grants when its licensed feature is unavailable", () => {
    expect(
      getProjectHomeAvailableActions({
        availability: "present",
        instanceAvailability: "unknown",
        isDefault: false,
        accessGrantsAvailable: false,
        hasPermission: (permission) => permission === "bb.accessGrants.list",
      })
    ).toEqual([]);
  });

  test("offers additional reachable destinations for customization", () => {
    expect(
      getProjectHomeAvailableActions({
        availability: "present",
        instanceAvailability: "unknown",
        isDefault: false,
        accessGrantsAvailable: true,
        hasPermission: (permission) =>
          [
            "bb.databases.list",
            "bb.plans.list",
            "bb.plans.create",
            "bb.issues.list",
            "bb.instances.list",
            "bb.projects.getIamPolicy",
            "bb.accessGrants.list",
          ].includes(permission),
      })
    ).toEqual([
      "plans",
      "query",
      "issues",
      "instances",
      "databases",
      "members",
      "access",
    ]);
  });

  test("offers GitOps in customization only when its route is reachable", () => {
    const input = {
      availability: "present" as const,
      instanceAvailability: "unknown" as const,
      isDefault: false,
      accessGrantsAvailable: true,
      hasPermission: (permission: Permission) =>
        ["bb.databases.list", "bb.workloadIdentities.list"].includes(
          permission
        ),
    };
    expect(getProjectHomeActions(input)).not.toContain("gitops");
    expect(getProjectHomeAvailableActions(input)).toContain("gitops");
    expect(
      getProjectHomeAvailableActions({
        ...input,
        hasPermission: (permission) => permission === "bb.databases.list",
      })
    ).not.toContain("gitops");
    expect(
      getProjectHomeAvailableActions({ ...input, isDefault: true })
    ).not.toContain("gitops");
  });

  test("does not duplicate a suggested Members shortcut in customization", () => {
    expect(
      getProjectHomeAvailableActions({
        availability: "unknown",
        instanceAvailability: "unknown",
        isDefault: false,
        accessGrantsAvailable: false,
        hasPermission: (permission) =>
          permission === "bb.projects.getIamPolicy",
      })
    ).toEqual(["members"]);
  });
});

describe("project Home shortcuts", () => {
  test("an explicit selection stays ordered while unavailable actions are hidden", () => {
    expect(
      getVisibleProjectHomeShortcuts(
        ["query", "access", "issues"],
        ["instances", "issues"],
        ["issues", "instances"]
      )
    ).toEqual(["issues"]);
    expect(
      getVisibleProjectHomeShortcuts([], ["instances"], ["instances"])
    ).toEqual([]);
    expect(
      getVisibleProjectHomeShortcuts(
        ["instances", "issues"],
        ["plans", "query"],
        ["plans", "query", "issues", "instances"]
      )
    ).toEqual(["instances", "issues"]);
  });

  test("reordering visible shortcuts retains hidden saved choices", () => {
    expect(
      reorderProjectHomeShortcuts(
        ["query", "access", "issues", "members"],
        ["query", "issues", "members"],
        "members",
        "query"
      )
    ).toEqual(["members", "access", "query", "issues"]);
  });

  test("reads known shortcut IDs and treats an explicit empty list as saved", () => {
    const storage = {
      getItem: (key: string) =>
        key === "empty" ? "[]" : '["query","removed","query","issues"]',
    };
    expect(readProjectHomeShortcuts(storage, "empty")).toEqual([]);
    expect(readProjectHomeShortcuts(storage, "saved")).toEqual([
      "query",
      "issues",
    ]);
  });

  test("retains a saved GitOps shortcut", () => {
    expect(
      readProjectHomeShortcuts({ getItem: () => '["gitops"]' }, "saved")
    ).toEqual(["gitops"]);
  });

  test("uses distinct browser keys for workspace, member, and project", () => {
    const key = storageKeyProjectHomeShortcuts(
      "workspaces/one",
      "alice@example.com",
      "projects/orders"
    );
    expect(key).not.toBe(
      storageKeyProjectHomeShortcuts(
        "workspaces/two",
        "alice@example.com",
        "projects/orders"
      )
    );
    expect(key).not.toBe(
      storageKeyProjectHomeShortcuts(
        "workspaces/one",
        "bob@example.com",
        "projects/orders"
      )
    );
    expect(key).not.toBe(
      storageKeyProjectHomeShortcuts(
        "workspaces/one",
        "alice@example.com",
        "projects/billing"
      )
    );
  });
});
