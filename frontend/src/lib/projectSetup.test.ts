import { describe, expect, test } from "vitest";
import { getProjectSetupSteps, type ProjectSetupState } from "./projectSetup";

const input = {
  isDefault: false,
  instance: "empty",
  projectInstance: "empty",
  transferableDatabase: "empty",
  canConnectInstance: true,
  canTransferDatabases: true,
  canCreateDatabase: true,
} as const;

describe("getProjectSetupSteps", () => {
  test("offers transfer first and connect second without a project instance", () => {
    expect(
      getProjectSetupSteps({ ...input, transferableDatabase: "present" })
    ).toEqual(["transferDatabases", "connect"]);
  });

  test("does not offer creation just because a workspace instance exists", () => {
    expect(getProjectSetupSteps({ ...input, instance: "present" })).toEqual([
      "connect",
    ]);
  });

  test("preserves creation for a project-owned instance", () => {
    expect(
      getProjectSetupSteps({
        ...input,
        instance: "present",
        projectInstance: "present",
        transferableDatabase: "present",
      })
    ).toEqual(["transferDatabases", "createDatabase"]);
  });

  test.each(["empty", "unknown"] as const)(
    "connects when transfer discovery is %s",
    (transferableDatabase) => {
      expect(getProjectSetupSteps({ ...input, transferableDatabase })).toEqual([
        "connect",
      ]);
    }
  );

  test("falls back to connect without transfer permission", () => {
    expect(
      getProjectSetupSteps({
        ...input,
        transferableDatabase: "present",
        canTransferDatabases: false,
      })
    ).toEqual(["connect"]);
  });

  test("only offers actions the member can take", () => {
    expect(
      getProjectSetupSteps({
        ...input,
        transferableDatabase: "present",
        canConnectInstance: false,
      })
    ).toEqual(["transferDatabases"]);
    expect(
      getProjectSetupSteps({ ...input, canConnectInstance: false })
    ).toEqual([]);
    expect(
      getProjectSetupSteps({
        ...input,
        instance: "present",
        projectInstance: "present",
        canCreateDatabase: false,
      })
    ).toEqual([]);
  });

  test.each(["loading", "unknown"] as const)(
    "does not infer project instance absence from %s",
    (projectInstance) => {
      expect(
        getProjectSetupSteps({ ...input, instance: "present", projectInstance })
      ).toEqual([]);
      expect(
        getProjectSetupSteps({
          ...input,
          instance: "present",
          projectInstance,
          transferableDatabase: "present",
        })
      ).toEqual(["transferDatabases"]);
    }
  );

  test.each([
    { ...input, isDefault: true },
    { ...input, transferableDatabase: "loading" },
  ] satisfies ProjectSetupState[])("waits or hides setup for %o", (state) => {
    expect(getProjectSetupSteps(state)).toEqual([]);
  });
});
