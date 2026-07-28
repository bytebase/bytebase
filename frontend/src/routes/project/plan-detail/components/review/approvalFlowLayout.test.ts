import { describe, expect, test } from "vitest";
import type { ApprovalStepStatus } from "./approvalFlowLayout";
import {
  computeApprovalFlowLayout,
  VERTICAL_BREAKPOINT_PX,
} from "./approvalFlowLayout";

const statuses = (s: string): ApprovalStepStatus[] =>
  s
    .split("")
    .map((c) =>
      c === "a"
        ? "approved"
        : c === "c"
          ? "current"
          : c === "r"
            ? "rejected"
            : "pending"
    );

describe("computeApprovalFlowLayout", () => {
  test("narrow container switches to vertical", () => {
    const layout = computeApprovalFlowLayout(statuses("aacpp"), 400);
    expect(layout.kind).toBe("vertical");
  });

  test("everything fits: nothing folds", () => {
    const layout = computeApprovalFlowLayout(statuses("aaacppp"), 2000);
    expect(layout).toEqual({
      kind: "horizontal",
      foldedApproved: 0,
      namedPending: 3,
    });
  });

  test("approved folds first; nearest pending stays named", () => {
    // 3 approved + current + 3 pending at ~900px: approved chip + current +
    // 1 named pending + pending chip (per the mockup's middle row).
    const layout = computeApprovalFlowLayout(statuses("aaacppp"), 900);
    expect(layout.kind).toBe("horizontal");
    if (layout.kind !== "horizontal") return;
    expect(layout.foldedApproved).toBe(3);
    expect(layout.namedPending).toBeGreaterThanOrEqual(1);
    expect(layout.namedPending).toBeLessThan(3);
  });

  test("minimum form: chip + current + chip", () => {
    const layout = computeApprovalFlowLayout(statuses("aaacppp"), 640);
    expect(layout).toEqual({
      kind: "horizontal",
      foldedApproved: 3,
      namedPending: 0,
    });
  });

  test("rejected step is the anchor like current", () => {
    const layout = computeApprovalFlowLayout(statuses("aarpp"), 640);
    expect(layout).toEqual({
      kind: "horizontal",
      foldedApproved: 2,
      namedPending: 0,
    });
  });

  test("fully approved: names as many nodes as fit, folding only the rest", () => {
    // 7 approved, no anchor. Wide enough for all 7 named.
    expect(computeApprovalFlowLayout(statuses("aaaaaaa"), 2000)).toEqual({
      kind: "horizontal",
      foldedApproved: 0,
      namedPending: 0,
    });
    // Medium width: partial fold (some named approved + a smaller chip), not
    // an all-or-nothing "7 approved" chip.
    const mid = computeApprovalFlowLayout(statuses("aaaaaaa"), 1200);
    expect(mid.kind).toBe("horizontal");
    if (mid.kind !== "horizontal") return;
    expect(mid.foldedApproved).toBeGreaterThan(0);
    expect(mid.foldedApproved).toBeLessThan(7);
  });

  test("no approved steps: no approved chip cost, pending folds only as needed", () => {
    const layout = computeApprovalFlowLayout(statuses("cpppp"), 2000);
    expect(layout).toEqual({
      kind: "horizontal",
      foldedApproved: 0,
      namedPending: 4,
    });
  });

  test("exports the vertical breakpoint", () => {
    expect(VERTICAL_BREAKPOINT_PX).toBeGreaterThan(0);
  });
});
