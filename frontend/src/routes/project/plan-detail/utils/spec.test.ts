import { describe, expect, test } from "vitest";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { getSelectedSpec } from "./spec";

describe("getSelectedSpec", () => {
  const specs = [{ id: "a" }, { id: "b" }] as Plan_Spec[];

  test("returns matching spec", () => {
    expect(getSelectedSpec({ selectedSpecId: "b", specs })?.id).toBe("b");
  });

  test("falls back to first spec", () => {
    expect(getSelectedSpec({ selectedSpecId: "missing", specs })?.id).toBe("a");
  });

  test("returns undefined for empty specs", () => {
    expect(getSelectedSpec({ selectedSpecId: "missing", specs: [] })).toBe(
      undefined
    );
  });
});
