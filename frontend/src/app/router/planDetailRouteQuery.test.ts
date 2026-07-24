import { describe, expect, test } from "vitest";
import {
  buildPlanDetailLegacySearch,
  stripPlanDetailSelectionQuery,
} from "./planDetailRouteQuery";

describe("plan detail route query", () => {
  test("preserves a valid legacy phase without accepting a new phase", () => {
    expect(
      buildPlanDetailLegacySearch(
        "http://localhost/issues/1?phase=review&stageId=s1&foo=a&foo=b"
      )
    ).toBe("phase=review&foo=a&foo=b");
    expect(
      buildPlanDetailLegacySearch(
        "http://localhost/issues/1?phase=unknown&foo=a"
      )
    ).toBe("foo=a");
  });

  test("strips selection keys from object queries", () => {
    expect(
      stripPlanDetailSelectionQuery({
        phase: "deploy",
        specId: "spec-1",
        stageId: "s1",
        taskId: "t1",
        line: "42",
      })
    ).toEqual({ line: "42" });
  });
});
