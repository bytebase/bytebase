import { describe, expect, test } from "vitest";
import { normalizeInstanceName } from "./resourceName";

describe("normalizeInstanceName", () => {
  test("normalizes a bare ID as a workspace instance", () => {
    expect(normalizeInstanceName("prod")).toBe("instances/prod");
  });

  test.each(["instances/prod", "projects/hr-system/instances/prod"])(
    "preserves the canonical instance name %s",
    (name) => {
      expect(normalizeInstanceName(name)).toBe(name);
    }
  );
});
