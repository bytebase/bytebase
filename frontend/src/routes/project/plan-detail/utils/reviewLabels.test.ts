import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({ sqlReviewV2: true }));

vi.mock("@/utils/featureGates", () => ({
  sqlReviewV2Enabled: () => mocks.sqlReviewV2,
}));

import { reviewerLabel, reviewerName } from "./reviewLabels";

const t = (key: string) => key;

beforeEach(() => {
  mocks.sqlReviewV2 = true;
});

describe("reviewerName", () => {
  test("names the one review under SQL Review V2", () => {
    expect(reviewerName(t)).toBe("sql-review.reviewer");
    expect(reviewerLabel(t, "review")).toBe("sql-review.reviewer");
  });

  test("names the AI review beside V1's plan checks", () => {
    mocks.sqlReviewV2 = false;
    expect(reviewerName(t)).toBe("sql-review.ai-review.self");
    expect(reviewerLabel(t, "review")).toBe("sql-review.ai-review.self");
  });
});

describe("reviewerLabel", () => {
  test.each([
    [true, "plan.automated-review.policy"],
    [false, "task.check-type.sql-review.self"],
  ])(
    "with SQL Review V2 %s, names the plan checks' SQL review advice %s",
    (v2, label) => {
      mocks.sqlReviewV2 = v2;
      expect(reviewerLabel(t, "policy")).toBe(label);
    }
  );
});
