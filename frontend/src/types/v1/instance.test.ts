import { describe, expect, test } from "vitest";
import { isValidInstanceName, UNKNOWN_INSTANCE_NAME } from "./instance";

describe("isValidInstanceName", () => {
  test("accepts workspace and project instance resource names", () => {
    expect(isValidInstanceName("instances/prod")).toBe(true);
    expect(isValidInstanceName("projects/app/instances/prod")).toBe(true);
  });

  test("rejects malformed, descendant, and unknown resource names", () => {
    expect(isValidInstanceName("projects/app/instances/")).toBe(false);
    expect(
      isValidInstanceName("projects/app/instances/prod/databases/main")
    ).toBe(false);
    expect(isValidInstanceName(UNKNOWN_INSTANCE_NAME)).toBe(false);
  });
});
