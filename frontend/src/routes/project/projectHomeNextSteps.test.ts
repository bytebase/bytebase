import { describe, expect, test } from "vitest";
import { getProjectHomeNextSteps } from "./projectHomeNextSteps";

const input = {
  database: "empty" as const,
  instance: "empty" as const,
  transferableDatabase: "empty" as const,
  change: "not-started" as const,
  membership: "solo" as const,
  isDefault: false,
  canConnectInstance: true,
  canTransferDatabases: true,
  canCreateDatabase: true,
  canCreateChange: true,
  canManageMembers: true,
};

describe("getProjectHomeNextSteps", () => {
  test("connects an instance first in a new project", () => {
    expect(getProjectHomeNextSteps(input)).toEqual(["connect", "members"]);
  });

  test("creates a database when a usable instance exists", () => {
    expect(getProjectHomeNextSteps({ ...input, instance: "present" })).toEqual([
      "createDatabase",
      "members",
    ]);
  });

  test("offers transfer first and creation second for unassigned databases", () => {
    expect(
      getProjectHomeNextSteps({
        ...input,
        instance: "present",
        transferableDatabase: "present",
      })
    ).toEqual(["transferDatabases", "createDatabase", "members"]);
  });

  test("waits for transfer discovery before choosing the primary action", () => {
    expect(
      getProjectHomeNextSteps({
        ...input,
        instance: "present",
        transferableDatabase: "loading",
      })
    ).toEqual([]);
  });

  test("falls back to creation when transfer is unavailable", () => {
    expect(
      getProjectHomeNextSteps({
        ...input,
        instance: "present",
        transferableDatabase: "present",
        canTransferDatabases: false,
      })
    ).toEqual(["createDatabase", "members"]);
    expect(
      getProjectHomeNextSteps({
        ...input,
        instance: "present",
        transferableDatabase: "unknown",
      })
    ).toEqual(["createDatabase", "members"]);
  });

  test("waits for sync or uncertain resource state", () => {
    expect(getProjectHomeNextSteps({ ...input, instance: "syncing" })).toEqual(
      []
    );
    expect(getProjectHomeNextSteps({ ...input, instance: "unknown" })).toEqual(
      []
    );
    expect(getProjectHomeNextSteps({ ...input, database: "unknown" })).toEqual(
      []
    );
  });

  test("offers a first database change and optional membership setup", () => {
    expect(getProjectHomeNextSteps({ ...input, database: "present" })).toEqual([
      "changeDatabase",
      "members",
    ]);
    expect(
      getProjectHomeNextSteps({
        ...input,
        database: "present",
        membership: "has-collaborators",
      })
    ).toEqual(["changeDatabase"]);
  });

  test("hides setup after the first submitted database change", () => {
    expect(
      getProjectHomeNextSteps({
        ...input,
        database: "present",
        change: "submitted",
      })
    ).toEqual([]);
  });

  test("does not show an action the member cannot take", () => {
    expect(
      getProjectHomeNextSteps({
        ...input,
        canConnectInstance: false,
        canManageMembers: false,
      })
    ).toEqual([]);
    expect(
      getProjectHomeNextSteps({
        ...input,
        database: "present",
        canCreateChange: false,
      })
    ).toEqual(["members"]);
  });

  test("does not show project setup on the default project", () => {
    expect(getProjectHomeNextSteps({ ...input, isDefault: true })).toEqual([]);
  });
});
