import { describe, expect, test } from "vitest";
import { isReviewSubmitDisabled } from "./reviewAction";

describe("isReviewSubmitDisabled", () => {
  test("disables approval for the last plan editor without blocking rejection", () => {
    expect(
      isReviewSubmitDisabled({
        action: "APPROVE",
        canApprove: false,
        commentMissing: false,
        loading: false,
      })
    ).toBe(true);
    expect(
      isReviewSubmitDisabled({
        action: "REJECT",
        canApprove: false,
        commentMissing: false,
        loading: false,
      })
    ).toBe(false);
  });
});
