import { describe, expect, test } from "vitest";
import { getPlanStatusTone } from "./planStatusLabel";

describe("getPlanStatusTone", () => {
  test("checking / in-review are neutral", () => {
    expect(getPlanStatusTone("checking")).toBe("neutral");
    expect(getPlanStatusTone("in-review")).toBe("neutral");
  });

  test("rejected / checks-failing are errors (hard blockers)", () => {
    expect(getPlanStatusTone("rejected")).toBe("error");
    expect(getPlanStatusTone("checks-failing")).toBe("error");
  });
});
